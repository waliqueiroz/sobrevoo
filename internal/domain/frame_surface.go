package domain

import "math"

// surface is the terrain of one grid of a slice, as a continuous surface: the
// height is interpolated bilinearly between the centers of the cells (the same
// convention as the exported grid: the center of cell (r, c) is at north − (r +
// 0.5) · cell), and from the last center to the edge of the grid it stays flat, so
// a grid covers exactly the rectangle of its cells and two neighbouring grids
// touch with no crack between them (research.md item 4).
//
// A cell with no value has a height too, or the ray could not pass it and the
// camera might stand over it: that of the sample with a value that is nearest,
// in blocks of city distance (research.md item 6). It is only geometry: the
// cell stays marked as one with no value (isHole), and is never drawn as if the
// height were data.
type surface struct {
	source     GeoDataSource
	rows, cols int

	// north, west, cellLat and cellLon are the geometry of the grid, in degrees.
	north, west      float64
	cellLat, cellLon float64

	// raw are the samples as they were read, a NaN for no value; heights are
	// what the geometry uses: the same slice when nothing is missing, else a
	// copy with the holes filled.
	raw, heights []float32
	hasHoles     bool

	// zmin and zmax are the lowest and the highest height of the surface.
	zmin, zmax float64

	// gradients is the surface's precomputed slope pyramid (016-terrain-
	// lighting, research.md item 5), built once here from raw — never from
	// heights, which would invent a slope out of a filled hole. Index 0 is
	// the finest level, at the grid's own resolution; a later stage adds
	// coarser levels on top of it, for a normal read from farther away.
	gradients []terrainGradient
}

// newSurface builds the surface of a grid. fallback is the height of a grid that
// has no sample with a value at all: the lowest one of the whole slice.
func newSurface(grid ElevationGrid, fallback float64) *surface {
	s := &surface{
		source:  grid.Source,
		rows:    grid.Rows(),
		cols:    grid.Cols(),
		north:   grid.NorthLatitude,
		west:    grid.WestLongitude,
		cellLat: grid.CellLatitude,
		cellLon: grid.CellLongitude,
		raw:     grid.values,
	}

	s.heights = grid.values
	for _, v := range grid.values {
		if v != v {
			s.hasHoles = true
			break
		}
	}
	if s.hasHoles {
		s.heights = fillHoles(grid.values, s.rows, s.cols, float32(fallback))
	}

	s.zmin, s.zmax = math.Inf(1), math.Inf(-1)
	for _, v := range s.heights {
		s.zmin, s.zmax = math.Min(s.zmin, float64(v)), math.Max(s.zmax, float64(v))
	}

	s.gradients = newTerrainGradientPyramid(grid.values, s.rows, s.cols)
	return s
}

// isHole says whether the file has no value for the cell.
func (s *surface) isHole(row, col int) bool {
	v := s.raw[row*s.cols+col]
	return v != v
}

// terrainGradient is one level of a surface's precomputed slope pyramid: the
// derivative of the height at each node, in each grid direction (meters of
// height change per cell, of this level's own resolution), and how much of
// it is backed by a real elevation sample — 0 at a node that is itself a
// hole, 1 at one with a real value (016-terrain-lighting, research.md item
// 5). A level coarser than 0 is a later stage's addition.
type terrainGradient struct {
	rows, cols int
	dRow, dCol []float32
	coverage   []float32
}

// newTerrainGradient builds the finest level of a slope pyramid from a
// grid's raw samples (a NaN marking a cell the file has no value for),
// never from the hole-filled heights the ray's geometry uses, which would
// invent a slope out of a hole. At each node: the central difference
// between its two opposite raw neighbors, in each direction, when both
// have a value; the one-sided difference when only one does; zero when
// neither does, without losing the node's own coverage, which depends only
// on whether the node itself has a value (research.md item 7).
func newTerrainGradient(raw []float32, rows, cols int) terrainGradient {
	g := terrainGradient{
		rows: rows, cols: cols,
		dRow:     make([]float32, rows*cols),
		dCol:     make([]float32, rows*cols),
		coverage: make([]float32, rows*cols),
	}

	at := func(row, col int) (value float32, ok bool) {
		if row < 0 || row >= rows || col < 0 || col >= cols {
			return 0, false
		}
		v := raw[row*cols+col]
		return v, v == v
	}

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			center, ok := at(row, col)
			if !ok {
				continue // coverage, dRow and dCol all stay zero: never read as real
			}
			i := row*cols + col
			g.coverage[i] = 1

			west, hasWest := at(row, col-1)
			east, hasEast := at(row, col+1)
			g.dCol[i] = float32(axisDerivative(float64(center), float64(west), float64(east), hasWest, hasEast))

			north, hasNorth := at(row-1, col)
			south, hasSouth := at(row+1, col)
			g.dRow[i] = float32(axisDerivative(float64(center), float64(north), float64(south), hasNorth, hasSouth))
		}
	}
	return g
}

