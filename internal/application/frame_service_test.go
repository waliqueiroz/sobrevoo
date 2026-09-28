package application_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mockdomain"
)

var frameResolution = domain.Resolution{Width: 64, Height: 36}

// framesPlan is a plan of n frames whose camera flies over a small plain, its
// marker walking east.
func framesPlan(n int) domain.CameraPlan {
	frames := make([]domain.CameraFrame, n)
	for i := range frames {
		frames[i] = builddomain.NewCameraFrameBuilder().
			WithIndex(i).
			WithCameraPosition(-0.004, 0).
			WithMarkerPosition(-0.0004, 0.0003*float64(i)).
			WithCameraAltitude(300).WithHeading(0).WithTilt(45).
			Build()
	}
	return builddomain.NewCameraPlanBuilder().WithFrames(frames...).Build()
}

// framesSlice is a slice of a plain of 100 m, 2.2 km on a side, whose only tile
// covers the whole world.
func framesSlice() domain.GeoSlice {
	const n = 40
	values := make([]float32, n*n)
	for i := range values {
		values[i] = 100
	}
	half := float64(n) / 2 * 0.0005
	grid := builddomain.NewElevationGridBuilder().
		WithWindow(domain.GridWindow{Rows: n, Cols: n}).WithOrigin(half, -half).WithCellSize(0.0005, 0.0005).WithValues(values...).Build()
	tileSet := builddomain.NewTileSetBuilder().
		WithDetail(domain.DetailLevel{Chosen: 0}).
		WithTiles(domain.Tile{ID: domain.TileID{Level: 0, X: 0, Y: 0}, Data: []byte("world")}).
		Build()
	return builddomain.NewGeoSliceBuilder().WithElevation(grid).WithTileSets(tileSet).WithContentID("slice-file").Build()
}

// forPlan makes a slice the one of a plan: it says it was made from it, and its
// area covers the whole world, so it holds whatever the plan needs.
func forPlan(slice domain.GeoSlice, plan domain.CameraPlan) domain.GeoSlice {
	slice.PlanID = plan.ID()
	slice.Area = domain.BoundingBox{MinLatitude: -90, MaxLatitude: 90, MinLongitude: -180, MaxLongitude: 180}
	return slice
}

func greenTile() domain.TileImage {
	pix := make([]uint8, 4*16)
	for i := 0; i < 16; i++ {
		pix[4*i], pix[4*i+1], pix[4*i+2], pix[4*i+3] = 10, 200, 10, 255
	}
	return domain.NewTileImage(4, 4, pix)
}

type frameMocks struct {
	decoder    *mockdomain.MockTileDecoder
	repository *mockdomain.MockFrameRepository
	exporter   *mockdomain.MockFrameExporter
	tuning     domain.RenderTuning
	service    application.FrameService
}

func newFrameMocks(t *testing.T) frameMocks {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	m := frameMocks{
		decoder:    mockdomain.NewMockTileDecoder(mockCtrl),
		repository: mockdomain.NewMockFrameRepository(mockCtrl),
		exporter:   mockdomain.NewMockFrameExporter(mockCtrl),
		tuning:     builddomain.NewRenderTuningBuilder().WithWorkers(2).Build(),
	}
	m.decoder.EXPECT().Decode("png", gomock.Any()).Return(greenTile(), nil).AnyTimes()
	m.service = application.NewFrameService(m.decoder, m.repository, m.exporter, m.tuning, builddomain.NewSliceTuningBuilder().Build())
	return m
}

// emptyDirectory makes the destination a directory with no frame in it.
func (m frameMocks) emptyDirectory() {
	m.repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(domain.FrameDirectory{}, nil)
}

