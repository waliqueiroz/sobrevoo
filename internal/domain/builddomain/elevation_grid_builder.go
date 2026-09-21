package builddomain

import (
	"math"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// ElevationGridBuilder builds an ElevationGrid. The default is a 3×3 grid of
// cells of 0.001° whose north-west corner is at (-23, -47), with elevations
// 100 to 108 m, from a registered GeoTIFF elevation source.
type ElevationGridBuilder struct {
	source domain.GeoDataSource
	window domain.GridWindow
	info   domain.ElevationGridInfo
	values []float32
}

func NewElevationGridBuilder() *ElevationGridBuilder {
	return &ElevationGridBuilder{
		source: NewGeoDataSourceBuilder().
			WithName("relevo").
			WithPath("/data/relevo.tif").
			WithType(domain.DataTypeElevation).
			WithFormat(domain.DataFormatGeoTIFF).
			Build(),
		window: domain.GridWindow{Rows: 3, Cols: 3},
		info: domain.ElevationGridInfo{
			Rows: 3, Cols: 3,
			NorthLatitude: -23, WestLongitude: -47,
			CellLatitude: 0.001, CellLongitude: 0.001,
			UnitToMeters: 1,
		},
		values: []float32{100, 101, 102, 103, 104, 105, 106, 107, 108},
	}
}

func (b *ElevationGridBuilder) WithSource(source domain.GeoDataSource) *ElevationGridBuilder {
	b.source = source
	return b
}

func (b *ElevationGridBuilder) WithWindow(window domain.GridWindow) *ElevationGridBuilder {
	b.window = window
	return b
}

// WithOrigin sets the north-west corner of the source file's grid.
func (b *ElevationGridBuilder) WithOrigin(northLatitude, westLongitude float64) *ElevationGridBuilder {
	b.info.NorthLatitude, b.info.WestLongitude = northLatitude, westLongitude
	return b
}

func (b *ElevationGridBuilder) WithCellSize(latitude, longitude float64) *ElevationGridBuilder {
	b.info.CellLatitude, b.info.CellLongitude = latitude, longitude
	return b
}

// WithValues sets the samples, in meters; they must fill the window exactly.
func (b *ElevationGridBuilder) WithValues(values ...float32) *ElevationGridBuilder {
	b.values = values
	return b
}

// WithNoValueAt marks the sample at (row, col) of the window as having no
// value.
func (b *ElevationGridBuilder) WithNoValueAt(row, col int) *ElevationGridBuilder {
	b.values[row*b.window.Cols+col] = float32(math.NaN())
	return b
}

func (b *ElevationGridBuilder) Build() domain.ElevationGrid {
	return domain.NewElevationGrid(b.source, b.window, b.info, b.values)
}
