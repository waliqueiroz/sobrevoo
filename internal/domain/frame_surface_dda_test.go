package domain

import (
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const traceCell = 0.001 * math.Pi / 180 * earthRadiusMeters // a cell of 0.001° on the equator, in meters

// towards is the unit direction from one point to another.
func towards(from, to [3]float64) (dx, dy, dz float64) {
	x, y, z := to[0]-from[0], to[1]-from[1], to[2]-from[2]
	length := math.Sqrt(x*x + y*y + z*z)
	return x / length, y / length, z / length
}

// flatTerrain is a square grid of n × n cells of 0.001° centered on the origin,
// all at the same height.
func flatTerrain(n int, height float32) terrain {
	values := make([]float32, n*n)
	for i := range values {
		values[i] = height
	}
	half := float64(n) / 2 * 0.001
	return terrain{newSurface(testGrid("dem", n, n, half, -half, 0.001, values), 0).place(equatorPlane(0))}
}

func Test_terrain_trace(t *testing.T) {
	t.Run("should hit flat ground where a ray descending at 45 degrees reaches it", func(t *testing.T) {
		// given: ground at 100 m; a camera at 300 m looking north and down at 45°
		ground := flatTerrain(8, 100)
		dx, dy, dz := towards([3]float64{0, 0, 300}, [3]float64{0, 200, 100})

		// when
		hit, ok := ground.trace(0, 0, 300, dx, dy, dz)

		// then
		require.True(t, ok)
		assert.InDelta(t, 200*math.Sqrt2, hit.t, 1e-3)
		assert.InDelta(t, 0.0, hit.x, 1e-3)
		assert.InDelta(t, 200.0, hit.y, 1e-3)
		assert.Equal(t, 0, hit.grid)
	})

	t.Run("should hit a ramp where the analytic intersection is", func(t *testing.T) {
		// given: ground that rises 10% to the north: height 100 + 0.1 y at the center of each row
		rows := 40
		values := make([]float32, rows*5)
		for r := 0; r < rows; r++ {
			y := (float64(rows)/2 - float64(r) - 0.5) * traceCell // the center of the row, north of the camera when positive
			for c := 0; c < 5; c++ {
				values[r*5+c] = float32(100 + 0.1*y)
			}
		}
		half := float64(rows) / 2 * 0.001
		g := terrain{newSurface(testGrid("dem", rows, 5, half, -2.5*0.001, 0.001, values), 0).place(equatorPlane(0))}
		dx, dy, dz := towards([3]float64{0, 0, 300}, [3]float64{0, 200, 100})

		// when
		hit, ok := g.trace(0, 0, 300, dx, dy, dz)

		// then: 300 − y = 100 + 0.1·y, along a ray whose horizontal part is cos 45°
		require.True(t, ok)
		wantY := 200 / 1.1
		assert.InDelta(t, wantY, hit.y, 1e-2)
		assert.InDelta(t, wantY/math.Cos(math.Pi/4), hit.t, 1e-2)
	})

	// a pyramid of 9 × 9 cells: 500 m at the middle, 100 m lower at each ring
	pyramid := func() terrain {
		values := make([]float32, 81)
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				ring := max(abs(r-4), abs(c-4))
				values[r*9+c] = float32(500 - 100*ring)
			}
		}
		half := 4.5 * 0.001
		return terrain{newSurface(testGrid("dem", 9, 9, half, -half, 0.001, values), 0).place(equatorPlane(0))}
	}

	t.Run("should not see the valley behind a peak, but see it from above the peak", func(t *testing.T) {
		// given
		ground := pyramid()
		low := [3]float64{0, -800, 300}
		lowDx, lowDy, lowDz := towards(low, [3]float64{0, 400, 100})
		high := [3]float64{0, -800, 2000}
		highDx, highDy, highDz := towards(high, [3]float64{0, 400, 0})

		// when
		behind, okBehind := ground.trace(low[0], low[1], low[2], lowDx, lowDy, lowDz)
		above, okAbove := ground.trace(high[0], high[1], high[2], highDx, highDy, highDz)

		// then
		require.True(t, okBehind)
		assert.Less(t, behind.y, 0.0, "the ray meets the south slope, in front of the peak")
		require.True(t, okAbove)
		assert.Greater(t, above.y, 0.0, "the ray that passes over the peak reaches the far side")
	})

	t.Run("should not hit anything when the ray goes up over the highest ground", func(t *testing.T) {
		// given
		ground := pyramid()
		dx, dy, dz := towards([3]float64{0, 0, 600}, [3]float64{0, 100, 700})

		// when
		_, ok := ground.trace(0, 0, 600, dx, dy, dz)

		// then
		assert.False(t, ok)
	})

	t.Run("should not hit anything when the ray runs level over the highest ground", func(t *testing.T) {
		// given
		ground := pyramid()

		// when
		_, ok := ground.trace(0, -800, 600, 0, 1, 0)

		// then
		assert.False(t, ok)
	})

	t.Run("should hit at the entrance when the camera is inside the ground", func(t *testing.T) {
		// given: a camera at 50 m under ground at 100 m
		ground := flatTerrain(8, 100)

		// when
		down, okDown := ground.trace(0, 0, 50, 0, 0.6, -0.8)
		up, okUp := ground.trace(0, 0, 50, 0, 0.6, 0.8)

		// then
		require.True(t, okDown)
		require.True(t, okUp)
		assert.InDelta(t, 0.0, down.t, 1e-3)
		assert.InDelta(t, 0.0, up.t, 1e-3)
	})

	t.Run("should hit at the entrance of a neighbouring grid whose ground is above the ray", func(t *testing.T) {
		// given: ground at 100 m to the west of x = 0 and at 150 m to the east
		west := testGrid("west", 4, 4, 0.002, -0.004, 0.001, repeat(16, 100))
		east := testGrid("east", 4, 4, 0.002, 0, 0.001, repeat(16, 150))
		plane := equatorPlane(0)
		ground := terrain{newSurface(west, 0).place(plane), newSurface(east, 0).place(plane)}
		dx, dy, dz := towards([3]float64{-200, 0, 130}, [3]float64{0, 0, 120})

		// when
		hit, ok := ground.trace(-200, 0, 130, dx, dy, dz)

		// then: it passes over the west (130 → 120 m over 100 m), and meets the east at its edge
		require.True(t, ok)
		assert.Equal(t, 1, hit.grid)
		assert.InDelta(t, 200/dx, hit.t, 1e-3)
		assert.InDelta(t, 0.0, hit.x, 1e-3)
	})

	t.Run("should not leave a crack between two neighbouring grids at the same height", func(t *testing.T) {
		// given
		west := testGrid("west", 4, 4, 0.002, -0.004, 0.001, repeat(16, 100))
		east := testGrid("east", 4, 4, 0.002, 0, 0.001, repeat(16, 100))
		plane := equatorPlane(0)
		ground := terrain{newSurface(west, 0).place(plane), newSurface(east, 0).place(plane)}

		// when: a shallow ray across the seam, at several heights of the ray over the ground
		hits := 0
		for _, oz := range []float64{100.5, 101, 105, 120} {
			dx, dy, dz := towards([3]float64{-350, 10, oz}, [3]float64{350, 10, 100})
			if _, ok := ground.trace(-350, 10, oz, dx, dy, dz); ok {
				hits++
			}
		}

		// then
		assert.Equal(t, 4, hits)
	})

	t.Run("should not hit anything outside the rectangle of the grid", func(t *testing.T) {
		// given
		ground := flatTerrain(4, 100)

		// when
		_, parallel := ground.trace(10000, 0, 300, 0, 1, -0.01)
		_, away := ground.trace(0, 0, 300, 0, -1, 0)
		_, past := ground.trace(0, 5000, 300, 0, 1, -0.5)

		// then
		assert.False(t, parallel)
		assert.False(t, away)
		assert.False(t, past)
	})

	t.Run("should say the cell that holds the point it hits, counting from the north west", func(t *testing.T) {
		// given: 8 × 8 cells; the ground is hit 1.5 cells east and 2.5 cells north of the center
		ground := flatTerrain(8, 100)
		target := [3]float64{1.5 * traceCell, 2.5 * traceCell, 100}
		dx, dy, dz := towards([3]float64{0, 0, 300}, target)

		// when
		hit, ok := ground.trace(0, 0, 300, dx, dy, dz)

		// then: row 1 (the center is between rows 3 and 4; 2.5 cells north is row 1), column 5
		require.True(t, ok)
		assert.Equal(t, 1, hit.row)
		assert.Equal(t, 5, hit.col)
	})

	t.Run("should cross the antimeridian with no jump", func(t *testing.T) {
		// given: the grid's west edge is at 179.998° and it has 4 columns, so it passes 180°; the camera is
		// at longitude -179.999°, that is 180.001°, inside it
		grid := testGrid("dem", 2, 4, 0.001, 179.998, 0.001, repeat(8, 10))
		plane := equatorPlane(-179.999)
		ground := terrain{newSurface(grid, 0).place(plane)}
		dx, dy, dz := towards([3]float64{0, 0, 100}, [3]float64{90, 0, 10})

		// when
		hit, ok := ground.trace(0, 0, 100, dx, dy, dz)

		// then
		require.True(t, ok)
		assert.InDelta(t, 90.0, hit.x, 1e-3)
		assert.Equal(t, 3, hit.col)
	})

	t.Run("should give the same hits with and without skipping what is above the highest ground", func(t *testing.T) {
		// given: a rough terrain and two hundred rays of a generator with a fixed seed
		values := make([]float32, 30*30)
		random := rand.New(rand.NewSource(7))
		for i := range values {
			values[i] = float32(200 + 150*random.Float64())
		}
		half := 15 * 0.001
		ground := terrain{newSurface(testGrid("dem", 30, 30, half, -half, 0.001, values), 0).place(equatorPlane(0))}

		type ray struct{ ox, oy, oz, dx, dy, dz float64 }
		rays := make([]ray, 200)
		for i := range rays {
			ox, oy, oz := (random.Float64()-0.5)*2000, (random.Float64()-0.5)*2000, 100+random.Float64()*1500
			dx, dy, dz := towards([3]float64{ox, oy, oz}, [3]float64{(random.Float64() - 0.5) * 2000, (random.Float64() - 0.5) * 2000, random.Float64() * 400})
			rays[i] = ray{ox, oy, oz, dx, dy, dz}
		}

		// when
		trace := func(prune bool) []surfaceHit {
			previous := pruneByHeight
			pruneByHeight = prune
			defer func() { pruneByHeight = previous }()

			hits := make([]surfaceHit, len(rays))
			for i, r := range rays {
				hit, ok := ground.trace(r.ox, r.oy, r.oz, r.dx, r.dy, r.dz)
				if !ok {
					hit = surfaceHit{t: -1}
				}
				hits[i] = hit
			}
			return hits
		}
		pruned, unpruned := trace(true), trace(false)

		// then
		assert.Equal(t, unpruned, pruned)
		found := 0
		for _, hit := range pruned {
			if hit.t >= 0 {
				found++
			}
		}
		assert.Greater(t, found, 100, "the rays hit the terrain often enough to prove something")
	})
}

func repeat(n int, value float32) []float32 {
	values := make([]float32, n)
	for i := range values {
		values[i] = value
	}
	return values
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
