package domain_test

import (
	"math"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_NewCameraPlan(t *testing.T) {
	t.Run("should compute the summary ranges from the frames", func(t *testing.T) {
		// given
		frames := []domain.CameraFrame{
			builddomain.NewCameraFrameBuilder().WithCameraAltitude(500).WithCameraToMarkerDistance(900).Build(),
			builddomain.NewCameraFrameBuilder().WithCameraAltitude(120).WithCameraToMarkerDistance(300).Build(),
			builddomain.NewCameraFrameBuilder().WithCameraAltitude(340).WithCameraToMarkerDistance(1200).Build(),
		}

		// when
		plan := builddomain.NewCameraPlanBuilder().WithFrames(frames...).Build()

		// then
		assert.Equal(t, 120.0, plan.Summary.MinCameraAltitude)
		assert.Equal(t, 500.0, plan.Summary.MaxCameraAltitude)
		assert.Equal(t, 300.0, plan.Summary.MinCameraDistance)
		assert.Equal(t, 1200.0, plan.Summary.MaxCameraDistance)
		assert.Equal(t, 3, plan.Summary.FrameCount)
	})

	t.Run("should copy the parameters, the duration mode and the time reference into the summary", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(42 * time.Second).WithFrameRate(24).Build()

		// when
		plan := builddomain.NewCameraPlanBuilder().
			WithParameters(parameters).
			WithDurationMode(domain.DurationModeAutomatic).
			WithTimeReference(domain.TimeReferenceDistance, "no time data").
			Build()

		// then
		assert.Equal(t, 42*time.Second, plan.Summary.Duration)
		assert.Equal(t, 24.0, plan.Summary.FrameRate)
		assert.Equal(t, domain.DurationModeAutomatic, plan.Summary.DurationMode)
		assert.Equal(t, domain.TimeReferenceDistance, plan.Summary.TimeReference)
		assert.Equal(t, "no time data", plan.TimeFallbackReason)
		assert.Equal(t, domain.TimeReferenceDistance, plan.TimeReference)
	})

	t.Run("should leave the duration zero when the parameters carry none", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithoutDuration().Build()

		// when
		plan := builddomain.NewCameraPlanBuilder().WithParameters(parameters).Build()

		// then
		assert.Zero(t, plan.Summary.Duration)
	})

	t.Run("should keep the smoothed spans in the order received, and an empty list when there are none", func(t *testing.T) {
		// given
		spans := []domain.SmoothedSpan{
			{Start: time.Second, End: 2 * time.Second, Quantity: domain.QuantityHeading},
			{Start: 5 * time.Second, End: 6 * time.Second, Quantity: domain.QuantityZoom},
		}

		// when
		with := builddomain.NewCameraPlanBuilder().WithSmoothedSpans(spans...).Build()
		without := builddomain.NewCameraPlanBuilder().Build()

		// then
		assert.Equal(t, spans, with.Summary.SmoothedSpans)
		assert.NotNil(t, without.Summary.SmoothedSpans)
		assert.Empty(t, without.Summary.SmoothedSpans)
	})

	t.Run("should produce a zeroed range summary for a plan without frames", func(t *testing.T) {
		// when
		plan := builddomain.NewCameraPlanBuilder().WithFrames().Build()

		// then
		assert.Zero(t, plan.Summary.MaxCameraAltitude)
		assert.Zero(t, plan.Summary.FrameCount)
	})

	t.Run("should copy elevationAvailable into the plan and into the summary", func(t *testing.T) {
		// given / when
		with := builddomain.NewCameraPlanBuilder().WithElevationAvailable(true).Build()
		without := builddomain.NewCameraPlanBuilder().WithElevationAvailable(false).Build()

		// then
		assert.True(t, with.ElevationAvailable)
		assert.True(t, with.Summary.ElevationAvailable)
		assert.False(t, without.ElevationAvailable)
		assert.False(t, without.Summary.ElevationAvailable)
	})
}

