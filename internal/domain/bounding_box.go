package domain

import (
	"math"
	"sort"
)

// BoundingBox is the geographic area covered by a set of points (FR-022,
// FR-023).
type BoundingBox struct {
	MinLatitude  float64
	MaxLatitude  float64
	MinLongitude float64
	MaxLongitude float64

	// CrossesAntimeridian is true when the route crosses the 180th
	// meridian. In that case, the occupied area is the one going from
	// MaxLongitude to MinLongitude "the outside way" (through ±180°), not
	// the direct span between the two values.
	CrossesAntimeridian bool
}

// BoundingBox returns the geographic area covered by the route. Latitude
// never needs special handling (it does not wrap around). Longitude is
// unwrapped by walking the points in order and accumulating each
// consecutive delta, so a route that crosses the antimeridian produces a
// short, correct occupied area instead of one spanning nearly the whole
// planet (FR-024) — no assumption about hemisphere or region is made
// (research.md item 6).
func (r Route) BoundingBox() BoundingBox {
	points := r.Points
	if len(points) == 0 {
		return BoundingBox{}
	}

	minLat, maxLat := points[0].Latitude, points[0].Latitude
	unwrapped := make([]float64, len(points))
	unwrapped[0] = points[0].Longitude

	for i := 1; i < len(points); i++ {
		lat := points[i].Latitude
		minLat = math.Min(minLat, lat)
		maxLat = math.Max(maxLat, lat)

		delta := points[i].Longitude - points[i-1].Longitude
		switch {
		case delta > 180:
			delta -= 360
		case delta < -180:
			delta += 360
		}
		unwrapped[i] = unwrapped[i-1] + delta
	}

	minLon, maxLon := unwrapped[0], unwrapped[0]
	for _, lon := range unwrapped {
		minLon = math.Min(minLon, lon)
		maxLon = math.Max(maxLon, lon)
	}

	return BoundingBox{
		MinLatitude:         minLat,
		MaxLatitude:         maxLat,
		MinLongitude:        normalizeLongitude(minLon),
		MaxLongitude:        normalizeLongitude(maxLon),
		CrossesAntimeridian: minLon < -180 || maxLon > 180,
	}
}

// normalizeLongitude brings a longitude value (potentially outside
// [-180, 180] after unwrapping) back into that range.
func normalizeLongitude(degrees float64) float64 {
	wrapped := math.Mod(degrees+180, 360)
	if wrapped < 0 {
		wrapped += 360
	}
	return wrapped - 180
}

// Contains reports whether the geographic point (lat, lon) falls inside b,
// used by CheckCoverageService to decide whether a registered source
// covers a given route point (FR-018). CrossesAntimeridian is handled the
// same way ComputeBoundingBox produces it: when b crosses the antimeridian,
// the occupied longitude range is the one going from MinLongitude to 180°
// and from -180° to MaxLongitude — not the direct span between the two
// values.
func (b BoundingBox) Contains(lat, lon float64) bool {
	if lat < b.MinLatitude || lat > b.MaxLatitude {
		return false
	}

	if b.CrossesAntimeridian {
		return lon >= b.MinLongitude || lon <= b.MaxLongitude
	}

	return lon >= b.MinLongitude && lon <= b.MaxLongitude
}

// AreaDegrees returns b's approximate area in square degrees (width ×
// height, with the same antimeridian "unwrap" as ComputeBoundingBox). It is
// not a true geodesic area — it exists only to compare how "specific" two
// overlapping sources of the same type are, to break ties deterministically
// (FR-016, research.md item 10).
func (b BoundingBox) AreaDegrees() float64 {
	height := b.MaxLatitude - b.MinLatitude

	width := b.MaxLongitude - b.MinLongitude
	if b.CrossesAntimeridian {
		width += 360
	}

	return width * height
}

// rounded returns b with every coordinate rounded to 1e-7 degrees.
func (b BoundingBox) rounded() BoundingBox {
	round := func(degrees float64) float64 { return math.Round(degrees*1e7) / 1e7 }
	b.MinLatitude, b.MaxLatitude = round(b.MinLatitude), round(b.MaxLatitude)
	b.MinLongitude, b.MaxLongitude = round(b.MinLongitude), round(b.MaxLongitude)
	return b
}

