package domain_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

// an 8 × 8 grid of 0.125° cells (exactly representable, so cell centers and
// limits are exact) whose north-west corner is at (41, 10)
func gridInfo() domain.ElevationGridInfo {
	return domain.ElevationGridInfo{
		Rows: 8, Cols: 8,
		NorthLatitude: 41, WestLongitude: 10,
		CellLatitude: 0.125, CellLongitude: 0.125,
		UnitToMeters: 1,
	}
}

// windowOf is the window of a region that is the whole area.
func windowOf(info domain.ElevationGridInfo, b domain.BoundingBox) []domain.GridWindow {
	return info.Window(b, b)
}

func Test_ElevationGridInfo_CellAt(t *testing.T) {
	info := gridInfo()

	t.Run("should find the cell that contains a point", func(t *testing.T) {
		// when
		row, col := info.CellAt(40.55, 10.35)

		// then
		assert.Equal(t, 3, row)
		assert.Equal(t, 2, col)
	})

	t.Run("should give a point on the limit between two cells to the southern and eastern one", func(t *testing.T) {
		// when
		row, col := info.CellAt(40.5, 10.5)

		// then
		assert.Equal(t, 4, row)
		assert.Equal(t, 4, col)
	})

	t.Run("should keep the north-west corner in cell (0, 0)", func(t *testing.T) {
		// when
		row, col := info.CellAt(41, 10)

		// then
		assert.Equal(t, 0, row)
		assert.Equal(t, 0, col)
	})

	t.Run("should keep the outer south and east edge in the last row and column", func(t *testing.T) {
		// when
		row, col := info.CellAt(40, 11)

		// then
		assert.Equal(t, 7, row)
		assert.Equal(t, 7, col)
	})

	t.Run("should measure the column eastwards across the antimeridian", func(t *testing.T) {
		// given
		crossing := domain.ElevationGridInfo{
			Rows: 10, Cols: 100,
			NorthLatitude: 1, WestLongitude: 175,
			CellLatitude: 0.1, CellLongitude: 0.1,
			UnitToMeters: 1,
		}

		// when
		_, colBefore := crossing.CellAt(0.5, 179.95)
		_, colAfter := crossing.CellAt(0.5, -179.95)
		_, colFar := crossing.CellAt(0.5, -175.05)

		// then
		assert.Equal(t, 49, colBefore)
		assert.Equal(t, 50, colAfter)
		assert.Equal(t, 99, colFar)
	})
}

func Test_ElevationGridInfo_Window(t *testing.T) {
	info := gridInfo()

	t.Run("should select the cells whose center is inside the box", func(t *testing.T) {
		// when
		windows := windowOf(info, box(40.2, 40.5, 10.3, 10.6))

		// then
		require.Len(t, windows, 1)
		assert.Equal(t, domain.GridWindow{FirstRow: 4, FirstCol: 2, Rows: 2, Cols: 3}, windows[0])
	})

	t.Run("should include the south and west edge centers and exclude the north and east edge centers of a region inside the area", func(t *testing.T) {
		// given: the box edges pass exactly through cell centers (row 4 is at 40.4375 and row 3 at 40.5625; columns 3 and 4 at 10.4375 and 10.5625)
		region := box(40.4375, 40.5625, 10.4375, 10.5625)
		area := box(40, 41, 10, 11)

		// when
		windows := info.Window(region, area)

		// then
		require.Len(t, windows, 1)
		assert.Equal(t, domain.GridWindow{FirstRow: 4, FirstCol: 3, Rows: 1, Cols: 1}, windows[0])
	})

	t.Run("should include the north and east edge centers when the edge is the area's outer edge", func(t *testing.T) {
		// given
		region := box(40.4375, 40.5625, 10.4375, 10.5625)

		// when
		windows := info.Window(region, region)

		// then
		require.Len(t, windows, 1)
		assert.Equal(t, domain.GridWindow{FirstRow: 3, FirstCol: 3, Rows: 2, Cols: 2}, windows[0])
	})

	t.Run("should select the whole grid for a box that contains it, including the outer edge", func(t *testing.T) {
		// when
		windows := windowOf(info, box(40, 41, 10, 11))

		// then
		require.Len(t, windows, 1)
		assert.Equal(t, domain.GridWindow{FirstRow: 0, FirstCol: 0, Rows: 8, Cols: 8}, windows[0])
	})

	t.Run("should give no window for a box that holds no cell center", func(t *testing.T) {
		// when
		windows := windowOf(info, box(40.51, 40.52, 10.51, 10.52))

		// then
		assert.Empty(t, windows)
	})

	t.Run("should give no window for a box outside the grid", func(t *testing.T) {
		// when
		windows := windowOf(info, box(0, 1, 0, 1))

		// then
		assert.Empty(t, windows)
	})

	t.Run("should give a single window for a grid and a box that cross the antimeridian, when the grid's columns are contiguous", func(t *testing.T) {
		// given
		crossing := domain.ElevationGridInfo{
			Rows: 10, Cols: 100,
			NorthLatitude: 1, WestLongitude: 175,
			CellLatitude: 0.1, CellLongitude: 0.1,
			UnitToMeters: 1,
		}

		// when
		windows := windowOf(crossing, box(0, 1, 179, -179))

		// then
		require.Len(t, windows, 1)
		assert.Equal(t, domain.GridWindow{FirstRow: 0, FirstCol: 40, Rows: 10, Cols: 20}, windows[0])
	})

	t.Run("should give two windows for a box that crosses the antimeridian over a grid that spans the whole world", func(t *testing.T) {
		// given
		global := domain.ElevationGridInfo{
			Rows: 10, Cols: 3600,
			NorthLatitude: 1, WestLongitude: -180,
			CellLatitude: 0.1, CellLongitude: 0.1,
			UnitToMeters: 1,
		}

		// when
		windows := windowOf(global, box(0, 1, 179, -179))

		// then
		require.Len(t, windows, 2)
		assert.Equal(t, domain.GridWindow{FirstRow: 0, FirstCol: 0, Rows: 10, Cols: 10}, windows[0])
		assert.Equal(t, domain.GridWindow{FirstRow: 0, FirstCol: 3590, Rows: 10, Cols: 10}, windows[1])
	})

	t.Run("should give each cell to exactly one of the regions that partition an area", func(t *testing.T) {
		// given
		area := box(40.13, 40.87, 10.21, 10.94)
		west := box(40.13, 40.87, 10.21, 10.5)
		east := box(40.13, 40.87, 10.5, 10.94)

		// when
		whole := windowOf(info, area)[0]
		first := windowOf(info, west)[0]
		second := windowOf(info, east)[0]

		// then
		assert.Equal(t, whole.Rows*whole.Cols, first.Rows*first.Cols+second.Rows*second.Cols)
	})
}

