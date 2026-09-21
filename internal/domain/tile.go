package domain

//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/base_map_reader.go -package mockdomain . BaseMapReader

import (
	"fmt"
	"math"
)

// BaseMapReader reads the content of a registered base map file: the levels
// of detail it offers and its image tiles. Concrete implementations (one per
// recognized format) live in internal/infra/outbound/basemapreader. Like
// GeoDataInspector, it takes a path: the adapter opens the file itself.
type BaseMapReader interface {
	// Levels reports the range of levels of detail the file offers.
	Levels(path string) (LevelRange, error)

	// ReadTiles reads the requested tiles at level. A requested tile the
	// file does not contain is not an error: it is reported in
	// TileRead.Missing. A file that cannot be read fails with
	// ErrGeoDataContentUnreadable.
	ReadTiles(path string, level int, ids []TileID) (TileRead, error)
}

// LevelRange is the range of levels of detail (zoom levels) a base map
// offers, both ends inclusive.
type LevelRange struct {
	Min, Max int
}

// TileID identifies a map tile in the XYZ scheme of Web Mercator: Level is
// the zoom level, X grows eastwards and Y grows southwards from the
// north-west corner of the map. Any other scheme a file uses (MBTiles counts
// rows from the south) is converted by the adapter.
type TileID struct {
	Level, X, Y int
}

// before reports whether id comes before other: by level, then column, then
// row.
func (id TileID) before(other TileID) bool {
	if id.Level != other.Level {
		return id.Level < other.Level
	}
	if id.X != other.X {
		return id.X < other.X
	}
	return id.Y < other.Y
}

// TileRange is a rectangle of tiles of one level, all ends inclusive.
type TileRange struct {
	Level                  int
	MinX, MaxX, MinY, MaxY int
}

// Bounds is the geographic area the tiles of the range cover, in the Web
// Mercator projection: columns split the longitudes evenly and rows the
// projected latitudes. A range never crosses the antimeridian.
func (r TileRange) Bounds() BoundingBox {
	size := math.Ldexp(1, r.Level)
	latitude := func(row float64) float64 {
		return math.Atan(math.Sinh(math.Pi*(1-2*row/size))) * 180 / math.Pi
	}

	return BoundingBox{
		MinLatitude:  latitude(float64(r.MaxY + 1)),
		MaxLatitude:  latitude(float64(r.MinY)),
		MinLongitude: float64(r.MinX)/size*360 - 180,
		MaxLongitude: float64(r.MaxX+1)/size*360 - 180,
	}
}

// Tile is one image tile, exactly as the file stores it (it is never
// decoded or reprocessed).
type Tile struct {
	ID   TileID
	Data []byte
}

// TileRead is what BaseMapReader.ReadTiles found.
type TileRead struct {
	// Format is the image format of the tiles ("png", "jpg", "webp", ...).
	Format string

	Tiles   []Tile
	Missing []TileID
}

// DetailLevel is the level of detail chosen for one base map (FR-006).
type DetailLevel struct {
	// Ideal is the level the camera distance asks for; Chosen is Ideal
	// limited to the range [Min, Max] the file offers.
	Ideal, Chosen, Min, Max int

	// Reason is the short reason for the choice ("within the source's
	// range", ...) and Explanation says how the ideal level was reached.
	Reason, Explanation string
}

// TileSet is the tiles of one registered base map, at one level of detail.
type TileSet struct {
	Source GeoDataSource
	Detail DetailLevel

	// Format is the image format of the tiles.
	Format string

	// Tiles are sorted by (level, x, y); Missing are the tiles the slice
	// needed but the source does not contain, sorted the same way.
	Tiles   []Tile
	Missing []TileID
}

// maxDetailLevel bounds the level of detail asked for, however close the
// camera gets.
const maxDetailLevel = 30

// DetailLevel chooses the level of detail of a base map for a flight
// (FR-006, FR-007): the smallest level whose tiles are at least as fine, on
// the ground, as a screen pixel is at the nearest the camera gets, with the
// tolerance of TexelScreenRatio, limited to what the source offers. The
// resolution of a tile depends on the latitude, and the finest one is needed
// where the ground is widest per degree, so the reference latitude is the
// one of the area closest to the equator.
func (t SliceTuning) DetailLevel(minCameraDistance, verticalFOVDegrees float64, area BoundingBox, offered LevelRange) DetailLevel {
	footprint := t.TexelScreenRatio * 2 * minCameraDistance * math.Tan(verticalFOVDegrees*math.Pi/360) / t.ReferenceHeightPixels
	needed := EquatorResolution * math.Cos(area.closestLatitudeToEquator()*math.Pi/180) / footprint

	ideal := maxDetailLevel
	if footprint > 0 && !math.IsInf(needed, 0) {
		ideal = int(math.Min(math.Max(math.Ceil(math.Log2(needed)), 0), maxDetailLevel))
	}

	chosen := min(max(ideal, offered.Min), offered.Max)

	reason := "within the source's range"
	switch {
	case ideal > offered.Max:
		reason = "above the source's maximum level"
	case ideal < offered.Min:
		reason = "below the source's minimum level"
	}

	explanation := fmt.Sprintf("nearest camera distance %.1f m, area closest to the equator at latitude %.2f, tiles of at most %.2f m/px",
		minCameraDistance, area.closestLatitudeToEquator(), footprint)

	return DetailLevel{
		Ideal:       ideal,
		Chosen:      chosen,
		Min:         offered.Min,
		Max:         offered.Max,
		Reason:      reason,
		Explanation: explanation,
	}
}
