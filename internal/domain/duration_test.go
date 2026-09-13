package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/build_domain"
)

func Test_Duration(t *testing.T) {
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

	t.Run("should return not ok for an empty route", func(t *testing.T) {
		// given
		var points []domain.TrackPoint

		// when
		duration, ok := domain.Duration(points)

		// then
		assert.False(t, ok)
		assert.Equal(t, time.Duration(0), duration)
	})

	t.Run("should return not ok when no point has time", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithoutTime().Build(),
			build_domain.NewTrackPointBuilder().WithoutTime().Build(),
		}

		// when
		duration, ok := domain.Duration(points)

		// then
		assert.False(t, ok)
		assert.Equal(t, time.Duration(0), duration)
	})

	t.Run("should return not ok when some points are missing time", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithTime(start).Build(),
			build_domain.NewTrackPointBuilder().WithoutTime().Build(),
		}

		// when
		duration, ok := domain.Duration(points)

		// then
		assert.False(t, ok)
		assert.Equal(t, time.Duration(0), duration)
	})

	t.Run("should return the elapsed time between the first and the last point when every point has time", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithTime(start).Build(),
			build_domain.NewTrackPointBuilder().WithTime(start.Add(30 * time.Minute)).Build(),
			build_domain.NewTrackPointBuilder().WithTime(start.Add(45 * time.Minute)).Build(),
		}

		// when
		duration, ok := domain.Duration(points)

		// then
		assert.True(t, ok)
		assert.Equal(t, 45*time.Minute, duration)
	})
}