// axisDerivative is the derivative at a node, in the unit of one cell, from
// its two opposite neighbors: a central difference when both are real, a
// one-sided difference when only one is, and zero (no information in this
// direction) when neither is.
func axisDerivative(center, prev, next float64, hasPrev, hasNext bool) float64 {
	switch {
	case hasPrev && hasNext:
		return (next - prev) / 2
	case hasNext:
		return next - center
	case hasPrev:
		return center - prev
	default:
		return 0
	}
}

// newTerrainGradientPyramid is a surface's whole slope pyramid: the finest
// level (newTerrainGradient, at the grid's own resolution), then each next
// one half the rows and columns of the last, down to 1×1 — the same shape
// of pyramid newTileTexture already builds for a tile's mipmaps, applied to
// the derivative of the height instead of to color (016-terrain-lighting,
// research.md item 5).
func newTerrainGradientPyramid(raw []float32, rows, cols int) []terrainGradient {
	levels := []terrainGradient{newTerrainGradient(raw, rows, cols)}
	for {
		last := levels[len(levels)-1]
		if last.rows <= 1 && last.cols <= 1 {
			return levels
		}
		levels = append(levels, halveGradient(last))
	}
}

// halveGradient is the next coarser level of a slope pyramid: half the rows
// and columns of level (at least one of each), each node the average of its
// (up to) four children — weighted by how much of each one is backed by a
// real elevation sample, so a hole never pulls a coarser level's derivative
// toward an invented value, only dilutes its coverage (research.md item 7).
// The same halving newTileTexture's halved does for a tile's mipmap, with a
// weight instead of a plain mean.
func halveGradient(level terrainGradient) terrainGradient {
	rows, cols := max(level.rows/2, 1), max(level.cols/2, 1)
	out := terrainGradient{rows: rows, cols: cols, dRow: make([]float32, rows*cols), dCol: make([]float32, rows*cols), coverage: make([]float32, rows*cols)}

	at := func(row, col int) (dc, dr, cov float64) {
		row = min(row, level.rows-1)
		col = min(col, level.cols-1)
		i := row*level.cols + col
		return float64(level.dCol[i]), float64(level.dRow[i]), float64(level.coverage[i])
	}

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			dc00, dr00, cov00 := at(2*row, 2*col)
			dc01, dr01, cov01 := at(2*row, 2*col+1)
			dc10, dr10, cov10 := at(2*row+1, 2*col)
			dc11, dr11, cov11 := at(2*row+1, 2*col+1)

			weight := cov00 + cov01 + cov10 + cov11
			if weight == 0 {
				continue // dRow, dCol and coverage all stay zero: no real child to average
			}
			i := row*cols + col
			out.coverage[i] = float32(weight / 4)
			out.dCol[i] = float32((float64(cov00*dc00) + float64(cov01*dc01) + float64(cov10*dc10) + float64(cov11*dc11)) / weight)
			out.dRow[i] = float32((float64(cov00*dr00) + float64(cov01*dr01) + float64(cov10*dr10) + float64(cov11*dr11)) / weight)
		}
	}
	return out
}

// fillHoles copies values, giving each NaN the value of the nearest sample that
// has one, by the distance in blocks (|Δrow| + |Δcolumn|). Two passes, one from
// the north-west and one back from the south-east, are exact for that distance
// and take time proportional to the size of the grid; on a tie the sample the
// first pass reaches — from the north, then from the west — wins. A grid with no
// value at all is filled with fallback.
func fillHoles(values []float32, rows, cols int, fallback float32) []float32 {
	const unreached = math.MaxInt32

	filled := make([]float32, len(values))
	distance := make([]int32, len(values))
	for i, v := range values {
		if v == v {
			filled[i], distance[i] = v, 0
		} else {
			filled[i], distance[i] = fallback, unreached
		}
	}

	take := func(i, from int) {
		if distance[from] != unreached && distance[from]+1 < distance[i] {
			distance[i], filled[i] = distance[from]+1, filled[from]
		}
	}

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			i := r*cols + c
			if r > 0 {
				take(i, i-cols)
			}
			if c > 0 {
				take(i, i-1)
			}
		}
	}
	for r := rows - 1; r >= 0; r-- {
		for c := cols - 1; c >= 0; c-- {
			i := r*cols + c
			if r < rows-1 {
				take(i, i+cols)
			}
			if c < cols-1 {
				take(i, i+1)
			}
		}
	}
	return filled
}

