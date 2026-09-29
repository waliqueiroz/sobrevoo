package application

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -destination mockapplication/flight_service.go -package mockapplication . FlightService

// FlightService runs the whole flight of the Sobrevoo pipeline in a single
// call (the seventh stage): it treats a track, plans the camera, gathers the
// geo data slice, draws the frames and encodes the video, delegating each
// stage entirely to the service that already implements it —
// CameraPlanService, GeoSliceService, FrameService and VideoService. Like
// every service, it only orchestrates: no business rule of its own is
// decided here (see specs/007-full-flight-pipeline/research.md).
type FlightService interface {
	// Fly runs the whole flight: it treats reader's track, plans the camera,
	// gathers the geo data slice, draws the frames and encodes the video
	// described by request, reusing whatever of the plan/slice/frames under
	// request.Keep is still valid for the same track and the same values.
	// progress is called as the run enters each stage and as the frame
	// rendering and video encoding stages report their own progress (may be
	// nil). The summary says what was done even when the run stopped early,
	// with ErrFlightInterrupted or another error.
	Fly(ctx context.Context, reader io.Reader, request domain.FlightRequest, progress func(domain.FlightProgress)) (domain.FlightSummary, error)
}

type flightService struct {
	cameraPlanService CameraPlanService
	geoSliceService   GeoSliceService
	frameService      FrameService
	videoService      VideoService
	workspace         domain.Workspace
}

// NewFlightService creates a FlightService backed by the given services and
// workspace.
func NewFlightService(
	cameraPlanService CameraPlanService,
	geoSliceService GeoSliceService,
	frameService FrameService,
	videoService VideoService,
	workspace domain.Workspace,
) FlightService {
	return &flightService{
		cameraPlanService: cameraPlanService,
		geoSliceService:   geoSliceService,
		frameService:      frameService,
		videoService:      videoService,
		workspace:         workspace,
	}
}

func (s *flightService) Fly(ctx context.Context, reader io.Reader, request domain.FlightRequest, progress func(domain.FlightProgress)) (domain.FlightSummary, error) {
	started := time.Now()
	var summary domain.FlightSummary
	finish := func(err error) (domain.FlightSummary, error) {
		summary.Elapsed = time.Since(started)
		summary.Interrupted = errors.Is(err, domain.ErrFlightInterrupted)
		return summary, err
	}
	report := func(stage domain.FlightStage) {
		if progress != nil {
			progress(domain.FlightProgress{Stage: stage})
		}
	}
	// reported, unlike report, is used for the two stages whose announcement
	// says whether they were reused — known only once the stage's own work
	// (or its reuse decision) is done, not before.
	reported := func(stage domain.FlightStage, reused bool) {
		if progress != nil {
			progress(domain.FlightProgress{Stage: stage, Reused: reused})
		}
	}

	// The cheapest checks, needing no information from the track, go first
	// (FR-005, FR-007): the video destination, then the encoder.
	if err := s.videoService.CheckDestination(request.Output, request.Overwrite); err != nil {
		return finish(err)
	}
	if _, err := s.videoService.CheckEncoder(ctx); err != nil {
		return finish(flightInterruption(err))
	}

	report(domain.StageTrackProcessing)
	plan, err := s.cameraPlanService.Generate(reader, request.Parameters)
	if err != nil {
		return finish(err)
	}
	summary.Completed = append(summary.Completed, domain.StageTrackProcessing)

	if request.Keep != "" {
		// The kept directory may not exist yet (a first run with --keep, or a
		// path the user has not used before) — the same way "render all"
		// creates its --output directory when it does not exist.
		if err := s.workspace.EnsureDirectory(request.Keep); err != nil {
			return finish(err)
		}
		if err := s.reusePlan(plan, filepath.Join(request.Keep, "plan.json"), request.Overwrite, &summary); err != nil {
			return finish(err)
		}
	}
	reported(domain.StageCameraPlanning, summary.PlanReused)
	summary.Completed = append(summary.Completed, domain.StageCameraPlanning)

	var slice domain.GeoSlice
	if request.Keep != "" {
		slice, err = s.reuseSlice(plan, filepath.Join(request.Keep, "slice.zip"), request.Overwrite, &summary)
	} else {
		slice, err = s.geoSliceService.Generate(plan)
	}
	if err != nil {
		return finish(err)
	}
	reported(domain.StageGeoDataSlicing, summary.SliceReused)
	summary.Completed = append(summary.Completed, domain.StageGeoDataSlicing)

	// A kept directory needs no workspace of its own: the frames simply live
	// in its "frames" subdirectory, which FrameService creates as needed —
	// the same way "render all" already creates its --output directory.
	framesDirectory := filepath.Join(request.Keep, "frames")
	cleanup := func() error { return nil }
	if request.Keep == "" {
		framesDirectory, cleanup, err = s.workspace.NewTemporary()
		if err != nil {
			return finish(err)
		}
	}
	defer cleanup()
	summary.FramesDirectory = framesDirectory

	report(domain.StageFrameRendering)
	renderSummary, err := s.frameService.DrawFrames(ctx, plan, slice, domain.FrameSetRequest{
		Directory:  framesDirectory,
		Resolution: request.Resolution,
		Appearance: request.Appearance,
		Overlay:    request.Overlay,
		Overwrite:  request.Overwrite,
	}, func(p domain.RenderProgress) {
		if progress != nil {
			progress(domain.FlightProgress{Stage: domain.StageFrameRendering, Render: &p})
		}
	})
	summary.Render = renderSummary
	if err != nil {
		return finish(flightInterruption(err))
	}
	summary.Completed = append(summary.Completed, domain.StageFrameRendering)

	report(domain.StageVideoEncoding)
	videoSummary, err := s.videoService.Assemble(ctx, plan, domain.VideoRequest{
		Directory: framesDirectory,
		Output:    request.Output,
		Quality:   request.Quality,
		Overwrite: request.Overwrite,
	}, func(p domain.VideoProgress) {
		if progress != nil {
			progress(domain.FlightProgress{Stage: domain.StageVideoEncoding, Video: &p})
		}
	})
	summary.Video = videoSummary
	if err != nil {
		return finish(flightInterruption(err))
	}
	summary.Completed = append(summary.Completed, domain.StageVideoEncoding)

	return finish(nil)
}

