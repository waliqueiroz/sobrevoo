package domain_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/mockdomain"
)

var (
	smallFrame = domain.Resolution{Width: 64, Height: 36}
	mapColor   = domain.RGB{R: 10, G: 200, B: 10}
)

// solidTile is a 4 × 4 image of one opaque color.
func solidTile(c domain.RGB) domain.TileImage {
	pix := make([]uint8, 4*16)
	for i := 0; i < 16; i++ {
		pix[4*i], pix[4*i+1], pix[4*i+2], pix[4*i+3] = c.R, c.G, c.B, 255
	}
	return domain.NewTileImage(4, 4, pix)
}

// worldTile is the tile that covers the whole world, at level 0.
func worldTile() domain.Tile {
	return domain.Tile{ID: domain.TileID{Level: 0, X: 0, Y: 0}, Data: []byte("world")}
}

// sceneSlice is a slice of one grid of n × n cells of cell degrees, centered
// on (0, 0), with the given heights, and one tile for the whole world.
func sceneSlice(n int, cell float64, values []float32, tiles ...domain.Tile) domain.GeoSlice {
	half := float64(n) / 2 * cell
	grid := builddomain.NewElevationGridBuilder().
		WithWindow(domain.GridWindow{Rows: n, Cols: n}).
		WithOrigin(half, -half).
		WithCellSize(cell, cell).
		WithValues(values...).
		Build()
	tileSet := builddomain.NewTileSetBuilder().
		WithDetail(domain.DetailLevel{Ideal: 0, Chosen: 0, Min: 0, Max: 0}).
		WithTiles(tiles...).
		Build()
	return builddomain.NewGeoSliceBuilder().
		WithArea(domain.BoundingBox{MinLatitude: -half, MaxLatitude: half, MinLongitude: -half, MaxLongitude: half}).
		WithElevation(grid).
		WithTileSets(tileSet).
		Build()
}

func flat(n int, height float32) []float32 {
	values := make([]float32, n*n)
	for i := range values {
		values[i] = height
	}
	return values
}

func solidDecoder(t *testing.T, c domain.RGB) *mockdomain.MockTileDecoder {
	t.Helper()
	decoder := mockdomain.NewMockTileDecoder(gomock.NewController(t))
	decoder.EXPECT().Decode("png", gomock.Any()).Return(solidTile(c), nil).AnyTimes()
	return decoder
}

// planOf is a plan of frames whose camera is `from` and whose marker is
// `marker`, all with the same heading and tilt.
func planOf(heading, tilt, altitude float64, from, marker [2]float64, count int) domain.CameraPlan {
	frames := make([]domain.CameraFrame, count)
	for i := range frames {
		frames[i] = builddomain.NewCameraFrameBuilder().
			WithIndex(i).
			WithCameraPosition(from[0], from[1]).
			WithMarkerPosition(marker[0], marker[1]).
			WithCameraAltitude(altitude).
			WithHeading(heading).
			WithTilt(tilt).
			Build()
	}
	return builddomain.NewCameraPlanBuilder().WithFrames(frames...).Build()
}

func tuningWithWorkers(workers int) domain.RenderTuning {
	return builddomain.NewRenderTuningBuilder().WithWorkers(workers).Build()
}

// sceneAppearance is the tool's appearance of before this feature let it be
// chosen.
var sceneAppearance = builddomain.NewAppearanceBuilder().Build()

// sceneOverlay is disabled: the zero value of domain.OverlayConfig, so
// existing pixel assertions (about the terrain, the trail, the marker) are
// never touched by a screen overlay block — tests of the overlay itself
// live in frame_screen_overlay_test.go, and Test_Scene_Render_ScreenOverlay
// below is the one exception that turns it on.
var sceneOverlay domain.OverlayConfig

func rowIsAll(image domain.FrameImage, y int, c domain.RGB) bool {
	for x := 0; x < image.Resolution.Width; x++ {
		if image.At(x, y) != c {
			return false
		}
	}
	return true
}

func Test_NewScene(t *testing.T) {
	t.Run("should refuse a slice in which no sample has a value", func(t *testing.T) {
		// given
		values := make([]float32, 9)
		for i := range values {
			values[i] = float32(math.NaN())
		}
		slice := sceneSlice(3, 0.001, values, worldTile())

		// when
		_, err := domain.NewScene(slice, solidDecoder(t, mapColor), tuningWithWorkers(1), sceneAppearance, sceneOverlay)

		// then
		assert.ErrorIs(t, err, domain.ErrNoElevationData)
	})
}

