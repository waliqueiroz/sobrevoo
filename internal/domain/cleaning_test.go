package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_ReorderByTime(t *testing.T) {
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

	t.Run("should sort the points chronologically when every point has time", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(3).WithTime(start.Add(2 * time.Minute)).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(1).WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(2).WithTime(start.Add(time.Minute)).Build(),
		}

		// when
		result := domain.ReorderByTime(points)

		// then
		assert.Equal(t, []float64{1, 2, 3}, latitudesOf(result))
	})

	t.Run("should leave the points unchanged when some are missing time", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(3).WithTime(start.Add(2 * time.Minute)).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(1).WithoutTime().Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(2).WithTime(start.Add(time.Minute)).Build(),
		}

		// when
		result := domain.ReorderByTime(points)

		// then
		assert.Equal(t, []float64{3, 1, 2}, latitudesOf(result))
	})

	t.Run("should leave the points unchanged when none has time", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(3).WithoutTime().Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(1).WithoutTime().Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(2).WithoutTime().Build(),
		}

		// when
		result := domain.ReorderByTime(points)

		// then
		assert.Equal(t, []float64{3, 1, 2}, latitudesOf(result))
	})
}

func Test_DiscardImpossibleCoordinates(t *testing.T) {
	t.Run("should discard a couple of impossible points scattered in the track", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(10).WithLongitude(20).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(200).WithLongitude(20).Build(),  // impossible latitude
			builddomain.NewTrackPointBuilder().WithLatitude(10).WithLongitude(-300).Build(), // impossible longitude
			builddomain.NewTrackPointBuilder().WithLatitude(-90).WithLongitude(180).Build(), // boundary values are possible
		}

		// when
		kept, discarded := domain.DiscardImpossibleCoordinates(points)

		// then
		assert.Equal(t, 2, discarded)
		assert.Equal(t, []float64{10, -90}, latitudesOf(kept))
	})

	t.Run("should discard many impossible points throughout a longer track", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(1).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(91).Build(), // impossible
			builddomain.NewTrackPointBuilder().WithLatitude(2).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(-91).Build(), // impossible
			builddomain.NewTrackPointBuilder().WithLatitude(3).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(3).WithLongitude(181).Build(),  // impossible
			builddomain.NewTrackPointBuilder().WithLatitude(3).WithLongitude(-181).Build(), // impossible
			builddomain.NewTrackPointBuilder().WithLatitude(4).Build(),
		}

		// when
		kept, discarded := domain.DiscardImpossibleCoordinates(points)

		// then
		assert.Equal(t, 4, discarded)
		assert.Equal(t, []float64{1, 2, 3, 4}, latitudesOf(kept))
	})
}

func Test_DiscardConsecutiveDuplicates(t *testing.T) {
	t.Run("should discard a single duplicate", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(1).WithLongitude(1).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(1).WithLongitude(1).Build(), // duplicate of the previous kept point
			builddomain.NewTrackPointBuilder().WithLatitude(2).WithLongitude(2).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(1).WithLongitude(1).Build(), // not consecutive with the first occurrence, so it is kept
		}

		// when
		kept, discarded := domain.DiscardConsecutiveDuplicates(points)

		// then
		assert.Equal(t, 1, discarded)
		assert.Equal(t, []float64{1, 2, 1}, latitudesOf(kept))
	})

	t.Run("should collapse a long run of consecutive duplicates to a single point", func(t *testing.T) {
		// given
		samePoint := builddomain.NewTrackPointBuilder().WithLatitude(1).WithLongitude(1).Build()
		points := []domain.TrackPoint{
			samePoint,
			samePoint,
			samePoint,
			samePoint,
			samePoint,
			builddomain.NewTrackPointBuilder().WithLatitude(2).WithLongitude(2).Build(),
		}

		// when
		kept, discarded := domain.DiscardConsecutiveDuplicates(points)

		// then
		assert.Equal(t, 4, discarded)
		assert.Equal(t, []float64{1, 2}, latitudesOf(kept))
	})
}

