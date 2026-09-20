package domain_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_PlanParameters_Validate(t *testing.T) {
	t.Run("should accept an automatic duration (nil) with a valid frame rate", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithoutDuration().Build()

		// when
		err := parameters.Validate()

		// then
		assert.NoError(t, err)
	})

	t.Run("should reject a zero duration", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(0).Build()

		// when
		err := parameters.Validate()

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidDuration)
	})

	t.Run("should reject a negative duration", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(-time.Second).Build()

		// when
		err := parameters.Validate()

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidDuration)
	})

	t.Run("should reject a duration above the maximum and mention the maximum", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(time.Hour + time.Second).Build()

		// when
		err := parameters.Validate()

		// then
		require.ErrorIs(t, err, domain.ErrInvalidDuration)
		assert.Contains(t, err.Error(), "1h0m0s")
	})

	t.Run("should accept a duration of exactly one hour", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(time.Hour).Build()

		// when
		err := parameters.Validate()

		// then
		assert.NoError(t, err)
	})

	t.Run("should reject frame rates that are not positive, are out of range or are not finite", func(t *testing.T) {
		// given
		for _, rate := range []float64{0, -30, 0.999, 120.001, math.NaN(), math.Inf(1), math.Inf(-1)} {
			parameters := builddomain.NewPlanParametersBuilder().WithFrameRate(rate).Build()

			// when
			err := parameters.Validate()

			// then
			assert.ErrorIs(t, err, domain.ErrInvalidFrameRate, "rate %v", rate)
		}
	})

	t.Run("should mention the valid frame rate range in the error", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithFrameRate(200).Build()

		// when
		err := parameters.Validate()

		// then
		require.ErrorIs(t, err, domain.ErrInvalidFrameRate)
		assert.Contains(t, err.Error(), "between 1 and 120")
	})

	t.Run("should accept the frame rates at both ends of the range", func(t *testing.T) {
		// given
		low := builddomain.NewPlanParametersBuilder().WithFrameRate(1).Build()
		high := builddomain.NewPlanParametersBuilder().WithFrameRate(120).Build()

		// when
		lowErr, highErr := low.Validate(), high.Validate()

		// then
		assert.NoError(t, lowErr)
		assert.NoError(t, highErr)
	})

	t.Run("should validate the frame rate even when the duration is automatic", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithoutDuration().WithFrameRate(0).Build()

		// when
		err := parameters.Validate()

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidFrameRate)
	})
}

func Test_PlanParameters_FrameCount(t *testing.T) {
	t.Run("should multiply the duration by the frame rate", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithFrameRate(30).Build()

		// when
		count := parameters.FrameCount(60 * time.Second)

		// then
		assert.Equal(t, 1800, count)
	})

	t.Run("should round a fractional product to the nearest whole frame", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithFrameRate(29.97).Build()

		// when
		count := parameters.FrameCount(45500 * time.Millisecond)

		// then
		assert.Equal(t, 1364, count) // 45.5 × 29.97 = 1363.635
	})

	t.Run("should round a product ending in one half up", func(t *testing.T) {
		// given
		parameters := builddomain.NewPlanParametersBuilder().WithFrameRate(1).Build()

		// when
		count := parameters.FrameCount(2500 * time.Millisecond)

		// then
		assert.Equal(t, 3, count)
	})
}