func Test_frameService_DrawFrame(t *testing.T) {
	plan := framesPlan(3)
	slice := forPlan(framesSlice(), plan)

	t.Run("should draw the frame asked for and export it once, marked with the set of the plan, the slice and the resolution, and with the plan itself", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		request := domain.SingleFrameRequest{Number: 2, Path: "/tmp/frame.png", Resolution: frameResolution, Overwrite: true}
		wantMark := domain.FrameMark{SetID: domain.NewFrameSetID(plan, slice, frameResolution, m.tuning, domain.Appearance{}), PlanID: plan.ID()}
		var exported domain.FrameImage
		m.exporter.EXPECT().Export(gomock.Any(), wantMark, "/tmp/frame.png", true).
			DoAndReturn(func(image domain.FrameImage, _ domain.FrameMark, _ string, _ bool) error {
				exported = image
				return nil
			})

		// when
		_, err := m.service.DrawFrame(context.Background(), plan, slice, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, frameResolution, exported.Resolution)
		assert.Len(t, exported.Pix, 3*64*36)

		scene, sceneErr := domain.NewScene(slice, m.decoder, m.tuning, domain.Appearance{})
		require.NoError(t, sceneErr)
		want, _, renderErr := scene.Render(context.Background(), plan, 2, frameResolution)
		require.NoError(t, renderErr)
		assert.Equal(t, want.Pix, exported.Pix, "it is frame 2 of the plan")

		other, _, _ := scene.Render(context.Background(), plan, 0, frameResolution)
		assert.NotEqual(t, other.Pix, exported.Pix, "and not another frame")
	})

	t.Run("should pass the request's appearance to the scene and to the mark, unaltered", func(t *testing.T) {
		// given
		orange := builddomain.NewAppearanceBuilder().Build()
		green := builddomain.NewAppearanceBuilder().WithTrailColor(domain.RGB{R: 0x00, G: 0xFF, B: 0x00}).Build()

		export := func(appearance domain.Appearance) (domain.FrameImage, domain.FrameMark) {
			m := newFrameMocks(t)
			var image domain.FrameImage
			var mark domain.FrameMark
			m.exporter.EXPECT().Export(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(i domain.FrameImage, mk domain.FrameMark, _ string, _ bool) error {
					image, mark = i, mk
					return nil
				})

			_, err := m.service.DrawFrame(context.Background(), plan, slice, domain.SingleFrameRequest{Number: 1, Path: "/tmp/f.png", Resolution: frameResolution, Appearance: appearance})
			require.NoError(t, err)
			return image, mark
		}

		// when
		orangeImage, orangeMark := export(orange)
		greenImage, greenMark := export(green)

		// then
		assert.NotEqual(t, orangeImage.Pix, greenImage.Pix)
		assert.NotEqual(t, orangeMark.SetID, greenMark.SetID)
	})

	t.Run("should say it drew one frame, at the resolution, and how long it took", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		m.exporter.EXPECT().Export(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		// when
		summary, err := m.service.DrawFrame(context.Background(), plan, slice, domain.SingleFrameRequest{Number: 1, Path: "/tmp/f.png", Resolution: frameResolution})

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, summary.Requested)
		assert.Equal(t, 1, summary.Drawn)
		assert.Equal(t, 0, summary.Kept)
		assert.Equal(t, frameResolution, summary.Resolution)
		assert.Greater(t, int64(summary.Elapsed), int64(0))
		assert.False(t, summary.Interrupted)
	})

	t.Run("should not touch the repository of a set of frames", func(t *testing.T) {
		// given: the mock repository has no expectations, so any call to it fails the test
		m := newFrameMocks(t)
		m.exporter.EXPECT().Export(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		// when
		_, err := m.service.DrawFrame(context.Background(), plan, slice, domain.SingleFrameRequest{Number: 0, Path: "/tmp/f.png", Resolution: frameResolution})

		// then
		require.NoError(t, err)
	})

	t.Run("should return the error of the exporter as it is, without counting the frame as drawn", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		want := domain.ErrFrameDestinationExists
		m.exporter.EXPECT().Export(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(want)

		// when
		summary, err := m.service.DrawFrame(context.Background(), plan, slice, domain.SingleFrameRequest{Number: 0, Path: "/tmp/f.png", Resolution: frameResolution})

		// then
		assert.Equal(t, want, err)
		assert.Equal(t, 0, summary.Drawn)
		assert.Equal(t, 1, summary.Requested)
	})

	t.Run("should be interrupted, exporting nothing, when the context is done", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// when
		summary, err := m.service.DrawFrame(ctx, plan, slice, domain.SingleFrameRequest{Number: 0, Path: "/tmp/f.png", Resolution: frameResolution})

		// then
		assert.ErrorIs(t, err, domain.ErrRenderInterrupted)
		assert.True(t, summary.Interrupted)
		assert.Equal(t, 0, summary.Drawn)
	})

	t.Run("should fail when a tile is not an image, exporting nothing", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		decoder := mockdomain.NewMockTileDecoder(mockCtrl)
		decoder.EXPECT().Decode("png", gomock.Any()).Return(domain.TileImage{}, errors.New("not a PNG")).AnyTimes()
		service := application.NewFrameService(decoder, mockdomain.NewMockFrameRepository(mockCtrl), mockdomain.NewMockFrameExporter(mockCtrl),
			builddomain.NewRenderTuningBuilder().WithWorkers(2).Build(), builddomain.NewSliceTuningBuilder().Build())

		// when
		summary, err := service.DrawFrame(context.Background(), plan, slice, domain.SingleFrameRequest{Number: 0, Path: "/tmp/f.png", Resolution: frameResolution})

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.Equal(t, 0, summary.Drawn)
	})

	t.Run("should refuse a slice with no elevation value at all", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		values := make([]float32, 9)
		for i := range values {
			values[i] = float32(math.NaN())
		}
		empty := forPlan(builddomain.NewGeoSliceBuilder().WithElevation(builddomain.NewElevationGridBuilder().WithValues(values...).Build()).Build(), plan)

		// when
		_, err := m.service.DrawFrame(context.Background(), plan, empty, domain.SingleFrameRequest{Number: 0, Path: "/tmp/f.png", Resolution: frameResolution})

		// then
		assert.ErrorIs(t, err, domain.ErrNoElevationData)
	})
}

