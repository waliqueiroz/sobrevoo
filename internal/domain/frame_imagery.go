package domain

import (
	"fmt"
	"math"
	"sync"
)

// pixelState says what a pixel of terrain shows.
type pixelState uint8

const (
	// stateImage: an image of the map.
	stateImage pixelState = iota

	// stateNoMap: no image of the map is there — the tile is one the slice lacks,
	// or none exists at that place (beyond the latitude of Web Mercator).
	stateNoMap

	// stateNoElevation: the terrain is over a cell the file has no value for.
	stateNoElevation
)

// imagerySet is the tiles of one base map of the slice, at its level.
type imagerySet struct {
	source  string
	format  string
	level   int
	data    map[TileID][]byte
	missing map[TileID]struct{}
}

// imagery is the map of a slice as something to look up by position: which tile
// holds a point, its decoded image, and the color of a point as the camera sees
// it (research.md item 7).
type imagery struct {
	decoder TileDecoder
	sets    []imagerySet
	cache   *tileCache
}

func newImagery(tileSets []TileSet, decoder TileDecoder, cacheBytes int64) *imagery {
	im := &imagery{decoder: decoder, cache: newTileCache(cacheBytes)}

	for _, tileSet := range tileSets {
		set := imagerySet{
			source:  tileSet.Source.Name,
			format:  tileSet.Format,
			level:   tileSet.Detail.Chosen,
			data:    make(map[TileID][]byte, len(tileSet.Tiles)),
			missing: make(map[TileID]struct{}, len(tileSet.Missing)),
		}
		for _, tile := range tileSet.Tiles {
			set.data[tile.ID] = tile.Data
		}
		for _, id := range tileSet.Missing {
			set.missing[id] = struct{}{}
		}
		im.sets = append(im.sets, set)
	}
	return im
}

// mercator is the position of a coordinate on the whole map of Web Mercator,
// from 0 to 1 eastwards and southwards from the north-west corner; ok is false
// beyond the latitude the projection reaches, where no tile exists.
func mercator(lat, lon float64) (u, v float64, ok bool) {
	if math.Abs(lat) > MaxMercatorLatitude {
		return 0, 0, false
	}
	u = (lon + 180) / 360
	v = 0.5 - math.Log(math.Tan(math.Pi/4+degreesToRadians(lat)/2))/(2*math.Pi)
	return u, v, true
}

// tileRef names a tile of a set of the imagery.
type tileRef struct {
	set int
	id  TileID
}

// tileAt is the tile, of the set at index, that holds the position (u, v) of the
// map.
func (im *imagery) tileAt(set int, u, v float64) TileID {
	level := im.sets[set].level
	size := math.Ldexp(1, level)
	last := int(size) - 1

	return TileID{
		Level: level,
		X:     min(max(int(math.Floor(u*size)), 0), last),
		Y:     min(max(int(math.Floor(v*size)), 0), last),
	}
}

// find is the tile that holds a position: the present one of the first set that
// has it; if none does, the position has no image of the map, whether a set
// lists the tile as one it lacks or none knows of it.
func (im *imagery) find(u, v float64) (ref tileRef, state pixelState) {
	for i := range im.sets {
		id := im.tileAt(i, u, v)
		if _, ok := im.sets[i].data[id]; ok {
			return tileRef{set: i, id: id}, stateImage
		}
	}
	return tileRef{}, stateNoMap
}

// tileTexture is the image of a tile, with its mipmaps: level 0 is the tile,
// and each next one is the last at half the width and height, every pixel the
// average of four.
type tileTexture struct {
	levels []textureLevel
}

// textureLevel is one image of a tileTexture: RGB, 8 bits per channel.
type textureLevel struct {
	width, height int
	pix           []uint8
}

