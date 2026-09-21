package domain

//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/elevation_reader.go -package mockdomain . ElevationReader

import "math"

// ElevationReader reads the content of a registered elevation file. Concrete
// implementations (one per recognized format) live in
// internal/infra/outbound/elevationreader. Like GeoDataInspector, it takes a
// path: the adapter opens the file itself.
type ElevationReader interface {
	// Describe reads the geometry of the file's grid — from its metadata
	// only, never its samples. It fails with ErrGeoDataContentUnreadable
	// when the metadata cannot be read and with ErrElevationUnitUnsupported
	// when the vertical unit cannot be converted to meters.
	Describe(path string) (ElevationGridInfo, error)

	// ReadWindow reads the samples of a rectangular window of the grid, in
	// meters. A sample the file marks as having no data comes back marked as
	// such (never as zero).
	ReadWindow(path string, window GridWindow) (ElevationWindow, error)
}

// ElevationGridInfo is the geometry of an elevation file's grid.
type ElevationGridInfo struct {
	Rows, Cols int

	// NorthLatitude and WestLongitude are the north-west corner of cell
	// (0, 0). CellLatitude and CellLongitude are the size of a cell, in
	// degrees (positive).
	NorthLatitude, WestLongitude float64
	CellLatitude, CellLongitude  float64

	// UnitToMeters converts the file's vertical unit to meters.
	UnitToMeters float64
}

// GridWindow is a rectangle of cells of a grid: rows run north to south and
// columns west to east. A window never crosses the antimeridian.
type GridWindow struct {
	FirstRow, FirstCol int
	Rows, Cols         int
}

// ElevationWindow is what ElevationReader.ReadWindow returns: Rows × Cols
// samples in meters, row by row, north to south. A sample without data is a
// NaN, which never leaves this package's types: use ElevationGrid.At.
type ElevationWindow struct {
	Values []float32
}

// ElevationGrid is the elevation samples of one region of a slice, in
// meters, taken as they are in the source file's grid.
type ElevationGrid struct {
	Source GeoDataSource

	// Window is where these samples sit in the source file's grid.
	Window GridWindow

	// NorthLatitude and WestLongitude are the north-west corner of this
	// grid's cell (0, 0); CellLatitude and CellLongitude are the cell size.
	NorthLatitude, WestLongitude float64
	CellLatitude, CellLongitude  float64

	values []float32
}

// NewElevationGrid builds the grid of window out of the source's geometry
// info and the samples read for it (ElevationWindow.Values).
func NewElevationGrid(source GeoDataSource, window GridWindow, info ElevationGridInfo, values []float32) ElevationGrid {
	return ElevationGrid{
		Source:        source,
		Window:        window,
		NorthLatitude: info.NorthLatitude - float64(window.FirstRow)*info.CellLatitude,
		WestLongitude: normalizeLongitude(info.WestLongitude + float64(window.FirstCol)*info.CellLongitude),
		CellLatitude:  info.CellLatitude,
		CellLongitude: info.CellLongitude,
		values:        values,
	}
}

// ElevationReading is the answer to "what is the elevation at this
// coordinate?" (FR-017): the value of the grid cell that contains it, or the
// fact that the file has no value for that cell.
type ElevationReading struct {
	Latitude, Longitude float64

	// Meters is meaningful only when HasValue is true.
	Meters   float64
	HasValue bool

	// Source is the elevation source used; Row and Col are the cell in its
	// grid.
	Source   GeoDataSource
	Row, Col int
}

// CellAt is the row and column of the grid cell that contains the point
// (lat, lon): rows count from the north, columns from the west and eastwards
// (across the antimeridian too). A point exactly on the limit between two
// cells belongs to the southern and eastern one, and the outer south and east
// edge to the last row and column. The point must be inside the grid.
func (i ElevationGridInfo) CellAt(lat, lon float64) (row, col int) {
	row = clampIndex(math.Floor((i.NorthLatitude-lat)/i.CellLatitude), i.Rows)
	col = clampIndex(math.Floor(eastwardsFrom(i.WestLongitude, lon)/i.CellLongitude), i.Cols)
	return row, col
}