func Test_frameService_DrawFrames(t *testing.T) {
	plan := framesPlan(3)
	slice := forPlan(framesSlice(), plan)
	request := domain.FrameSetRequest{Directory: "/tmp/frames", Resolution: frameResolution}

	t.Run("should draw every frame of the plan, in order, saving each with the set of the plan, the slice and the resolution and with the plan itself", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		m.emptyDirectory()
		wantMark := domain.FrameMark{SetID: domain.NewFrameSetID(plan, slice, frameResolution, m.tuning, domain.Appearance{}), PlanID: plan.ID()}
		saved := map[int]domain.FrameImage{}
		var order []int
		m.repository.EXPECT().Save("/tmp/frames", gomock.Any(), wantMark, gomock.Any()).Times(3).
			DoAndReturn(func(_ string, index int, _ domain.FrameMark, image domain.FrameImage) error {
				saved[index] = image
				order = append(order, index)
				return nil
			})

		// when
		summary, err := m.service.DrawFrames(context.Background(), plan, slice, request, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1, 2}, order)
		scene, sceneErr := domain.NewScene(slice, m.decoder, m.tuning, domain.Appearance{})
		require.NoError(t, sceneErr)
		for index, image := range saved {
			want, _, renderErr := scene.Render(context.Background(), plan, index, frameResolution)
			require.NoError(t, renderErr)
			assert.Equal(t, want.Pix, image.Pix, "frame %d", index)
		}
		assert.Equal(t, 3, summary.Requested)
		assert.Equal(t, 3, summary.Drawn)
		assert.Equal(t, 0, summary.Kept)
		assert.Equal(t, frameResolution, summary.Resolution)
		assert.Greater(t, int64(summary.Elapsed), int64(0))
	})

	t.Run("should stop at the frame it cannot save, saying what it had done, and return the error as it is", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		m.emptyDirectory()
		gomock.InOrder(
			m.repository.EXPECT().Save(gomock.Any(), 0, gomock.Any(), gomock.Any()).Return(nil),
			m.repository.EXPECT().Save(gomock.Any(), 1, gomock.Any(), gomock.Any()).Return(domain.ErrFrameDestinationInvalid),
		)

		// when
		summary, err := m.service.DrawFrames(context.Background(), plan, slice, request, nil)

		// then
		assert.Equal(t, domain.ErrFrameDestinationInvalid, err)
		assert.Equal(t, 1, summary.Drawn)
		assert.Equal(t, 3, summary.Requested)
		assert.False(t, summary.Interrupted)
	})

	t.Run("should be interrupted, saving nothing, when the context is done before the first frame", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		m.emptyDirectory()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// when
		summary, err := m.service.DrawFrames(ctx, plan, slice, request, nil)

		// then
		assert.ErrorIs(t, err, domain.ErrRenderInterrupted)
		assert.True(t, summary.Interrupted)
		assert.Equal(t, 0, summary.Drawn)
		assert.Equal(t, 3, summary.Requested)
	})

	t.Run("should stop when a tile is not an image, saying what it had done", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		decoder := mockdomain.NewMockTileDecoder(mockCtrl)
		decoder.EXPECT().Decode("png", gomock.Any()).Return(domain.TileImage{}, errors.New("not a PNG")).AnyTimes()
		repository := mockdomain.NewMockFrameRepository(mockCtrl)
		repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(domain.FrameDirectory{}, nil)
		service := application.NewFrameService(decoder, repository, mockdomain.NewMockFrameExporter(mockCtrl),
			builddomain.NewRenderTuningBuilder().WithWorkers(2).Build(), builddomain.NewSliceTuningBuilder().Build())

		// when
		summary, err := service.DrawFrames(context.Background(), plan, slice, request, nil)

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.Equal(t, 0, summary.Drawn)
	})
}