func Test_Scene_Render(t *testing.T) {
	// a flat plain of 100 m, 2.2 km on a side, whose camera looks north from its south, 300 m up
	plain := func() domain.GeoSlice { return sceneSlice(40, 0.0005, flat(40, 100), worldTile()) }
	southOfCenter := [2]float64{-0.004, 0}
	center := [2]float64{0, 0}
	behindCamera := [2]float64{-0.008, 0} // south of a camera that looks north: nothing to draw

	t.Run("should draw an image of the resolution asked for, the sky above and the terrain below", func(t *testing.T) {
		// given
		scene, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		plan := planOf(0, 20, 300, southOfCenter, behindCamera, 3)

		// when
		image, stats, err := scene.Render(context.Background(), plan, 1, smallFrame)

		// then
		require.NoError(t, err)
		assert.Equal(t, smallFrame, image.Resolution)
		assert.Len(t, image.Pix, 3*64*36)
		assert.Equal(t, domain.FrameStats{}, stats)
		assert.True(t, rowIsAll(image, 0, sceneAppearance.BackgroundColor), "the top row looks above the horizon")
		assert.True(t, rowIsAll(image, 35, mapColor), "the bottom row looks at the ground under the camera")
	})

	t.Run("should draw a different image for a different appearance", func(t *testing.T) {
		// given
		plan := planOf(0, 20, 300, southOfCenter, behindCamera, 3)
		orange, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		white, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(2),
			builddomain.NewAppearanceBuilder().WithBackgroundColor(domain.RGB{R: 0xFF, G: 0xFF, B: 0xFF}).Build(), sceneOverlay)
		require.NoError(t, err)

		// when
		orangeImage, _, err1 := orange.Render(context.Background(), plan, 1, smallFrame)
		whiteImage, _, err2 := white.Render(context.Background(), plan, 1, smallFrame)

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, orangeImage.Pix, whiteImage.Pix)
	})

	t.Run("should draw the same image, byte by byte, for the same appearance twice", func(t *testing.T) {
		// given
		plan := planOf(0, 20, 300, southOfCenter, behindCamera, 3)
		first, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		second, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)

		// when
		firstImage, _, err1 := first.Render(context.Background(), plan, 1, smallFrame)
		secondImage, _, err2 := second.Render(context.Background(), plan, 1, smallFrame)

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, firstImage.Pix, secondImage.Pix)
	})

	t.Run("should never put sky under ground in a column", func(t *testing.T) {
		// given
		scene, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		plan := planOf(0, 20, 300, southOfCenter, behindCamera, 3)

		// when
		image, _, err := scene.Render(context.Background(), plan, 1, smallFrame)

		// then
		require.NoError(t, err)
		for x := 0; x < 64; x++ {
			seenGround := false
			for y := 0; y < 36; y++ {
				isGround := image.At(x, y) == mapColor
				if seenGround {
					assert.True(t, isGround, "column %d row %d", x, y)
				}
				seenGround = seenGround || isGround
			}
		}
	})

	t.Run("should draw the marker where the camera looks at it, and none when it is behind the camera", func(t *testing.T) {
		// given: the marker is 400 m ahead of the camera, on the ground
		scene, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		ahead := planOf(0, 45, 300, southOfCenter, [2]float64{-0.0004, 0}, 3)
		behind := planOf(180, 45, 300, southOfCenter, [2]float64{-0.0004, 0}, 3)

		// when
		withMarker, _, err1 := scene.Render(context.Background(), ahead, 1, smallFrame)
		without, _, err2 := scene.Render(context.Background(), behind, 1, smallFrame)

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		count := func(image domain.FrameImage) int {
			total := 0
			for y := 0; y < 36; y++ {
				for x := 0; x < 64; x++ {
					if image.At(x, y) == sceneAppearance.MarkerColor {
						total++
					}
				}
			}
			return total
		}
		assert.Greater(t, count(withMarker), 0)
		assert.Equal(t, 0, count(without))
	})

	t.Run("should turn the picture around with the heading of the frame", func(t *testing.T) {
		// given: a plain that is the same in every direction, and a camera at the middle of it
		scene, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		north := planOf(0, 45, 300, center, center, 2)
		south := planOf(180, 45, 300, center, center, 2)

		// when
		facingNorth, _, err1 := scene.Render(context.Background(), north, 0, smallFrame)
		facingSouth, _, err2 := scene.Render(context.Background(), south, 0, smallFrame)

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, facingNorth.Pix, facingSouth.Pix)
	})

	t.Run("should refuse a frame that is not in the plan", func(t *testing.T) {
		// given
		scene, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(1), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		plan := planOf(0, 45, 300, center, center, 3)

		// when
		_, _, tooBig := scene.Render(context.Background(), plan, 3, smallFrame)
		_, _, negative := scene.Render(context.Background(), plan, -1, smallFrame)

		// then
		assert.ErrorIs(t, tooBig, domain.ErrFrameOutOfRange)
		assert.ErrorIs(t, negative, domain.ErrFrameOutOfRange)
	})

	t.Run("should see the same fraction of ground at any resolution of the same shape", func(t *testing.T) {
		// given: a plain of 100 km, so the ground reaches almost to the horizon
		slice := sceneSlice(100, 0.01, flat(100, 100), worldTile())
		scene, err := domain.NewScene(slice, solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		plan := planOf(0, 20, 300, center, center, 2)
		firstGround := func(resolution domain.Resolution) float64 {
			image, _, err := scene.Render(context.Background(), plan, 0, resolution)
			require.NoError(t, err)
			for y := 0; y < resolution.Height; y++ {
				if image.At(resolution.Width/2, y) == mapColor {
					return float64(y) / float64(resolution.Height)
				}
			}
			return 1
		}

		// when
		small, large := firstGround(smallFrame), firstGround(domain.Resolution{Width: 128, Height: 72})

		// then
		assert.InDelta(t, small, large, 1.0/36)
	})
}