// longitudeSpans returns the longitudes b occupies as one or two ranges of
// [-180, 180], both ends inclusive: two when b crosses the antimeridian.
func (b BoundingBox) longitudeSpans() [][2]float64 {
	if b.CrossesAntimeridian {
		return [][2]float64{{b.MinLongitude, 180}, {-180, b.MaxLongitude}}
	}
	return [][2]float64{{b.MinLongitude, b.MaxLongitude}}
}

// Intersects reports whether b and other share any point, touching at an
// edge included. Both may cross the antimeridian.
func (b BoundingBox) Intersects(other BoundingBox) bool {
	if b.MinLatitude > other.MaxLatitude || other.MinLatitude > b.MaxLatitude {
		return false
	}

	for _, mine := range b.longitudeSpans() {
		for _, theirs := range other.longitudeSpans() {
			if mine[0] <= theirs[1] && theirs[0] <= mine[1] {
				return true
			}
		}
	}
	return false
}

// TileRange returns the tiles of the given level (zoom) of the XYZ scheme of
// Web Mercator that cover b: one rectangle of tiles, or two when b crosses
// the antimeridian (the eastern part first, then the western one).
// Latitudes beyond the ones Web Mercator covers use the tiles of its edge, and
// indexes are limited to the level's grid.
func (b BoundingBox) TileRange(level int) []TileRange {
	size := math.Ldexp(1, level)
	last := int(size) - 1
	clamp := func(index float64) int {
		return int(math.Min(math.Max(math.Floor(index), 0), float64(last)))
	}
	column := func(lon float64) int { return clamp((lon + 180) / 360 * size) }
	row := func(lat float64) int {
		lat = math.Min(math.Max(lat, -MaxMercatorLatitude), MaxMercatorLatitude)
		return clamp((1 - math.Asinh(math.Tan(lat*math.Pi/180))/math.Pi) / 2 * size)
	}

	minY, maxY := row(b.MaxLatitude), row(b.MinLatitude)
	if !b.CrossesAntimeridian {
		return []TileRange{{Level: level, MinX: column(b.MinLongitude), MaxX: column(b.MaxLongitude), MinY: minY, MaxY: maxY}}
	}
	return []TileRange{
		{Level: level, MinX: column(b.MinLongitude), MaxX: last, MinY: minY, MaxY: maxY},
		{Level: level, MinX: 0, MaxX: column(b.MaxLongitude), MinY: minY, MaxY: maxY},
	}
}

// Regions splits b into rectangles inside each of which the winning source of
// each type (SelectSource) is the same everywhere, so that any part of b can
// be read from one base map and one elevation source. The limits of the
// regions are the ones of b and of the sources that intersect it, which is why
// the center of a region stands for all of it. Regions run from south to
// north and, within a row, from west to east, in longitudes unwrapped from
// b's western limit. A region no source covers has a zero-value source of that
// type. The route holds the center of each region, in the same order: what
// Route.Coverage checks to tell whether b is fully covered.
func (b BoundingBox) Regions(baseMaps, elevations []GeoDataSource) (SliceRegions, Route) {
	west, east := b.MinLongitude, b.MaxLongitude
	if b.CrossesAntimeridian {
		east += 360
	}

	latitudes := []float64{b.MinLatitude, b.MaxLatitude}
	longitudes := []float64{west, east}

	for _, candidates := range [][]GeoDataSource{baseMaps, elevations} {
		for _, candidate := range candidates {
			if !candidate.BoundingBox.Intersects(b) {
				continue
			}

			latitudes = append(latitudes, clampBetween(candidate.BoundingBox.MinLatitude, b.MinLatitude, b.MaxLatitude))
			latitudes = append(latitudes, clampBetween(candidate.BoundingBox.MaxLatitude, b.MinLatitude, b.MaxLatitude))
			// the source's longitudes, unwrapped so they do not break at the
			// antimeridian, tried a turn to each side to line up with b's
			low, high := candidate.BoundingBox.MinLongitude, candidate.BoundingBox.MaxLongitude
			if candidate.BoundingBox.CrossesAntimeridian {
				high += 360
			}
			for shift := -360.0; shift <= 360; shift += 360 {
				if high+shift < west || low+shift > east {
					continue
				}
				longitudes = append(longitudes, clampBetween(low+shift, west, east), clampBetween(high+shift, west, east))
			}
		}
	}

	latitudes, longitudes = sortedUnique(latitudes), sortedUnique(longitudes)

	var regions SliceRegions
	var centers []TrackPoint
	for i := 0; i+1 < len(latitudes); i++ {
		for j := 0; j+1 < len(longitudes); j++ {
			south, north := latitudes[i], latitudes[i+1]
			low, high := longitudes[j], longitudes[j+1]

			centerLat := (south + north) / 2
			centerLon := wrapEdge((low + high) / 2)
			baseMap, _ := SelectSource(baseMaps, centerLat, centerLon)
			elevation, _ := SelectSource(elevations, centerLat, centerLon)

			regions = append(regions, SliceRegion{
				Box: BoundingBox{
					MinLatitude: south, MaxLatitude: north,
					MinLongitude: wrapLowEdge(low), MaxLongitude: wrapEdge(high),
					CrossesAntimeridian: low < 180 && high > 180,
				},
				BaseMap:   baseMap,
				Elevation: elevation,
			})
			centers = append(centers, TrackPoint{Latitude: centerLat, Longitude: centerLon})
		}
	}

	return regions, Route{Points: centers}
}

