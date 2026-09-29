package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_Route_Duration(t *testing.T) {
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

	t.Run("should return not ok for an empty route", func(t *testing.T) {
		// given
		var points []domain.TrackPoint

		// when
		duration, ok := (domain.Route{Points: points}).Duration()

		// then
		assert.False(t, ok)
		assert.Equal(t, time.Duration(0), duration)
	})

	t.Run("should return not ok when no point has time", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithoutTime().Build(),
			builddomain.NewTrackPointBuilder().WithoutTime().Build(),
		}

		// when
		duration, ok := (domain.Route{Points: points}).Duration()

		// then
		assert.False(t, ok)
		assert.Equal(t, time.Duration(0), duration)
	})

	t.Run("should return not ok when some points are missing time", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithoutTime().Build(),
		}

		// when
		duration, ok := (domain.Route{Points: points}).Duration()

		// then
		assert.False(t, ok)
		assert.Equal(t, time.Duration(0), duration)
	})

	t.Run("should return the elapsed time between the first and the last point when every point has time", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithTime(start.Add(30 * time.Minute)).Build(),
			builddomain.NewTrackPointBuilder().WithTime(start.Add(45 * time.Minute)).Build(),
		}

		// when
		duration, ok := (domain.Route{Points: points}).Duration()

		// then
		assert.True(t, ok)
		assert.Equal(t, 45*time.Minute, duration)
	})
}

func Test_Route_TimeAt(t *testing.T) {
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

	t.Run("should return not ok when some points are missing time", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithoutTime().Build(),
		}
		distances := []float64{0, 100}

		// when
		elapsed, ok := (domain.Route{Points: points}).TimeAt(distances, 50)

		// then
		assert.False(t, ok)
		assert.Equal(t, time.Duration(0), elapsed)
	})

	t.Run("should return zero at the first point", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithTime(start.Add(10 * time.Minute)).Build(),
		}
		distances := []float64{0, 1000}

		// when
		elapsed, ok := (domain.Route{Points: points}).TimeAt(distances, 0)

		// then
		assert.True(t, ok)
		assert.Equal(t, time.Duration(0), elapsed)
	})

	t.Run("should return the total duration at the last point", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithTime(start.Add(7 * time.Minute)).Build(),
			builddomain.NewTrackPointBuilder().WithTime(start.Add(22 * time.Minute)).Build(),
		}
		distances := []float64{0, 400, 1000}

		// when
		elapsed, ok := (domain.Route{Points: points}).TimeAt(distances, 1000)

		// then
		assert.True(t, ok)
		assert.Equal(t, 22*time.Minute, elapsed)
	})

	t.Run("should interpolate linearly between two points", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithTime(start.Add(10 * time.Minute)).Build(),
		}
		distances := []float64{0, 1000}

		// when
		elapsed, ok := (domain.Route{Points: points}).TimeAt(distances, 250)

		// then: a quarter of the way, in distance, is a quarter of the way in time
		assert.True(t, ok)
		assert.Equal(t, 150*time.Second, elapsed)
	})

	t.Run("should clamp a distance beyond the end to the last point", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithTime(start.Add(5 * time.Minute)).Build(),
		}
		distances := []float64{0, 1000}

		// when
		elapsed, ok := (domain.Route{Points: points}).TimeAt(distances, 5000)

		// then
		assert.True(t, ok)
		assert.Equal(t, 5*time.Minute, elapsed)
	})
}