// holesSlice is a plain like framesSlice, with a block of cells with no value at
// its middle (which a camera south of it, looking north, sees) and, when withTile
// is false, no map tile at all.
func holesSlice(withTile bool) domain.GeoSlice {
	const n = 40
	values := make([]float32, n*n)
	for i := range values {
		values[i] = 100
	}
	for r := 18; r <= 22; r++ {
		for c := 18; c <= 22; c++ {
			values[r*n+c] = float32(math.NaN())
		}
	}
	half := float64(n) / 2 * 0.0005
	grid := builddomain.NewElevationGridBuilder().
		WithWindow(domain.GridWindow{Rows: n, Cols: n}).WithOrigin(half, -half).WithCellSize(0.0005, 0.0005).WithValues(values...).Build()

	tiles := []domain.Tile{}
	if withTile {
		tiles = append(tiles, domain.Tile{ID: domain.TileID{Level: 0, X: 0, Y: 0}, Data: []byte("world")})
	}
	tileSet := builddomain.NewTileSetBuilder().WithDetail(domain.DetailLevel{Chosen: 0}).WithTiles(tiles...).Build()
	return builddomain.NewGeoSliceBuilder().WithElevation(grid).WithTileSets(tileSet).WithContentID("slice-file").Build()
}

// lookingPlan is a plan whose first two frames look north at the middle of the
// plain, from the south, and whose third looks south, away from it.
func lookingPlan() domain.CameraPlan {
	frame := func(index int, heading float64) domain.CameraFrame {
		return builddomain.NewCameraFrameBuilder().WithIndex(index).
			WithCameraPosition(-0.004, 0).WithMarkerPosition(-0.008, 0).
			WithCameraAltitude(300).WithHeading(heading).WithTilt(45).Build()
	}
	return builddomain.NewCameraPlanBuilder().WithFrames(frame(0, 0), frame(1, 0), frame(2, 180)).Build()
}

func Test_frameService_Holes(t *testing.T) {
	plan := lookingPlan()

	t.Run("should count the frames with a hole of the elevation apart from those with a hole of the map", func(t *testing.T) {
		// given: the map is whole; the block with no value is seen by frames 0 and 1
		m := newFrameMocks(t)
		m.emptyDirectory()
		m.repository.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(3)

		// when
		summary, err := m.service.DrawFrames(context.Background(), plan, forPlan(holesSlice(true), plan), domain.FrameSetRequest{Directory: "/tmp/f", Resolution: frameResolution}, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, 3, summary.Drawn)
		assert.Equal(t, 2, summary.ElevationHoleFrames)
		assert.Equal(t, 0, summary.MapHoleFrames)
	})

	t.Run("should count a frame with both holes in both, and one with a single hole in that one", func(t *testing.T) {
		// given: no map tile at all: frames 0 and 1 also see the block with no value
		m := newFrameMocks(t)
		m.emptyDirectory()
		m.repository.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(3)

		// when
		summary, err := m.service.DrawFrames(context.Background(), plan, forPlan(holesSlice(false), plan), domain.FrameSetRequest{Directory: "/tmp/f", Resolution: frameResolution}, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, 3, summary.MapHoleFrames)
		assert.Equal(t, 2, summary.ElevationHoleFrames)
	})

	t.Run("should say, for a single frame, whether it had a hole of each kind", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		m.exporter.EXPECT().Export(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(2)

		// when
		looking, err1 := m.service.DrawFrame(context.Background(), plan, forPlan(holesSlice(true), plan), domain.SingleFrameRequest{Number: 0, Path: "/tmp/a.png", Resolution: frameResolution})
		away, err2 := m.service.DrawFrame(context.Background(), plan, forPlan(holesSlice(true), plan), domain.SingleFrameRequest{Number: 2, Path: "/tmp/b.png", Resolution: frameResolution})

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, 1, looking.ElevationHoleFrames)
		assert.Equal(t, 0, away.ElevationHoleFrames)
		assert.Equal(t, 0, away.MapHoleFrames)
	})
}