// Window returns the windows of cells whose center lies inside box, where box
// is a region of area: the south and west edges of a region are inclusive and
// the north and east ones exclusive, unless they are the edge of the whole
// area, so every cell belongs to exactly one region. The columns are
// contiguous across the antimeridian; a box crossing the antimeridian over a
// grid that spans the whole world gives two windows, the western one first.
// It gives none when no cell center is inside box.
func (i ElevationGridInfo) Window(box, area BoundingBox) []GridWindow {
	northInclusive := box.MaxLatitude == area.MaxLatitude
	eastInclusive := box.MaxLongitude == area.MaxLongitude && box.CrossesAntimeridian == area.CrossesAntimeridian

	// rows: the centers are at north - (row + 0.5) × cell
	first := (i.NorthLatitude-box.MaxLatitude)/i.CellLatitude - 0.5
	firstRow := math.Floor(first) + 1
	if northInclusive {
		firstRow = math.Ceil(first)
	}
	lastRow := math.Floor((i.NorthLatitude-box.MinLatitude)/i.CellLatitude - 0.5)
	rowFrom, rowTo, rowsOK := clampRange(firstRow, lastRow, i.Rows)
	if !rowsOK {
		return nil
	}

	// columns: the centers are at west + (col + 0.5) × cell, in a frame that
	// grows eastwards past 180°
	width := box.MaxLongitude - box.MinLongitude
	if box.CrossesAntimeridian {
		width += 360
	}
	low := box.MinLongitude
	for low < i.WestLongitude {
		low += 360
	}
	for low >= i.WestLongitude+360 {
		low -= 360
	}

	var windows []GridWindow
	for _, shift := range []float64{-360, 0} {
		from := (low+shift-i.WestLongitude)/i.CellLongitude - 0.5
		to := (low+shift+width-i.WestLongitude)/i.CellLongitude - 0.5
		firstCol := math.Ceil(from)
		lastCol := math.Ceil(to) - 1
		if eastInclusive {
			lastCol = math.Floor(to)
		}
		if colFrom, colTo, ok := clampRange(firstCol, lastCol, i.Cols); ok {
			windows = append(windows, GridWindow{FirstRow: rowFrom, FirstCol: colFrom, Rows: rowTo - rowFrom + 1, Cols: colTo - colFrom + 1})
		}
	}
	return windows
}

// Rows is the number of rows of the grid.
func (g ElevationGrid) Rows() int { return g.Window.Rows }

// Cols is the number of columns of the grid.
func (g ElevationGrid) Cols() int { return g.Window.Cols }

// At is the elevation of the sample at (row, col), in meters. hasValue is
// false when the file has no value for it; the meters are then zero and mean
// nothing.
func (g ElevationGrid) At(row, col int) (meters float64, hasValue bool) {
	value := g.values[row*g.Window.Cols+col]
	if value != value { // NaN: no value
		return 0, false
	}
	return float64(value), true
}

// NoValueCount is how many samples have no value.
func (g ElevationGrid) NoValueCount() int {
	count := 0
	for _, value := range g.values {
		if value != value {
			count++
		}
	}
	return count
}

// Range is the lowest and highest elevation among the samples that have a
// value; ok is false when none has.
func (g ElevationGrid) Range() (minimum, maximum float64, ok bool) {
	for _, value := range g.values {
		if value != value {
			continue
		}
		v := float64(value)
		if !ok {
			minimum, maximum, ok = v, v, true
			continue
		}
		minimum, maximum = math.Min(minimum, v), math.Max(maximum, v)
	}
	return minimum, maximum, ok
}

// eastwardsFrom is how far east of west lon is, in [0, 360).
func eastwardsFrom(west, lon float64) float64 {
	distance := math.Mod(lon-west, 360)
	if distance < 0 {
		distance += 360
	}
	return distance
}

func clampIndex(index float64, size int) int {
	return int(math.Min(math.Max(index, 0), float64(size-1)))
}

// clampRange limits [first, last] to [0, size-1]; ok is false when nothing is
// left.
func clampRange(first, last float64, size int) (from, to int, ok bool) {
	first = math.Max(first, 0)
	last = math.Min(last, float64(size-1))
	if first > last {
		return 0, 0, false
	}
	return int(first), int(last), true
}

// NewElevationReading is the reading of the sample of the cell (row, col) of
// source's grid at coordinate; a NaN sample is a cell the file has no value
// for.
func NewElevationReading(coordinate Coordinate, source GeoDataSource, row, col int, sample float32) ElevationReading {
	reading := ElevationReading{
		Latitude:  coordinate.Latitude,
		Longitude: coordinate.Longitude,
		Source:    source,
		Row:       row,
		Col:       col,
	}
	if sample == sample {
		reading.Meters, reading.HasValue = float64(sample), true
	}
	return reading
}
