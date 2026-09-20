package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_Route_ElevationGain(t *testing.T) {
	t.Run("should return not ok for an empty route", func(t *testing.T) {
		// given
		var points []domain.TrackPoint

		// when
		gain, ok := (domain.Route{Points: points}).ElevationGain()

		// then
		assert.False(t, ok)
		assert.Equal(t, 0.0, gain)
	})

	t.Run("should return not ok when no point has elevation", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithoutElevation().Build(),
			builddomain.NewTrackPointBuilder().WithoutElevation().Build(),
		}

		// when
		gain, ok := (domain.Route{Points: points}).ElevationGain()

		// then
		assert.False(t, ok)
		assert.Equal(t, 0.0, gain)
	})

	t.Run("should return not ok when some points are missing elevation", func(t *testing.T) {
		// given: a partial gain would misrepresent the real ascent (FR-019)
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithElevation(100).Build(),
			builddomain.NewTrackPointBuilder().WithoutElevation().Build(),
			builddomain.NewTrackPointBuilder().WithElevation(150).Build(),
		}

		// when
		gain, ok := (domain.Route{Points: points}).ElevationGain()

		// then
		assert.False(t, ok)
		assert.Equal(t, 0.0, gain)
	})

	t.Run("should sum every positive elevation delta when the route monotonically climbs", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithElevation(100).Build(),
			builddomain.NewTrackPointBuilder().WithElevation(150).Build(),
			builddomain.NewTrackPointBuilder().WithElevation(200).Build(),
		}

		// when
		gain, ok := (domain.Route{Points: points}).ElevationGain()

		// then
		assert.True(t, ok)
		assert.InDelta(t, 100.0, gain, 0.0001)
	})

	t.Run("should ignore descents and only sum positive deltas", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithElevation(100).Build(),
			builddomain.NewTrackPointBuilder().WithElevation(80).Build(),
			builddomain.NewTrackPointBuilder().WithElevation(120).Build(),
			builddomain.NewTrackPointBuilder().WithElevation(90).Build(),
		}

		// when
		gain, ok := (domain.Route{Points: points}).ElevationGain()

		// then: only the 80 -> 120 climb counts
		assert.True(t, ok)
		assert.InDelta(t, 40.0, gain, 0.0001)
	})

	t.Run("should return zero gain but still ok for a flat route", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithElevation(50).Build(),
			builddomain.NewTrackPointBuilder().WithElevation(50).Build(),
		}

		// when
		gain, ok := (domain.Route{Points: points}).ElevationGain()

		// then
		assert.True(t, ok)
		assert.InDelta(t, 0.0, gain, 0.0001)
	})
}
