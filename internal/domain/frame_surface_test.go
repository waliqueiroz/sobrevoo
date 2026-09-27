package domain

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

var nan32 = float32(math.NaN())

// testGrid is a grid of square cells of `cell` degrees, whose north-west corner
// is (north, west).
func testGrid(name string, rows, cols int, north, west, cell float64, values []float32) ElevationGrid {
	info := ElevationGridInfo{Rows: rows, Cols: cols, NorthLatitude: north, WestLongitude: west, CellLatitude: cell, CellLongitude: cell, UnitToMeters: 1}
	return NewElevationGrid(GeoDataSource{Name: name}, GridWindow{FirstRow: 0, FirstCol: 0, Rows: rows, Cols: cols}, info, values)
}

// equatorPlane is the plane of a camera on the equator, where a degree of
// longitude is as long as a degree of latitude.
func equatorPlane(lon float64) framePlane { return newFramePlane(0, lon) }

// placedGrid is a 4 × 4 grid of cells of 0.001° whose north-west corner is
// (0.002, -0.002): centered on the origin of the plane of a camera at (0, 0).
func placedGrid(values []float32) placedSurface {
	grid := testGrid("dem", 4, 4, 0.002, -0.002, 0.001, values)
	return newSurface(grid, 0).place(equatorPlane(0))
}

func ramp16() []float32 {
	values := make([]float32, 16)
	for i := range values {
		values[i] = float32(10 * i) // row r, column c: 40r + 10c
	}
	return values
}

func Test_surface_heightAt(t *testing.T) {
	g := placedGrid(ramp16())
	cell := 0.001 * math.Pi / 180 * earthRadiusMeters

	// the center of the cell of row r and column c
	center := func(r, c int) (x, y float64) {
		return g.x0 + (float64(c)+0.5)*g.cx, g.y0 - (float64(r)+0.5)*g.cy
	}

	t.Run("should place the grid in meters, with the cells as wide as the degrees are", func(t *testing.T) {
		// given / when / then
		assert.InDelta(t, cell, g.cx, 1e-9)
		assert.InDelta(t, cell, g.cy, 1e-9)
		assert.InDelta(t, -2*cell, g.x0, 1e-6)
		assert.InDelta(t, 2*cell, g.y0, 1e-6)
	})

	t.Run("should be the value of the cell at the center of a cell", func(t *testing.T) {
		for r := 0; r < 4; r++ {
			for c := 0; c < 4; c++ {
				// given
				x, y := center(r, c)

				// when
				height := g.heightAt(x, y)

				// then
				assert.InDelta(t, float64(40*r+10*c), height, 1e-6, "row %d column %d", r, c)
			}
		}
	})

	t.Run("should be the average halfway between two centers and among four", func(t *testing.T) {
		// given
		x1, y1 := center(1, 1)
		x2, _ := center(1, 2)
		_, y3 := center(2, 1)

		// when
		between := g.heightAt((x1+x2)/2, y1)
		below := g.heightAt(x1, (y1+y3)/2)
		amongFour := g.heightAt((x1+x2)/2, (y1+y3)/2)

		// then
		assert.InDelta(t, 55.0, between, 1e-6)
		assert.InDelta(t, 70.0, below, 1e-6)
		assert.InDelta(t, 75.0, amongFour, 1e-6)
	})

	t.Run("should stay flat between the corner of the grid and the first center", func(t *testing.T) {
		// given
		x, y := center(0, 0)

		// when
		atCorner := g.heightAt(g.x0+0.05*g.cx, g.y0-0.05*g.cy)
		atSouthEast := g.heightAt(g.x0+3.95*g.cx+0.04*g.cx, g.y0-3.95*g.cy-0.04*g.cy)

		// then
		assert.InDelta(t, g.heightAt(x, y), atCorner, 1e-6)
		assert.InDelta(t, 150.0, atSouthEast, 1e-6)
	})

	t.Run("should cover exactly the rectangle of its cells", func(t *testing.T) {
		// given / when / then
		assert.InDelta(t, g.x0+4*g.cx, g.xMax, 1e-9)
		assert.InDelta(t, g.y0-4*g.cy, g.yMin, 1e-9)
	})

	t.Run("should know the highest height it has", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, 150.0, g.zmax)
	})
}