func Test_frameService_DrawFrames_Resuming(t *testing.T) {
	plan := framesPlan(5)
	slice := forPlan(framesSlice(), plan)
	request := domain.FrameSetRequest{Directory: "/tmp/frames", Resolution: frameResolution}

	// directoryWith is a directory that holds the given frames of the set of plan, slice and resolution.
	directoryWith := func(m frameMocks, indexes ...int) domain.FrameDirectory {
		id := domain.NewFrameSetID(plan, slice, frameResolution, m.tuning, domain.Appearance{})
		builder := builddomain.NewFrameDirectoryBuilder()
		for _, index := range indexes {
			builder.WithOursFrame(index, id)
		}
		return builder.Build()
	}

	t.Run("should look at the directory, with the resolution, before drawing anything", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		gomock.InOrder(
			m.repository.EXPECT().Inspect("/tmp/frames", frameResolution).Return(domain.FrameDirectory{}, nil),
			m.repository.EXPECT().Save("/tmp/frames", 0, gomock.Any(), gomock.Any()).Return(nil),
		)
		m.repository.EXPECT().Save("/tmp/frames", gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(4)

		// when
		_, err := m.service.DrawFrames(context.Background(), plan, slice, request, nil)

		// then
		require.NoError(t, err)
	})

	t.Run("should keep the frames that are there and draw only the ones that are not", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		m.repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(directoryWith(m, 0, 1), nil)
		var saved []int
		m.repository.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(3).
			DoAndReturn(func(_ string, index int, _ domain.FrameMark, _ domain.FrameImage) error {
				saved = append(saved, index)
				return nil
			})

		// when
		summary, err := m.service.DrawFrames(context.Background(), plan, slice, request, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, []int{2, 3, 4}, saved)
		assert.Equal(t, 5, summary.Requested)
		assert.Equal(t, 3, summary.Drawn)
		assert.Equal(t, 2, summary.Kept)
	})

	t.Run("should not even prepare the scene when every frame is there already", func(t *testing.T) {
		// given: the decoder and the exporter have no expectations, so any use of them fails the test
		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockFrameRepository(mockCtrl)
		service := application.NewFrameService(mockdomain.NewMockTileDecoder(mockCtrl), repository, mockdomain.NewMockFrameExporter(mockCtrl),
			builddomain.NewRenderTuningBuilder().WithWorkers(2).Build(), builddomain.NewSliceTuningBuilder().Build())
		id := domain.NewFrameSetID(plan, slice, frameResolution, builddomain.NewRenderTuningBuilder().WithWorkers(2).Build(), domain.Appearance{})
		repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(builddomain.NewFrameDirectoryBuilder().WithOursFrames(0, 4, id).Build(), nil)

		// when
		summary, err := service.DrawFrames(context.Background(), plan, slice, request, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, summary.Drawn)
		assert.Equal(t, 5, summary.Kept)
		assert.Equal(t, 5, summary.Requested)
	})

	t.Run("should report the progress after each frame it draws, counting the frames it kept", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		m.repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(directoryWith(m, 0), nil)
		m.repository.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(4)
		var reports []domain.RenderProgress

		// when
		_, err := m.service.DrawFrames(context.Background(), plan, slice, request, func(p domain.RenderProgress) { reports = append(reports, p) })

		// then
		require.NoError(t, err)
		require.Len(t, reports, 4)
		for i, report := range reports {
			assert.Equal(t, i+2, report.Done)
			assert.Equal(t, 5, report.Total)
			assert.Greater(t, int64(report.Elapsed), int64(0))
			if i > 0 {
				assert.GreaterOrEqual(t, int64(report.Elapsed), int64(reports[i-1].Elapsed))
			}
		}
	})

	t.Run("should stop when the context is done meanwhile, keeping the frames it saved", func(t *testing.T) {
		// given: the user interrupts after the second frame is done
		m := newFrameMocks(t)
		m.emptyDirectory()
		ctx, cancel := context.WithCancel(context.Background())
		m.repository.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(2)

		// when
		summary, err := m.service.DrawFrames(ctx, plan, slice, request, func(p domain.RenderProgress) {
			if p.Done == 2 {
				cancel()
			}
		})

		// then
		assert.ErrorIs(t, err, domain.ErrRenderInterrupted)
		assert.True(t, summary.Interrupted)
		assert.Equal(t, 2, summary.Drawn)
		assert.Equal(t, 5, summary.Requested)
	})

	t.Run("should draw nothing when the directory cannot be looked at", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		m.repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(domain.FrameDirectory{}, domain.ErrFrameDestinationInvalid)

		// when
		summary, err := m.service.DrawFrames(context.Background(), plan, slice, request, nil)

		// then
		assert.Equal(t, domain.ErrFrameDestinationInvalid, err)
		assert.Equal(t, 0, summary.Drawn)
	})
}

