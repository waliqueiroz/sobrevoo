package domain_test

import (
	"math"
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