func Test_CameraPlan_Validate(t *testing.T) {
	// a plan of three frames: 0.1 s at 30 frames per second
	validPlan := func() *builddomain.CameraPlanBuilder {
		return builddomain.NewCameraPlanBuilder()
	}

	t.Run("should accept a coherent plan", func(t *testing.T) {
		// given
		plan := validPlan().Build()

		// when
		err := plan.Validate()

		// then
		assert.NoError(t, err)
	})

	t.Run("should refuse a plan without frames", func(t *testing.T) {
		// given
		plan := validPlan().WithFrames().Build()

		// when
		err := plan.Validate()

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
	})

	t.Run("should refuse a plan whose frame count does not match the duration times the frame rate", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(41 * time.Second).WithFrameRate(30).Build()
		plan := validPlan().WithParameters(parameters).Build()

		// when
		err := plan.Validate()

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "3 frames but duration × frame rate is 1230")
	})

	t.Run("should refuse a plan without a duration", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithoutDuration().Build()
		plan := validPlan().WithParameters(parameters).Build()

		// when
		err := plan.Validate()

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
	})

	t.Run("should refuse a frame whose index is not its position", func(t *testing.T) {
		// given
		plan := validPlan().WithFrames(
			builddomain.NewCameraFrameBuilder().WithIndex(0).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(5).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(2).Build(),
		).Build()

		// when
		err := plan.Validate()

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[1].index")
	})

	t.Run("should refuse a frame with an unknown phase", func(t *testing.T) {
		// given
		plan := validPlan().WithFrames(
			builddomain.NewCameraFrameBuilder().WithIndex(0).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(1).WithPhase("hovering").Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(2).Build(),
		).Build()

		// when
		err := plan.Validate()

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[1].phase")
	})

	t.Run("should refuse a camera latitude outside -90 to 90", func(t *testing.T) {
		// given
		plan := validPlan().WithFrames(
			builddomain.NewCameraFrameBuilder().WithIndex(0).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(1).WithCameraPosition(90.5, 0).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(2).Build(),
		).Build()

		// when
		err := plan.Validate()

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[1].camera.lat")
	})

	t.Run("should refuse a marker longitude outside -180 to 180", func(t *testing.T) {
		// given
		plan := validPlan().WithFrames(
			builddomain.NewCameraFrameBuilder().WithIndex(0).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(1).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(2).WithMarkerPosition(0, -181).Build(),
		).Build()

		// when
		err := plan.Validate()

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[2].marker.lon")
	})

	t.Run("should refuse a negative camera-to-marker distance", func(t *testing.T) {
		// given
		plan := validPlan().WithFrames(
			builddomain.NewCameraFrameBuilder().WithIndex(0).WithCameraToMarkerDistance(-1).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(1).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(2).Build(),
		).Build()

		// when
		err := plan.Validate()

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[0].camera_to_marker_m")
	})

	t.Run("should refuse a camera-to-marker distance that is not a finite number", func(t *testing.T) {
		// given
		plan := validPlan().WithFrames(
			builddomain.NewCameraFrameBuilder().WithIndex(0).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(1).WithCameraToMarkerDistance(math.NaN()).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(2).WithCameraToMarkerDistance(math.Inf(1)).Build(),
		).Build()

		// when
		err := plan.Validate()

		// then
		assert.ErrorIs(t, err, domain.ErrPlanFileInvalid)
		assert.ErrorContains(t, err, "frames[1].camera_to_marker_m")
	})
}

