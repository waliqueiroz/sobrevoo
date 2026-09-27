package domain_test

import (
	"fmt"
	"regexp"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_FrameFileName(t *testing.T) {
	t.Run("should be frame_ and the number in six digits", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, "frame_000000.png", domain.FrameFileName(0))
		assert.Equal(t, "frame_000300.png", domain.FrameFileName(300))
		assert.Equal(t, "frame_431999.png", domain.FrameFileName(431999))
	})

	t.Run("should sort alphabetically in the order of the frames", func(t *testing.T) {
		// given
		names := make([]string, 1260)
		for i := range names {
			names[i] = domain.FrameFileName(i)
		}
		reversed := append([]string(nil), names...)
		sort.Sort(sort.Reverse(sort.StringSlice(reversed)))

		// when
		sort.Strings(reversed)

		// then
		assert.Equal(t, names, reversed)
	})
}

func Test_ParseFrameFileName(t *testing.T) {
	t.Run("should give the number of a frame file name", func(t *testing.T) {
		// given / when
		index, ok := domain.ParseFrameFileName("frame_000300.png")

		// then
		assert.True(t, ok)
		assert.Equal(t, 300, index)
	})

	t.Run("should refuse a name that is not exactly frame_ six digits and .png", func(t *testing.T) {
		for _, name := range []string{"frame_1.png", "frame_0003000.png", "frame_000300.jpg", "Frame_000300.png", "frame_00030a.png", "", "frame_000300.png.tmp", ".frame_000300.png", "xframe_000300.png"} {
			// given / when
			_, ok := domain.ParseFrameFileName(name)

			// then
			assert.False(t, ok, "%q", name)
		}
	})
}

func Test_ParseFrameNumber(t *testing.T) {
	t.Run("should read a number inside the plan", func(t *testing.T) {
		// given / when
		first, errFirst := domain.ParseFrameNumber("0", 1260)
		middle, errMiddle := domain.ParseFrameNumber("300", 1260)
		last, errLast := domain.ParseFrameNumber("1259", 1260)

		// then
		require.NoError(t, errFirst)
		require.NoError(t, errMiddle)
		require.NoError(t, errLast)
		assert.Equal(t, []int{0, 300, 1259}, []int{first, middle, last})
	})

	t.Run("should refuse a number past the last frame, saying the valid range", func(t *testing.T) {
		// given / when
		_, err := domain.ParseFrameNumber("1260", 1260)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameOutOfRange)
		assert.ErrorContains(t, err, "1260")
		assert.ErrorContains(t, err, "frames 0 to 1259")
	})

	t.Run("should refuse a negative number or one too big to be a number of the plan", func(t *testing.T) {
		// given / when
		_, errNegative := domain.ParseFrameNumber("-1", 1260)
		_, errHuge := domain.ParseFrameNumber("99999999999999999999", 1260)

		// then
		assert.ErrorIs(t, errNegative, domain.ErrFrameOutOfRange)
		assert.ErrorIs(t, errHuge, domain.ErrFrameOutOfRange)
	})

	t.Run("should refuse text that is not a whole number, saying so", func(t *testing.T) {
		for _, text := range []string{"3.5", "abc", "", " 3", "3 ", "+3", "1e2", "0x10"} {
			// given / when
			_, err := domain.ParseFrameNumber(text, 1260)

			// then
			assert.ErrorIs(t, err, domain.ErrFrameOutOfRange, "%q", text)
			assert.ErrorContains(t, err, "not a whole number", "%q", text)
		}
	})
}