func Test_surface_cellAt(t *testing.T) {
	g := placedGrid(ramp16())

	t.Run("should be the cell that contains the point, counting rows from the north and columns from the west", func(t *testing.T) {
		// given / when
		row, col := g.cellAt(g.x0+2.5*g.cx, g.y0-1.2*g.cy)
		lastRow, lastCol := g.cellAt(g.xMax, g.yMin)

		// then
		assert.Equal(t, 1, row)
		assert.Equal(t, 2, col)
		assert.Equal(t, 3, lastRow)
		assert.Equal(t, 3, lastCol)
	})
}

func Test_surface_holes(t *testing.T) {
	t.Run("should take the value of the nearest sample with a value, in blocks of city distance", func(t *testing.T) {
		// given: a 3 x 3 block of holes in the middle of a 5 x 5 ramp
		values := make([]float32, 25)
		for i := range values {
			values[i] = float32(i)
		}
		for r := 1; r <= 3; r++ {
			for c := 1; c <= 3; c++ {
				values[5*r+c] = nan32
			}
		}
		s := newSurface(testGrid("dem", 5, 5, 0.0025, -0.0025, 0.001, values), 0)

		// when / then
		assert.True(t, s.hasHoles)
		// (1, 1) is one block from (0, 1) and (1, 0): the first found is the north one
		assert.Equal(t, float32(1), s.heights[5*1+1])
		// (2, 2) is two blocks from every border sample: the north-west pass finds (0, 2) first
		assert.Equal(t, float32(2), s.heights[5*2+2])
		// (3, 1) is one block from (4, 1) and (3, 0): the one from the west comes first
		assert.Equal(t, float32(15), s.heights[5*3+1])
		// a cell that had a value keeps it
		assert.Equal(t, float32(0), s.heights[0])
		assert.Equal(t, float32(24), s.heights[24])
	})

	t.Run("should resolve a tie by the order of the passes: the north-west first", func(t *testing.T) {
		// given: 10, hole, hole, hole, 50 in a row
		s := newSurface(testGrid("dem", 1, 5, 0.001, 0, 0.001, []float32{10, nan32, nan32, nan32, 50}), 0)

		// when / then
		assert.Equal(t, []float32{10, 10, 10, 50, 50}, s.heights)
	})

	t.Run("should keep the original samples to know which cells have no value", func(t *testing.T) {
		// given
		s := newSurface(testGrid("dem", 1, 3, 0.001, 0, 0.001, []float32{5, nan32, 7}), 0)

		// when / then
		assert.False(t, s.isHole(0, 0))
		assert.True(t, s.isHole(0, 1))
		assert.False(t, s.isHole(0, 2))
	})

	t.Run("should not copy the samples of a grid without holes", func(t *testing.T) {
		// given
		grid := testGrid("dem", 2, 2, 0.002, 0, 0.001, []float32{1, 2, 3, 4})

		// when
		s := newSurface(grid, 0)

		// then
		assert.False(t, s.hasHoles)
		assert.Same(t, &s.raw[0], &s.heights[0])
		assert.False(t, s.isHole(1, 1))
	})

	t.Run("should give a grid with no value at all the fallback height, the lowest with a value of the slice", func(t *testing.T) {
		// given
		s := newSurface(testGrid("dem", 2, 2, 0.002, 0, 0.001, []float32{nan32, nan32, nan32, nan32}), 42)

		// when / then
		assert.Equal(t, []float32{42, 42, 42, 42}, s.heights)
		assert.True(t, s.isHole(0, 0))
		assert.Equal(t, 42.0, s.zmax)
	})

	t.Run("should have the highest filled height as its zmax", func(t *testing.T) {
		// given
		s := newSurface(testGrid("dem", 1, 3, 0.001, 0, 0.001, []float32{5, nan32, 70}), 0)

		// when / then
		assert.Equal(t, 70.0, s.zmax)
	})
}

