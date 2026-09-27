package application_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mockdomain"
)

type videoMocks struct {
	repository *mockdomain.MockFrameRepository
	encoder    *mockdomain.MockVideoEncoder
	exporter   *mockdomain.MockVideoExporter
	service    application.VideoService
}

func newVideoMocks(t *testing.T) videoMocks {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	m := videoMocks{
		repository: mockdomain.NewMockFrameRepository(mockCtrl),
		encoder:    mockdomain.NewMockVideoEncoder(mockCtrl),
		exporter:   mockdomain.NewMockVideoExporter(mockCtrl),
	}
	m.service = application.NewVideoService(m.repository, m.encoder, m.exporter)
	return m
}

var videoResolution = domain.Resolution{Width: 360, Height: 640}

var anEncoder = domain.EncoderInfo{Name: "ffmpeg", Version: "7.1", Codec: "libx264"}

// framesOf is the directory of the frames of plan, all whole, 360 × 640, of one
// set and drawn from the plan.
func framesOf(plan domain.CameraPlan) domain.FrameDirectory {
	return builddomain.NewFrameDirectoryBuilder().
		WithResolution(videoResolution).
		WithPlanID(plan.ID()).
		WithOursFrames(0, len(plan.Frames)-1, "the-set").
		Build()
}

// videoRequest is a request for the video of the frames in /tmp/frames.
func videoRequest() domain.VideoRequest {
	return domain.VideoRequest{Directory: "/tmp/frames", Output: "/tmp/flight.mp4", Quality: domain.VideoQualityMedium}
}

// exports makes the exporter run what it is asked to produce, on a temporary
// path, and say the file it published has size bytes.
func (m videoMocks) exports(size int64) *gomock.Call {
	return m.exporter.EXPECT().Export("/tmp/flight.mp4", false, gomock.Any()).
		DoAndReturn(func(_ string, _ bool, produce func(string) error) (int64, error) {
			if err := produce("/tmp/.sobrevoo-1.tmp"); err != nil {
				return 0, err
			}
			return size, nil
		})
}

func Test_videoService_Assemble(t *testing.T) {
	plan := framesPlan(380)

	t.Run("should list the frames, check the destination, probe the encoder and export the video, in that order", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		gomock.InOrder(
			m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil),
			m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil),
			m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil),
			m.exports(1234),
		)
		m.encoder.EXPECT().Encode(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		require.NoError(t, err)
	})

	t.Run("should ask the encoder for the frames of the plan, at its frame rate and the quality asked, to the temporary file", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		request := videoRequest()
		request.Quality = domain.VideoQualityHigh
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil)
		m.exports(1234)
		var asked domain.EncodeJob
		m.encoder.EXPECT().Encode(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, job domain.EncodeJob, _ func(int)) error {
				asked = job
				return nil
			})

		// when
		_, err := m.service.Assemble(context.Background(), plan, request, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.EncodeJob{
			Directory: "/tmp/frames",
			Frames:    380,
			FrameRate: plan.Parameters.FrameRate,
			Quality:   domain.VideoQualityHigh,
			Output:    "/tmp/.sobrevoo-1.tmp",
		}, asked)
	})

	t.Run("should say in the summary what the video is: frames, frame rate, quality, encoder and size", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		request := videoRequest()
		request.Quality = domain.VideoQualityLow
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil)
		m.exports(19293798)
		m.encoder.EXPECT().Encode(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		// when
		summary, err := m.service.Assemble(context.Background(), plan, request, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, 380, summary.Frames)
		assert.Equal(t, plan.Parameters.FrameRate, summary.FrameRate)
		assert.Equal(t, domain.VideoQualityLow, summary.Quality)
		assert.Equal(t, anEncoder, summary.Encoder)
		assert.Equal(t, int64(19293798), summary.SizeBytes)
		assert.Equal(t, videoResolution, summary.Resolution)
		assert.GreaterOrEqual(t, int64(summary.Elapsed), int64(0))
		assert.False(t, summary.Interrupted)
	})

	t.Run("should refuse a directory with no frame, without probing the encoder or exporting anything", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		m.repository.EXPECT().List("/tmp/frames").Return(domain.FrameDirectory{}, nil)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDirectoryInvalid)
	})

	t.Run("should return the error of listing the frames as it is, and do nothing else", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		m.repository.EXPECT().List("/tmp/frames").Return(domain.FrameDirectory{}, domain.ErrFrameDirectoryInvalid)

		// when
		summary, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDirectoryInvalid)
		assert.Equal(t, 380, summary.Frames, "the summary says what is known even when it fails")
	})

	t.Run("should return the error of the encoder not being there as it is, without exporting anything", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(domain.EncoderInfo{}, domain.ErrEncoderUnavailable)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrEncoderUnavailable)
	})

	t.Run("should return the error of the encoding as it is", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil)
		m.exports(0)
		m.encoder.EXPECT().Encode(gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.ErrVideoEncodingFailed)

		// when
		summary, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrVideoEncodingFailed)
		assert.Equal(t, anEncoder, summary.Encoder)
		assert.Zero(t, summary.SizeBytes)
	})

	t.Run("should return the error of publishing the video as it is", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		failure := errors.New("the disk is full")
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil)
		m.exporter.EXPECT().Export("/tmp/flight.mp4", false, gomock.Any()).Return(int64(0), failure)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, failure)
	})
}