// placedSurface is a surface on the plane of a frame: the rectangle it covers,
// in meters, and the size of its cells.
type placedSurface struct {
	*surface

	// x0 and y0 are the north-west corner; cx and cy the size of a cell.
	x0, y0, cx, cy float64

	// xMax and yMin are the south-east corner.
	xMax, yMin float64
}

// place puts the surface on the plane of a frame. The grid is contiguous
// across the antimeridian: its west edge is put at its longitude relative to
// the camera, and its columns keep going east.
func (s *surface) place(plane framePlane) placedSurface {
	g := placedSurface{
		surface: s,
		x0:      float64(plane.eastwards(s.west) * plane.metersPerDegreeLon),
		y0:      float64((s.north - plane.latitude) * plane.metersPerDegreeLat),
		cx:      float64(s.cellLon * plane.metersPerDegreeLon),
		cy:      float64(s.cellLat * plane.metersPerDegreeLat),
	}
	g.xMax = g.x0 + float64(float64(s.cols)*g.cx)
	g.yMin = g.y0 - float64(float64(s.rows)*g.cy)
	return g
}

// node is the height at the center of cell (row, col); a cell outside the grid
// is the nearest one inside it.
func (g placedSurface) node(row, col int) float64 {
	row = min(max(row, 0), g.rows-1)
	col = min(max(col, 0), g.cols-1)
	return float64(g.heights[row*g.cols+col])
}

// nodeCoordinates are the position (x, y) in the space of the nodes: a is the
// column and b the row of the center it is at, so a whole number is a center.
func (g placedSurface) nodeCoordinates(x, y float64) (a, b float64) {
	return (x-g.x0)/g.cx - 0.5, (g.y0-y)/g.cy - 0.5
}

// heightAt is the height of the surface at (x, y); outside the rectangle it is
// that of the nearest edge.
func (g placedSurface) heightAt(x, y float64) float64 {
	a, b := g.nodeCoordinates(x, y)
	return g.interpolate(a, b)
}

// interpolate is the bilinear height at node coordinates (a, b).
func (g placedSurface) interpolate(a, b float64) float64 {
	column, row := math.Floor(a), math.Floor(b)
	fa, fb := a-column, b-row
	c0, r0 := int(column), int(row)

	north := float64(g.node(r0, c0)*(1-fa)) + float64(g.node(r0, c0+1)*fa)
	south := float64(g.node(r0+1, c0)*(1-fa)) + float64(g.node(r0+1, c0+1)*fa)
	return float64(north*(1-fb)) + float64(south*fb)
}

// levelGradientAt is the coverage-weighted bilinear blend of the four nodes
// of gradients[level] around node coordinates (a, b): the weighted average
// derivative in each grid direction, counting only the nodes that have a
// real elevation sample, and how much of the blend they cover — 0 when
// none of the four does (the slope there is unknown), up to 1 when all four
// do (016-terrain-lighting, research.md items 5, 7).
func (g placedSurface) levelGradientAt(level int, a, b float64) (dCol, dRow, coverage float64) {
	lvl := g.gradients[level]
	column, row := math.Floor(a), math.Floor(b)
	fa, fb := a-column, b-row
	c0, r0 := int(column), int(row)

	at := func(row, col int) (dc, dr, cov float64) {
		row = min(max(row, 0), lvl.rows-1)
		col = min(max(col, 0), lvl.cols-1)
		i := row*lvl.cols + col
		return float64(lvl.dCol[i]), float64(lvl.dRow[i]), float64(lvl.coverage[i])
	}

	dc00, dr00, cov00 := at(r0, c0)
	dc01, dr01, cov01 := at(r0, c0+1)
	dc10, dr10, cov10 := at(r0+1, c0)
	dc11, dr11, cov11 := at(r0+1, c0+1)

	w00 := float64(float64((1-fa)*(1-fb)) * cov00)
	w01 := float64(float64(fa*(1-fb)) * cov01)
	w10 := float64(float64((1-fa)*fb) * cov10)
	w11 := float64(float64(fa*fb) * cov11)

	weight := w00 + w01 + w10 + w11
	if weight == 0 {
		return 0, 0, 0
	}

	dCol = (float64(w00*dc00) + float64(w01*dc01) + float64(w10*dc10) + float64(w11*dc11)) / weight
	dRow = (float64(w00*dr00) + float64(w01*dr01) + float64(w10*dr10) + float64(w11*dr11)) / weight
	return dCol, dRow, weight
}