func Test_frameService_DrawFrames_Protection(t *testing.T) {
	plan := framesPlan(3)
	slice := forPlan(framesSlice(), plan)
	other := domain.FrameSetID("a-previous-flight")

	t.Run("should remove the frames of a previous set the plan has no number for before it draws, and count them", func(t *testing.T) {
		// given: the previous flight had six frames
		m := newFrameMocks(t)
		request := domain.FrameSetRequest{Directory: "/tmp/frames", Resolution: frameResolution, Overwrite: true}
		m.repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(builddomain.NewFrameDirectoryBuilder().WithOursFrames(0, 5, other).Build(), nil)
		gomock.InOrder(
			m.repository.EXPECT().Remove("/tmp/frames", []int{3, 4, 5}).Return(nil),
			m.repository.EXPECT().Save("/tmp/frames", 0, gomock.Any(), gomock.Any()).Return(nil),
		)
		m.repository.EXPECT().Save("/tmp/frames", gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(2)

		// when
		summary, err := m.service.DrawFrames(context.Background(), plan, slice, request, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, 3, summary.Drawn)
		assert.Equal(t, 0, summary.Kept)
		assert.Equal(t, 3, summary.Removed)
	})

	t.Run("should not call remove when there is nothing to remove", func(t *testing.T) {
		// given: the mock repository has no expectation for Remove, so calling it fails the test
		m := newFrameMocks(t)
		request := domain.FrameSetRequest{Directory: "/tmp/frames", Resolution: frameResolution, Overwrite: true}
		m.repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(builddomain.NewFrameDirectoryBuilder().WithOursFrames(0, 2, other).Build(), nil)
		m.repository.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(3)

		// when
		summary, err := m.service.DrawFrames(context.Background(), plan, slice, request, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, summary.Removed)
		assert.Equal(t, 3, summary.Drawn)
	})

	t.Run("should draw nothing when it cannot remove what it has to", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		request := domain.FrameSetRequest{Directory: "/tmp/frames", Resolution: frameResolution, Overwrite: true}
		m.repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(builddomain.NewFrameDirectoryBuilder().WithOursFrames(0, 5, other).Build(), nil)
		m.repository.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(domain.ErrFrameDestinationInvalid)

		// when
		summary, err := m.service.DrawFrames(context.Background(), plan, slice, request, nil)

		// then
		assert.Equal(t, domain.ErrFrameDestinationInvalid, err)
		assert.Equal(t, 0, summary.Drawn)
		assert.Equal(t, 0, summary.Removed)
	})

	t.Run("should touch nothing when the directory holds frames of another set and overwrite was not asked", func(t *testing.T) {
		// given: the decoder, the exporter and everything of the repository but Inspect have no expectations
		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockFrameRepository(mockCtrl)
		service := application.NewFrameService(mockdomain.NewMockTileDecoder(mockCtrl), repository, mockdomain.NewMockFrameExporter(mockCtrl),
			builddomain.NewRenderTuningBuilder().WithWorkers(2).Build(), builddomain.NewSliceTuningBuilder().Build())
		repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(builddomain.NewFrameDirectoryBuilder().WithOursFrames(0, 2, other).Build(), nil)

		// when
		summary, err := service.DrawFrames(context.Background(), plan, slice, domain.FrameSetRequest{Directory: "/tmp/frames", Resolution: frameResolution}, nil)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameSetConflict)
		assert.Equal(t, 0, summary.Drawn)
		assert.Equal(t, 0, summary.Removed)
	})

	t.Run("should hand the choice to overwrite to the exporter of a single frame, without asking the repository", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		m.exporter.EXPECT().Export(gomock.Any(), gomock.Any(), "/tmp/frame.png", false).Return(domain.ErrFrameDestinationExists)

		// when
		_, err := m.service.DrawFrame(context.Background(), plan, slice, domain.SingleFrameRequest{Number: 0, Path: "/tmp/frame.png", Resolution: frameResolution})

		// then
		assert.ErrorIs(t, err, domain.ErrFrameDestinationExists)
	})
}

