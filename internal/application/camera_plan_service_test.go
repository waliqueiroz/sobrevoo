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
	"github.com/waliqueiroz/sobrevoo/internal/application/mockapplication"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mockdomain"
)

func treatedRoute(lengthMeters float64) domain.TreatedTrack {
	points := builddomain.NewSyntheticRouteBuilder().WithLine(lengthMeters, 90).WithConstantSpeed(5).Build()
	return domain.TreatedTrack{
		Track:         builddomain.NewTrackBuilder().WithPoints(points...).Build(),
		CleanedPoints: points,
		Route:         domain.Route{Points: points},
	}
}

func Test_cameraPlanService_Generate(t *testing.T) {
	tuning := builddomain.NewCameraTuningBuilder().Build()

	t.Run("should refuse invalid parameters without reading the track", func(t *testing.T) {
		// given: a TrackService mock with no expectations fails the test if it is called
		mockCtrl := gomock.NewController(t)
		trackService := mockapplication.NewMockTrackService(mockCtrl)
		service := application.NewCameraPlanService(trackService, nil, domain.LevelMedium, tuning)
		parameters := builddomain.NewPlanParametersBuilder().WithFrameRate(0).Build()

		// when
		_, err := service.Generate(strings.NewReader(""), parameters)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidFrameRate)
	})

	t.Run("should propagate TrackService.Treat's error unchanged", func(t *testing.T) {
		// given
		wantErr := domain.ErrEmptyFile
		mockCtrl := gomock.NewController(t)
		trackService := mockapplication.NewMockTrackService(mockCtrl)
		trackService.EXPECT().Treat(gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.TreatedTrack{}, wantErr)
		service := application.NewCameraPlanService(trackService, nil, domain.LevelMedium, tuning)

		// when
		_, err := service.Generate(strings.NewReader(""), builddomain.NewPlanParametersBuilder().Build())

		// then
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("should treat the track with the default level for both simplification and smoothing", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		trackService := mockapplication.NewMockTrackService(mockCtrl)
		trackService.EXPECT().Treat(gomock.Any(), domain.LevelHigh, domain.LevelHigh).Return(treatedRoute(5000), nil)
		service := application.NewCameraPlanService(trackService, nil, domain.LevelHigh, tuning)

		// when
		_, err := service.Generate(strings.NewReader(""), builddomain.NewPlanParametersBuilder().Build())

		// then
		assert.NoError(t, err)
	})

	t.Run("should plan the camera over the treated route with the injected tuning", func(t *testing.T) {
		// given
		treated := treatedRoute(5000)
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(45 * time.Second).Build()
		want, err := domain.PlanCamera(treated, parameters, tuning)
		require.NoError(t, err)

		mockCtrl := gomock.NewController(t)
		trackService := mockapplication.NewMockTrackService(mockCtrl)
		trackService.EXPECT().Treat(gomock.Any(), gomock.Any(), gomock.Any()).Return(treated, nil)
		service := application.NewCameraPlanService(trackService, nil, domain.LevelMedium, tuning)

		// when
		plan, err := service.Generate(strings.NewReader(""), parameters)

		// then
		require.NoError(t, err)
		assert.Equal(t, want, plan)
	})

	t.Run("should propagate a duration too short for the track", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		trackService := mockapplication.NewMockTrackService(mockCtrl)
		trackService.EXPECT().Treat(gomock.Any(), gomock.Any(), gomock.Any()).Return(treatedRoute(20000), nil)
		service := application.NewCameraPlanService(trackService, nil, domain.LevelMedium, tuning)
		parameters := builddomain.NewPlanParametersBuilder().WithDuration(5 * time.Second).Build()

		// when
		_, err := service.Generate(strings.NewReader(""), parameters)

		// then
		assert.ErrorIs(t, err, domain.ErrDurationTooShort)
	})

	t.Run("should propagate a track too short to follow", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		trackService := mockapplication.NewMockTrackService(mockCtrl)
		trackService.EXPECT().Treat(gomock.Any(), gomock.Any(), gomock.Any()).Return(treatedRoute(10), nil)
		service := application.NewCameraPlanService(trackService, nil, domain.LevelMedium, tuning)

		// when
		_, err := service.Generate(strings.NewReader(""), builddomain.NewPlanParametersBuilder().Build())

		// then
		assert.ErrorIs(t, err, domain.ErrTrackTooShort)
	})

	t.Run("should propagate a track too large to plan", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		trackService := mockapplication.NewMockTrackService(mockCtrl)
		trackService.EXPECT().Treat(gomock.Any(), gomock.Any(), gomock.Any()).Return(treatedRoute(2_500_000), nil)
		service := application.NewCameraPlanService(trackService, nil, domain.LevelMedium, tuning)

		// when
		_, err := service.Generate(strings.NewReader(""), builddomain.NewPlanParametersBuilder().Build())

		// then
		assert.ErrorIs(t, err, domain.ErrTrackTooLarge)
	})
}

func Test_cameraPlanService_Export(t *testing.T) {
	t.Run("should hand the plan, the path and the overwrite flag to the exporter", func(t *testing.T) {
		// given
		plan := builddomain.NewCameraPlanBuilder().Build()
		mockCtrl := gomock.NewController(t)
		exporter := mockdomain.NewMockCameraPlanExporter(mockCtrl)
		exporter.EXPECT().Export(plan, "/tmp/plan.json", true).Return(nil)
		service := application.NewCameraPlanService(nil, exporter, domain.LevelMedium, domain.CameraTuning{})

		// when
		err := service.Export(plan, "/tmp/plan.json", true)

		// then
		assert.NoError(t, err)
	})

	t.Run("should propagate the exporter's errors unchanged", func(t *testing.T) {
		// given
		for _, wantErr := range []error{domain.ErrPlanDestinationExists, domain.ErrPlanDestinationInvalid, errors.New("boom")} {
			plan := builddomain.NewCameraPlanBuilder().Build()
			mockCtrl := gomock.NewController(t)
			exporter := mockdomain.NewMockCameraPlanExporter(mockCtrl)
			exporter.EXPECT().Export(gomock.Any(), gomock.Any(), false).Return(wantErr)
			service := application.NewCameraPlanService(nil, exporter, domain.LevelMedium, domain.CameraTuning{})

			// when
			err := service.Export(plan, "/tmp/plan.json", false)

			// then
			assert.ErrorIs(t, err, wantErr)
		}
	})
}