// reusePlan reuses the plan already at planPath when it is, by its content
// identification (CameraPlan.ID()), the one this run just computed — the
// same identity the plan's own file already carries, so no new comparison
// mechanism is invented (research.md item 4). Otherwise it exports plan to
// planPath, which already refuses an existing, different plan unless
// overwrite (ErrPlanDestinationExists) — the same "another set" protection
// FR-010b asks for, with no rule of its own.
func (s *flightService) reusePlan(plan domain.CameraPlan, planPath string, overwrite bool, summary *domain.FlightSummary) error {
	if existing, err := s.cameraPlanService.Load(planPath); err == nil && existing.ID() == plan.ID() {
		summary.PlanReused = true
		return nil
	}
	return s.cameraPlanService.Export(plan, planPath, overwrite)
}

// reuseSlice reuses the slice already at slicePath when it still matches
// plan (GeoSlice.EnsureMatches, the same check FrameService already makes
// before drawing). Otherwise it generates a fresh slice — which is where the
// coverage of the registered geo data is verified, as the first thing
// GeoSliceService.Generate does (research.md item 3) — and exports it to
// slicePath, with the same "another set" protection as reusePlan.
func (s *flightService) reuseSlice(plan domain.CameraPlan, slicePath string, overwrite bool, summary *domain.FlightSummary) (domain.GeoSlice, error) {
	if existing, err := s.geoSliceService.Load(slicePath); err == nil && existing.EnsureMatches(plan) == nil {
		summary.SliceReused = true
		return existing, nil
	}

	slice, err := s.geoSliceService.Generate(plan)
	if err != nil {
		return domain.GeoSlice{}, err
	}
	if err := s.geoSliceService.Export(slice, slicePath, overwrite); err != nil {
		return domain.GeoSlice{}, err
	}

	// A freshly generated slice has no ContentID — only Load, reading it back
	// from the file, sets it (GeoSlice.ContentID's own contract) — and
	// FrameMark.SetID is computed from it. Drawing with the freshly generated
	// slice would give the frames a SetID a later run's *reused* (loaded)
	// slice could never match, breaking reuse for this run's own frames
	// forever. Reading back what was just written keeps the identity the
	// same one a later run will compute.
	return s.geoSliceService.Load(slicePath)
}

// flightInterruption turns the interruption of a stage this run called
// (ErrRenderInterrupted, ErrVideoInterrupted, or the context simply being
// done) into ErrFlightInterrupted — the single command always exits with its
// own code for an interruption, never the stage's (FR-011). Any other error
// is left as it is.
func flightInterruption(err error) error {
	if errors.Is(err, domain.ErrRenderInterrupted) || errors.Is(err, domain.ErrVideoInterrupted) ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return domain.ErrFlightInterrupted
	}
	return err
}
