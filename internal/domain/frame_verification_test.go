package domain_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

// planOfNFrames is a plan of n frames.
func planOfNFrames(n int) domain.CameraPlan {
	frames := make([]domain.CameraFrame, n)
	for i := range frames {
		frames[i] = builddomain.NewCameraFrameBuilder().WithIndex(i).Build()
	}
	return builddomain.NewCameraPlanBuilder().WithFrames(frames...).Build()
}

// framesOfPlan is the directory of the n frames of plan, all whole, 360 × 640, of
// one set and drawn from the plan.
func framesOfPlan(plan domain.CameraPlan) *builddomain.FrameDirectoryBuilder {
	return builddomain.NewFrameDirectoryBuilder().
		WithResolution(domain.Resolution{Width: 360, Height: 640}).
		WithPlanID(plan.ID()).
		WithOursFrames(0, len(plan.Frames)-1, "set-a")
}

func Test_FrameDirectory_Verify_Success(t *testing.T) {
	t.Run("should give the resolution of the frames of a directory that holds exactly the frames of the plan", func(t *testing.T) {
		// given
		plan := planOfNFrames(20)
		directory := framesOfPlan(plan).Build()

		// when
		resolution, err := directory.Verify(plan)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.Resolution{Width: 360, Height: 640}, resolution)
	})

	t.Run("should accept a plan of a single frame with a single frame", func(t *testing.T) {
		// given
		plan := planOfNFrames(1)
		directory := framesOfPlan(plan).Build()

		// when
		_, err := directory.Verify(plan)

		// then
		assert.NoError(t, err)
	})

	t.Run("should not count as anything the files named like frames that are not this tool's", func(t *testing.T) {
		// given
		plan := planOfNFrames(20)
		directory := framesOfPlan(plan).WithForeignFrame(500).WithForeignFrame(20).Build()

		// when
		_, err := directory.Verify(plan)

		// then
		assert.NoError(t, err)
	})
}

func Test_FrameDirectory_Verify_NoFrames(t *testing.T) {
	plan := planOfNFrames(20)

	t.Run("should refuse a directory with no frame at all, saying how to draw them", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameDirectoryInvalid)
		assert.ErrorContains(t, err, `holds no frames drawn by this tool; draw them with "render all"`)
	})

	t.Run("should say how many files named like frames were ignored, and why", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().WithForeignFrame(0).WithForeignFrame(1).WithForeignFrame(2).Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameDirectoryInvalid)
		assert.ErrorContains(t, err, "3 files named like frames were ignored: they do not carry this tool's identification")
	})

	t.Run("should say it in the singular for a single file", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().WithForeignFrame(0).Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameDirectoryInvalid)
		assert.ErrorContains(t, err, "1 file named like a frame was ignored: it does not carry this tool's identification")
	})
}

func Test_FrameDirectory_Verify_PlanIdentification(t *testing.T) {
	plan := planOfNFrames(20)
	other := strings.Repeat("3fa9c1d2e4b7", 6)[:64]

	t.Run("should refuse frames that do not say which plan they were drawn from, saying to draw them again", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithPlanID("").WithOursFrames(0, 19, "set-a").Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFramesWithoutPlanID)
		assert.ErrorContains(t, err, `20 of 20 frames were drawn by an earlier version of Sobrevoo; draw them again with "render all --overwrite"`)
	})

	t.Run("should count only the frames without the identification", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithFrameWithoutPlan(3, "set-a").WithFrameWithoutPlan(4, "set-a").Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFramesWithoutPlanID)
		assert.ErrorContains(t, err, "2 of 20 frames were drawn by an earlier version of Sobrevoo")
	})

	t.Run("should refuse frames drawn from another plan, with both identifications shortened to 12 characters", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithPlanID(other).WithOursFrames(0, 19, "set-a").Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFramesDoNotMatchPlan)
		assert.ErrorContains(t, err, "20 of 20 frames carry another plan identification (plan "+plan.ID()[:12]+", frames "+other[:12]+")")
	})

	t.Run("should count how many frames are of another plan, when only some are", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithFrameOfPlan(9, "set-a", other).Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFramesDoNotMatchPlan)
		assert.ErrorContains(t, err, "1 of 20 frames carry another plan identification")
	})

	t.Run("should say a missing identification before another plan's", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithFrameWithoutPlan(0, "set-a").WithFrameOfPlan(1, "set-a", other).Build()

		// when
		_, err := directory.Verify(plan)

		// then
		assert.ErrorIs(t, err, domain.ErrFramesWithoutPlanID)
	})

	t.Run("should say the frames are of another plan even when they are not the number the plan has", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().WithPlanID(other).WithOursFrames(0, 5, "set-a").Build()

		// when
		_, err := directory.Verify(plan)

		// then
		assert.ErrorIs(t, err, domain.ErrFramesDoNotMatchPlan, "not that frames are missing")
	})
}

