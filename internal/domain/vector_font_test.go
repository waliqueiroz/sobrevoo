package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
)

func Test_FlattenQuad(t *testing.T) {
	t.Run("should flatten a quadratic curve into the documented fixed number of straight segments", func(t *testing.T) {
		// given
		p0, c, p1 := point{0, 0}, point{0, 10}, point{10, 10}
		var emitted []point

		// when
		flattenQuad(p0, c, p1, func(p point) { emitted = append(emitted, p) })

		// then
		require.Len(t, emitted, quadFlattenSteps)
		assert.Equal(t, p1, emitted[len(emitted)-1])
	})
}

func Test_FlattenCube(t *testing.T) {
	t.Run("should flatten a cubic curve into the documented fixed number of straight segments", func(t *testing.T) {
		// given
		p0, c1, c2, p1 := point{0, 0}, point{0, 10}, point{10, 10}, point{10, 0}
		var emitted []point

		// when
		flattenCube(p0, c1, c2, p1, func(p point) { emitted = append(emitted, p) })

		// then
		require.Len(t, emitted, cubeFlattenSteps)
		assert.Equal(t, p1, emitted[len(emitted)-1])
	})
}

// closedRectangleEdges is the four edges of a rectangle, closed, in the
// winding order that makes its inside produce a positive winding number
// (clockwise in image coordinates, where y grows downward).
func closedRectangleEdges(x0, y0, x1, y1 float64) []edge {
	corners := []point{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
	var edges []edge
	for i, p := range corners {
		next := corners[(i+1)%len(corners)]
		edges = append(edges, edge{p.x, p.y, next.x, next.y})
	}
	return edges
}

func Test_RasterizeOutline(t *testing.T) {
	t.Run("should fully cover a pixel entirely inside the shape", func(t *testing.T) {
		// given: a rectangle covering the whole 3x2 pixel grid
		edges := closedRectangleEdges(0, 0, 3, 2)

		// when
		coverage := rasterizeOutline(edges, 3, 2)

		// then
		assert.Equal(t, uint8(255), coverage[0*3+0])
		assert.Equal(t, uint8(255), coverage[1*3+1])
	})

	t.Run("should leave a pixel entirely outside the shape with zero coverage", func(t *testing.T) {
		// given: a rectangle covering only the first column
		edges := closedRectangleEdges(0, 0, 1, 2)

		// when
		coverage := rasterizeOutline(edges, 3, 2)

		// then
		assert.Equal(t, uint8(0), coverage[0*3+2])
		assert.Equal(t, uint8(0), coverage[1*3+2])
	})

	t.Run("should give a pixel a straight edge splits exactly in half the coverage 128", func(t *testing.T) {
		// given: a rectangle whose right edge falls exactly at the middle of
		// column 2 (x = 2.5, of a column spanning x = [2, 3)) — with the 4x4
		// supersampling grid documented in research.md item 4, the sample
		// x-offsets within a pixel are 0.125, 0.375, 0.625, 0.875; exactly two
		// of the four (0.125, 0.375) fall left of the 0.5 cut, so 8 of the 16
		// samples of that pixel are covered: 255 * 8/16 = 127.5, rounded to 128
		edges := closedRectangleEdges(0, 0, 2.5, 2)

		// when
		coverage := rasterizeOutline(edges, 3, 2)

		// then
		assert.Equal(t, uint8(255), coverage[0*3+0], "column 0, fully inside")
		assert.Equal(t, uint8(255), coverage[0*3+1], "column 1, fully inside")
		assert.Equal(t, uint8(128), coverage[0*3+2], "column 2, split exactly in half")
		assert.Equal(t, uint8(128), coverage[1*3+2], "column 2, second row, same split")
	})

	t.Run("should rasterize the same shape into exactly the same coverage on repeated calls", func(t *testing.T) {
		// given
		edges := closedRectangleEdges(0.3, 0.7, 2.2, 1.6)

		// when
		first := rasterizeOutline(edges, 3, 2)
		second := rasterizeOutline(edges, 3, 2)

		// then
		assert.Equal(t, first, second)
	})
}

func Test_VectorFace_Glyph(t *testing.T) {
	t.Run("should find a glyph for an ASCII rune the font has, with coverage lit somewhere", func(t *testing.T) {
		// given
		face := newVectorFace()

		// when
		mask, ok := face.glyph('0', 20)

		// then
		require.True(t, ok)
		assert.Greater(t, mask.width, 0)
		assert.Greater(t, mask.height, 0)
		assert.Greater(t, mask.advance, 0)
		assert.True(t, hasCoverage(mask), "expected some lit pixel in the glyph's own mask")
	})

	t.Run("should not find a glyph for a rune outside the font, without panicking", func(t *testing.T) {
		// given
		face := newVectorFace()

		// when
		mask, ok := face.glyph('\u0001', 20)

		// then
		assert.False(t, ok)
		assert.Equal(t, glyphMask{}, mask)
	})

	t.Run("should give a taller mask for a larger ppem of the same rune", func(t *testing.T) {
		// given
		face := newVectorFace()

		// when
		small, _ := face.glyph('0', 10)
		large, _ := face.glyph('0', 40)

		// then
		assert.Less(t, small.height, large.height)
	})

	t.Run("should cache a glyph after the first call and not grow the cache on a repeated call", func(t *testing.T) {
		// given
		face := newVectorFace()
		assert.Empty(t, face.cache)

		// when
		face.glyph('0', 20)
		afterFirst := len(face.cache)
		face.glyph('0', 20)
		afterSecond := len(face.cache)

		// then
		assert.Equal(t, 1, afterFirst)
		assert.Equal(t, 1, afterSecond)
	})

	t.Run("should cache a miss too, without growing on a repeated call for the same rune", func(t *testing.T) {
		// given
		face := newVectorFace()

		// when
		face.glyph('\u0001', 20)
		afterFirst := len(face.cache)
		face.glyph('\u0001', 20)
		afterSecond := len(face.cache)

		// then
		assert.Equal(t, 1, afterFirst)
		assert.Equal(t, 1, afterSecond)
	})

	t.Run("should rasterize the same glyph into exactly the same coverage on repeated calls", func(t *testing.T) {
		// given
		first := newVectorFace()
		second := newVectorFace()

		// when
		maskA, _ := first.glyph('G', 23)
		maskB, _ := second.glyph('G', 23)

		// then
		assert.Equal(t, maskA, maskB)
	})
}

// hasCoverage is true when any pixel of mask has non-zero coverage.
func hasCoverage(mask glyphMask) bool {
	for _, c := range mask.coverage {
		if c > 0 {
			return true
		}
	}
	return false
}

func Test_VectorFace_TextWidth(t *testing.T) {
	t.Run("should sum the advance of each rune at the given ppem", func(t *testing.T) {
		// given
		face := newVectorFace()
		const ppem = 20
		g1, _ := face.glyph('1', ppem)
		g2, _ := face.glyph('2', ppem)

		// when
		width := face.textWidth("12", ppem)

		// then
		assert.Equal(t, g1.advance+g2.advance, width)
	})

	t.Run("should skip a rune the font does not have, contributing zero width", func(t *testing.T) {
		// given
		face := newVectorFace()
		const ppem = 20
		g1, _ := face.glyph('1', ppem)

		// when
		width := face.textWidth("1\u0001", ppem)

		// then
		assert.Equal(t, g1.advance, width)
	})
}

func Test_VectorFace_LineHeight(t *testing.T) {
	t.Run("should match the height of a rasterized glyph's own box at the same ppem", func(t *testing.T) {
		// given
		face := newVectorFace()
		const ppem = 20
		mask, _ := face.glyph('0', ppem)

		// when
		height := face.lineHeight(ppem)

		// then
		assert.Equal(t, mask.height, height)
	})
}

func Test_VectorFace_Weight(t *testing.T) {
	t.Run("should rasterize a glyph with more ink than the regular weight would", func(t *testing.T) {
		// given: the font embedded in the binary is a bold weight
		// (012-overlay-ptbr-readability FR-006) — a face built here
		// directly from the regular weight only measures the difference;
		// production code never imports goregular again after this stage
		regularFont, err := sfnt.Parse(goregular.TTF)
		require.NoError(t, err)
		regular := &vectorFace{font: regularFont, cache: make(map[glyphKey]glyphCacheEntry)}
		bold := newVectorFace()
		const ppem = 40

		// when
		boldMask, ok := bold.glyph('0', ppem)
		require.True(t, ok)
		regularMask, ok := regular.glyph('0', ppem)
		require.True(t, ok)

		// then
		assert.Greater(t, coverageSum(boldMask), coverageSum(regularMask))
	})
}

// coverageSum is the total ink of mask — the sum of every pixel's coverage
// — used only to compare how heavy two rasterizations of the same glyph
// are, never for drawing.
func coverageSum(mask glyphMask) int {
	sum := 0
	for _, c := range mask.coverage {
		sum += int(c)
	}
	return sum
}
