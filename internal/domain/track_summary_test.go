package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/build_domain"
)

func Test_SummarizeTrack(t *testing.T) {
	t.Run("should build a summary from the track and its treated route when both have complete data", func(t *testing.T) {
		// given
		start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
		points := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithElevation(100).WithTime(start).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(1).WithElevation(150).WithTime(start.Add(time.Hour)).Build(),
		}
		track := build_domain.NewTrackBuilder().WithPoints(points...).Build()
		route := domain.Route{Points: points}

		// when
		summary := domain.SummarizeTrack(track, route, domain.DiscardStats{})

		// then
		assert.Equal(t, domain.FormatGPX, summary.Format)
		assert.Equal(t, 2, summary.PointCountOriginal)
		assert.Equal(t, 2, summary.PointCountTreated)
		assert.Greater(t, summary.TotalDistanceMeters, 0.0)
		require.NotNil(t, summary.ElevationGainMeters)
		assert.InDelta(t, 50.0, *summary.ElevationGainMeters, 0.0001)
		require.NotNil(t, summary.Duration)
		assert.Equal(t, time.Hour, *summary.Duration)
		assert.Equal(t, domain.ComputeBoundingBox(points), summary.BoundingBox)
	})

	t.Run("should report elevation gain and duration as unavailable when the route has no such data", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithoutElevation().WithoutTime().Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(1).WithoutElevation().WithoutTime().Build(),
		}
		track := build_domain.NewTrackBuilder().WithPoints(points...).Build()
		route := domain.Route{Points: points}

		// when
		summary := domain.SummarizeTrack(track, route, domain.DiscardStats{})

		// then
		assert.Nil(t, summary.ElevationGainMeters)
		assert.Nil(t, summary.Duration)
	})

	t.Run("should count original points from the track and treated points from the route, when they differ", func(t *testing.T) {
		// given: the track had 3 points, but only 2 survived into the route
		// (cleaned/simplified/smoothed away)
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().Build(),
			build_domain.NewTrackPointBuilder().Build(),
			build_domain.NewTrackPointBuilder().Build(),
		).Build()
		route := domain.Route{Points: []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().Build(),
			build_domain.NewTrackPointBuilder().Build(),
		}}

		// when
		summary := domain.SummarizeTrack(track, route, domain.DiscardStats{})

		// then
		assert.Equal(t, 3, summary.PointCountOriginal)
		assert.Equal(t, 2, summary.PointCountTreated)
	})

	t.Run("should carry the discard stats through unchanged", func(t *testing.T) {
		// given
		track := build_domain.NewTrackBuilder().Build()
		route := domain.Route{Points: track.Points}
		discarded := domain.DiscardStats{ImpossibleCoordinates: 1, ConsecutiveDuplicates: 2, ImplausibleJumps: 3}

		// when
		summary := domain.SummarizeTrack(track, route, discarded)

		// then
		assert.Equal(t, discarded, summary.Discarded)
	})
}