func Test_FrameDirectory_Verify_Resolution(t *testing.T) {
	plan := planOfNFrames(30)
	small := domain.Resolution{Width: 180, Height: 320}

	t.Run("should refuse frames of different resolutions, saying which is the most frequent and which frames differ", func(t *testing.T) {
		// given
		builder := framesOfPlan(plan)
		for index := 10; index <= 12; index++ {
			builder.WithFrameOfResolution(index, "set-a", small)
		}
		directory := builder.Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameResolutionInvalid)
		assert.ErrorContains(t, err, "360x640 is the resolution of 27 frames, but 3 differ (frame_000010.png is 180x320, frame_000011.png is 180x320, frame_000012.png is 180x320)")
	})

	t.Run("should list at most five of the frames that differ, and count the rest", func(t *testing.T) {
		// given
		builder := framesOfPlan(plan)
		for index := 10; index <= 17; index++ {
			builder.WithFrameOfResolution(index, "set-a", small)
		}
		directory := builder.Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameResolutionInvalid)
		assert.ErrorContains(t, err, "but 8 differ (frame_000010.png is 180x320, frame_000011.png is 180x320, frame_000012.png is 180x320, frame_000013.png is 180x320, frame_000014.png is 180x320, and 3 more)")
	})

	t.Run("should take, when two resolutions have as many frames, the one of the lowest numbered frame", func(t *testing.T) {
		// given
		plan := planOfNFrames(10)
		builder := framesOfPlan(plan)
		for index := 5; index <= 9; index++ {
			builder.WithFrameOfResolution(index, "set-a", small)
		}
		directory := builder.Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameResolutionInvalid)
		assert.ErrorContains(t, err, "360x640 is the resolution of 5 frames, but 5 differ")
	})

	t.Run("should refuse a width or a height that is odd, which the video cannot have", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithResolution(domain.Resolution{Width: 1081, Height: 1921}).WithOursFrames(0, 29, "set-a").Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameResolutionInvalid)
		assert.ErrorContains(t, err, "the frames are 1081x1921, but width and height must be even for the video")
	})

	t.Run("should refuse a width that is odd with an even height, and the other way round", func(t *testing.T) {
		// given
		oddWidth := framesOfPlan(plan).WithResolution(domain.Resolution{Width: 361, Height: 640}).WithOursFrames(0, 29, "set-a").Build()
		oddHeight := framesOfPlan(plan).WithResolution(domain.Resolution{Width: 360, Height: 641}).WithOursFrames(0, 29, "set-a").Build()

		// when
		_, errWidth := oddWidth.Verify(plan)
		_, errHeight := oddHeight.Verify(plan)

		// then
		assert.ErrorIs(t, errWidth, domain.ErrFrameResolutionInvalid)
		assert.ErrorIs(t, errHeight, domain.ErrFrameResolutionInvalid)
	})

	t.Run("should not judge the size of the frames beyond being even, since the encoder is the one that says what it can do", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithResolution(domain.Resolution{Width: 64, Height: 48}).WithOursFrames(0, 29, "set-a").Build()

		// when
		resolution, err := directory.Verify(plan)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.Resolution{Width: 64, Height: 48}, resolution)
	})

	t.Run("should say the resolution, and not that the sets are mixed, when frames of the same plan differ in both", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithFrameOfResolution(3, "set-b", small).Build()

		// when
		_, err := directory.Verify(plan)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameResolutionInvalid)
		assert.NotErrorIs(t, err, domain.ErrFramesDoNotMatchPlan)
	})
}