func Test_DiscardImplausibleJumps(t *testing.T) {
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

	t.Run("should keep a plausible walking/running/cycling pace", func(t *testing.T) {
		// given: ~11m in 10s, ~4 km/h
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0.0001).WithTime(start.Add(10 * time.Second)).Build(),
		}

		// when
		kept, discarded := domain.DiscardImplausibleJumps(points, 130)

		// then
		assert.Equal(t, 0, discarded)
		assert.Len(t, kept, 2)
	})

	t.Run("should discard a jump implying a speed far beyond any human-powered activity", func(t *testing.T) {
		// given: ~55km in 1s
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0.5).WithLongitude(0).WithTime(start.Add(time.Second)).Build(),
		}

		// when
		kept, discarded := domain.DiscardImplausibleJumps(points, 130)

		// then
		assert.Equal(t, 1, discarded)
		assert.Len(t, kept, 1)
	})

	t.Run("should never evaluate a jump when either point is missing time", func(t *testing.T) {
		// given: would be an enormous jump, but time is unknown
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithoutTime().Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(50).WithLongitude(50).WithoutTime().Build(),
		}

		// when
		kept, discarded := domain.DiscardImplausibleJumps(points, 130)

		// then
		assert.Equal(t, 0, discarded)
		assert.Len(t, kept, 2)
	})

	t.Run("should keep a jump right at the threshold speed", func(t *testing.T) {
		// given: one degree of latitude is almost exactly 111.19 km; covering
		// it in exactly one hour is a ~111.19 km/h pace, safely below a
		// 130 km/h threshold.
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(1).WithLongitude(0).WithTime(start.Add(time.Hour)).Build(),
		}

		// when
		kept, discarded := domain.DiscardImplausibleJumps(points, 130)

		// then
		assert.Equal(t, 0, discarded)
		assert.Len(t, kept, 2)
	})

	t.Run("should treat zero or negative elapsed time with real distance as always implausible", func(t *testing.T) {
		// given: same timestamp, but the point moved
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0.001).WithTime(start).Build(),
		}

		// when
		kept, discarded := domain.DiscardImplausibleJumps(points, 130)

		// then
		assert.Equal(t, 1, discarded)
		assert.Len(t, kept, 1)
	})

	t.Run("should not consider zero elapsed time with zero distance implausible", func(t *testing.T) {
		// given: identical point and timestamp
		point := builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithTime(start).Build()
		points := []domain.TrackPoint{point, point}

		// when
		kept, discarded := domain.DiscardImplausibleJumps(points, 130)

		// then
		assert.Equal(t, 0, discarded)
		assert.Len(t, kept, 2)
	})

	t.Run("should compare a point against the last point actually kept, not the discarded one", func(t *testing.T) {
		// given: the middle point is an implausible jump from the first
		// point; the third point is plausible relative to the first (kept)
		// point, not the discarded middle one
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0.5).WithLongitude(0).WithTime(start.Add(time.Second)).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0.00002).WithLongitude(0).WithTime(start.Add(2 * time.Second)).Build(),
		}

		// when
		kept, discarded := domain.DiscardImplausibleJumps(points, 130)

		// then
		assert.Equal(t, 1, discarded)
		require.Len(t, kept, 2)
		assert.Equal(t, []float64{0, 0.00002}, latitudesOf(kept))
	})
}

func Test_CleanTrack(t *testing.T) {
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

	t.Run("should reject fewer than the minimum points before any cleaning", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().Build(),
		}

		// when
		_, _, err := domain.CleanTrack(points, 2, 130)

		// then
		assert.ErrorIs(t, err, domain.ErrInsufficientPoints)
	})

	t.Run("should reject a track with enough raw points but too few after cleaning, with a distinguishable error", func(t *testing.T) {
		// given: two of the three points have an impossible latitude
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(200).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(300).Build(),
		}

		// when
		_, _, err := domain.CleanTrack(points, 2, 130)

		// then
		assert.ErrorIs(t, err, domain.ErrInsufficientPointsAfterCleaning)
		assert.NotErrorIs(t, err, domain.ErrInsufficientPoints, "the two errors must be distinguishable (FR-006)")
	})

	t.Run("should reorder by time before discarding, and report discard counts by reason", func(t *testing.T) {
		// given: out of chronological order; the earliest point has an
		// impossible coordinate. Spaced an hour apart so the ~111km/degree
		// of latitude between the two kept points stays a plausible pace
		// (~111 km/h), not an implausible jump.
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(3).WithLongitude(0).WithTime(start.Add(2 * time.Hour)).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(1).WithLongitude(-300).WithTime(start).Build(), // impossible longitude
			builddomain.NewTrackPointBuilder().WithLatitude(2).WithLongitude(0).WithTime(start.Add(time.Hour)).Build(),
		}

		// when
		kept, discarded, err := domain.CleanTrack(points, 2, 130)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, discarded.ImpossibleCoordinates)
		assert.Equal(t, []float64{2, 3}, latitudesOf(kept), "reordered chronologically, then the impossible point removed")
	})
}

func latitudesOf(points []domain.TrackPoint) []float64 {
	lats := make([]float64, len(points))
	for i, p := range points {
		lats[i] = p.Latitude
	}
	return lats
}