func Test_surface_placement(t *testing.T) {
	t.Run("should continue east of 180 degrees with no jump, for a camera on the other side", func(t *testing.T) {
		// given: a grid whose west edge is at 179.998° and whose 4 columns pass 180°
		grid := testGrid("dem", 2, 4, 0.001, 179.998, 0.001, []float32{1, 2, 3, 4, 5, 6, 7, 8})
		plane := equatorPlane(-179.999)

		// when
		g := newSurface(grid, 0).place(plane)

		// then: the west edge is 0.003° west of the camera (179.998 is 0.003 west of -179.999)
		cell := 0.001 * math.Pi / 180 * earthRadiusMeters
		assert.InDelta(t, -3*cell, g.x0, 1e-6)
		assert.InDelta(t, 1*cell, g.xMax, 1e-6)
	})
}

func Test_terrain_groundAt(t *testing.T) {
	west := testGrid("west", 2, 2, 0.001, -0.002, 0.001, []float32{10, 10, 10, 10})
	east := testGrid("east", 2, 2, 0.001, 0, 0.001, []float32{50, 50, 50, 50})
	plane := equatorPlane(0)
	ground := terrain{newSurface(west, 0).place(plane), newSurface(east, 0).place(plane)}
	cell := 0.001 * math.Pi / 180 * earthRadiusMeters

	t.Run("should be the height of the grid that holds the position", func(t *testing.T) {
		// given / when / then
		assert.InDelta(t, 10.0, ground.groundAt(-cell, -0.5*cell), 1e-6)
		assert.InDelta(t, 50.0, ground.groundAt(cell, -0.5*cell), 1e-6)
	})

	t.Run("should use the nearest grid, with the position pulled into it, outside all of them", func(t *testing.T) {
		// given / when / then
		assert.InDelta(t, 50.0, ground.groundAt(10*cell, -0.5*cell), 1e-6)
		assert.InDelta(t, 10.0, ground.groundAt(-10*cell, -0.5*cell), 1e-6)
		assert.InDelta(t, 10.0, ground.groundAt(-1.5*cell, 5*cell), 1e-6)
	})
}

func Test_cameraHeight(t *testing.T) {
	tuning := RenderTuning{MinTiltForTargetDegrees: 1, MinCameraClearanceMeters: 2}
	plane := equatorPlane(0)

	// a strip of 1 row and 8 columns of cells of 0.001° (111 m), centered on the
	// camera, which stands at its middle, between the fourth and the fifth cell
	build := func(values []float32) terrain {
		grid := testGrid("dem", 1, len(values), 0.0005, -0.004, 0.001, values)
		return terrain{newSurface(grid, 0).place(plane)}
	}

	t.Run("should be the terrain under the target plus the altitude of the frame", func(t *testing.T) {
		// given: flat ground at 300 m
		ground := build([]float32{300, 300, 300, 300, 300, 300, 300, 300})
		frame := CameraFrame{Heading: 90, Tilt: 45, CameraAltitude: 100}

		// when
		height := cameraHeight(ground, plane, frame, tuning)

		// then
		assert.InDelta(t, 400.0, height, 1e-6)
	})

	t.Run("should count the terrain under the observed point, not the one under the camera", func(t *testing.T) {
		// given: ground at 100 m to the west and at 200 m to the east of the camera; with a tilt of 45° and an
		// altitude of 100 m the observed point is 100 m ahead of the camera
		ground := build([]float32{100, 100, 100, 100, 200, 200, 200, 200})
		lookingEast := CameraFrame{Heading: 90, Tilt: 45, CameraAltitude: 100}
		lookingWest := CameraFrame{Heading: 270, Tilt: 45, CameraAltitude: 100}

		// when
		east := cameraHeight(ground, plane, lookingEast, tuning)
		west := cameraHeight(ground, plane, lookingWest, tuning)

		// then
		assert.InDelta(t, 300.0, east, 1e-6)
		assert.InDelta(t, 200.0, west, 1e-6)
	})

	t.Run("should keep the camera above the terrain it stands on, when the target is lower", func(t *testing.T) {
		// given: the camera stands on a slope that climbs from 100 m to 400 m across it, and looks west, down the
		// slope, at a point 20 m ahead
		ground := build([]float32{100, 100, 100, 100, 400, 400, 400, 400})
		frame := CameraFrame{Heading: 270, Tilt: 45, CameraAltitude: 20}

		// when
		height := cameraHeight(ground, plane, frame, tuning)

		// then: the terrain under the camera is 250 m; the one under the target plus 20 m is less
		assert.InDelta(t, 252.0, height, 1e-6)
	})
}