func Test_NewFrameSetID(t *testing.T) {
	plan := builddomain.NewCameraPlanBuilder().Build()
	slice := builddomain.NewGeoSliceBuilder().WithContentID("content-a").Build()
	resolution := domain.Resolution{Width: 1920, Height: 1080}
	tuning := builddomain.NewRenderTuningBuilder().Build()

	t.Run("should be 64 lowercase hexadecimal characters", func(t *testing.T) {
		// given / when
		id := domain.NewFrameSetID(plan, slice, resolution, tuning)

		// then
		assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{64}$`), string(id))
	})

	t.Run("should be the same for the same plan, slice, resolution and tuning", func(t *testing.T) {
		// given
		samePlan := builddomain.NewCameraPlanBuilder().Build()
		sameSlice := builddomain.NewGeoSliceBuilder().WithContentID("content-a").Build()

		// when / then
		assert.Equal(t, domain.NewFrameSetID(plan, slice, resolution, tuning), domain.NewFrameSetID(samePlan, sameSlice, resolution, tuning))
	})

	t.Run("should differ when the plan differs", func(t *testing.T) {
		// given
		other := builddomain.NewCameraPlanBuilder().WithFrames(
			builddomain.NewCameraFrameBuilder().WithIndex(0).WithCameraAltitude(101).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(1).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(2).Build(),
		).Build()

		// when / then
		assert.NotEqual(t, domain.NewFrameSetID(plan, slice, resolution, tuning), domain.NewFrameSetID(other, slice, resolution, tuning))
	})

	t.Run("should differ when the slice file differs", func(t *testing.T) {
		// given
		other := builddomain.NewGeoSliceBuilder().WithContentID("content-b").Build()

		// when / then
		assert.NotEqual(t, domain.NewFrameSetID(plan, slice, resolution, tuning), domain.NewFrameSetID(plan, other, resolution, tuning))
	})

	t.Run("should differ when the width or the height differs", func(t *testing.T) {
		// given
		base := domain.NewFrameSetID(plan, slice, resolution, tuning)

		// when / then
		assert.NotEqual(t, base, domain.NewFrameSetID(plan, slice, domain.Resolution{Width: 1280, Height: 1080}, tuning))
		assert.NotEqual(t, base, domain.NewFrameSetID(plan, slice, domain.Resolution{Width: 1920, Height: 720}, tuning))
	})

	t.Run("should not mix the width with the height", func(t *testing.T) {
		// given / when
		first := domain.NewFrameSetID(plan, slice, domain.Resolution{Width: 1000, Height: 2000}, tuning)
		second := domain.NewFrameSetID(plan, slice, domain.Resolution{Width: 2000, Height: 1000}, tuning)

		// then
		assert.NotEqual(t, first, second)
	})

	t.Run("should differ when a tuning that changes the look differs, but not for one that does not", func(t *testing.T) {
		// given
		base := domain.NewFrameSetID(plan, slice, resolution, tuning)
		looks := builddomain.NewRenderTuningBuilder().WithVerticalFOVDegrees(60).Build()
		speed := builddomain.NewRenderTuningBuilder().WithWorkers(1).WithTileCacheBytes(1 << 20).Build()

		// when / then
		assert.NotEqual(t, base, domain.NewFrameSetID(plan, slice, resolution, looks))
		assert.Equal(t, base, domain.NewFrameSetID(plan, slice, resolution, speed))
	})

	t.Run("should differ when the version of the drawing differs", func(t *testing.T) {
		// given
		current := domain.NewFrameSetIDForVersion(domain.RenderVersion, plan, slice, resolution, tuning)
		next := domain.NewFrameSetIDForVersion(domain.RenderVersion+1, plan, slice, resolution, tuning)

		// when / then
		assert.Equal(t, domain.NewFrameSetID(plan, slice, resolution, tuning), current)
		assert.NotEqual(t, current, next)
	})
}

func Test_ParseFrameNumber_MessageForNotWhole(t *testing.T) {
	t.Run("should quote what it was given", func(t *testing.T) {
		// given / when
		_, err := domain.ParseFrameNumber("3.5", 1260)

		// then
		assert.ErrorContains(t, err, fmt.Sprintf("%q", "3.5"))
	})
}

func Test_FrameDirectory_Plan_Resume(t *testing.T) {
	const id = domain.FrameSetID("this-set")

	t.Run("should draw every frame of an empty directory, in order, and keep none", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().Build()

		// when
		work, err := directory.Plan(id, 5, false)

		// then
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1, 2, 3, 4}, work.Draw)
		assert.Empty(t, work.Keep)
		assert.Empty(t, work.Remove)
	})

	t.Run("should keep every frame of the set that is whole, and draw nothing", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().WithOursFrames(0, 4, id).Build()

		// when
		work, err := directory.Plan(id, 5, false)

		// then
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1, 2, 3, 4}, work.Keep)
		assert.Empty(t, work.Draw)
	})

	t.Run("should keep the frames that are there and draw the ones that are not, each in order, none in both", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().
			WithOursFrame(0, id).WithOursFrame(1, id).WithOursFrame(3, id).Build()

		// when
		work, err := directory.Plan(id, 6, false)

		// then
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1, 3}, work.Keep)
		assert.Equal(t, []int{2, 4, 5}, work.Draw)
		all := append(append([]int{}, work.Keep...), work.Draw...)
		sort.Ints(all)
		assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, all)
	})

	t.Run("should draw again a frame of the set that is not whole", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().
			WithOursFrame(0, id).WithIncompleteFrame(1, id).WithOursFrame(2, id).Build()

		// when
		work, err := directory.Plan(id, 3, false)

		// then
		require.NoError(t, err)
		assert.Equal(t, []int{0, 2}, work.Keep)
		assert.Equal(t, []int{1}, work.Draw)
	})

	t.Run("should leave alone a frame of the set that has a number the plan does not", func(t *testing.T) {
		// given: it cannot happen with the same set, which has the same number of frames; it is still not the plan's to touch
		directory := builddomain.NewFrameDirectoryBuilder().WithOursFrames(0, 2, id).WithOursFrame(9, id).Build()

		// when
		work, err := directory.Plan(id, 3, false)

		// then
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1, 2}, work.Keep)
		assert.Empty(t, work.Draw)
		assert.Empty(t, work.Remove)
	})
}

func Test_FrameDirectory_Plan_Protection(t *testing.T) {
	const id = domain.FrameSetID("this-set")
	const other = domain.FrameSetID("another-set")

	t.Run("should refuse a directory with a frame of another set, whatever its number, saying how to go on", func(t *testing.T) {
		// given: one inside the plan, one outside it
		inside := builddomain.NewFrameDirectoryBuilder().WithOursFrame(1, other).Build()
		outside := builddomain.NewFrameDirectoryBuilder().WithOursFrame(9, other).Build()

		// when
		_, errInside := inside.Plan(id, 3, false)
		_, errOutside := outside.Plan(id, 3, false)

		// then
		assert.ErrorIs(t, errInside, domain.ErrFrameSetConflict)
		assert.ErrorIs(t, errOutside, domain.ErrFrameSetConflict)
		assert.ErrorContains(t, errInside, "1 frame is of another set")
		assert.ErrorContains(t, errInside, "use --overwrite to replace them, or another --output")
	})

	t.Run("should count the frames of another set in the message", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().WithOursFrames(0, 3, other).WithOursFrame(4, id).Build()

		// when
		_, err := directory.Plan(id, 5, false)

		// then
		assert.ErrorContains(t, err, "4 frames are of another set")
	})

	t.Run("should refuse a file that has the name of a frame in the plan but is not the tool's, and count it", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().WithForeignFrame(0).WithForeignFrame(2).Build()

		// when
		_, err := directory.Plan(id, 3, false)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameSetConflict)
		assert.ErrorContains(t, err, "2 files named like frames are not this tool's")
	})

	t.Run("should count both kinds of conflict", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().WithOursFrame(0, other).WithForeignFrame(1).Build()

		// when
		_, err := directory.Plan(id, 3, false)

		// then
		assert.ErrorContains(t, err, "1 frame is of another set")
		assert.ErrorContains(t, err, "1 file named like a frame is not this tool's")
	})

	t.Run("should ignore a file that is not the tool's and has a number the plan does not", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().WithForeignFrame(9).Build()

		// when
		work, err := directory.Plan(id, 3, false)

		// then
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1, 2}, work.Draw)
		assert.Empty(t, work.Remove)
	})

	t.Run("should draw all the frames with overwrite, keeping none, even those of the same set", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().WithOursFrames(0, 1, id).WithOursFrame(2, other).WithForeignFrame(0).Build()

		// when
		work, err := directory.Plan(id, 4, true)

		// then
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1, 2, 3}, work.Draw)
		assert.Empty(t, work.Keep)
	})

	t.Run("should remove, with overwrite, the frames of another set that the plan has no number for, and only those", func(t *testing.T) {
		// given: 1350 frames of a previous plan; the new one has 3
		directory := builddomain.NewFrameDirectoryBuilder().
			WithOursFrames(0, 5, other). // 3, 4, 5 are outside the new plan
			WithForeignFrame(8).         // not the tool's: never removed
			WithOursFrame(7, id).        // the same set: not touched either
			Build()

		// when
		work, err := directory.Plan(id, 3, true)

		// then
		require.NoError(t, err)
		assert.Equal(t, []int{3, 4, 5}, work.Remove)
		assert.Equal(t, []int{0, 1, 2}, work.Draw)
	})

	t.Run("should have nothing to remove when the directory holds nothing of another set outside the plan", func(t *testing.T) {
		// given
		directory := builddomain.NewFrameDirectoryBuilder().WithOursFrames(0, 2, other).Build()

		// when
		work, err := directory.Plan(id, 3, true)

		// then
		require.NoError(t, err)
		assert.Empty(t, work.Remove)
		assert.Equal(t, []int{0, 1, 2}, work.Draw)
	})
}
