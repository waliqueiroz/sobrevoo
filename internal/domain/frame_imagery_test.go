package domain

import (
	"errors"
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDecoder decodes the "images" a test made up: the bytes of a tile are the
// key of an image, and it counts how many times each was decoded.
type fakeDecoder struct {
	mu     sync.Mutex
	images map[string]TileImage
	calls  map[string]int
	fail   map[string]error
}

func newFakeDecoder() *fakeDecoder {
	return &fakeDecoder{images: map[string]TileImage{}, calls: map[string]int{}, fail: map[string]error{}}
}

func (d *fakeDecoder) Decode(_ string, data []byte) (TileImage, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	key := string(data)
	d.calls[key]++
	if err := d.fail[key]; err != nil {
		return TileImage{}, err
	}
	return d.images[key], nil
}

// solidImage is a 4 × 4 tile of one color, with the given alpha.
func solidImage(r, g, b, a uint8) TileImage {
	pix := make([]uint8, 4*16)
	for i := 0; i < 16; i++ {
		pix[4*i], pix[4*i+1], pix[4*i+2], pix[4*i+3] = r, g, b, a
	}
	return NewTileImage(4, 4, pix)
}

// geoOf is the coordinate of a position (u, v) of the map of Web Mercator.
func geoOf(u, v float64) (lat, lon float64) {
	return (2*math.Atan(math.Exp((0.5-v)*2*math.Pi)) - math.Pi/2) * 180 / math.Pi, u*360 - 180
}

// oneSet is imagery of one base map at level 1: two tiles side by side in the
// north row, (0, 0) red and (1, 0) blue, both of which the decoder has.
func oneSet(decoder *fakeDecoder, cache int64, tiles ...string) *imagery {
	decoder.images["red"] = solidImage(255, 0, 0, 255)
	decoder.images["blue"] = solidImage(0, 0, 255, 255)

	set := TileSet{Source: GeoDataSource{Name: "map"}, Format: "png", Detail: DetailLevel{Chosen: 1}}
	for _, name := range tiles {
		x := 0
		if name == "blue" {
			x = 1
		}
		set.Tiles = append(set.Tiles, Tile{ID: TileID{Level: 1, X: x, Y: 0}, Data: []byte(name)})
	}
	return newImagery([]TileSet{set}, decoder, cache)
}

func Test_mercator(t *testing.T) {
	t.Run("should put the origin of the map at its middle and the west edge at zero", func(t *testing.T) {
		// given / when
		u, v, ok := mercator(0, 0)
		west, _, _ := mercator(0, -180)

		// then
		require.True(t, ok)
		assert.InDelta(t, 0.5, u, 1e-12)
		assert.InDelta(t, 0.5, v, 1e-12)
		assert.InDelta(t, 0.0, west, 1e-12)
	})

	t.Run("should put the north limit of the projection at the top of the map", func(t *testing.T) {
		// given / when
		_, v, ok := mercator(MaxMercatorLatitude, 0)

		// then
		require.True(t, ok)
		assert.InDelta(t, 0.0, v, 1e-9)
	})

	t.Run("should have nothing beyond the latitude of the projection", func(t *testing.T) {
		// given / when
		_, _, north := mercator(85.06, 0)
		_, _, south := mercator(-89, 0)

		// then
		assert.False(t, north)
		assert.False(t, south)
	})

	t.Run("should be the inverse of the coordinates of the map", func(t *testing.T) {
		// given
		lat, lon := geoOf(0.3, 0.6)

		// when
		u, v, ok := mercator(lat, lon)

		// then
		require.True(t, ok)
		assert.InDelta(t, 0.3, u, 1e-9)
		assert.InDelta(t, 0.6, v, 1e-9)
	})
}

func Test_imagery_find(t *testing.T) {
	first := TileSet{
		Source: GeoDataSource{Name: "a"}, Detail: DetailLevel{Chosen: 1},
		Tiles:   []Tile{{ID: TileID{Level: 1, X: 0, Y: 0}, Data: []byte("a00")}},
		Missing: []TileID{{Level: 1, X: 1, Y: 0}},
	}
	second := TileSet{
		Source: GeoDataSource{Name: "b"}, Detail: DetailLevel{Chosen: 1},
		Tiles: []Tile{{ID: TileID{Level: 1, X: 0, Y: 0}, Data: []byte("b00")}, {ID: TileID{Level: 1, X: 1, Y: 0}, Data: []byte("b10")}},
	}
	im := newImagery([]TileSet{first, second}, newFakeDecoder(), 1<<20)

	t.Run("should be the tile that holds the position", func(t *testing.T) {
		// given / when
		ref, state := im.find(0.25, 0.25)

		// then
		assert.Equal(t, stateImage, state)
		assert.Equal(t, TileID{Level: 1, X: 0, Y: 0}, ref.id)
	})

	t.Run("should prefer the present tile of the first base map that has it", func(t *testing.T) {
		// given / when
		ref, state := im.find(0.25, 0.25)

		// then
		assert.Equal(t, stateImage, state)
		assert.Equal(t, 0, ref.set)
	})

	t.Run("should use a present tile of another base map when the first lacks it", func(t *testing.T) {
		// given / when
		ref, state := im.find(0.75, 0.25)

		// then
		assert.Equal(t, stateImage, state)
		assert.Equal(t, 1, ref.set)
		assert.Equal(t, TileID{Level: 1, X: 1, Y: 0}, ref.id)
	})

	t.Run("should have no image where every base map lacks the tile, or knows nothing of it", func(t *testing.T) {
		// given
		onlyMissing := newImagery([]TileSet{first}, newFakeDecoder(), 1<<20)

		// when
		_, lacking := onlyMissing.find(0.75, 0.25)
		_, unknown := onlyMissing.find(0.25, 0.75)

		// then
		assert.Equal(t, stateNoMap, lacking)
		assert.Equal(t, stateNoMap, unknown)
	})

	t.Run("should keep a position on the last row or column inside the map", func(t *testing.T) {
		// given / when
		id := im.tileAt(0, 1, 1)

		// then
		assert.Equal(t, TileID{Level: 1, X: 1, Y: 1}, id)
	})
}

func Test_tileTexture(t *testing.T) {
	t.Run("should have the tile as its first level and a level at half the size after each", func(t *testing.T) {
		// given / when
		texture := newTileTexture(solidImage(10, 20, 30, 255))

		// then
		require.Len(t, texture.levels, 3)
		assert.Equal(t, 4, texture.levels[0].width)
		assert.Equal(t, 2, texture.levels[1].width)
		assert.Equal(t, 1, texture.levels[2].width)
	})

	t.Run("should average four pixels, rounding half up, for each pixel of the next level", func(t *testing.T) {
		// given: the four pixels at the top left have reds of 10, 20, 30 and 41
		image := solidImage(0, 0, 0, 255)
		set := func(x, y int, r uint8) { image.Pix[4*(y*4+x)] = r }
		set(0, 0, 10)
		set(1, 0, 20)
		set(0, 1, 30)
		set(1, 1, 41)

		// when
		texture := newTileTexture(image)

		// then: (10 + 20 + 30 + 41 + 2) >> 2 = 25
		assert.Equal(t, uint8(25), texture.levels[1].pix[0])
		assert.Equal(t, uint8(0), texture.levels[1].pix[3])
	})

	t.Run("should put a transparent pixel over the background and mix a half transparent one", func(t *testing.T) {
		// given
		transparent := newTileTexture(solidImage(200, 100, 50, 0))
		half := newTileTexture(solidImage(200, 100, 50, 128))

		// when
		r0, g0, b0 := transparent.levels[0].at(0, 0)
		r1, g1, b1 := half.levels[0].at(0, 0)

		// then
		assert.Equal(t, [3]float64{float64(BackgroundColor.R), float64(BackgroundColor.G), float64(BackgroundColor.B)}, [3]float64{r0, g0, b0})
		assert.Equal(t, float64((200*128+int(BackgroundColor.R)*127+127)/255), r1)
		assert.Equal(t, float64((100*128+int(BackgroundColor.G)*127+127)/255), g1)
		assert.Equal(t, float64((50*128+int(BackgroundColor.B)*127+127)/255), b1)
	})
}

func Test_sampler_color(t *testing.T) {
	t.Run("should give the color of a solid tile whatever the distance", func(t *testing.T) {
		// given
		decoder := newFakeDecoder()
		im := oneSet(decoder, 1<<20, "red")
		sampler := im.newSampler(1)
		lat, lon := geoOf(0.25, 0.25)

		// when
		near, nearState, nearErr := sampler.color(lat, lon, 10, 1)
		far, farState, farErr := sampler.color(lat, lon, 1e9, 0.1)

		// then
		require.NoError(t, nearErr)
		require.NoError(t, farErr)
		assert.Equal(t, stateImage, nearState)
		assert.Equal(t, stateImage, farState)
		assert.Equal(t, RGB{255, 0, 0}, near)
		assert.Equal(t, RGB{255, 0, 0}, far)
	})

	t.Run("should be the average of a checkerboard when the pixel covers many texels", func(t *testing.T) {
		// given: black and white, pixel by pixel
		decoder := newFakeDecoder()
		image := solidImage(0, 0, 0, 255)
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				if (x+y)%2 == 1 {
					offset := 4 * (y*4 + x)
					image.Pix[offset], image.Pix[offset+1], image.Pix[offset+2] = 255, 255, 255
				}
			}
		}
		decoder.images["checker"] = image
		set := TileSet{Source: GeoDataSource{Name: "map"}, Format: "png", Detail: DetailLevel{Chosen: 1},
			Tiles: []Tile{{ID: TileID{Level: 1, X: 0, Y: 0}, Data: []byte("checker")}}}
		im := newImagery([]TileSet{set}, decoder, 1<<20)
		lat, lon := geoOf(0.1875, 0.1875) // the center of the texel of column 1 and row 1, a black one

		// when: a pixel that covers dozens of texels, then one that covers a fraction of one
		far, _, err := im.newSampler(1).color(lat, lon, 1e9, 1)
		require.NoError(t, err)
		near, _, err := im.newSampler(1e-9).color(lat, lon, 1, 1)
		require.NoError(t, err)

		// then
		assert.Equal(t, RGB{128, 128, 128}, far)
		assert.Contains(t, []RGB{{0, 0, 0}, {255, 255, 255}}, near, "the sharp level keeps a texel's own color, near its center")
	})

	t.Run("should get coarser as the distance grows", func(t *testing.T) {
		// given: a tile of a black half and a white half
		decoder := newFakeDecoder()
		image := solidImage(0, 0, 0, 255)
		for y := 0; y < 4; y++ {
			for x := 2; x < 4; x++ {
				offset := 4 * (y*4 + x)
				image.Pix[offset], image.Pix[offset+1], image.Pix[offset+2] = 255, 255, 255
			}
		}
		decoder.images["halves"] = image
		set := TileSet{Source: GeoDataSource{Name: "map"}, Format: "png", Detail: DetailLevel{Chosen: 1},
			Tiles: []Tile{{ID: TileID{Level: 1, X: 0, Y: 0}, Data: []byte("halves")}}}
		im := newImagery([]TileSet{set}, decoder, 1<<20)
		sampler := im.newSampler(1)
		lat, lon := geoOf(0.25/8*3, 0.2) // texel column 1, the black half, in the world of 8 texels

		// when
		var previous uint8
		monotonic := true
		for i, distance := range []float64{1, 1e7, 3e7, 1e8, 1e9} {
			color, _, err := sampler.color(lat, lon, distance, 1)
			require.NoError(t, err)
			if i > 0 && color.R < previous {
				monotonic = false
			}
			previous = color.R
		}

		// then: the black half fades into the average of the tile
		assert.True(t, monotonic)
		assert.Greater(t, previous, uint8(100))
	})

	t.Run("should blend with the neighbouring tile at the border when the same base map has it", func(t *testing.T) {
		// given: red to the west of the border, blue to the east
		decoder := newFakeDecoder()
		im := oneSet(decoder, 1<<20, "red", "blue")
		lat, lon := geoOf(0.5, 0.2) // right on the border

		// when
		color, state, err := im.newSampler(1e-9).color(lat, lon, 1, 1)

		// then
		require.NoError(t, err)
		assert.Equal(t, stateImage, state)
		assert.InDelta(t, 128, float64(color.R), 1)
		assert.InDelta(t, 128, float64(color.B), 1)
	})

	t.Run("should use the nearest texel of the tile when the neighbour is missing", func(t *testing.T) {
		// given: only the blue tile, east of the border
		decoder := newFakeDecoder()
		im := oneSet(decoder, 1<<20, "blue")
		lat, lon := geoOf(0.5, 0.2)

		// when
		color, state, err := im.newSampler(1e-9).color(lat, lon, 1, 1)

		// then
		require.NoError(t, err)
		assert.Equal(t, stateImage, state)
		assert.Equal(t, RGB{0, 0, 255}, color)
	})

	t.Run("should have no image where there is no tile or no projection", func(t *testing.T) {
		// given
		decoder := newFakeDecoder()
		im := oneSet(decoder, 1<<20, "red")
		sampler := im.newSampler(1)
		lat, lon := geoOf(0.25, 0.75)

		// when
		_, noTile, err1 := sampler.color(lat, lon, 10, 1)
		_, noProjection, err2 := sampler.color(88, 10, 10, 1)

		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, stateNoMap, noTile)
		assert.Equal(t, stateNoMap, noProjection)
	})

	t.Run("should decode each tile once, however many goroutines ask for it at the same time", func(t *testing.T) {
		// given
		decoder := newFakeDecoder()
		im := oneSet(decoder, 1<<20, "red", "blue")
		lat, lon := geoOf(0.25, 0.25)

		// when
		var wait sync.WaitGroup
		for i := 0; i < 8; i++ {
			wait.Add(1)
			go func() {
				defer wait.Done()
				sampler := im.newSampler(1)
				for j := 0; j < 50; j++ {
					_, _, _ = sampler.color(lat, lon, 10, 1)
				}
			}()
		}
		wait.Wait()

		// then
		assert.Equal(t, 1, decoder.calls["red"])
	})

	t.Run("should draw the same colors with a cache too small to keep the tiles", func(t *testing.T) {
		// given
		roomy := oneSet(newFakeDecoder(), 1<<20, "red", "blue")
		decoder := newFakeDecoder()
		tiny := oneSet(decoder, 1, "red", "blue")
		west, westLon := geoOf(0.25, 0.25)
		east, eastLon := geoOf(0.75, 0.25)

		// when
		for i := 0; i < 5; i++ {
			for _, at := range [][2]float64{{west, westLon}, {east, eastLon}} {
				want, _, err := roomy.newSampler(1).color(at[0], at[1], 10, 1)
				require.NoError(t, err)
				got, _, err := tiny.newSampler(1).color(at[0], at[1], 10, 1)
				require.NoError(t, err)

				// then
				assert.Equal(t, want, got)
			}
		}
		assert.Greater(t, decoder.calls["red"]+decoder.calls["blue"], 2, "the tiles were dropped and decoded again")
	})

	t.Run("should fail as an invalid slice, saying which tile, when it is not an image", func(t *testing.T) {
		// given
		decoder := newFakeDecoder()
		im := oneSet(decoder, 1<<20, "red")
		decoder.fail["red"] = errors.New("unexpected EOF")
		lat, lon := geoOf(0.25, 0.25)

		// when
		_, _, err := im.newSampler(1).color(lat, lon, 10, 1)

		// then
		assert.ErrorIs(t, err, ErrSliceFileInvalid)
		assert.ErrorContains(t, err, `base map "map" level 1 x=0 y=0`)
		assert.ErrorContains(t, err, "unexpected EOF")
	})
}
