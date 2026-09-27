package application

import (
	"context"
	"errors"
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -destination mockapplication/video_service.go -package mockapplication . VideoService

// VideoService assembles the video of a flight (the sixth stage): it joins the
// frames the fifth stage drew, in the order and at the frame rate of the camera
// plan, into one file. Like every service, it groups all the operations on one
// resource and only orchestrates: what makes the frames the ones of the plan, and
// how long the video lasts, live in internal/domain; the frames in a directory,
// the encoder and the file the video is kept in are behind the
// domain.FrameRepository, domain.VideoEncoder and domain.VideoExporter ports.
type VideoService interface {
	// Assemble joins the frames in request.Directory into the video of plan,
	// written to request.Output, calling progress as frames are encoded (progress
	// may be nil). The summary says what was done even when the run stopped early,
	// with ErrVideoInterrupted or another error.
	Assemble(ctx context.Context, plan domain.CameraPlan, request domain.VideoRequest, progress func(domain.VideoProgress)) (domain.VideoSummary, error)

	// CheckEncoder probes the video encoder the same way Assemble does before
	// encoding, and returns what it found.
	CheckEncoder(ctx context.Context) (domain.EncoderInfo, error)

	// CheckDestination checks the video destination the same way Assemble does
	// before encoding: refuses an existing file unless overwrite.
	CheckDestination(output string, overwrite bool) error
}

type videoService struct {
	repository domain.FrameRepository
	encoder    domain.VideoEncoder
	exporter   domain.VideoExporter
}

// NewVideoService creates a VideoService backed by the given ports.
func NewVideoService(
	repository domain.FrameRepository,
	encoder domain.VideoEncoder,
	exporter domain.VideoExporter,
) VideoService {
	return &videoService{
		repository: repository,
		encoder:    encoder,
		exporter:   exporter,
	}
}

func (s *videoService) Assemble(ctx context.Context, plan domain.CameraPlan, request domain.VideoRequest, progress func(domain.VideoProgress)) (domain.VideoSummary, error) {
	started := time.Now()
	summary := domain.VideoSummary{
		Frames:    len(plan.Frames),
		FrameRate: plan.Parameters.FrameRate,
		Quality:   request.Quality,
	}
	finish := func(err error) (domain.VideoSummary, error) {
		summary.Elapsed = time.Since(started)
		summary.Interrupted = errors.Is(err, domain.ErrVideoInterrupted)
		return summary, err
	}

	directory, err := s.repository.List(request.Directory)
	if err != nil {
		return finish(err)
	}
	resolution, err := directory.Verify(plan)
	if err != nil {
		return finish(err)
	}
	summary.Resolution = resolution

	if err := s.CheckDestination(request.Output, request.Overwrite); err != nil {
		return finish(err)
	}

	info, err := s.CheckEncoder(ctx)
	if err != nil {
		return finish(err)
	}
	summary.Encoder = info

	report := func(encoded int) {
		summary.Encoded = min(encoded, summary.Frames)
		if progress != nil {
			progress(domain.VideoProgress{Done: summary.Encoded, Total: summary.Frames, Elapsed: time.Since(started)})
		}
	}

	size, err := s.exporter.Export(request.Output, request.Overwrite, func(temporary string) error {
		return s.encoder.Encode(ctx, domain.EncodeJob{
			Directory: request.Directory,
			Frames:    len(plan.Frames),
			FrameRate: plan.Parameters.FrameRate,
			Quality:   request.Quality,
			Output:    temporary,
		}, report)
	})
	if err != nil {
		return finish(videoInterruption(ctx, err))
	}
	summary.SizeBytes = size

	// The video is whole: every frame is in it, whatever the encoder last said.
	report(summary.Frames)
	return finish(nil)
}

func (s *videoService) CheckEncoder(ctx context.Context) (domain.EncoderInfo, error) {
	info, err := s.encoder.Probe(ctx)
	if err != nil {
		return domain.EncoderInfo{}, videoInterruption(ctx, err)
	}
	return info, nil
}

func (s *videoService) CheckDestination(output string, overwrite bool) error {
	return s.exporter.Check(output, overwrite)
}

// videoInterruption turns the error of an operation that stopped because the
// context is done into ErrVideoInterrupted; any other error is left as it is.
func videoInterruption(ctx context.Context, err error) error {
	if ctx.Err() != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		return domain.ErrVideoInterrupted
	}
	return err
}