func Test_ElevationGrid(t *testing.T) {
	t.Run("should give the samples in meters", func(t *testing.T) {
		// given
		grid := builddomain.NewElevationGridBuilder().WithValues(1, 2, 3, 4, 5, 6, 7, 8, 9).Build()

		// when
		meters, hasValue := grid.At(1, 2)

		// then
		assert.True(t, hasValue)
		assert.Equal(t, 6.0, meters)
	})

	t.Run("should never turn a sample without value into zero", func(t *testing.T) {
		// given
		grid := builddomain.NewElevationGridBuilder().WithNoValueAt(0, 1).Build()

		// when
		meters, hasValue := grid.At(0, 1)
		zero, zeroHasValue := builddomain.NewElevationGridBuilder().WithValues(0, 0, 0, 0, 0, 0, 0, 0, 0).Build().At(0, 1)

		// then
		assert.False(t, hasValue)
		assert.True(t, math.IsNaN(meters) || meters == 0)
		assert.True(t, zeroHasValue, "a real zero is a value")
		assert.Equal(t, 0.0, zero)
	})

	t.Run("should count the samples without value", func(t *testing.T) {
		// given
		grid := builddomain.NewElevationGridBuilder().WithNoValueAt(0, 0).WithNoValueAt(2, 2).Build()

		// when
		count := grid.NoValueCount()

		// then
		assert.Equal(t, 2, count)
	})

	t.Run("should give the range over the samples that have a value only", func(t *testing.T) {
		// given
		grid := builddomain.NewElevationGridBuilder().
			WithValues(50, -3, 700, 12, 0, 99, 1, 2, 3).
			WithNoValueAt(0, 2).
			Build()

		// when
		minimum, maximum, ok := grid.Range()

		// then
		assert.True(t, ok)
		assert.Equal(t, -3.0, minimum)
		assert.Equal(t, 99.0, maximum)
	})

	t.Run("should report no range when no sample has a value", func(t *testing.T) {
		// given
		builder := builddomain.NewElevationGridBuilder()
		for row := 0; row < 3; row++ {
			for col := 0; col < 3; col++ {
				builder.WithNoValueAt(row, col)
			}
		}

		// when
		_, _, ok := builder.Build().Range()

		// then
		assert.False(t, ok)
	})

	t.Run("should place the grid of a window at the window's corner of the source's grid", func(t *testing.T) {
		// given
		info := gridInfo()
		window := domain.GridWindow{FirstRow: 2, FirstCol: 3, Rows: 1, Cols: 2}
		source := builddomain.NewGeoDataSourceBuilder().Build()

		// when
		grid := domain.NewElevationGrid(source, window, info, []float32{10, 20})

		// then
		assert.InDelta(t, 40.75, grid.NorthLatitude, 1e-9)
		assert.InDelta(t, 10.375, grid.WestLongitude, 1e-9)
		assert.Equal(t, 1, grid.Rows())
		assert.Equal(t, 2, grid.Cols())
		assert.Equal(t, 0.125, grid.CellLatitude)
	})
}

func Test_NewElevationReading(t *testing.T) {
	coordinate, err := domain.NewCoordinate(-23.5, -46.6)
	require.NoError(t, err)
	source := builddomain.NewGeoDataSourceBuilder().WithName("dem").Build()

	t.Run("should give the sample in meters when the cell has a value", func(t *testing.T) {
		// when
		reading := domain.NewElevationReading(coordinate, source, 4, 7, 760.5)

		// then
		assert.True(t, reading.HasValue)
		assert.Equal(t, 760.5, reading.Meters)
		assert.Equal(t, 4, reading.Row)
		assert.Equal(t, 7, reading.Col)
		assert.Equal(t, "dem", reading.Source.Name)
		assert.Equal(t, -23.5, reading.Latitude)
		assert.Equal(t, -46.6, reading.Longitude)
	})

	t.Run("should say there is no value for a sample without one, never zero", func(t *testing.T) {
		// when
		reading := domain.NewElevationReading(coordinate, source, 4, 7, float32(math.NaN()))

		// then
		assert.False(t, reading.HasValue)
		assert.Equal(t, 0.0, reading.Meters)
	})

	t.Run("should keep a real zero as a value", func(t *testing.T) {
		// when
		reading := domain.NewElevationReading(coordinate, source, 4, 7, 0)

		// then
		assert.True(t, reading.HasValue)
	})
}