// gradientAt is levelGradientAt(level, ...), from (x, y) in meters (the
// plane of a frame) instead of level 0's node coordinates: level's own
// nodes are half as dense as level-1's, so the node coordinates are scaled
// by 2⁻ˡᵉᵛᵉˡ before the blend.
func (g placedSurface) gradientAt(level int, x, y float64) (dCol, dRow, coverage float64) {
	a, b := g.nodeCoordinates(x, y)
	scale := math.Ldexp(1, -level)
	return g.levelGradientAt(level, float64(a*scale), float64(b*scale))
}

// climbGradientAt is gradientAt(level, x, y), climbing to the next coarser
// level, and the next, up to the coarsest one of the pyramid, whenever the
// level asked for has no coverage at (x, y) — the neighborhood is widened
// until a real sample is found, never invented (016-terrain-lighting
// FR-008, research.md item 7).
func (g placedSurface) climbGradientAt(level int, x, y float64) (dCol, dRow, coverage float64) {
	last := len(g.gradients) - 1
	for l := max(level, 0); l <= last; l++ {
		if dCol, dRow, coverage = g.gradientAt(l, x, y); coverage > 0 {
			return dCol, dRow, coverage
		}
	}
	return 0, 0, 0
}

// normalAt is the unit surface normal at (x, y) (meters, the plane of a
// frame), read from the gradient pyramid at the level proportional to
// footprintMeters — how much ground the screen pixel this normal is for
// actually covers there — blended between the two nearest levels the same
// way sampler.color blends between two mipmaps of a tile's texture
// (016-terrain-lighting FR-007, research.md item 6); it climbs to a coarser
// level wherever the one chosen has no coverage at (x, y) (climbGradientAt).
// It always succeeds: the flat normal (0, 0, 1) where no level of the
// pyramid has any real sample there (research.md items 4, 7).
func (g placedSurface) normalAt(x, y, footprintMeters float64) (nx, ny, nz float64) {
	last := len(g.gradients) - 1
	cellSize := math.Sqrt(float64(g.cx * g.cy))
	lod := clamp(math.Log2(math.Max(footprintMeters/cellSize, 1)), 0, float64(last))
	fine := int(math.Floor(lod))
	fraction := lod - float64(fine)
	coarse := min(fine+1, last)

	dCol, dRow, coverage := g.climbGradientAt(fine, x, y)
	if fraction > 0 && coarse != fine {
		dColCoarse, dRowCoarse, coverageCoarse := g.climbGradientAt(coarse, x, y)
		switch {
		case coverage == 0:
			dCol, dRow, coverage = dColCoarse, dRowCoarse, coverageCoarse
		case coverageCoarse > 0:
			dCol = float64(float64(1-fraction)*dCol) + float64(float64(fraction)*dColCoarse)
			dRow = float64(float64(1-fraction)*dRow) + float64(float64(fraction)*dRowCoarse)
		}
	}
	if coverage == 0 {
		return 0, 0, 1
	}

	slopeX := dCol / g.cx
	slopeY := -dRow / g.cy
	length := math.Sqrt(float64(slopeX*slopeX) + float64(slopeY*slopeY) + 1)
	return -slopeX / length, -slopeY / length, 1 / length
}

// cellAt is the cell that contains (x, y): rows count from the north, columns
// from the west, and a point on the outer south or east edge belongs to the
// last row or column.
func (g placedSurface) cellAt(x, y float64) (row, col int) {
	return min(max(int(math.Floor((g.y0-y)/g.cy)), 0), g.rows-1),
		min(max(int(math.Floor((x-g.x0)/g.cx)), 0), g.cols-1)
}

// distanceTo is how far (x, y) is from the rectangle of the surface, zero
// inside it.
func (g placedSurface) distanceTo(x, y float64) float64 {
	dx := math.Max(math.Max(g.x0-x, x-g.xMax), 0)
	dy := math.Max(math.Max(y-g.y0, g.yMin-y), 0)
	return math.Sqrt(float64(dx*dx) + float64(dy*dy))
}

// terrain is all the surfaces of a slice on the plane of a frame.
type terrain []placedSurface