func Test_CameraPlan_ID(t *testing.T) {
	frames := func(mutate func(f *domain.CameraFrame)) []domain.CameraFrame {
		list := []domain.CameraFrame{
			builddomain.NewCameraFrameBuilder().WithIndex(0).WithPhase(domain.PhaseOpening).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(1).WithPhase(domain.PhaseFollowing).Build(),
			builddomain.NewCameraFrameBuilder().WithIndex(2).WithPhase(domain.PhaseClosing).Build(),
		}
		if mutate != nil {
			mutate(&list[1])
		}
		return list
	}
	planWith := func(mutate func(f *domain.CameraFrame)) domain.CameraPlan {
		return builddomain.NewCameraPlanBuilder().WithFrames(frames(mutate)...).Build()
	}

	t.Run("should be 64 lowercase hexadecimal characters", func(t *testing.T) {
		// given
		plan := planWith(nil)

		// when
		id := plan.ID()

		// then
		assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{64}$`), id)
	})

	t.Run("should be the same for two plans with the same content", func(t *testing.T) {
		// given
		first, second := planWith(nil), planWith(nil)

		// when / then
		assert.Equal(t, first.ID(), second.ID())
	})

	t.Run("should be the same for values that differ by less than the step of the plan", func(t *testing.T) {
		// given: a plan read back from a file holds values rounded to the plan's steps
		exact := planWith(nil)
		noisy := planWith(func(f *domain.CameraFrame) {
			f.CameraLatitude += 1e-9
			f.CameraLongitude -= 1e-9
			f.MarkerLatitude += 1e-9
			f.CameraAltitude += 1e-5
			f.Heading += 1e-5
			f.Tilt -= 1e-5
			f.MarkerDistance += 1e-5
			f.CameraToMarkerDistance -= 1e-5
		})

		// when / then
		assert.Equal(t, exact.ID(), noisy.ID())
	})

	t.Run("should change when one value of one frame changes by one step", func(t *testing.T) {
		// given
		base := planWith(nil).ID()

		// when / then
		assert.NotEqual(t, base, planWith(func(f *domain.CameraFrame) { f.CameraLatitude += 1e-7 }).ID(), "camera latitude")
		assert.NotEqual(t, base, planWith(func(f *domain.CameraFrame) { f.CameraLongitude += 1e-7 }).ID(), "camera longitude")
		assert.NotEqual(t, base, planWith(func(f *domain.CameraFrame) { f.MarkerLatitude += 1e-7 }).ID(), "marker latitude")
		assert.NotEqual(t, base, planWith(func(f *domain.CameraFrame) { f.MarkerLongitude += 1e-7 }).ID(), "marker longitude")
		assert.NotEqual(t, base, planWith(func(f *domain.CameraFrame) { f.CameraAltitude += 1e-3 }).ID(), "altitude")
		assert.NotEqual(t, base, planWith(func(f *domain.CameraFrame) { f.Heading += 1e-3 }).ID(), "heading")
		assert.NotEqual(t, base, planWith(func(f *domain.CameraFrame) { f.Tilt += 1e-3 }).ID(), "tilt")
		assert.NotEqual(t, base, planWith(func(f *domain.CameraFrame) { f.MarkerDistance += 1e-3 }).ID(), "marker distance")
		assert.NotEqual(t, base, planWith(func(f *domain.CameraFrame) { f.CameraToMarkerDistance += 1e-3 }).ID(), "camera to marker distance")
	})

	t.Run("should change when the index or the phase of a frame changes", func(t *testing.T) {
		// given
		base := planWith(nil).ID()

		// when / then
		assert.NotEqual(t, base, planWith(func(f *domain.CameraFrame) { f.Index = 7 }).ID(), "index")
		assert.NotEqual(t, base, planWith(func(f *domain.CameraFrame) { f.Phase = domain.PhaseClosing }).ID(), "phase")
	})

	t.Run("should change when a parameter changes", func(t *testing.T) {
		// given
		base := builddomain.NewCameraPlanBuilder().Build().ID()
		with := func(parameters domain.PlanParameters) string {
			return builddomain.NewCameraPlanBuilder().WithParameters(parameters).Build().ID()
		}
		builder := func() *builddomain.PlanParametersBuilder {
			return builddomain.NewPlanParametersBuilder().WithDuration(100 * time.Millisecond).WithFrameRate(30)
		}

		// when / then
		assert.NotEqual(t, base, with(builder().WithFrameRate(24).Build()), "frame rate")
		assert.NotEqual(t, base, with(builder().WithDuration(133*time.Millisecond).Build()), "duration")
		assert.NotEqual(t, base, with(builder().WithDistance(domain.LevelHigh).Build()), "distance level")
		assert.NotEqual(t, base, with(builder().WithTilt(domain.LevelLow).Build()), "tilt level")
	})

	t.Run("should change when a frame is added", func(t *testing.T) {
		// given
		base := planWith(nil)
		longer := builddomain.NewCameraPlanBuilder().WithFrames(append(frames(nil), builddomain.NewCameraFrameBuilder().WithIndex(3).Build())...).Build()

		// when / then
		assert.NotEqual(t, base.ID(), longer.ID())
	})

	t.Run("should not depend on the summary, which is derived from the frames", func(t *testing.T) {
		// given
		plan := planWith(nil)
		other := plan
		other.Summary.MaxCameraAltitude += 1000
		other.TimeFallbackReason = "something else"
		other.Summary.SmoothedSpans = []domain.SmoothedSpan{{Quantity: domain.QuantityHeading}}

		// when / then
		assert.Equal(t, plan.ID(), other.ID())
	})

	t.Run("should not depend on ActivityElapsed, TrackElevation or TrackElevationGain, which are pure functions of what already identifies the plan (research.md item 9)", func(t *testing.T) {
		// given
		plan := planWith(func(f *domain.CameraFrame) {
			f.ActivityElapsed = 0
			f.TrackElevation = 0
			f.TrackElevationGain = 0
		})
		other := planWith(func(f *domain.CameraFrame) {
			f.ActivityElapsed = 90 * time.Second
			f.TrackElevation = 842.5
			f.TrackElevationGain = 120.3
		})

		// when / then
		assert.Equal(t, plan.ID(), other.ID())
	})

	t.Run("should not depend on ElevationAvailable", func(t *testing.T) {
		// given
		with := builddomain.NewCameraPlanBuilder().WithElevationAvailable(true).Build()
		without := builddomain.NewCameraPlanBuilder().WithElevationAvailable(false).Build()

		// when / then
		assert.Equal(t, with.ID(), without.ID())
	})
}