// newTileTexture builds the texture of a decoded tile. A pixel with some
// transparency is put over BackgroundColor, so a tile with a transparent stretch
// looks like an absence of map, not like a color that is not in the file.
func newTileTexture(image TileImage) *tileTexture {
	base := textureLevel{width: image.Width, height: image.Height, pix: make([]uint8, 3*image.Width*image.Height)}
	background := [3]int{int(BackgroundColor.R), int(BackgroundColor.G), int(BackgroundColor.B)}

	for i := 0; i < image.Width*image.Height; i++ {
		r, g, b, a := image.Pix[4*i], image.Pix[4*i+1], image.Pix[4*i+2], image.Pix[4*i+3]
		if a == 255 {
			base.pix[3*i], base.pix[3*i+1], base.pix[3*i+2] = r, g, b
			continue
		}
		alpha := int(a)
		base.pix[3*i] = uint8((int(r)*alpha + background[0]*(255-alpha) + 127) / 255)
		base.pix[3*i+1] = uint8((int(g)*alpha + background[1]*(255-alpha) + 127) / 255)
		base.pix[3*i+2] = uint8((int(b)*alpha + background[2]*(255-alpha) + 127) / 255)
	}

	texture := &tileTexture{levels: []textureLevel{base}}
	for last := base; last.width > 1 || last.height > 1; {
		last = halved(last)
		texture.levels = append(texture.levels, last)
	}
	return texture
}

// halved is a level at half the width and height, each pixel the average of
// four (with integer arithmetic, so it is the same everywhere).
func halved(level textureLevel) textureLevel {
	width, height := max(level.width/2, 1), max(level.height/2, 1)
	out := textureLevel{width: width, height: height, pix: make([]uint8, 3*width*height)}

	for y := 0; y < height; y++ {
		y0, y1 := min(2*y, level.height-1), min(2*y+1, level.height-1)
		for x := 0; x < width; x++ {
			x0, x1 := min(2*x, level.width-1), min(2*x+1, level.width-1)
			for channel := 0; channel < 3; channel++ {
				sum := int(level.pix[3*(y0*level.width+x0)+channel]) + int(level.pix[3*(y0*level.width+x1)+channel]) +
					int(level.pix[3*(y1*level.width+x0)+channel]) + int(level.pix[3*(y1*level.width+x1)+channel])
				out.pix[3*(y*width+x)+channel] = uint8((sum + 2) >> 2)
			}
		}
	}
	return out
}

func (l textureLevel) at(x, y int) (r, g, b float64) {
	offset := 3 * (y*l.width + x)
	return float64(l.pix[offset]), float64(l.pix[offset+1]), float64(l.pix[offset+2])
}

func (t *tileTexture) bytes() int64 {
	var total int64
	for _, level := range t.levels {
		total += int64(len(level.pix))
	}
	return total
}

// tileCache keeps the decoded tiles, up to a budget of memory: each is decoded
// once, however many goroutines ask for it at the same time, and the oldest
// are dropped to stay in the budget. What it keeps changes how long drawing
// takes, never what it draws.
type tileCache struct {
	mu      sync.Mutex
	entries map[tileRef]*tileEntry
	order   []tileRef
	bytes   int64
	budget  int64
}

type tileEntry struct {
	once    sync.Once
	texture *tileTexture
	err     error
}

func newTileCache(budget int64) *tileCache {
	return &tileCache{entries: map[tileRef]*tileEntry{}, budget: budget}
}