// groundAt is the height of the terrain at (x, y): that of the grid that holds
// the point or, outside all of them, of the nearest one, with the point pulled
// into it. When two grids are as near, the first of them.
func (t terrain) groundAt(x, y float64) float64 {
	best, bestDistance := 0, math.Inf(1)
	for i, g := range t {
		if d := g.distanceTo(x, y); d < bestDistance {
			best, bestDistance = i, d
		}
	}
	return t[best].heightAt(x, y)
}

// pruneByHeight makes a ray skip the stretch in which it is above the highest
// ground of a surface: there it cannot hit anything, so skipping it changes no
// result. It is a variable only so a test can prove that.
var pruneByHeight = true

// bisections is how many times the stretch of a ray in the quad it hits is
// halved to find where: the last one is under a tenth of a millimeter for the
// cells of a real grid.
const bisections = 20

// surfaceHit is where a ray met the terrain: how far along the ray (of length 1
// per unit), the position on the plane, the cell of the grid that holds it and
// which surface, of the terrain, it is.
type surfaceHit struct {
	t, x, y  float64
	row, col int
	grid     int
}

// clip is the stretch of the ray, from tEnter to tExit, that is over the
// rectangle of the surface; ok is false when there is none.
func (g placedSurface) clip(ox, oy, dx, dy float64) (tEnter, tExit float64, ok bool) {
	tEnter, tExit = 0, math.Inf(1)

	slab := func(origin, direction, low, high float64) bool {
		if direction == 0 {
			return origin >= low && origin <= high
		}
		t1, t2 := (low-origin)/direction, (high-origin)/direction
		tEnter, tExit = math.Max(tEnter, math.Min(t1, t2)), math.Min(tExit, math.Max(t1, t2))
		return true
	}
	if !slab(ox, dx, g.x0, g.xMax) || !slab(oy, dy, g.yMin, g.y0) {
		return 0, 0, false
	}
	return tEnter, tExit, tEnter < tExit
}

// trace follows a ray, from (ox, oy, oz) along the unit direction (dx, dy, dz),
// to the first point at which it is not above the surface: at the entrance of
// the surface if the ray comes from under the ground, else inside the quad — the
// space between four neighbouring centers — where it goes through, found by
// halving that stretch. Quads are visited one by one along the ray, so nothing
// finer than the grid is missed (Amanatides and Woo).
func (g placedSurface) trace(ox, oy, oz, dx, dy, dz float64) (surfaceHit, bool) {
	rectEnter, rectExit, ok := g.clip(ox, oy, dx, dy)
	if !ok {
		return surfaceHit{}, false
	}

	// The stretch worth visiting: not the part in which the ray is above all the
	// ground. It only chooses which quads to visit; the stretch of each quad
	// comes from the quad and the rectangle alone, so pruning cannot change a hit.
	start, stop := rectEnter, rectExit
	if dx == 0 && dy == 0 {
		// straight down (or up): there is no horizontal stretch to bound it
		if dz >= 0 {
			return surfaceHit{}, false
		}
		stop = (oz - g.zmin) / -dz
		rectExit = stop
	}
	if pruneByHeight {
		switch {
		case dz > 0:
			if oz+float64(start*dz) > g.zmax {
				return surfaceHit{}, false
			}
			stop = math.Min(stop, (g.zmax-oz)/dz)
		case dz < 0:
			start = math.Max(start, (g.zmax-oz)/dz)
		default:
			if oz > g.zmax {
				return surfaceHit{}, false
			}
		}
		if start >= stop {
			return surfaceHit{}, false
		}
	}

	column, row := g.quadAt(ox, oy, dx, dy, start)
	stepColumn, stepRow := 0, 0
	if dx > 0 {
		stepColumn = 1
	} else if dx < 0 {
		stepColumn = -1
	}
	if dy < 0 {
		stepRow = 1 // the rows count southwards
	} else if dy > 0 {
		stepRow = -1
	}

	// the time at which the ray is at the node coordinate a (of columns) or b (of rows)
	atColumn := func(a float64) float64 { return (g.x0 + (a+0.5)*g.cx - ox) / dx }
	atRow := func(b float64) float64 { return (g.y0 - (b+0.5)*g.cy - oy) / dy }

	for column >= -1 && column <= g.cols-1 && row >= -1 && row <= g.rows-1 {
		columnIn, columnOut := math.Inf(-1), math.Inf(1)
		switch {
		case dx > 0:
			columnIn, columnOut = atColumn(float64(column)), atColumn(float64(column+1))
		case dx < 0:
			columnIn, columnOut = atColumn(float64(column+1)), atColumn(float64(column))
		}
		rowIn, rowOut := math.Inf(-1), math.Inf(1)
		switch {
		case dy < 0:
			rowIn, rowOut = atRow(float64(row)), atRow(float64(row+1))
		case dy > 0:
			rowIn, rowOut = atRow(float64(row+1)), atRow(float64(row))
		}

		in := math.Max(rectEnter, math.Max(columnIn, rowIn))
		out := math.Min(rectExit, math.Min(columnOut, rowOut))
		if in >= stop {
			break
		}

		if out > in {
			if g.aboveQuad(ox, oy, oz, dx, dy, dz, in, row, column) <= 0 {
				return g.hitAt(ox, oy, dx, dy, in), true
			}
			if g.aboveQuad(ox, oy, oz, dx, dy, dz, out, row, column) <= 0 {
				low, high := in, out
				for i := 0; i < bisections; i++ {
					middle := (low + high) / 2
					if g.aboveQuad(ox, oy, oz, dx, dy, dz, middle, row, column) <= 0 {
						high = middle
					} else {
						low = middle
					}
				}
				return g.hitAt(ox, oy, dx, dy, (low+high)/2), true
			}
		}

		if out >= rectExit {
			break
		}
		switch {
		case columnOut < rowOut:
			column += stepColumn
		case rowOut < columnOut:
			row += stepRow
		default:
			column, row = column+stepColumn, row+stepRow
		}
	}
	return surfaceHit{}, false
}

