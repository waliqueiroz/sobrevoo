package application_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/application/mockapplication"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mockdomain"
)

type flightMocks struct {
	cameraPlanService *mockapplication.MockCameraPlanService
	geoSliceService   *mockapplication.MockGeoSliceService
	frameService      *mockapplication.MockFrameService
	videoService      *mockapplication.MockVideoService
	workspace         *mockdomain.MockWorkspace
	service           application.FlightService
}

func newFlightMocks(t *testing.T) flightMocks {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	m := flightMocks{
		cameraPlanService: mockapplication.NewMockCameraPlanService(mockCtrl),
		geoSliceService:   mockapplication.NewMockGeoSliceService(mockCtrl),
		frameService:      mockapplication.NewMockFrameService(mockCtrl),
		videoService:      mockapplication.NewMockVideoService(mockCtrl),
		workspace:         mockdomain.NewMockWorkspace(mockCtrl),
	}
	m.service = application.NewFlightService(m.cameraPlanService, m.geoSliceService, m.frameService, m.videoService, m.workspace)
	return m
}

// flightRequest is a request for the video at /tmp/flight.mp4, with no kept
// intermediates directory.
func flightRequest() domain.FlightRequest {
	return builddomain.NewFlightRequestBuilder().Build()
}

// readyToFly sets up the mocks for a whole, successful run without a kept
// directory: the plan, the slice, a temporary frames directory, the frames
// drawn and the video assembled.
func readyToFly(m flightMocks, plan domain.CameraPlan, slice domain.GeoSlice) {
	m.videoService.EXPECT().CheckDestination("/tmp/flight.mp4", false).Return(nil)
	m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
	m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
	m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
	m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { return nil }, nil)
	m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
		Return(domain.RenderSummary{Requested: len(plan.Frames), Drawn: len(plan.Frames)}, nil)
	m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).
		Return(domain.VideoSummary{Frames: len(plan.Frames)}, nil)
}