func Test_FrameDirectory_Verify_Sets(t *testing.T) {
	plan := planOfNFrames(20)

	t.Run("should refuse frames of the same plan and resolution that belong to two sets", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithOursFrame(4, "set-b").Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFramesDoNotMatchPlan)
		assert.ErrorContains(t, err, "the directory mixes frames of 2 different sets (another slice, resolution or drawing version); draw them again into an empty directory")
	})

	t.Run("should count the sets", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithOursFrame(4, "set-b").WithOursFrame(5, "set-c").Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFramesDoNotMatchPlan)
		assert.ErrorContains(t, err, "mixes frames of 3 different sets")
	})
}

func Test_FrameDirectory_Verify_Numbering(t *testing.T) {
	plan := planOfNFrames(20)

	t.Run("should say which frames are missing, in ranges, and how to draw them", func(t *testing.T) {
		// given
		builder := builddomain.NewFrameDirectoryBuilder().WithResolution(domain.Resolution{Width: 360, Height: 640}).WithPlanID(plan.ID())
		builder.WithOursFrames(0, 11, "set-a").WithOursFrames(16, 19, "set-a")
		directory := builder.Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameSequenceInvalid)
		assert.ErrorContains(t, err, `4 missing (12-15)`)
		assert.ErrorContains(t, err, `draw the missing frames with "render all"`)
	})

	t.Run("should write a single number alone and a range with a dash", func(t *testing.T) {
		// given
		builder := builddomain.NewFrameDirectoryBuilder().WithResolution(domain.Resolution{Width: 360, Height: 640}).WithPlanID(plan.ID())
		builder.WithOursFrames(4, 6, "set-a").WithOursFrames(8, 9, "set-a").WithOursFrames(13, 19, "set-a")
		directory := builder.Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameSequenceInvalid)
		assert.ErrorContains(t, err, "8 missing (0-3, 7, 10-12)")
	})

	t.Run("should list at most ten ranges, keep the total in view and count what is left out", func(t *testing.T) {
		// given
		plan := planOfNFrames(80)
		builder := builddomain.NewFrameDirectoryBuilder().WithResolution(domain.Resolution{Width: 360, Height: 640}).WithPlanID(plan.ID())
		for index := 1; index < 80; index += 2 {
			builder.WithOursFrame(index, "set-a")
		}
		directory := builder.Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameSequenceInvalid)
		assert.ErrorContains(t, err, "40 missing (0, 2, 4, 6, 8, 10, 12, 14, 16, 18, and 30 more)")
	})

	t.Run("should say which frames the plan has no number for, and which numbers it has", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithOursFrame(20, "set-a").WithOursFrame(21, "set-a").Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameSequenceInvalid)
		assert.ErrorContains(t, err, "2 not in the plan (20-21; the plan has frames 0 to 19)")
	})

	t.Run("should say a number that appears more than once", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithRepeatedFrame(3, "set-a").Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameSequenceInvalid)
		assert.ErrorContains(t, err, "number 3 appears more than once")
	})

	t.Run("should say the numbers that appear more than once, when there are several", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithRepeatedFrame(3, "set-a").WithRepeatedFrame(4, "set-a").WithRepeatedFrame(9, "set-a").Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameSequenceInvalid)
		assert.ErrorContains(t, err, "3 numbers appear more than once (3-4, 9)")
	})

	t.Run("should say what is missing, what the plan has no number for and what repeats, in one message", func(t *testing.T) {
		// given
		builder := builddomain.NewFrameDirectoryBuilder().WithResolution(domain.Resolution{Width: 360, Height: 640}).WithPlanID(plan.ID())
		builder.WithOursFrames(0, 11, "set-a").WithOursFrames(16, 21, "set-a").WithRepeatedFrame(2, "set-a")
		directory := builder.Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameSequenceInvalid)
		assert.ErrorContains(t, err, "4 missing (12-15), 2 not in the plan (20-21; the plan has frames 0 to 19), number 2 appears more than once")
	})

	t.Run("should say that a file named like a missing frame was ignored, and why", func(t *testing.T) {
		// given
		builder := builddomain.NewFrameDirectoryBuilder().WithResolution(domain.Resolution{Width: 360, Height: 640}).WithPlanID(plan.ID())
		builder.WithOursFrames(0, 11, "set-a").WithForeignFrame(12).WithOursFrames(13, 19, "set-a")
		directory := builder.Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameSequenceInvalid)
		assert.ErrorContains(t, err, "1 missing (12)")
		assert.ErrorContains(t, err, "frame_000012.png was ignored: it does not carry this tool's identification")
	})

	t.Run("should not mention a foreign file that is not in the place of a missing frame", func(t *testing.T) {
		// given
		builder := builddomain.NewFrameDirectoryBuilder().WithResolution(domain.Resolution{Width: 360, Height: 640}).WithPlanID(plan.ID())
		builder.WithOursFrames(0, 12, "set-a").WithOursFrames(14, 19, "set-a").WithForeignFrame(500)
		directory := builder.Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameSequenceInvalid)
		assert.NotContains(t, err.Error(), "ignored")
	})

	t.Run("should say the numbering before that a frame is not whole", func(t *testing.T) {
		// given
		builder := builddomain.NewFrameDirectoryBuilder().WithResolution(domain.Resolution{Width: 360, Height: 640}).WithPlanID(plan.ID())
		builder.WithOursFrames(0, 5, "set-a").WithTruncatedFrame(6, "set-a").WithOursFrames(8, 19, "set-a")
		directory := builder.Build()

		// when
		_, err := directory.Verify(plan)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameSequenceInvalid)
	})
}

