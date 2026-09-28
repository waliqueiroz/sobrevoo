package application

import (
	"context"
	"errors"
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -destination mockapplication/frame_service.go -package mockapplication . FrameService

// FrameService draws the frames of a flight (the fifth stage): the images of
// what the camera of a camera plan sees of the geo data slice made for it. Like
// every service, it groups all the operations on one resource and only
// orchestrates: how a frame is drawn — the camera, the terrain, the map, the
// marks of what is missing, the trail and the marker — and what to do about the
// frames a directory already holds live in internal/domain; decoding tiles and
// keeping the images are behind the domain.TileDecoder, domain.FrameRepository
// and domain.FrameExporter ports.
type FrameService interface {
	// DrawFrame draws the frame request.Number of plan into the file
	// request.Path.
	DrawFrame(ctx context.Context, plan domain.CameraPlan, slice domain.GeoSlice, request domain.SingleFrameRequest) (domain.RenderSummary, error)

	// DrawFrames draws the frames of plan that request.Directory does not have
	// yet — all of them with request.Overwrite —, calling progress after each.
	// The summary says what was done even when the drawing stopped early, with
	// ErrRenderInterrupted or another error.
	DrawFrames(ctx context.Context, plan domain.CameraPlan, slice domain.GeoSlice, request domain.FrameSetRequest, progress func(domain.RenderProgress)) (domain.RenderSummary, error)
}

type frameService struct {
	decoder    domain.TileDecoder
	repository domain.FrameRepository
	exporter   domain.FrameExporter

	// renderTuning holds the constants of drawing and sliceTuning those of the
	// slice a plan needs. Both are resolved by an outbound configuration
	// adapter and injected by whoever assembles the service (Constitution
	// Principle VIII).
	renderTuning domain.RenderTuning
	sliceTuning  domain.SliceTuning
}

// NewFrameService creates a FrameService backed by the given ports.
func NewFrameService(
	decoder domain.TileDecoder,
	repository domain.FrameRepository,
	exporter domain.FrameExporter,
	renderTuning domain.RenderTuning,
	sliceTuning domain.SliceTuning,
) FrameService {
	return &frameService{
		decoder:      decoder,
		repository:   repository,
		exporter:     exporter,
		renderTuning: renderTuning,
		sliceTuning:  sliceTuning,
	}
}

func (s *frameService) DrawFrame(ctx context.Context, plan domain.CameraPlan, slice domain.GeoSlice, request domain.SingleFrameRequest) (domain.RenderSummary, error) {
	started := time.Now()
	summary := domain.RenderSummary{Requested: 1, Resolution: request.Resolution}
	finish := func(err error) (domain.RenderSummary, error) {
		summary.Elapsed = time.Since(started)
		return summary, err
	}

	if err := s.check(plan, slice); err != nil {
		return finish(err)
	}

	scene, err := domain.NewScene(slice, s.decoder, s.renderTuning, request.Appearance)
	if err != nil {
		return finish(err)
	}

	image, stats, err := scene.Render(ctx, plan, request.Number, request.Resolution)
	if err != nil {
		return finish(interruption(ctx, &summary, err))
	}

	mark := domain.NewFrameMark(plan, slice, request.Resolution, s.renderTuning, request.Appearance)
	if err := s.exporter.Export(image, mark, request.Path, request.Overwrite); err != nil {
		return finish(err)
	}

	summary.Add(stats)
	return finish(nil)
}

func (s *frameService) DrawFrames(ctx context.Context, plan domain.CameraPlan, slice domain.GeoSlice, request domain.FrameSetRequest, progress func(domain.RenderProgress)) (domain.RenderSummary, error) {
	started := time.Now()
	summary := domain.RenderSummary{Requested: len(plan.Frames), Resolution: request.Resolution}
	finish := func(err error) (domain.RenderSummary, error) {
		summary.Elapsed = time.Since(started)
		return summary, err
	}

	if err := s.check(plan, slice); err != nil {
		return finish(err)
	}

	directory, err := s.repository.Inspect(request.Directory, request.Resolution)
	if err != nil {
		return finish(err)
	}

	mark := domain.NewFrameMark(plan, slice, request.Resolution, s.renderTuning, request.Appearance)
	work, err := directory.Plan(mark.SetID, len(plan.Frames), request.Overwrite)
	if err != nil {
		return finish(err)
	}
	summary.Kept = len(work.Keep)

	// The frames of a previous set that this plan has no number for would be
	// joined into the video after the frames of this one: they go first.
	if len(work.Remove) > 0 {
		if err := s.repository.Remove(request.Directory, work.Remove); err != nil {
			return finish(err)
		}
		summary.Removed = len(work.Remove)
	}

	if len(work.Draw) == 0 {
		return finish(nil)
	}
	scene, err := domain.NewScene(slice, s.decoder, s.renderTuning, request.Appearance)
	if err != nil {
		return finish(err)
	}

	for _, index := range work.Draw {
		image, stats, err := scene.Render(ctx, plan, index, request.Resolution)
		if err != nil {
			return finish(interruption(ctx, &summary, err))
		}
		if err := s.repository.Save(request.Directory, index, mark, image); err != nil {
			return finish(err)
		}

		summary.Add(stats)
		if progress != nil {
			progress(domain.RenderProgress{Done: summary.Kept + summary.Drawn, Total: summary.Requested, Elapsed: time.Since(started)})
		}
	}
	return finish(nil)
}

// check refuses, before anything is read or drawn, a slice that cannot be drawn
// for the plan, from the cheapest question to the dearest: whether it was made
// from this plan, whether its area holds what the plan needs, whether its tiles
// are images, whether it has any elevation at all.
func (s *frameService) check(plan domain.CameraPlan, slice domain.GeoSlice) error {
	if err := slice.EnsureMatches(plan); err != nil {
		return err
	}
	if err := slice.EnsureCovers(plan, s.sliceTuning); err != nil {
		return err
	}
	return slice.EnsureDrawable()
}

// interruption turns the error of a drawing that was stopped because the context
// is done into ErrRenderInterrupted, and marks the summary; any other error is
// left as it is.
func interruption(ctx context.Context, summary *domain.RenderSummary, err error) error {
	if ctx.Err() != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		summary.Interrupted = true
		return domain.ErrRenderInterrupted
	}
	return err
}