func Test_flightService_Fly(t *testing.T) {
	plan := framesPlan(20)
	slice := forPlan(framesSlice(), plan)

	t.Run("should check the destination, check the encoder, generate the plan, generate the slice, draw the frames into a temporary directory and assemble the video, in that order", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		removed := false
		gomock.InOrder(
			m.videoService.EXPECT().CheckDestination("/tmp/flight.mp4", false).Return(nil),
			m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil),
			m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil),
			m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil),
			m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { removed = true; return nil }, nil),
			m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, domain.FrameSetRequest{
				Directory: "/tmp/sobrevoo-fly-1", Resolution: flightRequest().Resolution, Appearance: flightRequest().Appearance, Overlay: flightRequest().Overlay,
			}, gomock.Any()).Return(domain.RenderSummary{Requested: 20, Drawn: 20}, nil),
			m.videoService.EXPECT().Assemble(gomock.Any(), plan, domain.VideoRequest{
				Directory: "/tmp/sobrevoo-fly-1", Output: "/tmp/flight.mp4", Quality: domain.VideoQualityMedium,
			}, gomock.Any()).Return(domain.VideoSummary{Frames: 20}, nil),
		)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), nil)

		// then
		require.NoError(t, err)
		assert.True(t, removed, "the temporary frames directory is removed at the end")
		assert.Equal(t, []domain.FlightStage{
			domain.StageTrackProcessing, domain.StageCameraPlanning, domain.StageGeoDataSlicing,
			domain.StageFrameRendering, domain.StageVideoEncoding,
		}, summary.Completed)
		assert.GreaterOrEqual(t, int64(summary.Elapsed), int64(0))
		assert.False(t, summary.PlanReused)
		assert.False(t, summary.SliceReused)
	})

	t.Run("should pass the request's appearance to the frames, unaltered", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		green := builddomain.NewAppearanceBuilder().WithTrailColor(domain.RGB{R: 0x00, G: 0xFF, B: 0x00}).Build()
		request := builddomain.NewFlightRequestBuilder().WithAppearance(green).Build()
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { return nil }, nil)
		var got domain.Appearance
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.CameraPlan, _ domain.GeoSlice, r domain.FrameSetRequest, _ func(domain.RenderProgress)) (domain.RenderSummary, error) {
				got = r.Appearance
				return domain.RenderSummary{Requested: 20, Drawn: 20}, nil
			})
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).
			Return(domain.VideoSummary{Frames: 20}, nil)

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, green, got)
	})

	t.Run("should pass the request's overlay to the frames, unaltered", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		full, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{domain.OverlayBlockDistance})
		require.NoError(t, err)
		request := builddomain.NewFlightRequestBuilder().WithOverlay(full).Build()
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { return nil }, nil)
		var got domain.OverlayConfig
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.CameraPlan, _ domain.GeoSlice, r domain.FrameSetRequest, _ func(domain.RenderProgress)) (domain.RenderSummary, error) {
				got = r.Overlay
				return domain.RenderSummary{Requested: 20, Drawn: 20}, nil
			})
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).
			Return(domain.VideoSummary{Frames: 20}, nil)

		// when
		_, err = m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, full, got)
	})

	t.Run("should pass the request's source selection to GeoSliceService.Generate, unaltered", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		name := "mapa-b"
		selection := domain.SourceSelection{BaseMapName: &name}
		request := builddomain.NewFlightRequestBuilder().WithSelection(selection).Build()
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, selection).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { return nil }, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
			Return(domain.RenderSummary{Requested: 20, Drawn: 20}, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).
			Return(domain.VideoSummary{Frames: 20}, nil)

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
	})

	t.Run("should say what the run did in the summary: the render summary and the video summary", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { return nil }, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
			Return(domain.RenderSummary{Requested: 20, Drawn: 20, Resolution: flightRequest().Resolution}, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).
			Return(domain.VideoSummary{Frames: 20, Encoder: anEncoder}, nil)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, 20, summary.Render.Drawn)
		assert.Equal(t, 20, summary.Video.Frames)
		assert.Equal(t, anEncoder, summary.Video.Encoder)
	})

	t.Run("should return the error of the destination check as it is, without generating anything", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(domain.ErrVideoDestinationExists)

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrVideoDestinationExists)
	})

	t.Run("should return the error of the encoder check as it is, without generating anything", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(domain.EncoderInfo{}, domain.ErrEncoderUnavailable)

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrEncoderUnavailable)
	})

	t.Run("should return the error of planning the camera as it is, without slicing or drawing anything", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(domain.CameraPlan{}, domain.ErrTrackTooShort)

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrTrackTooShort)
	})

	t.Run("should return the error of slicing the geo data as it is, without drawing anything", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(domain.GeoSlice{}, &domain.AreaNotCoveredError{})

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrAreaNotCovered)
	})

	t.Run("should return the error of creating the temporary workspace as it is", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		failure := errors.New("no space left on device")
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("", nil, failure)

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), nil)

		// then
		assert.ErrorIs(t, err, failure)
	})

	t.Run("should return the error of drawing the frames as it is, and still remove the temporary directory", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		removed := false
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { removed = true; return nil }, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
			Return(domain.RenderSummary{}, domain.ErrNoElevationData)

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrNoElevationData)
		assert.True(t, removed)
	})

	t.Run("should return the error of assembling the video as it is, and still remove the temporary directory", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		removed := false
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { removed = true; return nil }, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
			Return(domain.RenderSummary{Requested: 20, Drawn: 20}, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).
			Return(domain.VideoSummary{}, domain.ErrVideoEncodingFailed)

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), nil)

		// then
		assert.ErrorIs(t, err, domain.ErrVideoEncodingFailed)
		assert.True(t, removed)
	})

	t.Run("should pass the reader to the camera plan service so it can treat the track", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		reader := strings.NewReader("a track")
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { return nil }, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).Return(domain.RenderSummary{}, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).Return(domain.VideoSummary{}, nil)
		var received io.Reader
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).DoAndReturn(func(r io.Reader, _ domain.PlanParameters) (domain.CameraPlan, error) {
			received = r
			return plan, nil
		})

		// when
		_, err := m.service.Fly(context.Background(), reader, flightRequest(), nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, reader, received)
	})

	t.Run("should pass the request's parameters to the camera plan service", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		request := flightRequest()
		request.Parameters = builddomain.NewPlanParametersBuilder().WithFrameRate(24).Build()
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { return nil }, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).Return(domain.RenderSummary{}, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).Return(domain.VideoSummary{}, nil)
		var received domain.PlanParameters
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ io.Reader, p domain.PlanParameters) (domain.CameraPlan, error) {
			received = p
			return plan, nil
		})

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, float64(24), received.FrameRate)
	})

	t.Run("should pass the request's resolution and quality to the frame and video services", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		request := flightRequest()
		request.Resolution = domain.Resolution{Width: 640, Height: 360}
		request.Quality = domain.VideoQualityHigh
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { return nil }, nil)
		var frameRequest domain.FrameSetRequest
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.CameraPlan, _ domain.GeoSlice, r domain.FrameSetRequest, _ func(domain.RenderProgress)) (domain.RenderSummary, error) {
				frameRequest = r
				return domain.RenderSummary{}, nil
			})
		var videoRequest domain.VideoRequest
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.CameraPlan, r domain.VideoRequest, _ func(domain.VideoProgress)) (domain.VideoSummary, error) {
				videoRequest = r
				return domain.VideoSummary{}, nil
			})

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.Resolution{Width: 640, Height: 360}, frameRequest.Resolution)
		assert.Equal(t, domain.VideoQualityHigh, videoRequest.Quality)
	})
}

