package domain_test

import (
	"context"
	"math"
	"runtime"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mockdomain"
)

// benchmarkScene is a terrain of 2000 × 2000 cells of 30 m (about 60 km on a
// side) over rolling relief, dressed with a mosaic of tiles of level 14, and a
// camera 600 m from its target at the given tilt.
func benchmarkScene(b *testing.B, tilt float64) (*domain.Scene, domain.CameraPlan) {
	b.Helper()
	const n = 2000
	cell := 30 / (math.Pi / 180 * 6371000)

	values := make([]float32, n*n)
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			x, y := float64(c)/40, float64(r)/40
			values[r*n+c] = float32(400 + 120*math.Sin(x)*math.Cos(y) + 40*math.Sin(3*x+y))
		}
	}
	half := float64(n) / 2 * cell
	grid := builddomain.NewElevationGridBuilder().
		WithWindow(domain.GridWindow{Rows: n, Cols: n}).WithOrigin(half, -half).WithCellSize(cell, cell).WithValues(values...).Build()

	var tiles []domain.Tile
	for x := 8192 - 12; x <= 8192+12; x++ {
		for y := 8192 - 12; y <= 8192+12; y++ {
			tiles = append(tiles, domain.Tile{ID: domain.TileID{Level: 14, X: x, Y: y}, Data: []byte{byte(x), byte(y)}})
		}
	}
	tileSet := builddomain.NewTileSetBuilder().WithDetail(domain.DetailLevel{Chosen: 14, Max: 14}).WithTiles(tiles...).Build()
	slice := builddomain.NewGeoSliceBuilder().WithElevation(grid).WithTileSets(tileSet).Build()

	decoder := mockdomain.NewMockTileDecoder(gomock.NewController(b))
	decoder.EXPECT().Decode("png", gomock.Any()).DoAndReturn(func(_ string, data []byte) (domain.TileImage, error) {
		pix := make([]uint8, 4*256*256)
		for py := 0; py < 256; py++ {
			for px := 0; px < 256; px++ {
				i := py*256 + px
				v := uint8(80)
				if ((px/16)+(py/16))%2 == 0 {
					v = 170
				}
				pix[4*i], pix[4*i+1], pix[4*i+2], pix[4*i+3] = v, uint8(int(data[0])*13), uint8(int(data[1])*17), 255
			}
		}
		return domain.NewTileImage(256, 256, pix), nil
	}).AnyTimes()

	scene, err := domain.NewScene(slice, decoder, builddomain.NewRenderTuningBuilder().WithWorkers(runtime.NumCPU()).Build(), builddomain.NewAppearanceBuilder().Build(), domain.OverlayConfig{})
	if err != nil {
		b.Fatal(err)
	}

	tiltRadians := tilt * math.Pi / 180
	frame := builddomain.NewCameraFrameBuilder().
		WithCameraPosition(-0.004, 0).WithMarkerPosition(0, 0).
		WithCameraAltitude(600 * math.Sin(tiltRadians)).WithHeading(20).WithTilt(tilt).Build()
	return scene, builddomain.NewCameraPlanBuilder().WithFrames(frame).Build()
}

func benchmarkRender(b *testing.B, tilt float64) {
	scene, plan := benchmarkScene(b, tilt)
	resolution := domain.Resolution{Width: 1920, Height: 1080}

	// the first frame decodes the tiles it needs; the ones after find them in the cache
	if _, _, err := scene.Render(context.Background(), plan, 0, resolution); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := scene.Render(context.Background(), plan, 0, resolution); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScene_Render_1080p(b *testing.B)         { benchmarkRender(b, 45) }
func BenchmarkScene_Render_1080p_LowTilt(b *testing.B) { benchmarkRender(b, 25) }
