package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application/mockapplication"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

type videoCommandMocks struct {
	planService  *mockapplication.MockCameraPlanService
	videoService *mockapplication.MockVideoService
}

func newVideoCommandMocks(t *testing.T) videoCommandMocks {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	return videoCommandMocks{
		planService:  mockapplication.NewMockCameraPlanService(mockCtrl),
		videoService: mockapplication.NewMockVideoService(mockCtrl),
	}
}

func executeVideoCommand(t *testing.T, m videoCommandMocks, defaultQuality domain.VideoQuality, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	cmd := cli.NewVideoCommand(m.planService, m.videoService, defaultQuality)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), errOut.String(), err
}

// aVideo is what the summary of an assembly of 1350 frames at 30 frames per second
// looks like.
func aVideo() domain.VideoSummary {
	return domain.VideoSummary{
		Frames:     1350,
		FrameRate:  30,
		Resolution: domain.Resolution{Width: 1080, Height: 1920},
		Quality:    domain.VideoQualityMedium,
		Encoder:    domain.EncoderInfo{Name: "ffmpeg", Version: "7.1", Codec: "libx264"},
		SizeBytes:  19293798,
		Elapsed:    112 * time.Second,
	}
}

func Test_VideoCommand_Args(t *testing.T) {
	t.Run("should return a usage error when there is no plan and no frame directory", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "--output", "flight.mp4")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return a usage error when there is no frame directory", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "--output", "flight.mp4")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return a usage error when there is one argument too many", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "extra", "--output", "flight.mp4")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should return a usage error, without asking for anything, when --output is missing", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.ErrorContains(t, err, "--output is required")
	})

	t.Run("should return a usage error for an unknown flag", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4", "--nonsense")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})
}

func Test_VideoCommand_Execute(t *testing.T) {
	plan := planOfFrames(1350)

	t.Run("should load the plan and assemble the video of the frames in the directory, of the default quality, without overwriting", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)
		m.planService.EXPECT().Load("plan.json").Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, domain.VideoRequest{
			Directory: "frames",
			Output:    "flight.mp4",
			Quality:   domain.VideoQualityMedium,
			Overwrite: false,
		}, gomock.Any()).Return(aVideo(), nil)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.NoError(t, err)
	})

	t.Run("should print the summary, in the labels and order of the contract", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(aVideo(), nil)

		// when
		stdout, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.NoError(t, err)
		assert.Equal(t, "Video written to flight.mp4\n"+
			"Frames: 1350\n"+
			"Duration: 00:00:45.000\n"+
			"Resolution: 1080x1920\n"+
			"Frame rate: 30 fps\n"+
			"Quality: medium\n"+
			"Size: 18.4 MiB\n"+
			"Encoder: ffmpeg 7.1 (libx264)\n"+
			"Time: 00:01:52\n", stdout)
	})

	t.Run("should write a frame rate that is not whole as the plan has it, and the duration to the millisecond", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)
		summary := aVideo()
		summary.FrameRate = 29.97
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, nil)

		// when
		stdout, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Frame rate: 29.97 fps\n")
		assert.Contains(t, stdout, "Duration: 00:00:45.045\n")
	})

	t.Run("should write the size in bytes, KiB, MiB and GiB, with one decimal from KiB", func(t *testing.T) {
		// given
		sizes := map[int64]string{
			0:                      "Size: 0 B\n",
			512:                    "Size: 512 B\n",
			1023:                   "Size: 1023 B\n",
			1024:                   "Size: 1.0 KiB\n",
			1536:                   "Size: 1.5 KiB\n",
			19293798:               "Size: 18.4 MiB\n",
			3 * 1024 * 1024 * 1024: "Size: 3.0 GiB\n",
		}

		for size, want := range sizes {
			m := newVideoCommandMocks(t)
			summary := aVideo()
			summary.SizeBytes = size
			m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
			m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, nil)

			// when
			stdout, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4")

			// then
			require.NoError(t, err)
			assert.Contains(t, stdout, want, "%d bytes", size)
		}
	})

	t.Run("should return the error of loading the plan as it is, without assembling anything", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)
		m.planService.EXPECT().Load("plan.json").Return(domain.CameraPlan{}, domain.ErrPlanFileInvalid)

		// when
		stdout, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.Empty(t, stdout)
	})

	t.Run("should return the error of the assembly as it is, and print no summary", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)
		failure := errors.New("the video encoder broke")
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.VideoSummary{Frames: 1350}, failure)

		// when
		stdout, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.ErrorIs(t, err, failure)
		assert.Empty(t, stdout)
	})
}

func Test_VideoCommand_Execute_FrameChecks(t *testing.T) {
	plan := planOfFrames(1350)

	t.Run("should return the error of a frame check as it is, keeping its message, and print no summary", func(t *testing.T) {
		// given
		failures := []error{
			errors.Join(domain.ErrFrameDirectoryInvalid),
			errors.Join(domain.ErrFrameSequenceInvalid),
			errors.Join(domain.ErrFrameResolutionInvalid),
			errors.Join(domain.ErrFramesDoNotMatchPlan),
			errors.Join(domain.ErrFramesWithoutPlanID),
			errors.Join(domain.ErrFrameFileInvalid),
		}

		for _, failure := range failures {
			m := newVideoCommandMocks(t)
			wrapped := fmt.Errorf("%w: 4 missing (12-15)", failure)
			m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
			m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.VideoSummary{}, wrapped)

			// when
			stdout, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4")

			// then
			require.ErrorIs(t, err, failure)
			assert.Equal(t, wrapped.Error(), err.Error())
			assert.Empty(t, stdout)
		}
	})
}

