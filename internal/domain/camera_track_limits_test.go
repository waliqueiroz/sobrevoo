package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_PlanCamera_TrackTooShort(t *testing.T) {
	tuning := defaultTuning()
	parameters := builddomain.NewPlanParametersBuilder().WithoutDuration().Build()

	t.Run("should reject a track shorter than the minimum length and report both lengths", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(40, 90).Build()

		// when
		_, err := domain.PlanCamera(treatedOf(points), parameters, tuning)

		// then
		require.ErrorIs(t, err, domain.ErrTrackTooShort)
		assert.Contains(t, err.Error(), "length is 40.0 m")
		assert.Contains(t, err.Error(), "minimum is 50.0 m")
	})

	t.Run("should accept a track exactly at the minimum length", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(50, 90).Build()
		exact := tuning
		exact.MinTrackLengthMeters = domain.TotalDistance(points)

		// when
		_, err := domain.PlanCamera(treatedOf(points), parameters, exact)

		// then
		assert.NoError(t, err)
	})

	t.Run("should accept a track just above the minimum length", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(60, 90).Build()

		// when
		plan, err := domain.PlanCamera(treatedOf(points), parameters, tuning)

		// then
		require.NoError(t, err)
		assertSmooth(t, plan, tuning)
	})
}

func Test_PlanCamera_TrackTooLarge(t *testing.T) {
	tuning := defaultTuning()
	parameters := builddomain.NewPlanParametersBuilder().WithoutDuration().Build()

	t.Run("should reject a track whose span exceeds the maximum and report both", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(2_100_000, 90).WithPointCount(400).Build()

		// when
		_, err := domain.PlanCamera(treatedOf(points), parameters, tuning)

		// then
		require.ErrorIs(t, err, domain.ErrTrackTooLarge)
		assert.Contains(t, err.Error(), "span is 2100")
		assert.Contains(t, err.Error(), "maximum is 2000.0 km")
	})

	t.Run("should accept a track just within the maximum span", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(1_990_000, 90).WithPointCount(400).Build()

		// when
		plan, err := domain.PlanCamera(treatedOf(points), parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Equal(t, 120.0, plan.Summary.Duration.Seconds())
	})

	t.Run("should accept a very long track of small span, since the limit is on the span, not on the length", func(t *testing.T) {
		// given: 100 laps of a 100 m radius circle, 63 km long but 200 m across
		points := builddomain.NewSyntheticRouteBuilder().WithCircle(100, 100).Build()

		// when
		plan, err := domain.PlanCamera(treatedOf(points), parameters, tuning)

		// then
		require.NoError(t, err)
		assertMarkerMonotonic(t, plan)
	})
}
