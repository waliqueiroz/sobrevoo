package application_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/build_domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mock_domain"
)

const (
	testMinPoints            = 2
	testMaxPlausibleSpeedKmh = 130.0
)

// passthrough configures a mocked Simplifier/Smoother to return whatever
// points they receive unchanged, for tests where the treatment stages
// themselves are not what is being verified.
func passthroughSimplifier(ctrl *gomock.Controller) domain.Simplifier {
	simplifier := mock_domain.NewMockSimplifier(ctrl)
	simplifier.EXPECT().Simplify(gomock.Any(), gomock.Any()).
		DoAndReturn(func(points []domain.TrackPoint, _ domain.Level) []domain.TrackPoint { return points }).
		AnyTimes()
	return simplifier
}

func passthroughSmoother(ctrl *gomock.Controller) domain.Smoother {
	smoother := mock_domain.NewMockSmoother(ctrl)
	smoother.EXPECT().Smooth(gomock.Any(), gomock.Any()).
		DoAndReturn(func(points []domain.TrackPoint, _ domain.Level) []domain.TrackPoint { return points }).
		AnyTimes()
	return smoother
}

func Test_inspectTrackService_Execute(t *testing.T) {
	t.Run("should propagate the parser's error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		mockedParser := mock_domain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(domain.Track{}, wantErr)

		service := application.NewInspectTrackService(mockedParser, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Inspect(application.InspectTrackInput{Reader: strings.NewReader("")})

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should build a summary from the parsed track when it has complete data", func(t *testing.T) {
		// given
		start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithElevation(100).WithTime(start).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(1).WithElevation(150).WithTime(start.Add(time.Hour)).Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		mockedParser := mock_domain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		service := application.NewInspectTrackService(mockedParser, passthroughSimplifier(mockCtrl), passthroughSmoother(mockCtrl), testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		output, err := service.Inspect(application.InspectTrackInput{Reader: strings.NewReader("irrelevant")})

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.FormatGPX, output.Format)
		assert.Equal(t, 2, output.PointCountOriginal)
		assert.Equal(t, 2, output.PointCountTreated, "neither point is problematic, so none is discarded")
		assert.Greater(t, output.TotalDistanceMeters, 0.0)
		require.NotNil(t, output.ElevationGainMeters)
		assert.InDelta(t, 50.0, *output.ElevationGainMeters, 0.0001)
		require.NotNil(t, output.Duration)
		assert.Equal(t, time.Hour, *output.Duration)
	})

	t.Run("should report elevation and duration as unavailable when the track has no such data", func(t *testing.T) {
		// given
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithoutElevation().WithoutTime().Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(1).WithoutElevation().WithoutTime().Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		mockedParser := mock_domain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		service := application.NewInspectTrackService(mockedParser, passthroughSimplifier(mockCtrl), passthroughSmoother(mockCtrl), testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		output, err := service.Inspect(application.InspectTrackInput{Reader: strings.NewReader("irrelevant")})

		// then
		require.NoError(t, err)
		assert.Nil(t, output.ElevationGainMeters)
		assert.Nil(t, output.Duration)
	})

	t.Run("should reject a track with fewer than the minimum points before cleaning", func(t *testing.T) {
		// given
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		mockedParser := mock_domain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		service := application.NewInspectTrackService(mockedParser, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Inspect(application.InspectTrackInput{Reader: strings.NewReader("irrelevant")})

		// then
		assert.ErrorIs(t, err, domain.ErrInsufficientPoints)
	})

	t.Run("should reject a track that has enough raw points but too few after cleaning", func(t *testing.T) {
		// given: two of the three points have an impossible latitude
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(0).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(200).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(300).Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		mockedParser := mock_domain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		service := application.NewInspectTrackService(mockedParser, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Inspect(application.InspectTrackInput{Reader: strings.NewReader("irrelevant")})

		// then
		assert.ErrorIs(t, err, domain.ErrInsufficientPointsAfterCleaning)
		assert.NotErrorIs(t, err, domain.ErrInsufficientPoints, "the two errors must be distinguishable (FR-006)")
	})

	t.Run("should exclude discarded points from the computed distance", func(t *testing.T) {
		// given: the middle point is an implausible jump (~55km in 1s)
		start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithTime(start).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0.5).WithLongitude(0).WithTime(start.Add(time.Second)).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0.00002).WithLongitude(0).WithTime(start.Add(2*time.Second)).Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		mockedParser := mock_domain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		service := application.NewInspectTrackService(mockedParser, passthroughSimplifier(mockCtrl), passthroughSmoother(mockCtrl), testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		output, err := service.Inspect(application.InspectTrackInput{Reader: strings.NewReader("irrelevant")})

		// then
		require.NoError(t, err)
		assert.Equal(t, 2, output.PointCountTreated, "the middle point is discarded as an implausible jump")
		// Without the implausible jump, the remaining two points are only a
		// couple of meters apart; with it, the distance would be tens of
		// kilometers.
		assert.Less(t, output.TotalDistanceMeters, 100.0)
	})

	t.Run("should run simplification then smoothing, in that order, with the requested levels", func(t *testing.T) {
		// given
		track := build_domain.NewTrackBuilder().WithPoints(
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(1).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(2).Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		mockedParser := mock_domain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		simplified := []domain.TrackPoint{build_domain.NewTrackPointBuilder().WithLatitude(10).WithLongitude(10).Build()}
		mockedSimplifier := mock_domain.NewMockSimplifier(mockCtrl)
		mockedSimplifier.EXPECT().Simplify(track.Points, domain.LevelHigh).Return(simplified)

		smoothed := []domain.TrackPoint{build_domain.NewTrackPointBuilder().WithLatitude(20).WithLongitude(20).Build()}
		mockedSmoother := mock_domain.NewMockSmoother(mockCtrl)
		// Smooth must receive Simplify's output, not the original points —
		// simplification runs first.
		mockedSmoother.EXPECT().Smooth(simplified, domain.LevelLow).Return(smoothed)

		service := application.NewInspectTrackService(mockedParser, mockedSimplifier, mockedSmoother, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		output, err := service.Inspect(application.InspectTrackInput{
			Reader:              strings.NewReader("irrelevant"),
			SimplificationLevel: domain.LevelHigh,
			SmoothingLevel:      domain.LevelLow,
		})

		// then: the final route must be Smooth's output, confirming both
		// ports were actually applied to build the summary
		require.NoError(t, err)
		assert.Equal(t, 1, output.PointCountTreated)
		assert.Equal(t, domain.ComputeBoundingBox(smoothed), output.BoundingBox)
	})
}