func Test_VideoCommand_Quality(t *testing.T) {
	plan := planOfFrames(1350)

	// asks runs the command with the args, expecting the assembly to be asked
	// with the quality.
	asks := func(t *testing.T, defaultQuality, want domain.VideoQuality, args ...string) {
		t.Helper()
		m := newVideoCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			return x.(domain.VideoRequest).Quality == want
		}), gomock.Any()).Return(aVideo(), nil)

		_, _, err := executeVideoCommand(t, m, defaultQuality, append([]string{"plan.json", "frames", "--output", "flight.mp4"}, args...)...)

		require.NoError(t, err)
	}

	t.Run("should ask for the low quality with --quality low", func(t *testing.T) {
		// given / when / then
		asks(t, domain.VideoQualityMedium, domain.VideoQualityLow, "--quality", "low")
	})

	t.Run("should ask for the medium quality with --quality medium", func(t *testing.T) {
		// given / when / then
		asks(t, domain.VideoQualityLow, domain.VideoQualityMedium, "--quality", "medium")
	})

	t.Run("should ask for the high quality with --quality high", func(t *testing.T) {
		// given / when / then
		asks(t, domain.VideoQualityMedium, domain.VideoQualityHigh, "--quality", "high")
	})

	t.Run("should ask for the quality it was given as the default when none is chosen", func(t *testing.T) {
		// given / when / then
		asks(t, domain.VideoQualityLow, domain.VideoQualityLow)
		asks(t, domain.VideoQualityHigh, domain.VideoQualityHigh)
	})

	t.Run("should refuse a quality that does not exist as a usage error, listing the ones accepted, without asking for anything", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)

		// when
		stdout, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4", "--quality", "ultra")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.EqualError(t, err, `--quality "ultra": use one of low, medium, high`)
		assert.Empty(t, stdout)
	})

	t.Run("should say the quality of the video in the summary", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)
		summary := aVideo()
		summary.Quality = domain.VideoQualityLow
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, nil)

		// when
		stdout, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4", "--quality", "low")

		// then
		require.NoError(t, err)
		assert.Contains(t, stdout, "Quality: low\n")
	})
}

func Test_VideoCommand_Output(t *testing.T) {
	plan := planOfFrames(1350)

	t.Run("should refuse an output that does not end in .mp4 as a usage error, saying the video is always an MP4 file", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mkv")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
		assert.EqualError(t, err, "--output must end in .mp4: the video is always an MP4 file")
	})

	t.Run("should refuse an output with no extension", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should refuse an output that only has .mp4 inside its name", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4.txt")

		// then
		require.Error(t, err)
		assert.Equal(t, 2, cli.ExitCode(err))
	})

	t.Run("should refuse it before reading the plan", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "missing.json", "frames", "--output", "flight.mov")

		// then
		assert.Equal(t, 2, cli.ExitCode(err), "not the error of the plan, which was not read")
	})

	t.Run("should accept the extension in capitals", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(aVideo(), nil)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "FLIGHT.MP4")

		// then
		assert.NoError(t, err)
	})

	t.Run("should ask to overwrite with --overwrite", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Cond(func(x any) bool {
			return x.(domain.VideoRequest).Overwrite
		}), gomock.Any()).Return(aVideo(), nil)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4", "--overwrite")

		// then
		assert.NoError(t, err)
	})
}

func Test_VideoCommand_Interrupted(t *testing.T) {
	plan := planOfFrames(1350)

	t.Run("should say the assembly was interrupted, how many frames were encoded and that nothing was written, and return the error", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)
		summary := aVideo()
		summary.Encoded = 412
		summary.Elapsed = 41 * time.Second
		summary.Interrupted = true
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, domain.ErrVideoInterrupted)

		// when
		stdout, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.ErrorIs(t, err, domain.ErrVideoInterrupted)
		assert.Equal(t, 49, cli.ExitCode(err))
		assert.Equal(t, "Interrupted: 412 of 1350 frames were encoded; no video was written, run the same command again to start over\n"+
			"Time: 00:00:41\n", stdout)
	})

	t.Run("should not print the summary of a video that was made", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)
		summary := aVideo()
		summary.Interrupted = true
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(summary, domain.ErrVideoInterrupted)

		// when
		stdout, _, _ := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4")

		// then
		assert.NotContains(t, stdout, "Video written")
		assert.NotContains(t, stdout, "Size:")
	})

	t.Run("should print nothing on stdout for any other error", func(t *testing.T) {
		// given
		m := newVideoCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.VideoSummary{Encoded: 5}, domain.ErrVideoEncodingFailed)

		// when
		stdout, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4")

		// then
		require.ErrorIs(t, err, domain.ErrVideoEncodingFailed)
		assert.Empty(t, stdout)
	})
}