func (c *tileCache) get(ref tileRef, decode func() (*tileTexture, error)) (*tileTexture, error) {
	c.mu.Lock()
	entry, ok := c.entries[ref]
	if !ok {
		entry = &tileEntry{}
		c.entries[ref] = entry
		c.order = append(c.order, ref)
	}
	c.mu.Unlock()

	entry.once.Do(func() {
		entry.texture, entry.err = decode()
		if entry.texture == nil {
			return
		}

		c.mu.Lock()
		defer c.mu.Unlock()
		c.bytes += entry.texture.bytes()
		for c.bytes > c.budget && len(c.order) > 1 && c.order[0] != ref {
			oldest := c.order[0]
			c.order = c.order[1:]
			if dropped := c.entries[oldest]; dropped != nil && dropped.texture != nil {
				c.bytes -= dropped.texture.bytes()
			}
			delete(c.entries, oldest)
		}
	})
	if entry.texture == nil && entry.err == nil {
		// the decoding never finished (its goroutine was stopped): there is nothing
		// to read, and no one to say why
		return nil, fmt.Errorf("%w: a tile could not be decoded", ErrSliceFileInvalid)
	}
	return entry.texture, entry.err
}

// texture is the decoded image of a tile. A tile that is not an image is an
// invalid slice, and the error says which tile.
func (im *imagery) texture(ref tileRef) (*tileTexture, error) {
	return im.cache.get(ref, func() (*tileTexture, error) {
		set := im.sets[ref.set]
		image, err := im.decoder.Decode(set.format, set.data[ref.id])
		if err != nil {
			return nil, fmt.Errorf("%w: base map %q level %d x=%d y=%d: %w", ErrSliceFileInvalid, set.source, ref.id.Level, ref.id.X, ref.id.Y, err)
		}
		return newTileTexture(image), nil
	})
}

// sampler reads colors off the imagery for one goroutine: it remembers the
// last few tiles it used, so most lookups need no lock.
type sampler struct {
	imagery    *imagery
	pixelAngle float64

	recent [8]recentTile
	next   int
}

type recentTile struct {
	ref     tileRef
	texture *tileTexture
	used    bool
}

// newSampler makes a sampler for a camera whose pixels each cover pixelAngle
// radians.
func (im *imagery) newSampler(pixelAngle float64) *sampler {
	return &sampler{imagery: im, pixelAngle: pixelAngle}
}

func (s *sampler) texture(ref tileRef) (*tileTexture, error) {
	for i := range s.recent {
		if s.recent[i].used && s.recent[i].ref == ref {
			return s.recent[i].texture, nil
		}
	}

	texture, err := s.imagery.texture(ref)
	if err != nil {
		return nil, err
	}
	s.recent[s.next] = recentTile{ref: ref, texture: texture, used: true}
	s.next = (s.next + 1) % len(s.recent)
	return texture, nil
}

// color is the color of the map at a coordinate, as a camera sees it from
// distance meters, along a ray that descends by descent (the sine of its angle
// below the horizontal): the trilinear filter of the mipmaps by how many texels
// a pixel of the screen covers there. state is stateNoMap, with no color, where
// there is no image.
func (s *sampler) color(lat, lon, distance, descent float64) (RGB, pixelState, error) {
	u, v, ok := mercator(lat, lon)
	if !ok {
		return RGB{}, stateNoMap, nil
	}
	ref, state := s.imagery.find(u, v)
	if state != stateImage {
		return RGB{}, state, nil
	}

	center, err := s.texture(ref)
	if err != nil {
		return RGB{}, stateNoMap, err
	}

	// how many texels of the tile one pixel of the screen covers
	groundTexel := float64(EquatorResolution*math.Cos(degreesToRadians(lat))) / math.Ldexp(1, ref.id.Level) * 256 / float64(center.levels[0].width)
	texels := float64(s.pixelAngle*distance) / groundTexel / math.Sqrt(math.Max(descent, 0.1))
	lod := math.Log2(math.Max(texels, 1))

	last := len(center.levels) - 1
	lod = math.Min(lod, float64(last))
	fine := int(math.Floor(lod))
	weight := lod - float64(fine)
	coarse := min(fine+1, last)

	r0, g0, b0, err := s.bilinear(ref, center, fine, u, v)
	if err != nil {
		return RGB{}, stateNoMap, err
	}
	if weight == 0 || coarse == fine {
		return RGB{rounded(r0), rounded(g0), rounded(b0)}, stateImage, nil
	}
	r1, g1, b1, err := s.bilinear(ref, center, coarse, u, v)
	if err != nil {
		return RGB{}, stateNoMap, err
	}

	mix := func(a, b float64) uint8 { return rounded(float64(a*(1-weight)) + float64(b*weight)) }
	return RGB{mix(r0, r1), mix(g0, g1), mix(b0, b1)}, stateImage, nil
}