// quadAt is the quad, of column and row of its north-west center, that holds the
// ray at time t; the quads at the edges are -1 (before the first center) and the
// last row or column.
func (g placedSurface) quadAt(ox, oy, dx, dy, t float64) (column, row int) {
	a, b := g.nodeCoordinates(ox+float64(t*dx), oy+float64(t*dy))
	column = min(max(int(math.Floor(a)), -1), g.cols-1)
	row = min(max(int(math.Floor(b)), -1), g.rows-1)
	return column, row
}

// aboveQuad is how far the ray is above the surface at time t, taking the
// surface as the bilinear one of the quad (row, column).
func (g placedSurface) aboveQuad(ox, oy, oz, dx, dy, dz, t float64, row, column int) float64 {
	a, b := g.nodeCoordinates(ox+float64(t*dx), oy+float64(t*dy))
	fa := math.Min(math.Max(a-float64(column), 0), 1)
	fb := math.Min(math.Max(b-float64(row), 0), 1)

	north := float64(g.node(row, column)*(1-fa)) + float64(g.node(row, column+1)*fa)
	south := float64(g.node(row+1, column)*(1-fa)) + float64(g.node(row+1, column+1)*fa)
	ground := float64(north*(1-fb)) + float64(south*fb)

	return oz + float64(t*dz) - ground
}

// hitAt is the hit of the ray at time t.
func (g placedSurface) hitAt(ox, oy, dx, dy, t float64) surfaceHit {
	x, y := ox+float64(t*dx), oy+float64(t*dy)
	row, col := g.cellAt(x, y)
	return surfaceHit{t: t, x: x, y: y, row: row, col: col}
}

// trace is the first hit, over all the surfaces, of a ray. The surfaces the ray
// crosses are followed in the order it enters them (the first of two that it
// enters together first), and it stops as soon as no surface that is left can be
// nearer than the hit it has.
func (t terrain) trace(ox, oy, oz, dx, dy, dz float64) (surfaceHit, bool) {
	type candidate struct {
		index  int
		tEnter float64
	}
	var buffer [8]candidate
	candidates := buffer[:0]

	for i, g := range t {
		tEnter, _, ok := g.clip(ox, oy, dx, dy)
		if !ok {
			continue
		}
		position := len(candidates)
		candidates = append(candidates, candidate{})
		for position > 0 && candidates[position-1].tEnter > tEnter {
			candidates[position] = candidates[position-1]
			position--
		}
		candidates[position] = candidate{index: i, tEnter: tEnter}
	}

	var best surfaceHit
	found := false
	for _, c := range candidates {
		if found && best.t <= c.tEnter {
			break
		}
		if hit, ok := t[c.index].trace(ox, oy, oz, dx, dy, dz); ok && (!found || hit.t < best.t) {
			best, found = hit, true
			best.grid = c.index
		}
	}
	return best, found
}