func Test_frameService_DrawFrames_Appearance(t *testing.T) {
	plan := framesPlan(3)
	slice := forPlan(framesSlice(), plan)
	orange := builddomain.NewAppearanceBuilder().Build()
	green := builddomain.NewAppearanceBuilder().WithTrailColor(domain.RGB{R: 0x00, G: 0xFF, B: 0x00}).Build()

	t.Run("should refuse, as any other set, a directory that holds frames of the same plan/slice/resolution but a different appearance, without overwrite", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		repository := mockdomain.NewMockFrameRepository(mockCtrl)
		service := application.NewFrameService(mockdomain.NewMockTileDecoder(mockCtrl), repository, mockdomain.NewMockFrameExporter(mockCtrl),
			builddomain.NewRenderTuningBuilder().WithWorkers(2).Build(), builddomain.NewSliceTuningBuilder().Build())
		tuning := builddomain.NewRenderTuningBuilder().WithWorkers(2).Build()
		orangeID := domain.NewFrameSetID(plan, slice, frameResolution, tuning, orange)
		repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(builddomain.NewFrameDirectoryBuilder().WithOursFrames(0, 2, orangeID).Build(), nil)

		// when: asking for the green appearance now, over a directory drawn in orange
		summary, err := service.DrawFrames(context.Background(), plan, slice, domain.FrameSetRequest{Directory: "/tmp/frames", Resolution: frameResolution, Appearance: green}, nil)

		// then
		assert.ErrorIs(t, err, domain.ErrFrameSetConflict)
		assert.Equal(t, 0, summary.Drawn)
	})

	t.Run("should keep the frames already there when the appearance asked for is the same as before", func(t *testing.T) {
		// given
		m := newFrameMocks(t)
		id := domain.NewFrameSetID(plan, slice, frameResolution, m.tuning, orange)
		m.repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(builddomain.NewFrameDirectoryBuilder().WithOursFrames(0, 2, id).Build(), nil)

		// when
		summary, err := m.service.DrawFrames(context.Background(), plan, slice, domain.FrameSetRequest{Directory: "/tmp/frames", Resolution: frameResolution, Appearance: orange}, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, 3, summary.Kept)
		assert.Equal(t, 0, summary.Drawn)
	})
}

// strictFrameService is a service whose ports have no expectations, so any use of
// a port fails the test: a slice refused by a check must touch nothing.
func strictFrameService(t *testing.T) application.FrameService {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	return application.NewFrameService(mockdomain.NewMockTileDecoder(mockCtrl), mockdomain.NewMockFrameRepository(mockCtrl), mockdomain.NewMockFrameExporter(mockCtrl),
		builddomain.NewRenderTuningBuilder().WithWorkers(2).Build(), builddomain.NewSliceTuningBuilder().Build())
}