// roughScene is a terrain of random heights and a map whose color depends on
// the place, so a change in the drawing shows in the image.
func roughScene(t *testing.T, workers int) (*domain.Scene, domain.CameraPlan) {
	t.Helper()
	random := rand.New(rand.NewSource(11))
	values := make([]float32, 40*40)
	for i := range values {
		values[i] = float32(100 + 60*random.Float64())
	}
	slice := sceneSlice(40, 0.0005, values, worldTile())

	decoder := mockdomain.NewMockTileDecoder(gomock.NewController(t))
	pix := make([]uint8, 4*16)
	for i := 0; i < 16; i++ {
		pix[4*i], pix[4*i+1], pix[4*i+2], pix[4*i+3] = uint8(16*(i%4)), uint8(16*(i/4)), 128, 255
	}
	decoder.EXPECT().Decode("png", gomock.Any()).Return(domain.NewTileImage(4, 4, pix), nil).AnyTimes()

	scene, err := domain.NewScene(slice, decoder, tuningWithWorkers(workers), sceneAppearance, sceneOverlay)
	require.NoError(t, err)

	frames := make([]domain.CameraFrame, 6)
	for i := range frames {
		frames[i] = builddomain.NewCameraFrameBuilder().
			WithIndex(i).
			WithCameraPosition(-0.004+0.0002*float64(i), 0.0003*float64(i)).
			WithMarkerPosition(-0.001+0.0003*float64(i), 0.0004*float64(i)).
			WithCameraAltitude(250).WithHeading(10 * float64(i)).WithTilt(40).
			Build()
	}
	return scene, builddomain.NewCameraPlanBuilder().WithFrames(frames...).Build()
}

func Test_Scene_Render_Determinism(t *testing.T) {
	t.Run("should draw the same image with one goroutine and with eight", func(t *testing.T) {
		// given
		one, plan := roughScene(t, 1)
		eight, _ := roughScene(t, 8)

		// when
		first, _, err1 := one.Render(context.Background(), plan, 3, smallFrame)
		second, _, err2 := eight.Render(context.Background(), plan, 3, smallFrame)

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, first.Pix, second.Pix)
	})

	t.Run("should draw each frame the same whatever the order and the frames drawn before", func(t *testing.T) {
		// given
		alone, plan := roughScene(t, 3)
		shuffled, _ := roughScene(t, 5)
		want := map[int][]uint8{}
		for _, index := range []int{4, 1, 3} {
			image, _, err := alone.Render(context.Background(), plan, index, smallFrame)
			require.NoError(t, err)
			want[index] = image.Pix
		}

		// when
		for _, index := range []int{3, 4, 0, 1, 5, 2} {
			image, _, err := shuffled.Render(context.Background(), plan, index, smallFrame)
			require.NoError(t, err)

			// then
			if pix, ok := want[index]; ok {
				assert.Equal(t, pix, image.Pix, "frame %d", index)
			}
		}
	})

	t.Run("should draw the same image every time", func(t *testing.T) {
		// given
		scene, plan := roughScene(t, 4)

		// when
		first, _, _ := scene.Render(context.Background(), plan, 2, smallFrame)
		second, _, _ := scene.Render(context.Background(), plan, 2, smallFrame)

		// then
		assert.Equal(t, first.Pix, second.Pix)
	})

	t.Run("should draw the image the reference says, on any machine", func(t *testing.T) {
		// given: the pixels of one small frame, hashed. It guards the arithmetic: a compiler that fuses
		// a multiplication with a sum, or a function that differs between processors, changes it. If it
		// fails on another machine, fix the arithmetic, not this value; change it only with RenderVersion.
		scene, plan := roughScene(t, 2)

		// when
		image, _, err := scene.Render(context.Background(), plan, 2, smallFrame)

		// then
		require.NoError(t, err)
		sum := sha256.Sum256(image.Pix)
		assert.Equal(t, "7c65df9a868038182d4b902aa3f67e8de18e21d0950bda293316efbcb39ad399", hex.EncodeToString(sum[:]))
	})
}

// eastWestSlope is an n × n grid whose height grows only eastward, flat
// north-south — row r, column c: base + step*c. A negative step grows
// westward instead.
func eastWestSlope(n int, base, step float32) []float32 {
	values := make([]float32, n*n)
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			values[r*n+c] = base + step*float32(c)
		}
	}
	return values
}

// sawtoothSlope is an n × n grid whose height rises by step then resets
// every 4 columns, flat north-south — a real, strong local slope that
// alternates in sign every few cells, like the project's own synthetic
// relevo-sp.tif sample (test/samples): a coarse reading of it averages the
// alternation toward flat; a fine one does not.
func sawtoothSlope(n int, base, step float32) []float32 {
	values := make([]float32, n*n)
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			values[r*n+c] = base + step*float32(c%4)
		}
	}
	return values
}

