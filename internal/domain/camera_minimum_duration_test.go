package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func lineOfKm(km float64) []domain.TrackPoint {
	return builddomain.NewSyntheticRouteBuilder().WithLine(km*1000, 90).WithConstantSpeed(5).Build()
}

func Test_MinimumDuration(t *testing.T) {
	tuning := defaultTuning()
	minimumFor := func(km float64) time.Duration {
		return minimumDuration(lineOfKm(km), 30, domain.LevelMedium, domain.LevelMedium, tuning)
	}

	t.Run("should be 20 seconds for a track of a few kilometers: the two second phase floor over a tenth", func(t *testing.T) {
		// when / then
		assert.Equal(t, 20*time.Second, minimumFor(0.2))
	})

	t.Run("should grow with the track's size", func(t *testing.T) {
		// when
		short, medium, long := minimumFor(5), minimumFor(20), minimumFor(100)

		// then
		assert.InDelta(t, 25.0, short.Seconds(), 1.5)
		assert.InDelta(t, 39.0, medium.Seconds(), 1.5)
		assert.InDelta(t, 55.0, long.Seconds(), 2.0)
	})

	t.Run("should stay below 120 seconds even for a track of 2000 km of span", func(t *testing.T) {
		// when
		minimum := minimumFor(1990)

		// then
		assert.Less(t, minimum.Seconds(), 120.0)
		assert.Greater(t, minimum.Seconds(), 80.0)
	})

	t.Run("should be a whole number of frames", func(t *testing.T) {
		// when
		minimum := minimumDuration(lineOfKm(20), 24, domain.LevelMedium, domain.LevelMedium, tuning)

		// then
		frames := minimum.Seconds() * 24
		assert.InDelta(t, float64(int(frames+0.5)), frames, 1e-6)
	})

	t.Run("should not depend on the duration the user chooses", func(t *testing.T) {
		// when / then: only track, frame rate, levels and tuning matter
		assert.Equal(t, minimumFor(20), minimumFor(20))
	})

	t.Run("should be smaller for a lower distance level only by the zoom the opening needs", func(t *testing.T) {
		// when
		low := minimumDuration(lineOfKm(20), 30, domain.LevelLow, domain.LevelMedium, tuning)
		high := minimumDuration(lineOfKm(20), 30, domain.LevelHigh, domain.LevelMedium, tuning)

		// then: closer flights need a larger zoom, so they need more time
		assert.Greater(t, low, high)
	})
}

func Test_PlanCamera_Duration(t *testing.T) {
	tuning := defaultTuning()
	points := lineOfKm(20)

	t.Run("should accept a duration exactly equal to the minimum and produce a smooth plan", func(t *testing.T) {
		// given
		minimum := minimumDuration(points, 30, domain.LevelMedium, domain.LevelMedium, tuning)
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(minimum).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assertSmooth(t, plan, tuning)
		assertPhaseOrder(t, plan)
	})

	t.Run("should reject a duration one frame below the minimum and mention the minimum", func(t *testing.T) {
		// given
		minimum := minimumDuration(points, 30, domain.LevelMedium, domain.LevelMedium, tuning)
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(minimum - time.Second/30).Build()

		// when
		_, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.ErrorIs(t, err, domain.ErrDurationTooShort)
		assert.Contains(t, err.Error(), "minimum for this track is 38.")
	})

	t.Run("should never apply the automatic curve to a duration the user requested", func(t *testing.T) {
		// given: 45 s where the automatic duration for this track would be 42 s
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(45 * time.Second).Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Equal(t, 45*time.Second, *plan.Parameters.Duration)
	})

	t.Run("should be smooth for every shape when the duration is automatic", func(t *testing.T) {
		for name, shape := range routeShapes() {
			// given
			parameters := builddomain.NewPlanParametersBuilder().WithoutDuration().Build()

			// when
			plan, err := treatedOf(shape).PlanCamera(parameters, tuning)

			// then
			require.NoError(t, err, name)
			assertSmooth(t, plan, tuning)
			assertPhaseOrder(t, plan)
			assertMarkerMonotonic(t, plan)
		}
	})
}

func Test_DefaultDuration_AtLeastMinimum(t *testing.T) {
	t.Run("should be the minimum, and above 120 seconds, when a restrictive tuning makes the minimum exceed the cap", func(t *testing.T) {
		// given: ten times slower zoom than the initial tuning
		tuning := builddomain.NewCameraTuningBuilder().WithMaxLogDistanceRatePerSecond(0.1).Build()
		points := lineOfKm(200)
		minimum := minimumDuration(points, 30, domain.LevelMedium, domain.LevelMedium, tuning)
		require.Greater(t, minimum, 120*time.Second)

		// when
		duration := defaultDuration(points, 30, domain.LevelMedium, domain.LevelMedium, tuning)

		// then
		assert.Equal(t, minimum, duration)
	})

	t.Run("should never produce a duration PlanCamera refuses as too short", func(t *testing.T) {
		// given
		tuning := builddomain.NewCameraTuningBuilder().WithMaxLogDistanceRatePerSecond(0.1).Build()
		points := lineOfKm(200)
		parameters := builddomain.NewPlanParametersBuilder().WithoutDuration().Build()

		// when
		plan, err := treatedOf(points).PlanCamera(parameters, tuning)

		// then
		require.NoError(t, err)
		assert.Greater(t, *plan.Parameters.Duration, 120*time.Second)
	})

	t.Run("should stay below the cap with the initial tuning for any accepted track", func(t *testing.T) {
		// given
		tuning := defaultTuning()

		for _, km := range []float64{0.06, 1, 20, 100, 1000, 1990} {
			// when
			minimum := minimumDuration(lineOfKm(km), 30, domain.LevelLow, domain.LevelLow, tuning)

			// then
			assert.Less(t, minimum, tuning.AutoDurationMax, "%v km", km)
		}
	})

	t.Run("should never decrease as the track grows, with the minimum applied", func(t *testing.T) {
		// given
		tuning := defaultTuning()
		previous := time.Duration(0)

		for _, km := range []float64{0.1, 1, 5, 20, 100, 400, 1000, 1990} {
			// when
			current := defaultDuration(lineOfKm(km), 30, domain.LevelMedium, domain.LevelMedium, tuning)

			// then
			assert.GreaterOrEqual(t, current, previous, "%v km", km)
			previous = current
		}
	})
}