// wrapEdge brings the eastern limit or a center of an unwrapped longitude
// back to [-180, 180]: 180 itself stays.
func wrapEdge(lon float64) float64 {
	if lon > 180 {
		return lon - 360
	}
	return lon
}

// wrapLowEdge is wrapEdge for a western limit, where 180 is -180.
func wrapLowEdge(lon float64) float64 {
	if lon >= 180 {
		return lon - 360
	}
	return lon
}

func clampBetween(v, low, high float64) float64 {
	return math.Min(math.Max(v, low), high)
}

func sortedUnique(values []float64) []float64 {
	sort.Float64s(values)
	var unique []float64
	for _, v := range values {
		if len(unique) == 0 || v != unique[len(unique)-1] {
			unique = append(unique, v)
		}
	}
	return unique
}

// closestLatitudeToEquator is the smallest |latitude| inside b: 0 when b
// contains the equator.
func (b BoundingBox) closestLatitudeToEquator() float64 {
	if b.MinLatitude <= 0 && b.MaxLatitude >= 0 {
		return 0
	}
	return math.Min(math.Abs(b.MinLatitude), math.Abs(b.MaxLatitude))
}

// Extent is the width and the height of b, in kilometers: the height from the
// latitude span and the width from the longitude span at the middle latitude
// (across the antimeridian too).
func (b BoundingBox) Extent() (widthKm, heightKm float64) {
	width := b.MaxLongitude - b.MinLongitude
	if b.CrossesAntimeridian {
		width += 360
	}
	middle := (b.MinLatitude + b.MaxLatitude) / 2

	return width * MetersPerDegree * math.Cos(middle*math.Pi/180) / 1000, (b.MaxLatitude - b.MinLatitude) * MetersPerDegree / 1000
}

// ClippedTo is the part of b that is inside limits, for when the limits are
// more trustworthy than b on a side — a file whose declared bounds run past
// where its data is. If either box crosses the antimeridian, or they do not
// overlap, b is returned as it is: there is no reliable way to tell what to
// keep.
func (b BoundingBox) ClippedTo(limits BoundingBox) BoundingBox {
	if b.CrossesAntimeridian || limits.CrossesAntimeridian || !b.Intersects(limits) {
		return b
	}

	return BoundingBox{
		MinLatitude:  math.Max(b.MinLatitude, limits.MinLatitude),
		MaxLatitude:  math.Min(b.MaxLatitude, limits.MaxLatitude),
		MinLongitude: math.Max(b.MinLongitude, limits.MinLongitude),
		MaxLongitude: math.Min(b.MaxLongitude, limits.MaxLongitude),
	}
}