// Test_Scene_Render_TerrainLight is the one file of tests that checks the
// terrain's directional lighting (016-terrain-lighting): every other test
// of this file draws a flat plain (flat), over which the lighting is
// always neutral, so none of their pixel assertions about the terrain are
// touched by it.
func Test_Scene_Render_TerrainLight(t *testing.T) {
	southOfCenter := [2]float64{-0.004, 0}
	behindCamera := [2]float64{-0.008, 0}
	plan := planOf(0, 20, 300, southOfCenter, behindCamera, 3)

	t.Run("should lighten a slope whose horizontal lean faces the light more than the map's own color", func(t *testing.T) {
		// given: a slope that rises eastward — its exposed face leans west, toward part of the
		// fixed light's direction (from the north-west)
		slice := sceneSlice(40, 0.0005, eastWestSlope(40, 100, 10), worldTile())
		scene, err := domain.NewScene(slice, solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)

		// when
		image, _, err := scene.Render(context.Background(), plan, 1, smallFrame)

		// then: the whole visible ground is one color (the slope's gradient, and so its lighting
		// factor, is the same everywhere on this uniformly sloped grid), lighter than the map's own
		require.NoError(t, err)
		lit := image.At(32, 35)
		assert.True(t, rowIsAll(image, 35, lit))
		assert.NotEqual(t, mapColor, lit)
		assert.Greater(t, int(lit.G), int(mapColor.G))
	})

	t.Run("should darken a slope whose horizontal lean faces away from the light more than the map's own color", func(t *testing.T) {
		// given: the mirrored slope, rising westward — its exposed face leans east, away from the light
		slice := sceneSlice(40, 0.0005, eastWestSlope(40, 100, -10), worldTile())
		scene, err := domain.NewScene(slice, solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)

		// when
		image, _, err := scene.Render(context.Background(), plan, 1, smallFrame)

		// then
		require.NoError(t, err)
		shaded := image.At(32, 35)
		assert.True(t, rowIsAll(image, 35, shaded))
		assert.NotEqual(t, mapColor, shaded)
		assert.Less(t, int(shaded.G), int(mapColor.G))
	})

	t.Run("should leave a flat surface's color exactly unchanged", func(t *testing.T) {
		// given
		slice := sceneSlice(40, 0.0005, flat(40, 100), worldTile())
		scene, err := domain.NewScene(slice, solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)

		// when
		image, _, err := scene.Render(context.Background(), plan, 1, smallFrame)

		// then
		require.NoError(t, err)
		assert.True(t, rowIsAll(image, 35, mapColor))
	})

	t.Run("should leave the no-map hatch exactly as it is, untouched by the terrain's lighting", func(t *testing.T) {
		// given: no tile at all, over a slope (not a flat plain) — the hatch never reaches
		// sampler.color, so it never goes through the lighting branch either
		slice := holeSlice(0, eastWestSlope(40, 100, 10), worldTileSet(nil))
		scene, err := domain.NewScene(slice, solidDecoder(t, mapColor), tuningWithWorkers(3), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		view := planOf(0, 45, 300, southOfCenter, behindCamera, 2)

		// when
		image, _, err := scene.Render(context.Background(), view, 1, smallFrame)

		// then
		require.NoError(t, err)
		for y := 0; y < 36; y++ {
			for x := 0; x < 64; x++ {
				if c := image.At(x, y); c != sceneAppearance.BackgroundColor {
					assert.Equal(t, noMapAt(x, y), c, "pixel %d,%d", x, y)
				}
			}
		}
	})

	t.Run("should leave the no-elevation checkerboard exactly as it is, untouched by the terrain's lighting", func(t *testing.T) {
		// given: a block of cells with no value, over a slope, with its tile
		slice := holeSlice(0, withHoleBlock(eastWestSlope(40, 100, 10)), worldTileSet([]domain.Tile{worldTile()}))
		scene, err := domain.NewScene(slice, solidDecoder(t, mapColor), tuningWithWorkers(3), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		view := planOf(0, 45, 300, southOfCenter, behindCamera, 2)

		// when
		image, _, err := scene.Render(context.Background(), view, 1, smallFrame)

		// then
		require.NoError(t, err)
		checkered := pixelsOf(image, func(x, y int, c domain.RGB) bool { return c == noElevationAt(x, y) })
		assert.Greater(t, checkered, 100)
		assert.Equal(t, checkered, pixelsOf(image, func(_, _ int, c domain.RGB) bool {
			return c == domain.NoElevationColors[0] || c == domain.NoElevationColors[1]
		}), "no other pixel has the colors of the checkerboard")
	})

	t.Run("should read a smoother lighting from farther away than from close up, over terrain whose local slope alternates", func(t *testing.T) {
		// given: a terrain with a real, strong, alternating local slope (like the project's own
		// synthetic relevo-sp.tif sample), its period a few dozen meters so a close-up pixel and a
		// far-away one differ by several periods, not a fraction of one; seen almost straight down,
		// at the same angle, from very close and from very far away
		slice := sceneSlice(200, 0.0001, sawtoothSlope(200, 10, 5), worldTile())
		scene, err := domain.NewScene(slice, solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		centerPoint := [2]float64{0, 0}
		nearPlan := planOf(0, 80, 100, centerPoint, centerPoint, 1)
		farPlan := planOf(0, 80, 5000, centerPoint, centerPoint, 1)

		// when
		nearImage, _, err1 := scene.Render(context.Background(), nearPlan, 0, smallFrame)
		farImage, _, err2 := scene.Render(context.Background(), farPlan, 0, smallFrame)

		// then: the average difference between neighboring pixels of the bottom row (ground, at
		// this angle, in both images) is smaller far away — a coarser level of the pyramid
		// smooths the alternation that a finer one still shows up close
		require.NoError(t, err1)
		require.NoError(t, err2)
		roughness := func(image domain.FrameImage, y int) float64 {
			total := 0
			for x := 1; x < image.Resolution.Width; x++ {
				a, b := image.At(x-1, y), image.At(x, y)
				diff := int(a.G) - int(b.G)
				if diff < 0 {
					diff = -diff
				}
				total += diff
			}
			return float64(total) / float64(image.Resolution.Width-1)
		}
		assert.Less(t, roughness(farImage, 35), roughness(nearImage, 35))
	})
}

// Test_Scene_Render_ScreenOverlay is the one test of this file that turns
// the screen overlay on — every other test keeps sceneOverlay disabled, so
// its pixel assertions about the terrain, the trail and the marker are
// never touched by an overlay block (009-frame-overlays). The overlay's own
// behavior is tested directly in frame_screen_overlay_test.go.
func Test_Scene_Render_ScreenOverlay(t *testing.T) {
	plain := func() domain.GeoSlice { return sceneSlice(40, 0.0005, flat(40, 100), worldTile()) }
	southOfCenter := [2]float64{-0.004, 0}
	behindCamera := [2]float64{-0.008, 0}

	full, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{domain.OverlayBlockDistance, domain.OverlayBlockElevation, domain.OverlayBlockTime, domain.OverlayBlockProfile})
	require.NoError(t, err)

	sceneWith := func(overlay domain.OverlayConfig) *domain.Scene {
		scene, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, overlay)
		require.NoError(t, err)
		return scene
	}

	t.Run("should draw a different image for a different overlay configuration, same plan, slice, resolution and appearance", func(t *testing.T) {
		// given
		plan := planOf(0, 20, 300, southOfCenter, behindCamera, 3)
		off, on := sceneWith(domain.OverlayConfig{}), sceneWith(full)

		// when
		offImage, _, err1 := off.Render(context.Background(), plan, 1, smallFrame)
		onImage, _, err2 := on.Render(context.Background(), plan, 1, smallFrame)

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, offImage.Pix, onImage.Pix)
	})

	t.Run("should draw the same image every time for the same overlay configuration", func(t *testing.T) {
		// given
		plan := planOf(0, 20, 300, southOfCenter, behindCamera, 3)
		first, second := sceneWith(full), sceneWith(full)

		// when
		firstImage, _, err1 := first.Render(context.Background(), plan, 1, smallFrame)
		secondImage, _, err2 := second.Render(context.Background(), plan, 1, smallFrame)

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, firstImage.Pix, secondImage.Pix)
	})

	t.Run("should draw the same frame twice from the same Scene into byte-identical images", func(t *testing.T) {
		// given: the same *Scene, so its vector-font glyph cache
		// (011-overlay-polish) is reused the second time, not rebuilt
		plan := planOf(0, 20, 300, southOfCenter, behindCamera, 3)
		scene := sceneWith(full)

		// when
		first, _, err1 := scene.Render(context.Background(), plan, 1, smallFrame)
		second, _, err2 := scene.Render(context.Background(), plan, 1, smallFrame)

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, first.Pix, second.Pix)
	})

}