func Test_flightService_Fly_Interruption(t *testing.T) {
	plan := framesPlan(20)
	slice := forPlan(framesSlice(), plan)

	t.Run("should return ErrFlightInterrupted, and mark the summary, when drawing the frames was interrupted", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { return nil }, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
			Return(domain.RenderSummary{Drawn: 7, Interrupted: true}, domain.ErrRenderInterrupted)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), nil)

		// then
		require.ErrorIs(t, err, domain.ErrFlightInterrupted)
		assert.NotErrorIs(t, err, domain.ErrRenderInterrupted)
		assert.True(t, summary.Interrupted)
		assert.Equal(t, 7, summary.Render.Drawn)
	})

	t.Run("should return ErrFlightInterrupted, and mark the summary, when assembling the video was interrupted", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { return nil }, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
			Return(domain.RenderSummary{Drawn: 20}, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).
			Return(domain.VideoSummary{Encoded: 5, Interrupted: true}, domain.ErrVideoInterrupted)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), nil)

		// then
		require.ErrorIs(t, err, domain.ErrFlightInterrupted)
		assert.NotErrorIs(t, err, domain.ErrVideoInterrupted)
		assert.True(t, summary.Interrupted)
		assert.Equal(t, 5, summary.Video.Encoded)
	})

	t.Run("should return ErrFlightInterrupted when the encoder check stops as the context is done", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		ctx, cancel := context.WithCancel(context.Background())
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).DoAndReturn(func(context.Context) (domain.EncoderInfo, error) {
			cancel()
			return domain.EncoderInfo{}, domain.ErrVideoInterrupted
		})

		// when
		summary, err := m.service.Fly(ctx, strings.NewReader("track"), flightRequest(), nil)

		// then
		require.ErrorIs(t, err, domain.ErrFlightInterrupted)
		assert.True(t, summary.Interrupted)
	})
}

func Test_flightService_Fly_Progress(t *testing.T) {
	plan := framesPlan(20)
	slice := forPlan(framesSlice(), plan)

	t.Run("should report entering each of the five stages, in order", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		readyToFly(m, plan, slice)
		var stages []domain.FlightStage

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), func(p domain.FlightProgress) {
			stages = append(stages, p.Stage)
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, stages, domain.StageTrackProcessing)
		assert.Contains(t, stages, domain.StageCameraPlanning)
		assert.Contains(t, stages, domain.StageGeoDataSlicing)
		assert.Contains(t, stages, domain.StageFrameRendering)
		assert.Contains(t, stages, domain.StageVideoEncoding)
	})

	t.Run("should say the camera-planning and geo-data-slicing stages were not reused when nothing was kept", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		readyToFly(m, plan, slice)
		reusedByStage := map[domain.FlightStage]bool{}

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), func(p domain.FlightProgress) {
			if p.Render == nil && p.Video == nil {
				reusedByStage[p.Stage] = p.Reused
			}
		})

		// then
		require.NoError(t, err)
		assert.False(t, reusedByStage[domain.StageCameraPlanning])
		assert.False(t, reusedByStage[domain.StageGeoDataSlicing])
	})

	t.Run("should say the camera-planning and geo-data-slicing stages were reused when the kept plan and slice already matched", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").Build()
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.workspace.EXPECT().EnsureDirectory("/tmp/kept").Return(nil)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(plan, nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).Return(domain.RenderSummary{}, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).Return(domain.VideoSummary{}, nil)
		reusedByStage := map[domain.FlightStage]bool{}

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, func(p domain.FlightProgress) {
			if p.Render == nil && p.Video == nil {
				reusedByStage[p.Stage] = p.Reused
			}
		})

		// then
		require.NoError(t, err)
		assert.True(t, reusedByStage[domain.StageCameraPlanning])
		assert.True(t, reusedByStage[domain.StageGeoDataSlicing])
	})

	t.Run("should forward the render progress reported during the frame-rendering stage", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { return nil }, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.CameraPlan, _ domain.GeoSlice, _ domain.FrameSetRequest, progress func(domain.RenderProgress)) (domain.RenderSummary, error) {
				progress(domain.RenderProgress{Done: 5, Total: 20})
				return domain.RenderSummary{Drawn: 20}, nil
			})
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).Return(domain.VideoSummary{}, nil)
		var renders []domain.RenderProgress

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), func(p domain.FlightProgress) {
			if p.Render != nil {
				renders = append(renders, *p.Render)
			}
		})

		// then
		require.NoError(t, err)
		require.Len(t, renders, 1)
		assert.Equal(t, 5, renders[0].Done)
	})

	t.Run("should forward the video progress reported during the video-encoding stage", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.workspace.EXPECT().NewTemporary().Return("/tmp/sobrevoo-fly-1", func() error { return nil }, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).Return(domain.RenderSummary{Drawn: 20}, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.CameraPlan, _ domain.VideoRequest, progress func(domain.VideoProgress)) (domain.VideoSummary, error) {
				progress(domain.VideoProgress{Done: 10, Total: 20})
				return domain.VideoSummary{}, nil
			})
		var videos []domain.VideoProgress

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), func(p domain.FlightProgress) {
			if p.Video != nil {
				videos = append(videos, *p.Video)
			}
		})

		// then
		require.NoError(t, err)
		require.Len(t, videos, 1)
		assert.Equal(t, 10, videos[0].Done)
	})

	t.Run("should accept no function to report to", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		readyToFly(m, plan, slice)

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), flightRequest(), nil)

		// then
		assert.NoError(t, err)
	})
}