func Test_FrameDirectory_Verify_Whole(t *testing.T) {
	plan := planOfNFrames(20)

	t.Run("should refuse a frame that is not a whole image, saying which and how to draw it again", func(t *testing.T) {
		// given
		directory := framesOfPlan(plan).WithTruncatedFrame(7, "set-a").Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameFileInvalid)
		assert.ErrorContains(t, err, `frame_000007.png is not a whole PNG image (truncated?); draw it again with "render all"`)
	})

	t.Run("should list at most five of the files, and count the rest", func(t *testing.T) {
		// given
		builder := framesOfPlan(plan)
		for index := 2; index <= 8; index++ {
			builder.WithTruncatedFrame(index, "set-a")
		}
		directory := builder.Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameFileInvalid)
		assert.ErrorContains(t, err, `7 frames are not whole PNG images (truncated?): frame_000002.png, frame_000003.png, frame_000004.png, frame_000005.png, frame_000006.png, and 2 more; draw them again with "render all"`)
	})
}

func Test_FrameDirectory_Verify_Resolution_Grammar(t *testing.T) {
	t.Run("should say a single frame differs, not differ", func(t *testing.T) {
		// given
		plan := planOfNFrames(30)
		directory := framesOfPlan(plan).WithFrameOfResolution(10, "set-a", domain.Resolution{Width: 180, Height: 320}).Build()

		// when
		_, err := directory.Verify(plan)

		// then
		require.ErrorIs(t, err, domain.ErrFrameResolutionInvalid)
		assert.ErrorContains(t, err, "but 1 differs (frame_000010.png is 180x320)")
	})
}