func Test_videoService_Assemble_Verification(t *testing.T) {
	plan := framesPlan(20)

	t.Run("should refuse frames that do not form the flight of the plan, before probing the encoder or exporting anything", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		directory := builddomain.NewFrameDirectoryBuilder().
			WithResolution(videoResolution).WithPlanID(plan.ID()).
			WithOursFrames(0, 11, "the-set").WithOursFrames(16, 19, "the-set").Build()
		m.repository.EXPECT().List("/tmp/frames").Return(directory, nil)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		require.ErrorIs(t, err, domain.ErrFrameSequenceInvalid)
		assert.ErrorContains(t, err, "4 missing (12-15)")
	})

	t.Run("should refuse frames drawn from another plan with the error of the domain as it is", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		directory := builddomain.NewFrameDirectoryBuilder().
			WithResolution(videoResolution).WithPlanID("another-plan").
			WithOursFrames(0, 19, "the-set").Build()
		m.repository.EXPECT().List("/tmp/frames").Return(directory, nil)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrFramesDoNotMatchPlan)
	})

	t.Run("should refuse frames that do not say which plan they came from", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		directory := builddomain.NewFrameDirectoryBuilder().
			WithResolution(videoResolution).
			WithOursFrames(0, 19, "the-set").Build()
		m.repository.EXPECT().List("/tmp/frames").Return(directory, nil)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrFramesWithoutPlanID)
	})

	t.Run("should take the resolution of the summary from the frames the domain checked", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		directory := builddomain.NewFrameDirectoryBuilder().
			WithResolution(domain.Resolution{Width: 1080, Height: 1920}).WithPlanID(plan.ID()).
			WithOursFrames(0, 19, "the-set").Build()
		m.repository.EXPECT().List("/tmp/frames").Return(directory, nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil)
		m.exports(10)
		m.encoder.EXPECT().Encode(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		// when
		summary, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.Resolution{Width: 1080, Height: 1920}, summary.Resolution)
	})

	t.Run("should refuse a directory with no frame of this tool with the error of the domain", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		m.repository.EXPECT().List("/tmp/frames").Return(builddomain.NewFrameDirectoryBuilder().WithForeignFrame(0).Build(), nil)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		require.ErrorIs(t, err, domain.ErrFrameDirectoryInvalid)
		assert.ErrorContains(t, err, "1 file named like a frame was ignored")
	})
}

func Test_videoService_Assemble_Order(t *testing.T) {
	plan := framesPlan(20)

	t.Run("should say the frames are wrong before the encoder is missing, without even asking for the encoder", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		directory := builddomain.NewFrameDirectoryBuilder().
			WithResolution(videoResolution).WithPlanID(plan.ID()).
			WithOursFrames(0, 11, "the-set").Build()
		m.repository.EXPECT().List("/tmp/frames").Return(directory, nil)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameSequenceInvalid)
		assert.NotErrorIs(t, err, domain.ErrEncoderUnavailable)
	})

	t.Run("should not export anything when the encoder is not there", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(domain.EncoderInfo{}, domain.ErrEncoderUnavailable)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrEncoderUnavailable)
	})

	t.Run("should keep the message of the encoder that is not there as it came", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		unavailable := errors.Join(domain.ErrEncoderUnavailable, errors.New(`"ffmpeg" was not found on the PATH`))
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(domain.EncoderInfo{}, unavailable)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.Same(t, unavailable, err)
	})
}

