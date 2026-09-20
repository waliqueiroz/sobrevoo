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
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mockdomain"
)

const (
	testMinPoints            = 2
	testMaxPlausibleSpeedKmh = 130.0
)

// passthrough configures a mocked Simplifier/Smoother to return whatever
// points they receive unchanged, for tests where the treatment stages
// themselves are not what is being verified.
func passthroughSimplifier(ctrl *gomock.Controller) domain.Simplifier {
	simplifier := mockdomain.NewMockSimplifier(ctrl)
	simplifier.EXPECT().Simplify(gomock.Any(), gomock.Any()).
		DoAndReturn(func(points []domain.TrackPoint, _ domain.Level) []domain.TrackPoint { return points }).
		AnyTimes()
	return simplifier
}

func passthroughSmoother(ctrl *gomock.Controller) domain.Smoother {
	smoother := mockdomain.NewMockSmoother(ctrl)
	smoother.EXPECT().Smooth(gomock.Any(), gomock.Any()).
		DoAndReturn(func(points []domain.TrackPoint, _ domain.Level) []domain.TrackPoint { return points }).
		AnyTimes()
	return smoother
}

func Test_trackService_Inspect(t *testing.T) {
	t.Run("should propagate the parser's error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		mockedParser := mockdomain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(domain.Track{}, wantErr)

		service := application.NewTrackService(mockedParser, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Inspect(strings.NewReader(""), domain.LevelMedium, domain.LevelMedium)

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should propagate domain.CleanTrack's error unchanged", func(t *testing.T) {
		// given: domain/cleaning_test.go covers CleanTrack's own rules
		// (including distinguishing this from ErrInsufficientPointsAfterCleaning)
		// in detail — this only checks the service does not swallow it.
		track := builddomain.NewTrackBuilder().WithPoints(
			builddomain.NewTrackPointBuilder().Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		mockedParser := mockdomain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		service := application.NewTrackService(mockedParser, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Inspect(strings.NewReader("irrelevant"), domain.LevelMedium, domain.LevelMedium)

		// then
		assert.ErrorIs(t, err, domain.ErrInsufficientPoints)
	})

	t.Run("should run simplification then smoothing, in that order, with the requested levels", func(t *testing.T) {
		// given
		track := builddomain.NewTrackBuilder().WithPoints(
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(1).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(2).Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		mockedParser := mockdomain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		simplified := []domain.TrackPoint{builddomain.NewTrackPointBuilder().WithLatitude(10).WithLongitude(10).Build()}
		mockedSimplifier := mockdomain.NewMockSimplifier(mockCtrl)
		mockedSimplifier.EXPECT().Simplify(track.Points, domain.LevelHigh).Return(simplified)

		smoothed := []domain.TrackPoint{builddomain.NewTrackPointBuilder().WithLatitude(20).WithLongitude(20).Build()}
		mockedSmoother := mockdomain.NewMockSmoother(mockCtrl)
		// Smooth must receive Simplify's output, not the original points —
		// simplification runs first.
		mockedSmoother.EXPECT().Smooth(simplified, domain.LevelLow).Return(smoothed)

		service := application.NewTrackService(mockedParser, mockedSimplifier, mockedSmoother, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		summary, err := service.Inspect(strings.NewReader("irrelevant"), domain.LevelHigh, domain.LevelLow)

		// then: the final route must be Smooth's output, confirming both
		// ports were actually applied to build the summary
		require.NoError(t, err)
		assert.Equal(t, 1, summary.PointCountTreated)
		assert.Equal(t, domain.ComputeBoundingBox(smoothed), summary.BoundingBox)
	})

	t.Run("should return the summary built from the cleaned, simplified and smoothed route", func(t *testing.T) {
		// given: SummarizeTrack's own rules (elevation gain, duration,
		// discard stats) are covered in domain/track_summary_test.go — this
		// only checks the service wires the cleaned/treated route into it.
		start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
		track := builddomain.NewTrackBuilder().WithPoints(
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).WithElevation(100).WithTime(start).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(1).WithElevation(150).WithTime(start.Add(time.Hour)).Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		mockedParser := mockdomain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		service := application.NewTrackService(mockedParser, passthroughSimplifier(mockCtrl), passthroughSmoother(mockCtrl), testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		summary, err := service.Inspect(strings.NewReader("irrelevant"), domain.LevelMedium, domain.LevelMedium)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.FormatGPX, summary.Format)
		assert.Equal(t, 2, summary.PointCountOriginal)
		assert.Equal(t, 2, summary.PointCountTreated)
		require.NotNil(t, summary.ElevationGainMeters)
		assert.InDelta(t, 50.0, *summary.ElevationGainMeters, 0.0001)
		require.NotNil(t, summary.Duration)
		assert.Equal(t, time.Hour, *summary.Duration)
	})
}

func Test_trackService_Clean(t *testing.T) {
	t.Run("should propagate the parser's error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		mockedParser := mockdomain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(domain.Track{}, wantErr)

		service := application.NewTrackService(mockedParser, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Clean(strings.NewReader(""))

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should propagate domain.CleanTrack's error unchanged", func(t *testing.T) {
		// given
		track := builddomain.NewTrackBuilder().WithPoints(
			builddomain.NewTrackPointBuilder().Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		mockedParser := mockdomain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		service := application.NewTrackService(mockedParser, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Clean(strings.NewReader("irrelevant"))

		// then
		assert.ErrorIs(t, err, domain.ErrInsufficientPoints)
	})

	t.Run("should return the cleaned points and discard stats without simplifying or smoothing", func(t *testing.T) {
		// given: nil simplifier and smoother would panic if Clean called them
		track := builddomain.NewTrackBuilder().WithPoints(
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(1).Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		mockedParser := mockdomain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		service := application.NewTrackService(mockedParser, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		cleaned, err := service.Clean(strings.NewReader("irrelevant"))

		// then
		require.NoError(t, err)
		assert.Equal(t, track, cleaned.Track)
		assert.Len(t, cleaned.Points, 2)
		assert.Equal(t, 1, cleaned.Discarded.ConsecutiveDuplicates)
	})
}

func Test_trackService_Treat(t *testing.T) {
	t.Run("should propagate Clean's error unchanged", func(t *testing.T) {
		// given
		wantErr := errors.New("boom")

		mockCtrl := gomock.NewController(t)
		mockedParser := mockdomain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(domain.Track{}, wantErr)

		service := application.NewTrackService(mockedParser, nil, nil, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		_, err := service.Treat(strings.NewReader(""), domain.LevelMedium, domain.LevelMedium)

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should simplify then smooth the cleaned points, in that order, with the requested levels", func(t *testing.T) {
		// given
		track := builddomain.NewTrackBuilder().WithPoints(
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(1).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(2).Build(),
		).Build()

		mockCtrl := gomock.NewController(t)
		mockedParser := mockdomain.NewMockTrackParser(mockCtrl)
		mockedParser.EXPECT().Parse(gomock.Any()).Return(track, nil)

		simplified := []domain.TrackPoint{builddomain.NewTrackPointBuilder().WithLatitude(10).WithLongitude(10).Build()}
		mockedSimplifier := mockdomain.NewMockSimplifier(mockCtrl)
		mockedSimplifier.EXPECT().Simplify(track.Points, domain.LevelHigh).Return(simplified)

		smoothed := []domain.TrackPoint{builddomain.NewTrackPointBuilder().WithLatitude(20).WithLongitude(20).Build()}
		mockedSmoother := mockdomain.NewMockSmoother(mockCtrl)
		mockedSmoother.EXPECT().Smooth(simplified, domain.LevelLow).Return(smoothed)

		service := application.NewTrackService(mockedParser, mockedSimplifier, mockedSmoother, testMinPoints, testMaxPlausibleSpeedKmh)

		// when
		treated, err := service.Treat(strings.NewReader("irrelevant"), domain.LevelHigh, domain.LevelLow)

		// then
		require.NoError(t, err)
		assert.Equal(t, track, treated.Track)
		assert.Equal(t, domain.Route{Points: smoothed}, treated.Route)
		assert.Equal(t, track.Points, treated.CleanedPoints)
		assert.Zero(t, treated.Discarded.Total())
	})
}
