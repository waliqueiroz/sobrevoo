package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_Route_ElevationProfile(t *testing.T) {
	t.Run("should return not ok when some points are missing elevation", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithElevation(100).Build(),
			builddomain.NewTrackPointBuilder().WithoutElevation().Build(),
		}
		distances := []float64{0, 100}

		// when
		_, ok := (domain.Route{Points: points}).ElevationProfile(distances)

		// then
		assert.False(t, ok)
	})
}

func Test_ElevationProfile_At(t *testing.T) {
	// A non-monotonic profile: up, down, up — so the gain at the end can
	// only be reproduced by a prefix sum over the route's own points, never
	// by resampling at coarser distances (research.md item 7).
	points := []domain.TrackPoint{
		builddomain.NewTrackPointBuilder().WithElevation(100).Build(), // 0 m
		builddomain.NewTrackPointBuilder().WithElevation(150).Build(), // 400 m, +50
		builddomain.NewTrackPointBuilder().WithElevation(120).Build(), // 700 m, -30
		builddomain.NewTrackPointBuilder().WithElevation(180).Build(), // 1000 m, +60
	}
	distances := []float64{0, 400, 700, 1000}

	t.Run("should return the elevation and zero gain at the first point", func(t *testing.T) {
		// given
		profile, ok := (domain.Route{Points: points}).ElevationProfile(distances)
		require.True(t, ok)

		// when
		elevation, gain := profile.At(0)

		// then
		assert.Equal(t, 100.0, elevation)
		assert.Equal(t, 0.0, gain)
	})

	t.Run("should return the elevation and the total gain at the last point, matching Route.ElevationGain bit for bit", func(t *testing.T) {
		// given
		profile, ok := (domain.Route{Points: points}).ElevationProfile(distances)
		require.True(t, ok)
		expectedGain, ok := (domain.Route{Points: points}).ElevationGain()
		require.True(t, ok)

		// when
		elevation, gain := profile.At(1000)

		// then
		assert.Equal(t, 180.0, elevation)
		assert.Equal(t, expectedGain, gain)
	})

	t.Run("should interpolate the elevation linearly within a climbing segment", func(t *testing.T) {
		// given
		profile, ok := (domain.Route{Points: points}).ElevationProfile(distances)
		require.True(t, ok)

		// when: halfway from 0m (100m elevation) to 400m (150m elevation)
		elevation, _ := profile.At(200)

		// then
		assert.InDelta(t, 125.0, elevation, 1e-9)
	})

	t.Run("should hold the gain steady while descending", func(t *testing.T) {
		// given: the 400m->700m segment descends, so it adds nothing to the gain
		profile, ok := (domain.Route{Points: points}).ElevationProfile(distances)
		require.True(t, ok)

		// when
		_, gainAtDescentStart := profile.At(400)
		_, gainMidDescent := profile.At(550)
		_, gainAtDescentEnd := profile.At(700)

		// then
		assert.Equal(t, 50.0, gainAtDescentStart)
		assert.Equal(t, 50.0, gainMidDescent)
		assert.Equal(t, 50.0, gainAtDescentEnd)
	})

	t.Run("should never decrease as distance grows", func(t *testing.T) {
		// given
		profile, ok := (domain.Route{Points: points}).ElevationProfile(distances)
		require.True(t, ok)

		// when
		var previous float64
		var gains []float64
		for _, d := range []float64{0, 100, 300, 400, 500, 700, 850, 1000} {
			_, gain := profile.At(d)
			gains = append(gains, gain)
		}

		// then
		for _, gain := range gains {
			assert.GreaterOrEqual(t, gain, previous)
			previous = gain
		}
	})

	t.Run("should clamp a distance beyond the end to the last point", func(t *testing.T) {
		// given
		profile, ok := (domain.Route{Points: points}).ElevationProfile(distances)
		require.True(t, ok)

		// when
		elevation, gain := profile.At(5000)

		// then
		assert.Equal(t, 180.0, elevation)
		assert.Equal(t, 110.0, gain) // 50 (0->400) + 0 (400->700) + 60 (700->1000)
	})
}