func Test_frameService_Checks(t *testing.T) {
	plan := framesPlan(3)
	good := forPlan(framesSlice(), plan)
	single := domain.SingleFrameRequest{Number: 0, Path: "/tmp/f.png", Resolution: frameResolution}
	set := domain.FrameSetRequest{Directory: "/tmp/frames", Resolution: frameResolution}

	fromOtherPlan := func() domain.GeoSlice {
		slice := good
		slice.PlanID = framesPlan(4).ID()
		return slice
	}
	tooSmall := func() domain.GeoSlice {
		slice := good
		slice.Area = domain.BoundingBox{MinLatitude: -0.0001, MaxLatitude: 0.0001, MinLongitude: -0.0001, MaxLongitude: 0.0001}
		return slice
	}
	vector := func() domain.GeoSlice {
		slice := good
		slice.TileSets = []domain.TileSet{builddomain.NewTileSetBuilder().WithFormat("pbf").Build()}
		return slice
	}
	noElevation := func() domain.GeoSlice {
		nan := float32(math.NaN())
		slice := forPlan(builddomain.NewGeoSliceBuilder().WithElevation(builddomain.NewElevationGridBuilder().
			WithValues(nan, nan, nan, nan, nan, nan, nan, nan, nan).Build()).Build(), plan)
		return slice
	}

	t.Run("should refuse a slice of another plan, touching nothing, for one frame and for all", func(t *testing.T) {
		// given / when
		_, errOne := strictFrameService(t).DrawFrame(context.Background(), plan, fromOtherPlan(), single)
		summary, errAll := strictFrameService(t).DrawFrames(context.Background(), plan, fromOtherPlan(), set, nil)

		// then
		assert.ErrorIs(t, errOne, domain.ErrSliceDoesNotMatchPlan)
		assert.ErrorIs(t, errAll, domain.ErrSliceDoesNotMatchPlan)
		assert.Equal(t, 0, summary.Drawn)
	})

	t.Run("should refuse a slice that does not cover the plan, touching nothing, for one frame and for all", func(t *testing.T) {
		// given / when
		_, errOne := strictFrameService(t).DrawFrame(context.Background(), plan, tooSmall(), single)
		_, errAll := strictFrameService(t).DrawFrames(context.Background(), plan, tooSmall(), set, nil)

		// then
		assert.ErrorIs(t, errOne, domain.ErrSliceDoesNotCoverPlan)
		assert.ErrorIs(t, errAll, domain.ErrSliceDoesNotCoverPlan)
	})

	t.Run("should refuse vector tiles, touching nothing, for one frame and for all", func(t *testing.T) {
		// given / when
		_, errOne := strictFrameService(t).DrawFrame(context.Background(), plan, vector(), single)
		_, errAll := strictFrameService(t).DrawFrames(context.Background(), plan, vector(), set, nil)

		// then
		assert.ErrorIs(t, errOne, domain.ErrTileFormatUnsupported)
		assert.ErrorIs(t, errAll, domain.ErrTileFormatUnsupported)
	})

	t.Run("should refuse a slice with no elevation value, touching nothing, for one frame and for all", func(t *testing.T) {
		// given / when
		_, errOne := strictFrameService(t).DrawFrame(context.Background(), plan, noElevation(), single)
		_, errAll := strictFrameService(t).DrawFrames(context.Background(), plan, noElevation(), set, nil)

		// then
		assert.ErrorIs(t, errOne, domain.ErrNoElevationData)
		assert.ErrorIs(t, errAll, domain.ErrNoElevationData)
	})

	t.Run("should check the plan first, then the coverage, then the format, then the elevation", func(t *testing.T) {
		// given: a slice that is wrong in every way, and then in fewer
		everything := noElevation()
		everything.PlanID = "another-plan"
		everything.Area = tooSmall().Area
		everything.TileSets = vector().TileSets
		noPlan := everything
		noPlan.PlanID = plan.ID()
		noCoverNoElevation := noPlan
		noCoverNoElevation.Area = good.Area

		// when
		_, first := strictFrameService(t).DrawFrame(context.Background(), plan, everything, single)
		_, second := strictFrameService(t).DrawFrame(context.Background(), plan, noPlan, single)
		_, third := strictFrameService(t).DrawFrame(context.Background(), plan, noCoverNoElevation, single)

		// then
		assert.ErrorIs(t, first, domain.ErrSliceDoesNotMatchPlan)
		assert.ErrorIs(t, second, domain.ErrSliceDoesNotCoverPlan)
		assert.ErrorIs(t, third, domain.ErrTileFormatUnsupported)
	})

	t.Run("should stop at a tile that is not an image, naming it, with the frames it had drawn saved and counted", func(t *testing.T) {
		// given: two frames look at the west of the meridian of 0°, whose tiles are fine, and the third at the east, whose are not
		mockCtrl := gomock.NewController(t)
		decoder := mockdomain.NewMockTileDecoder(mockCtrl)
		decoder.EXPECT().Decode("png", []byte("west")).Return(greenTile(), nil).AnyTimes()
		decoder.EXPECT().Decode("png", []byte("east")).Return(domain.TileImage{}, errors.New("unexpected EOF")).AnyTimes()
		repository := mockdomain.NewMockFrameRepository(mockCtrl)
		service := application.NewFrameService(decoder, repository, mockdomain.NewMockFrameExporter(mockCtrl),
			builddomain.NewRenderTuningBuilder().WithWorkers(1).Build(), builddomain.NewSliceTuningBuilder().Build())

		frame := func(index int, lon float64) domain.CameraFrame {
			return builddomain.NewCameraFrameBuilder().WithIndex(index).
				WithCameraPosition(-0.004, lon).WithMarkerPosition(-0.008, lon).WithCameraAltitude(300).WithHeading(0).WithTilt(45).Build()
		}
		twoSides := builddomain.NewCameraPlanBuilder().WithFrames(frame(0, -0.008), frame(1, -0.008), frame(2, 0.008)).Build()
		tiles := builddomain.NewTileSetBuilder().WithDetail(domain.DetailLevel{Chosen: 14, Max: 14}).WithTiles(
			domain.Tile{ID: domain.TileID{Level: 14, X: 8191, Y: 8191}, Data: []byte("west")}, domain.Tile{ID: domain.TileID{Level: 14, X: 8191, Y: 8192}, Data: []byte("west")},
			domain.Tile{ID: domain.TileID{Level: 14, X: 8192, Y: 8191}, Data: []byte("east")}, domain.Tile{ID: domain.TileID{Level: 14, X: 8192, Y: 8192}, Data: []byte("east")},
		).Build()
		slice := forPlan(framesSlice(), twoSides)
		slice.TileSets = []domain.TileSet{tiles}

		repository.EXPECT().Inspect(gomock.Any(), gomock.Any()).Return(domain.FrameDirectory{}, nil)
		gomock.InOrder(
			repository.EXPECT().Save(gomock.Any(), 0, gomock.Any(), gomock.Any()).Return(nil),
			repository.EXPECT().Save(gomock.Any(), 1, gomock.Any(), gomock.Any()).Return(nil),
		)

		// when
		summary, err := service.DrawFrames(context.Background(), twoSides, slice, set, nil)

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.ErrorContains(t, err, "level 14 x=8192")
		assert.Equal(t, 2, summary.Drawn)
		assert.Equal(t, 3, summary.Requested)
	})
}