func Test_flightService_Fly_Keep(t *testing.T) {
	plan := framesPlan(20)
	slice := forPlan(framesSlice(), plan)

	// readyToFlyKept sets up a whole, successful run with intermediates kept
	// at /tmp/kept, assuming neither the plan nor the slice already there is
	// valid.
	readyToFlyKept := func(m flightMocks) {
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.workspace.EXPECT().EnsureDirectory("/tmp/kept").Return(nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).Return(domain.RenderSummary{}, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).Return(domain.VideoSummary{}, nil)
	}

	t.Run("should write the plan and the slice under the kept directory, and draw the frames into its frames subdirectory, without a temporary workspace", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").Build()
		reloaded := slice
		reloaded.ContentID = "the-content-id-only-a-read-back-slice-has"
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.workspace.EXPECT().EnsureDirectory("/tmp/kept").Return(nil)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(domain.CameraPlan{}, errors.New("no such file"))
		m.cameraPlanService.EXPECT().Export(plan, "/tmp/kept/plan.json", false).Return(nil)
		gomock.InOrder(
			m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(domain.GeoSlice{}, errors.New("no such file")),
			m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil),
			m.geoSliceService.EXPECT().Export(slice, "/tmp/kept/slice.zip", false).Return(nil),
			m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(reloaded, nil),
		)
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).Return(domain.VideoSummary{}, nil)
		var frameRequest domain.FrameSetRequest
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, reloaded, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.CameraPlan, _ domain.GeoSlice, r domain.FrameSetRequest, _ func(domain.RenderProgress)) (domain.RenderSummary, error) {
				frameRequest = r
				return domain.RenderSummary{}, nil
			})

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, "/tmp/kept/frames", frameRequest.Directory)
		assert.Equal(t, "/tmp/kept/frames", summary.FramesDirectory)
		assert.False(t, summary.PlanReused)
		assert.False(t, summary.SliceReused)
	})

	t.Run("should pass the request's source selection to GeoSliceService.Generate when regenerating a kept slice", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		name := "mapa-b"
		selection := domain.SourceSelection{BaseMapName: &name}
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").WithSelection(selection).Build()
		reloaded := slice
		reloaded.ContentID = "the-content-id-only-a-read-back-slice-has"
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.workspace.EXPECT().EnsureDirectory("/tmp/kept").Return(nil)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(domain.CameraPlan{}, errors.New("no such file"))
		m.cameraPlanService.EXPECT().Export(plan, "/tmp/kept/plan.json", false).Return(nil)
		gomock.InOrder(
			m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(domain.GeoSlice{}, errors.New("no such file")),
			m.geoSliceService.EXPECT().Generate(plan, selection).Return(slice, nil),
			m.geoSliceService.EXPECT().Export(slice, "/tmp/kept/slice.zip", false).Return(nil),
			m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(reloaded, nil),
		)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, reloaded, gomock.Any(), gomock.Any()).Return(domain.RenderSummary{}, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).Return(domain.VideoSummary{}, nil)

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
	})

	t.Run("should return the error of ensuring the kept directory exists as it is, without loading or generating anything", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").Build()
		failure := errors.New("permission denied")
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.workspace.EXPECT().EnsureDirectory("/tmp/kept").Return(failure)

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		assert.ErrorIs(t, err, failure)
	})

	t.Run("should reuse the plan under the kept directory when it already matches, without exporting it again", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").Build()
		readyToFlyKept(m)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(plan, nil)
		m.cameraPlanService.EXPECT().Export(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(domain.GeoSlice{}, errors.New("no such file"))
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.geoSliceService.EXPECT().Export(slice, "/tmp/kept/slice.zip", false).Return(nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.True(t, summary.PlanReused)
	})

	t.Run("should not reuse a plan under the kept directory that is of another track or values, and export over it with the request's overwrite", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").WithOverwrite().Build()
		readyToFlyKept(m)
		another := framesPlan(5)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(another, nil)
		m.cameraPlanService.EXPECT().Export(plan, "/tmp/kept/plan.json", true).Return(nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(domain.GeoSlice{}, errors.New("no such file"))
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.geoSliceService.EXPECT().Export(slice, "/tmp/kept/slice.zip", true).Return(nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.False(t, summary.PlanReused)
	})

	t.Run("should return the error of exporting the plan as it is", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").Build()
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.workspace.EXPECT().EnsureDirectory("/tmp/kept").Return(nil)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(domain.CameraPlan{}, errors.New("no such file"))
		m.cameraPlanService.EXPECT().Export(plan, "/tmp/kept/plan.json", false).Return(domain.ErrPlanDestinationExists)

		// when
		_, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		assert.ErrorIs(t, err, domain.ErrPlanDestinationExists)
	})

	t.Run("should reuse the slice under the kept directory when it already matches, without generating or exporting it again", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").Build()
		readyToFlyKept(m)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(plan, nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)
		m.geoSliceService.EXPECT().Generate(gomock.Any(), gomock.Any()).Times(0)
		m.geoSliceService.EXPECT().Export(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.True(t, summary.SliceReused)
	})

	t.Run("should not reuse a slice under the kept directory that was made for another plan, and generate and export over it", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").WithOverwrite().Build()
		readyToFlyKept(m)
		another := forPlan(framesSlice(), framesPlan(5))
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(plan, nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(another, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{}).Return(slice, nil)
		m.geoSliceService.EXPECT().Export(slice, "/tmp/kept/slice.zip", true).Return(nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.False(t, summary.SliceReused)
	})

	t.Run("should not reuse a kept slice whose recorded provenance uses a different base map than the one now requested (010-geo-data-source-control FR-011)", func(t *testing.T) {
		// given: slice's recorded base map is "europa-central-mapa" (framesSlice's default);
		// this run explicitly requests a different one
		m := newFlightMocks(t)
		name := "mapa-b"
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").WithSelection(domain.SourceSelection{BaseMapName: &name}).Build()
		readyToFlyKept(m)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(plan, nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{BaseMapName: &name}).Return(slice, nil)
		m.geoSliceService.EXPECT().Export(slice, "/tmp/kept/slice.zip", false).Return(nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.False(t, summary.SliceReused)
	})

	t.Run("should reuse a kept slice whose recorded provenance uses exactly the base map now requested (010-geo-data-source-control FR-011)", func(t *testing.T) {
		// given
		m := newFlightMocks(t)
		name := "europa-central-mapa"
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").WithSelection(domain.SourceSelection{BaseMapName: &name}).Build()
		readyToFlyKept(m)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(plan, nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)
		m.geoSliceService.EXPECT().Generate(gomock.Any(), gomock.Any()).Times(0)
		m.geoSliceService.EXPECT().Export(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.True(t, summary.SliceReused)
	})

	t.Run("should not reuse a kept slice made with automatic selection when this run requests an explicit source (010-geo-data-source-control FR-011)", func(t *testing.T) {
		// given: the kept slice's recorded base map ("europa-central-mapa")
		// happens to differ from the one requested now — automatic-to-explicit
		// is just another case of "the source requested now differs from the
		// one recorded"
		m := newFlightMocks(t)
		name := "mapa-b"
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").WithSelection(domain.SourceSelection{BaseMapName: &name}).Build()
		readyToFlyKept(m)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(plan, nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)
		m.geoSliceService.EXPECT().Generate(plan, domain.SourceSelection{BaseMapName: &name}).Return(slice, nil)
		m.geoSliceService.EXPECT().Export(slice, "/tmp/kept/slice.zip", false).Return(nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.False(t, summary.SliceReused)
	})

	t.Run("should reuse only what still matches when only a later parameter changed: the plan and the slice, not the frames, which the frame service's own set rule already decides", func(t *testing.T) {
		// given: the plan and slice are both reused; drawing is always delegated
		// as-is, so whether frames are kept is FrameService's own concern.
		m := newFlightMocks(t)
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").WithQuality(domain.VideoQualityHigh).Build()
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.workspace.EXPECT().EnsureDirectory("/tmp/kept").Return(nil)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(plan, nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).Return(domain.RenderSummary{Kept: 20}, nil)
		var videoRequest domain.VideoRequest
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.CameraPlan, r domain.VideoRequest, _ func(domain.VideoProgress)) (domain.VideoSummary, error) {
				videoRequest = r
				return domain.VideoSummary{}, nil
			})

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.True(t, summary.PlanReused)
		assert.True(t, summary.SliceReused)
		assert.Equal(t, domain.VideoQualityHigh, videoRequest.Quality)
	})

	t.Run("should reuse the plan and the slice when only the appearance changed, and pass the new appearance to the frames", func(t *testing.T) {
		// given: the plan and slice under --keep still match; only the
		// appearance differs from a previous run — FR-009
		m := newFlightMocks(t)
		green := builddomain.NewAppearanceBuilder().WithTrailColor(domain.RGB{R: 0x00, G: 0xFF, B: 0x00}).Build()
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").WithAppearance(green).Build()
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.workspace.EXPECT().EnsureDirectory("/tmp/kept").Return(nil)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(plan, nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)
		var frameRequest domain.FrameSetRequest
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.CameraPlan, _ domain.GeoSlice, r domain.FrameSetRequest, _ func(domain.RenderProgress)) (domain.RenderSummary, error) {
				frameRequest = r
				return domain.RenderSummary{Drawn: 20}, nil
			})
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).Return(domain.VideoSummary{}, nil)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.True(t, summary.PlanReused)
		assert.True(t, summary.SliceReused)
		assert.Equal(t, green, frameRequest.Appearance)
	})

	t.Run("should reuse the plan and the slice when only the overlay configuration changed, and pass the new overlay to the frames", func(t *testing.T) {
		// given: the plan and slice under --keep still match; only the
		// overlay configuration differs from a previous run (009-frame-overlays)
		m := newFlightMocks(t)
		distanceOnly, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{domain.OverlayBlockDistance})
		require.NoError(t, err)
		request := builddomain.NewFlightRequestBuilder().WithKeep("/tmp/kept").WithOverlay(distanceOnly).Build()
		m.videoService.EXPECT().CheckDestination(gomock.Any(), gomock.Any()).Return(nil)
		m.videoService.EXPECT().CheckEncoder(gomock.Any()).Return(anEncoder, nil)
		m.cameraPlanService.EXPECT().Generate(gomock.Any(), gomock.Any()).Return(plan, nil)
		m.workspace.EXPECT().EnsureDirectory("/tmp/kept").Return(nil)
		m.cameraPlanService.EXPECT().Load("/tmp/kept/plan.json").Return(plan, nil)
		m.geoSliceService.EXPECT().Load("/tmp/kept/slice.zip").Return(slice, nil)
		var frameRequest domain.FrameSetRequest
		m.frameService.EXPECT().DrawFrames(gomock.Any(), plan, slice, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ domain.CameraPlan, _ domain.GeoSlice, r domain.FrameSetRequest, _ func(domain.RenderProgress)) (domain.RenderSummary, error) {
				frameRequest = r
				return domain.RenderSummary{Drawn: 20}, nil
			})
		m.videoService.EXPECT().Assemble(gomock.Any(), plan, gomock.Any(), gomock.Any()).Return(domain.VideoSummary{}, nil)

		// when
		summary, err := m.service.Fly(context.Background(), strings.NewReader("track"), request, nil)

		// then
		require.NoError(t, err)
		assert.True(t, summary.PlanReused)
		assert.True(t, summary.SliceReused)
		assert.Equal(t, distanceOnly, frameRequest.Overlay)
	})
}