func Test_videoService_Assemble_Quality(t *testing.T) {
	plan := framesPlan(20)

	// assembles runs an assembly of the quality and gives the job the encoder got
	// and the summary.
	assembles := func(t *testing.T, quality domain.VideoQuality) (domain.EncodeJob, domain.VideoSummary) {
		t.Helper()
		m := newVideoMocks(t)
		request := videoRequest()
		request.Quality = quality
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil)
		m.exports(10)
		var asked domain.EncodeJob
		m.encoder.EXPECT().Encode(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, job domain.EncodeJob, _ func(int)) error {
				asked = job
				return nil
			})

		summary, err := m.service.Assemble(context.Background(), plan, request, nil)

		require.NoError(t, err)
		return asked, summary
	}

	t.Run("should ask the encoder for the low quality and say it in the summary", func(t *testing.T) {
		// given / when
		job, summary := assembles(t, domain.VideoQualityLow)

		// then
		assert.Equal(t, domain.VideoQualityLow, job.Quality)
		assert.Equal(t, domain.VideoQualityLow, summary.Quality)
	})

	t.Run("should ask the encoder for the medium quality and say it in the summary", func(t *testing.T) {
		// given / when
		job, summary := assembles(t, domain.VideoQualityMedium)

		// then
		assert.Equal(t, domain.VideoQualityMedium, job.Quality)
		assert.Equal(t, domain.VideoQualityMedium, summary.Quality)
	})

	t.Run("should ask the encoder for the high quality and say it in the summary", func(t *testing.T) {
		// given / when
		job, summary := assembles(t, domain.VideoQualityHigh)

		// then
		assert.Equal(t, domain.VideoQualityHigh, job.Quality)
		assert.Equal(t, domain.VideoQualityHigh, summary.Quality)
	})
}

func Test_videoService_Assemble_Progress(t *testing.T) {
	plan := framesPlan(20)

	// encodesReporting makes the encoder report the frames it encoded, and the
	// exporter run it.
	encodesReporting := func(m videoMocks, reported ...int) {
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil)
		m.exports(10)
		m.encoder.EXPECT().Encode(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.EncodeJob, progress func(int)) error {
				for _, encoded := range reported {
					progress(encoded)
				}
				return nil
			})
	}

	t.Run("should report each number of frames the encoder reports, with the total of the plan", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		encodesReporting(m, 5, 12)
		var progress []domain.VideoProgress

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), func(p domain.VideoProgress) { progress = append(progress, p) })

		// then
		require.NoError(t, err)
		require.Len(t, progress, 3)
		assert.Equal(t, 5, progress[0].Done)
		assert.Equal(t, 12, progress[1].Done)
		for _, p := range progress {
			assert.Equal(t, 20, p.Total)
			assert.GreaterOrEqual(t, int64(p.Elapsed), int64(0))
		}
	})

	t.Run("should report a last time, with every frame done, when the video is made", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		encodesReporting(m, 5)
		var progress []domain.VideoProgress

		// when
		summary, err := m.service.Assemble(context.Background(), plan, videoRequest(), func(p domain.VideoProgress) { progress = append(progress, p) })

		// then
		require.NoError(t, err)
		assert.Equal(t, 20, progress[len(progress)-1].Done)
		assert.Equal(t, 20, progress[len(progress)-1].Total)
		assert.Equal(t, 20, summary.Encoded)
	})

	t.Run("should not report the last time when the encoding fails, and say in the summary how many were encoded", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil)
		m.exports(0)
		m.encoder.EXPECT().Encode(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.EncodeJob, progress func(int)) error {
				progress(7)
				return domain.ErrVideoEncodingFailed
			})
		var progress []domain.VideoProgress

		// when
		summary, err := m.service.Assemble(context.Background(), plan, videoRequest(), func(p domain.VideoProgress) { progress = append(progress, p) })

		// then
		require.ErrorIs(t, err, domain.ErrVideoEncodingFailed)
		require.Len(t, progress, 1)
		assert.Equal(t, 7, progress[0].Done)
		assert.Equal(t, 7, summary.Encoded)
	})

	t.Run("should never report more frames than the plan has", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		encodesReporting(m, 25)
		var progress []domain.VideoProgress

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), func(p domain.VideoProgress) { progress = append(progress, p) })

		// then
		require.NoError(t, err)
		for _, p := range progress {
			assert.LessOrEqual(t, p.Done, p.Total)
		}
	})

	t.Run("should accept no function to report to", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		encodesReporting(m, 5, 12)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.NoError(t, err)
	})
}