func rounded(v float64) uint8 {
	return uint8(math.Min(math.Max(math.Floor(v+0.5), 0), 255))
}

// bilinear reads the mip level of a tile at a position of the map, mixing the
// four texels around it. A texel across the border of the tile is the one of
// the neighbouring tile when the same base map has it, and the nearest one of
// this tile when it does not.
func (s *sampler) bilinear(ref tileRef, center *tileTexture, level int, u, v float64) (r, g, b float64, err error) {
	texture := center.levels[level]
	size := math.Ldexp(1, ref.id.Level)

	x := float64(u*size*float64(texture.width)) - float64(ref.id.X*texture.width) - 0.5
	y := float64(v*size*float64(texture.height)) - float64(ref.id.Y*texture.height) - 0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	wx, wy := x-x0, y-y0

	var channels [3]float64
	taps := [4]struct {
		dx, dy int
		weight float64
	}{
		{0, 0, float64((1 - wx) * (1 - wy))}, {1, 0, float64(wx * (1 - wy))},
		{0, 1, float64((1 - wx) * wy)}, {1, 1, float64(wx * wy)},
	}
	for _, tap := range taps {
		if tap.weight == 0 {
			continue
		}
		tr, tg, tb, err := s.texel(ref, texture, level, int(x0)+tap.dx, int(y0)+tap.dy)
		if err != nil {
			return 0, 0, 0, err
		}
		channels[0] += float64(tr * tap.weight)
		channels[1] += float64(tg * tap.weight)
		channels[2] += float64(tb * tap.weight)
	}
	return channels[0], channels[1], channels[2], nil
}

// texel reads the pixel (x, y) of a mip level of a tile, where x and y may be
// one outside the tile.
func (s *sampler) texel(ref tileRef, texture textureLevel, level, x, y int) (r, g, b float64, err error) {
	if x >= 0 && x < texture.width && y >= 0 && y < texture.height {
		r, g, b = texture.at(x, y)
		return r, g, b, nil
	}

	// which neighbouring tile, if any, the texel is in
	neighbour := ref.id
	switch {
	case x < 0:
		neighbour.X--
	case x >= texture.width:
		neighbour.X++
	}
	switch {
	case y < 0:
		neighbour.Y--
	case y >= texture.height:
		neighbour.Y++
	}
	last := 1<<ref.id.Level - 1
	if neighbour.X < 0 {
		neighbour.X = last // the map wraps around the world
	} else if neighbour.X > last {
		neighbour.X = 0
	}

	if neighbour.Y >= 0 && neighbour.Y <= last {
		if _, ok := s.imagery.sets[ref.set].data[neighbour]; ok {
			other, err := s.texture(tileRef{set: ref.set, id: neighbour})
			if err != nil {
				return 0, 0, 0, err
			}
			if level < len(other.levels) {
				level := other.levels[level]
				x, y = wrapped(x, texture.width, level.width), wrapped(y, texture.height, level.height)
				r, g, b = level.at(x, y)
				return r, g, b, nil
			}
		}
	}

	r, g, b = texture.at(min(max(x, 0), texture.width-1), min(max(y, 0), texture.height-1))
	return r, g, b, nil
}

// wrapped is the position of a texel that is one outside a mip level of a size,
// in the neighbouring tile, whose level has otherSize texels.
func wrapped(position, size, otherSize int) int {
	inside := ((position % size) + size) % size
	return min(inside*otherSize/size, otherSize-1)
}