func Test_Scene_Render_Stopping(t *testing.T) {
	t.Run("should draw nothing when the context is already done", func(t *testing.T) {
		// given
		scene, plan := roughScene(t, 2)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// when
		image, _, err := scene.Render(ctx, plan, 1, smallFrame)

		// then
		assert.ErrorIs(t, err, context.Canceled)
		assert.Empty(t, image.Pix)
	})

	t.Run("should stop drawing when the context is done meanwhile", func(t *testing.T) {
		// given: the tile that draws the first pixel cancels the context
		ctx, cancel := context.WithCancel(context.Background())
		decoder := mockdomain.NewMockTileDecoder(gomock.NewController(t))
		decoder.EXPECT().Decode("png", gomock.Any()).DoAndReturn(func(string, []byte) (domain.TileImage, error) {
			cancel()
			return solidTile(mapColor), nil
		}).AnyTimes()
		scene, err := domain.NewScene(sceneSlice(40, 0.0005, flat(40, 100), worldTile()), decoder, tuningWithWorkers(1), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		plan := planOf(0, 45, 300, [2]float64{0, 0}, [2]float64{0, 0}, 2)

		// when
		image, _, err := scene.Render(ctx, plan, 0, smallFrame)

		// then
		assert.ErrorIs(t, err, context.Canceled)
		assert.Empty(t, image.Pix)
	})

	t.Run("should fail as an invalid slice, and draw nothing, when a tile is not an image", func(t *testing.T) {
		// given
		decoder := mockdomain.NewMockTileDecoder(gomock.NewController(t))
		decoder.EXPECT().Decode("png", gomock.Any()).Return(domain.TileImage{}, errors.New("not a PNG")).AnyTimes()
		scene, err := domain.NewScene(sceneSlice(40, 0.0005, flat(40, 100), worldTile()), decoder, tuningWithWorkers(3), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		plan := planOf(0, 45, 300, [2]float64{0, 0}, [2]float64{0, 0}, 2)

		// when
		image, _, err := scene.Render(context.Background(), plan, 0, smallFrame)

		// then
		assert.ErrorIs(t, err, domain.ErrSliceFileInvalid)
		assert.Empty(t, image.Pix)
	})
}

// holeSlice is a plain of 40 × 40 cells of 0.0005° centered on (lat, 0), with the
// heights given, and the tiles given for the whole world (none, for a slice with
// no map at all).
func holeSlice(lat float64, values []float32, tileSet domain.TileSet) domain.GeoSlice {
	const n = 40
	const cell = 0.0005
	half := float64(n) / 2 * cell
	grid := builddomain.NewElevationGridBuilder().
		WithWindow(domain.GridWindow{Rows: n, Cols: n}).
		WithOrigin(lat+half, -half).
		WithCellSize(cell, cell).
		WithValues(values...).
		Build()
	return builddomain.NewGeoSliceBuilder().WithElevation(grid).WithTileSets(tileSet).Build()
}

func worldTileSet(tiles []domain.Tile, missing ...domain.TileID) domain.TileSet {
	return builddomain.NewTileSetBuilder().
		WithDetail(domain.DetailLevel{Chosen: 0}).
		WithTiles(tiles...).
		WithMissing(missing...).
		Build()
}

// withHoleBlock has no value for the cells of rows 18 to 22 and columns 18 to
// 22 of a 40 × 40 grid, which is where the camera of the frames below looks.
func withHoleBlock(values []float32) []float32 {
	for r := 18; r <= 22; r++ {
		for c := 18; c <= 22; c++ {
			values[r*40+c] = float32(math.NaN())
		}
	}
	return values
}

// patternColor is what the marks of a frame look like at a pixel.
func noMapAt(x, y int) domain.RGB {
	if (x+y)%domain.PatternPeriod < domain.PatternPeriod/2 {
		return domain.NoMapColors[0]
	}
	return domain.NoMapColors[1]
}

func noElevationAt(x, y int) domain.RGB {
	if (x/domain.PatternPeriod+y/domain.PatternPeriod)%2 == 0 {
		return domain.NoElevationColors[0]
	}
	return domain.NoElevationColors[1]
}

// pixelsOf counts the pixels of an image that satisfy a condition on their color
// and place.
func pixelsOf(image domain.FrameImage, keep func(x, y int, c domain.RGB) bool) int {
	count := 0
	for y := 0; y < image.Resolution.Height; y++ {
		for x := 0; x < image.Resolution.Width; x++ {
			if keep(x, y, image.At(x, y)) {
				count++
			}
		}
	}
	return count
}

func Test_Scene_Render_MissingData(t *testing.T) {
	southOfCenter := [2]float64{-0.004, 0}
	behindCamera := [2]float64{-0.008, 0}
	// the camera looks north and down at 45°, from 300 m up: it sees the ground from about 120 m to 720 m ahead
	view := func(marker [2]float64) domain.CameraPlan { return planOf(0, 45, 300, southOfCenter, marker, 2) }
	render := func(t *testing.T, slice domain.GeoSlice, decoder domain.TileDecoder, plan domain.CameraPlan) (domain.FrameImage, domain.FrameStats) {
		t.Helper()
		scene, err := domain.NewScene(slice, decoder, tuningWithWorkers(3), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		image, stats, err := scene.Render(context.Background(), plan, 1, smallFrame)
		require.NoError(t, err)
		return image, stats
	}
	isBackground := func(_, _ int, c domain.RGB) bool { return c == sceneAppearance.BackgroundColor }

	t.Run("should hatch the terrain of a tile the slice lacks, keeping its relief, and count a hole of the map", func(t *testing.T) {
		// given: the same plain, once with its tile and once with the tile listed as missing
		present := holeSlice(0, flat(40, 100), worldTileSet([]domain.Tile{worldTile()}))
		lacking := holeSlice(0, flat(40, 100), worldTileSet(nil, domain.TileID{Level: 0, X: 0, Y: 0}))

		// when
		withTile, withTileStats := render(t, present, solidDecoder(t, mapColor), view(behindCamera))
		without, withoutStats := render(t, lacking, solidDecoder(t, mapColor), view(behindCamera))

		// then
		assert.Equal(t, domain.FrameStats{}, withTileStats)
		assert.Equal(t, domain.FrameStats{MapHole: true}, withoutStats)
		assert.Equal(t, pixelsOf(withTile, isBackground), pixelsOf(without, isBackground), "the terrain is where it was")
		for y := 0; y < 36; y++ {
			for x := 0; x < 64; x++ {
				if without.At(x, y) != sceneAppearance.BackgroundColor {
					assert.Equal(t, noMapAt(x, y), without.At(x, y), "pixel %d,%d", x, y)
				}
			}
		}
	})

	t.Run("should hatch terrain where the slice knows nothing of the tile, and count a hole of the map", func(t *testing.T) {
		// given: a slice with no tile at all
		slice := holeSlice(0, flat(40, 100), worldTileSet(nil))

		// when
		image, stats := render(t, slice, solidDecoder(t, mapColor), view(behindCamera))

		// then
		assert.Equal(t, domain.FrameStats{MapHole: true}, stats)
		assert.Greater(t, pixelsOf(image, func(x, y int, c domain.RGB) bool { return c == noMapAt(x, y) }), 500)
	})

	t.Run("should checker the terrain over cells with no elevation, and count a hole of the elevation only", func(t *testing.T) {
		// given: a plain with a block of cells with no value, and its tile
		slice := holeSlice(0, withHoleBlock(flat(40, 100)), worldTileSet([]domain.Tile{worldTile()}))

		// when
		image, stats := render(t, slice, solidDecoder(t, mapColor), view(behindCamera))

		// then
		assert.Equal(t, domain.FrameStats{ElevationHole: true}, stats)
		checkered := pixelsOf(image, func(x, y int, c domain.RGB) bool { return c == noElevationAt(x, y) })
		assert.Greater(t, checkered, 100)
		assert.Equal(t, checkered, pixelsOf(image, func(_, _ int, c domain.RGB) bool {
			return c == domain.NoElevationColors[0] || c == domain.NoElevationColors[1]
		}), "no other pixel has the colors of the checkerboard")
		assert.Greater(t, pixelsOf(image, func(_, _ int, c domain.RGB) bool { return c == mapColor }), 100, "the rest is the map")
	})

	t.Run("should put the checkerboard before the hatch where both are missing, and count both kinds only if both show", func(t *testing.T) {
		// given: the block with no value, and no tile
		slice := holeSlice(0, withHoleBlock(flat(40, 100)), worldTileSet(nil))

		// when
		image, stats := render(t, slice, solidDecoder(t, mapColor), view(behindCamera))

		// then
		assert.Equal(t, domain.FrameStats{MapHole: true, ElevationHole: true}, stats)
		assert.Greater(t, pixelsOf(image, func(x, y int, c domain.RGB) bool { return c == noElevationAt(x, y) }), 100)
		assert.Greater(t, pixelsOf(image, func(x, y int, c domain.RGB) bool { return c == noMapAt(x, y) }), 100)
	})

	t.Run("should not count a hole of the map when every visible pixel of terrain is over a cell with no elevation", func(t *testing.T) {
		// given: every cell has no value but one, far from what the camera sees, and there is no tile
		values := make([]float32, 40*40)
		for i := range values {
			values[i] = float32(math.NaN())
		}
		values[0] = 100
		slice := holeSlice(0, values, worldTileSet(nil))

		// when
		_, stats := render(t, slice, solidDecoder(t, mapColor), view(behindCamera))

		// then
		assert.Equal(t, domain.FrameStats{ElevationHole: true}, stats)
	})

	t.Run("should have no mark and no count in a frame whose terrain is whole", func(t *testing.T) {
		// given: a plain with a hole of elevation that the camera does not see (the far north-east corner)
		values := flat(40, 100)
		values[39*40+39] = float32(math.NaN())
		slice := holeSlice(0, values, worldTileSet([]domain.Tile{worldTile()}))

		// when
		image, stats := render(t, slice, solidDecoder(t, mapColor), view(behindCamera))

		// then
		assert.Equal(t, domain.FrameStats{}, stats)
		marks := pixelsOf(image, func(_, _ int, c domain.RGB) bool {
			return c == domain.NoMapColors[0] || c == domain.NoMapColors[1] || c == domain.NoElevationColors[0] || c == domain.NoElevationColors[1]
		})
		assert.Equal(t, 0, marks)
	})

	t.Run("should not count as a hole what is background, outside the slice", func(t *testing.T) {
		// given: the camera looks at the horizon over a plain that ends before it
		slice := holeSlice(0, flat(40, 100), worldTileSet([]domain.Tile{worldTile()}))
		plan := planOf(0, 10, 300, southOfCenter, behindCamera, 2)

		// when
		image, stats := render(t, slice, solidDecoder(t, mapColor), plan)

		// then
		assert.Equal(t, domain.FrameStats{}, stats)
		assert.Greater(t, pixelsOf(image, isBackground), 200)
	})

	t.Run("should show a slice that has no map image at all with its relief and the hatch, not as a success in silence", func(t *testing.T) {
		// given: every tile missing
		slice := holeSlice(0, flat(40, 100), worldTileSet(nil, domain.TileID{Level: 0, X: 0, Y: 0}))

		// when
		image, stats := render(t, slice, solidDecoder(t, mapColor), view(behindCamera))

		// then
		assert.True(t, stats.MapHole)
		assert.Equal(t, 0, pixelsOf(image, func(_, _ int, c domain.RGB) bool { return c == mapColor }))
	})

	t.Run("should count the holes under the trail and the marker, which do not hide them", func(t *testing.T) {
		// given: the marker is on the block with no value
		slice := holeSlice(0, withHoleBlock(flat(40, 100)), worldTileSet([]domain.Tile{worldTile()}))

		// when
		image, stats := render(t, slice, solidDecoder(t, mapColor), view([2]float64{0, 0}))

		// then
		assert.True(t, stats.ElevationHole)
		assert.Greater(t, pixelsOf(image, func(_, _ int, c domain.RGB) bool { return c == sceneAppearance.MarkerColor }), 0)
	})

	t.Run("should hatch the terrain beyond the latitude of the map, where no tile can exist", func(t *testing.T) {
		// given: a plain at 85.5° north, with the tile of the whole world
		slice := holeSlice(85.5, flat(40, 100), worldTileSet([]domain.Tile{worldTile()}))
		plan := planOf(0, 45, 300, [2]float64{85.496, 0}, [2]float64{85.492, 0}, 2)

		// when
		image, stats := render(t, slice, solidDecoder(t, mapColor), plan)

		// then
		assert.True(t, stats.MapHole)
		assert.Greater(t, pixelsOf(image, func(x, y int, c domain.RGB) bool { return c == noMapAt(x, y) }), 500)
		assert.Equal(t, 0, pixelsOf(image, func(_, _ int, c domain.RGB) bool { return c == mapColor }))
	})

	t.Run("should draw the same frame, marks and counts with one goroutine and with eight", func(t *testing.T) {
		// given
		slice := holeSlice(0, withHoleBlock(flat(40, 100)), worldTileSet(nil))
		plan := view(behindCamera)
		draw := func(workers int) (domain.FrameImage, domain.FrameStats) {
			scene, err := domain.NewScene(slice, solidDecoder(t, mapColor), tuningWithWorkers(workers), sceneAppearance, sceneOverlay)
			require.NoError(t, err)
			image, stats, err := scene.Render(context.Background(), plan, 1, smallFrame)
			require.NoError(t, err)
			return image, stats
		}

		// when
		one, oneStats := draw(1)
		eight, eightStats := draw(8)

		// then
		assert.Equal(t, one.Pix, eight.Pix)
		assert.Equal(t, oneStats, eightStats)
	})
}

func Test_Scene_Render_Resolution(t *testing.T) {
	plain := func() domain.GeoSlice { return sceneSlice(40, 0.0005, flat(40, 100), worldTile()) }
	southOfCenter := [2]float64{-0.004, 0}
	// the marker is on the ground 300 m ahead of the camera, which looks north and down
	plan := planOf(0, 45, 300, southOfCenter, [2]float64{-0.0013, 0}, 2)

	markerAt := func(image domain.FrameImage) (x, y float64) {
		var sumX, sumY float64
		var count int
		for py := 0; py < image.Resolution.Height; py++ {
			for px := 0; px < image.Resolution.Width; px++ {
				if image.At(px, py) == sceneAppearance.MarkerColor {
					sumX, sumY, count = sumX+float64(px)+0.5, sumY+float64(py)+0.5, count+1
				}
			}
		}
		require.Greater(t, count, 0, "the marker is in the image")
		return sumX / float64(count), sumY / float64(count)
	}

	t.Run("should put the marker at the same relative place in two resolutions of the same shape", func(t *testing.T) {
		// given
		scene, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)

		// when
		small, _, err1 := scene.Render(context.Background(), plan, 1, domain.Resolution{Width: 320, Height: 180})
		large, _, err2 := scene.Render(context.Background(), plan, 1, domain.Resolution{Width: 640, Height: 360})

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		smallX, smallY := markerAt(small)
		largeX, largeY := markerAt(large)
		assert.InDelta(t, smallX, largeX/2, 1.0)
		assert.InDelta(t, smallY, largeY/2, 1.0)
	})

	t.Run("should see the same stretch of ground from top to bottom at any width, and more of it to the sides on a wider image", func(t *testing.T) {
		// given
		scene, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)
		behind := planOf(0, 45, 300, southOfCenter, [2]float64{-0.008, 0}, 2)

		// when
		narrow, _, err1 := scene.Render(context.Background(), behind, 0, domain.Resolution{Width: 180, Height: 180})
		wide, _, err2 := scene.Render(context.Background(), behind, 0, domain.Resolution{Width: 360, Height: 180})

		// then: the middle column of both shows the same vertical stretch: the same pixels
		require.NoError(t, err1)
		require.NoError(t, err2)
		for y := 0; y < 180; y++ {
			assert.Equal(t, narrow.At(90, y), wide.At(180, y), "row %d", y)
		}
	})

	t.Run("should draw a portrait and an ultrawide image", func(t *testing.T) {
		// given
		scene, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(2), sceneAppearance, sceneOverlay)
		require.NoError(t, err)

		// when
		portrait, _, err1 := scene.Render(context.Background(), plan, 1, domain.Resolution{Width: 180, Height: 320})
		ultrawide, _, err2 := scene.Render(context.Background(), plan, 1, domain.Resolution{Width: 640, Height: 180})

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, domain.Resolution{Width: 180, Height: 320}, portrait.Resolution)
		assert.Equal(t, domain.Resolution{Width: 640, Height: 180}, ultrawide.Resolution)
		assert.Len(t, portrait.Pix, 3*180*320)
		assert.Len(t, ultrawide.Pix, 3*640*180)
		markerAt(portrait)
		markerAt(ultrawide)
	})

	t.Run("should size the image of the biggest resolution allowed", func(t *testing.T) {
		// given
		biggest, err := domain.NewResolution(3840, 2160)
		require.NoError(t, err)

		// when
		image := domain.NewFrameImage(biggest, sceneAppearance.BackgroundColor)

		// then
		assert.Len(t, image.Pix, 3*3840*2160)
	})

	t.Run("should draw at the default resolution", func(t *testing.T) {
		// given
		scene, err := domain.NewScene(plain(), solidDecoder(t, mapColor), tuningWithWorkers(4), sceneAppearance, sceneOverlay)
		require.NoError(t, err)

		// when
		image, _, err := scene.Render(context.Background(), plan, 1, domain.Resolution{Width: 1920, Height: 1080})

		// then
		require.NoError(t, err)
		assert.Len(t, image.Pix, 3*1920*1080)
	})
}