func Test_videoService_Assemble_Destination(t *testing.T) {
	plan := framesPlan(20)

	t.Run("should check the destination after the frames and before the encoder", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		gomock.InOrder(
			m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil),
			m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil),
			m.encoder.EXPECT().Probe(gomock.Any()).Return(domain.EncoderInfo{}, domain.ErrEncoderUnavailable),
		)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrEncoderUnavailable)
	})

	t.Run("should refuse a destination that exists, as it is, without asking for the encoder or exporting anything", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(domain.ErrVideoDestinationExists)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrVideoDestinationExists)
	})

	t.Run("should refuse a destination that cannot be written, as it is, without asking for the encoder or exporting anything", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(domain.ErrVideoDestinationInvalid)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrVideoDestinationInvalid)
	})

	t.Run("should say the frames are wrong before the destination is, without checking the destination", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		m.repository.EXPECT().List("/tmp/frames").Return(builddomain.NewFrameDirectoryBuilder().Build(), nil)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDirectoryInvalid)
	})

	t.Run("should say the destination is wrong before the encoder is missing", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(domain.ErrVideoDestinationExists)

		// when
		_, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		assert.NotErrorIs(t, err, domain.ErrEncoderUnavailable)
	})

	t.Run("should ask to overwrite in the check and in the export when the request does", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		request := videoRequest()
		request.Overwrite = true
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", true).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil)
		m.exporter.EXPECT().Export("/tmp/flight.mp4", true, gomock.Any()).Return(int64(10), nil)

		// when
		_, err := m.service.Assemble(context.Background(), plan, request, nil)

		// then
		assert.NoError(t, err)
	})
}

func Test_videoService_Assemble_Interruption(t *testing.T) {
	plan := framesPlan(20)

	t.Run("should say the assembly was interrupted, and how many frames were encoded, when the encoding stops because the context is done", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		ctx, cancel := context.WithCancel(context.Background())
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil)
		m.exports(0)
		m.encoder.EXPECT().Encode(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.EncodeJob, progress func(int)) error {
				progress(12)
				cancel()
				return fmt.Errorf("encoding stopped: %w", context.Canceled)
			})

		// when
		summary, err := m.service.Assemble(ctx, plan, videoRequest(), nil)

		// then
		require.ErrorIs(t, err, domain.ErrVideoInterrupted)
		assert.True(t, summary.Interrupted)
		assert.Equal(t, 12, summary.Encoded)
		assert.Zero(t, summary.SizeBytes)
	})

	t.Run("should say it was interrupted when the encoder is probed as the context is done", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		ctx, cancel := context.WithCancel(context.Background())
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).DoAndReturn(func(context.Context) (domain.EncoderInfo, error) {
			cancel()
			return domain.EncoderInfo{}, fmt.Errorf("could not be run: %w", context.Canceled)
		})

		// when
		summary, err := m.service.Assemble(ctx, plan, videoRequest(), nil)

		// then
		require.ErrorIs(t, err, domain.ErrVideoInterrupted)
		assert.True(t, summary.Interrupted)
	})

	t.Run("should not take an error of the context for an interruption when the context is not done", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil)
		m.exporter.EXPECT().Export("/tmp/flight.mp4", false, gomock.Any()).Return(int64(0), context.Canceled)

		// when
		summary, err := m.service.Assemble(context.Background(), plan, videoRequest(), nil)

		// then
		require.ErrorIs(t, err, context.Canceled)
		assert.NotErrorIs(t, err, domain.ErrVideoInterrupted)
		assert.False(t, summary.Interrupted)
	})

	t.Run("should not take another error for an interruption when the context is done", func(t *testing.T) {
		// given
		m := newVideoMocks(t)
		ctx, cancel := context.WithCancel(context.Background())
		m.repository.EXPECT().List("/tmp/frames").Return(framesOf(plan), nil)
		m.exporter.EXPECT().Check("/tmp/flight.mp4", false).Return(nil)
		m.encoder.EXPECT().Probe(gomock.Any()).Return(anEncoder, nil)
		m.exporter.EXPECT().Export("/tmp/flight.mp4", false, gomock.Any()).DoAndReturn(func(string, bool, func(string) error) (int64, error) {
			cancel()
			return 0, domain.ErrVideoDestinationInvalid
		})

		// when
		summary, err := m.service.Assemble(ctx, plan, videoRequest(), nil)

		// then
		require.ErrorIs(t, err, domain.ErrVideoDestinationInvalid)
		assert.False(t, summary.Interrupted)
	})
}
