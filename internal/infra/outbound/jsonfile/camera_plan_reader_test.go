package jsonfile_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/jsonfile"
	"github.com/waliqueiroz/sobrevoo/test/helper"
)

func writePlanFile(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plan.json")
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func Test_CameraPlanReader_Read(t *testing.T) {
	t.Run("should read the frames, the parameters and the summary of a plan file", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.ValidPlanFile(helper.DefaultPlanFileSpec()))

		// when
		plan, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.NoError(t, err)
		require.Len(t, plan.Frames, 3)
		assert.Equal(t, 30.0, plan.Parameters.FrameRate)
		require.NotNil(t, plan.Parameters.Duration)
		assert.Equal(t, 100*time.Millisecond, *plan.Parameters.Duration)
		assert.Equal(t, domain.LevelMedium, plan.Parameters.Distance)
		// The fixture's parameters have no "simplification"/"smoothing" keys
		// at all — the same shape a plan file from before this feature has.
		// Reading that absence as medium is the Clarification session's
		// decision (spec.md), already satisfied by parseLevel's existing
		// "unknown or empty text reads as medium" rule.
		assert.Equal(t, domain.LevelMedium, plan.Parameters.Simplification)
		assert.Equal(t, domain.LevelMedium, plan.Parameters.Smoothing)
		assert.Equal(t, domain.TimeReferenceClock, plan.TimeReference)
		assert.Equal(t, domain.DurationModeAutomatic, plan.Summary.DurationMode)
		assert.Equal(t, 3, plan.Summary.FrameCount)
		assert.True(t, plan.ElevationAvailable)
		assert.True(t, plan.Summary.ElevationAvailable)

		second := plan.Frames[1]
		assert.Equal(t, 1, second.Index)
		assert.Equal(t, domain.PhaseFollowing, second.Phase)
		assert.InDelta(t, -23.499, second.MarkerLatitude, 1e-9)
		assert.InDelta(t, -46.6, second.MarkerLongitude, 1e-9)
		assert.InDelta(t, -23.504, second.CameraLatitude, 1e-9)
		assert.InDelta(t, 111.0, second.MarkerDistance, 1e-9)
		assert.InDelta(t, 600.0, second.CameraToMarkerDistance, 1e-9)
		assert.InDelta(t, float64(time.Second/30), float64(second.Time), float64(time.Millisecond))
		assert.InDelta(t, 22.2, second.ActivityElapsed.Seconds(), 1e-9)
		assert.InDelta(t, 771.1, second.TrackElevation, 1e-9)
		assert.InDelta(t, 11.1, second.TrackElevationGain, 1e-9)
		assert.InDelta(t, 5.0, second.MarkerSpeed, 1e-9)
		assert.Equal(t, 400.0, plan.Summary.MaxCameraAltitude)
		assert.NoError(t, plan.Validate())
	})

	t.Run("should read back what the exporter wrote, within the precision of the file", func(t *testing.T) {
		// given
		original := examplePlan()
		path, _ := exportToTemp(t, original)

		// when
		plan, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.NoError(t, err)
		require.Len(t, plan.Frames, len(original.Frames))
		assert.Equal(t, original.Parameters.Distance, plan.Parameters.Distance)
		assert.Equal(t, original.Parameters.Tilt, plan.Parameters.Tilt)
		assert.Equal(t, original.Parameters.Simplification, plan.Parameters.Simplification)
		assert.Equal(t, original.Parameters.Smoothing, plan.Parameters.Smoothing)
		assert.Equal(t, *original.Parameters.Duration, *plan.Parameters.Duration)
		assert.Equal(t, original.TimeReference, plan.TimeReference)
		assert.Equal(t, original.TimeFallbackReason, plan.TimeFallbackReason)
		assert.Equal(t, original.Summary.DurationMode, plan.Summary.DurationMode)
		assert.Equal(t, original.Summary.SmoothedSpans, plan.Summary.SmoothedSpans)
		assert.Equal(t, original.ElevationAvailable, plan.ElevationAvailable)
		assert.Equal(t, original.Summary.ElevationAvailable, plan.Summary.ElevationAvailable)
		for i, frame := range original.Frames {
			got := plan.Frames[i]
			assert.Equal(t, frame.Index, got.Index)
			assert.Equal(t, frame.Phase, got.Phase)
			assert.Equal(t, frame.Time, got.Time)
			assert.InDelta(t, frame.CameraLatitude, got.CameraLatitude, 1e-7)
			assert.InDelta(t, frame.CameraLongitude, got.CameraLongitude, 1e-7)
			assert.InDelta(t, frame.CameraAltitude, got.CameraAltitude, 1e-3)
			assert.InDelta(t, frame.Heading, got.Heading, 1e-3)
			assert.InDelta(t, frame.Tilt, got.Tilt, 1e-3)
			assert.InDelta(t, frame.MarkerLatitude, got.MarkerLatitude, 1e-7)
			assert.InDelta(t, frame.MarkerLongitude, got.MarkerLongitude, 1e-7)
			assert.InDelta(t, frame.MarkerDistance, got.MarkerDistance, 1e-3)
			assert.InDelta(t, frame.CameraToMarkerDistance, got.CameraToMarkerDistance, 1e-3)
			assert.InDelta(t, frame.ActivityElapsed.Seconds(), got.ActivityElapsed.Seconds(), 1e-9)
			assert.InDelta(t, frame.TrackElevation, got.TrackElevation, 1e-3)
			assert.InDelta(t, frame.TrackElevationGain, got.TrackElevationGain, 1e-3)
			assert.InDelta(t, frame.MarkerSpeed, got.MarkerSpeed, 1e-3)
		}
	})

	t.Run("should ignore fields it does not know", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithUnknownField())

		// when
		plan, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.NoError(t, err)
		assert.Len(t, plan.Frames, 3)
	})

	t.Run("should refuse a format version it does not know, saying which it found and which it accepts", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithVersion(4))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFormatVersionUnsupported)
		assert.ErrorContains(t, err, "found 4, accepted: 3")
	})

	t.Run("should refuse a plan of the previous format version, telling the user to generate it again", func(t *testing.T) {
		// given: a plan made before this stage, without the per-frame
		// marker speed (014-speed-overlay-block FR-013)
		path := writePlanFile(t, helper.PlanFileWithVersion(2))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFormatVersionUnsupported)
		assert.ErrorContains(t, err, "found 2, accepted: 3")
		assert.ErrorContains(t, err, "generate the plan again")
	})

	t.Run("should refuse a plan from two format versions back the same way", func(t *testing.T) {
		// given: a plan made before even the previous stage
		// (009-frame-overlays), without the per-frame activity time and
		// track elevation either
		path := writePlanFile(t, helper.PlanFileWithVersion(1))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFormatVersionUnsupported)
		assert.ErrorContains(t, err, "found 1, accepted: 3")
		assert.ErrorContains(t, err, "generate the plan again")
	})

	t.Run("should refuse a file without a format version", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithoutFormatVersion())

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "format_version")
	})

	t.Run("should refuse content that is not JSON", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.NotJSONContent())

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
	})

	t.Run("should refuse a truncated file", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.TruncatedPlanFile())

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
	})

	t.Run("should refuse JSON that is not a plan", func(t *testing.T) {
		// given
		path := writePlanFile(t, []byte(`{"a": 1}`))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
	})

	t.Run("should name the missing marker of a frame", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithoutFrameField("marker"))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[1].marker")
	})

	t.Run("should name the missing camera of a frame", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithoutFrameField("camera"))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[1].camera")
	})

	t.Run("should name the missing camera_to_marker_m of a frame", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithoutFrameField("camera_to_marker_m"))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[1].camera_to_marker_m")
	})

	t.Run("should name a frame missing activity_time_s", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithoutFrameField("activity_time_s"))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[1].activity_time_s")
	})

	t.Run("should name a marker missing elevation_m", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithoutMarkerField("elevation_m"))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[1].marker.elevation_m")
	})

	t.Run("should name a marker missing gain_m", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithoutMarkerField("gain_m"))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[1].marker.gain_m")
	})

	t.Run("should name a marker missing speed_mps", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithoutMarkerField("speed_mps"))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[1].marker.speed_mps")
	})

	t.Run("should name a summary missing elevation_available", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithoutSummaryField("elevation_available"))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "summary.elevation_available")
	})

	t.Run("should read the aspect ratio of the plan", func(t *testing.T) {
		// given
		spec := helper.DefaultPlanFileSpec()
		spec.AspectRatio = "9:16"
		path := writePlanFile(t, helper.ValidPlanFile(spec))

		// when
		plan, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.AspectRatio{Width: 9, Height: 16}, plan.Parameters.Aspect)
	})

	t.Run("should read a plan made before the aspect ratio existed as 16:9", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.ValidPlanFile(helper.DefaultPlanFileSpec()))

		// when
		plan, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.LandscapeAspectRatio, plan.Parameters.Aspect)
	})

	t.Run("should refuse an aspect ratio that is not valid, naming the field", func(t *testing.T) {
		// given
		spec := helper.DefaultPlanFileSpec()
		spec.AspectRatio = "tall"
		path := writePlanFile(t, helper.ValidPlanFile(spec))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "parameters.aspect_ratio")
	})

	t.Run("should refuse a file without parameters", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithoutSection("parameters"))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "parameters")
	})

	t.Run("should refuse a file without a summary", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithoutSection("summary"))

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "summary")
	})

	t.Run("should refuse a summary without a frame count", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithoutFrameCount())

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "summary.frame_count")
	})

	t.Run("should refuse a summary whose frame count is not the number of frames listed", func(t *testing.T) {
		// given
		path := writePlanFile(t, helper.PlanFileWithFrameCountMismatch())

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "summary.frame_count is 7 but the file lists 3 frames")
	})

	t.Run("should report a file that does not exist as an error without a domain sentinel", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "missing.json")

		// when
		_, err := jsonfile.NewCameraPlanReader().Read(path)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.NotErrorIs(t, err, domain.ErrPlanFileInvalid)
	})
}
