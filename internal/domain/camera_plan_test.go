package domain_test

import (
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
