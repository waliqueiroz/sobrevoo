package domain

import (
	"fmt"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// glyphSupersample is how many subpixel samples, per axis, rasterizeOutline
// takes of every pixel (4x4 = 16 samples) to compute antialiased coverage —
// a fixed grid, never adaptive, so the result never depends on anything but
// the shape itself (research.md item 4).
const glyphSupersample = 4

// quadFlattenSteps and cubeFlattenSteps are the fixed number of straight
// segments a quadratic or cubic curve of a glyph outline is cut into, by
// uniform parametric evaluation — never adaptive by an error estimate, for
// the same reason: a fixed count is simpler and already enough at the small
// sizes this overlay's text uses (research.md item 4).
const (
	quadFlattenSteps = 8
	cubeFlattenSteps = 12
)

// point is a 2-D coordinate in pixels, in whatever space the caller put it
// (font space, with the Y axis flipped to image space, by flattenSegments).
type point struct{ x, y float64 }

// edge is one straight segment of a flattened glyph outline, in image-pixel
// coordinates (Y growing downward). rasterizeOutline tests a point against
// a whole glyph's edges by the nonzero winding rule.
type edge struct{ x0, y0, x1, y1 float64 }

// glyphKey selects one rasterized glyph: a rune at a pixel size. The pixel
// size is the same for every frame of one execution (it depends only on the
// frame's height, fixed for a render), so in practice every key shares its
// ppem — but the cache does not assume that.
type glyphKey struct {
	r    rune
	ppem int
}

// glyphMask is a glyph's own rasterized coverage (0 to 255 per pixel of its
// box) plus how far, in pixels, the pen advances before the next glyph.
// The box is [0, width) x [0, height): column 0 is the glyph's left edge
// (every glyph of this font has a positive left-side bearing, so the ink
// never starts before column 0); row 0 is the top of the line (the font's
// ascent above the baseline), row height-1 the bottom of the line (the
// descent below it) — the same line height for every glyph of the same
// ppem, whatever its own ink's extent.
type glyphMask struct {
	width, height int
	advance       int
	coverage      []uint8
}

// glyphCacheEntry is what vectorFace.cache keeps for one glyphKey: the mask,
// and whether the font has that rune at all (ok mirrors glyph's own second
// return, including a cached "no" for a rune the font does not have).
type glyphCacheEntry struct {
	mask glyphMask
	ok   bool
}

// vectorFace rasterizes the "Go Bold" font embedded in the binary
// (golang.org/x/image/font/gofont/gobold, BSD-3-Clause, compatible with
// this project's MIT license — 011-overlay-polish research.md item 2). It
// is the bold weight of the same "Go" family the regular weight used
// before 012-overlay-ptbr-readability, chosen for its stronger contrast
// over a bright satellite image (that stage's research.md item 3), one
// glyph at a time, with its own deterministic rasterizer (never
// golang.org/x/image/vector.Rasterizer, which has an amd64-only assembly
// path with no equal on other architectures — research.md item 1). A glyph
// is rasterized the first time it is asked for, at its (rune, ppem), and
// kept in cache for the rest of the execution: the pixel size of the
// overlay's text is the same for every frame of one render, so in practice
// every glyph used is rasterized once, however many frames reuse it.
//
// A *vectorFace is never shared between goroutines at once: screenOverlay.
// draw, its only caller, always runs sequentially within one Scene
// (frame_scene.go), never concurrently with itself.
type vectorFace struct {
	font  *sfnt.Font
	buf   sfnt.Buffer
	cache map[glyphKey]glyphCacheEntry
}

// newVectorFace parses the embedded font once. It panics on a parse failure:
// the font is static data compiled into the binary, so a failure would be a
// build-time bug, never a runtime situation to handle.
func newVectorFace() *vectorFace {
	parsed, err := sfnt.Parse(gobold.TTF)
	if err != nil {
		panic(fmt.Sprintf("sobrevoo: embedded vector font failed to parse: %v", err))
	}
	return &vectorFace{font: parsed, cache: make(map[glyphKey]glyphCacheEntry)}
}

// glyph rasterizes rune r at the given pixel size (ppem — "pixels per em"),
// or reads it from cache. ok is false when the font has no glyph for r; mask
// is then the zero value.
func (f *vectorFace) glyph(r rune, ppem int) (glyphMask, bool) {
	key := glyphKey{r, ppem}
	if entry, found := f.cache[key]; found {
		return entry.mask, entry.ok
	}

	mask, ok := f.rasterize(r, ppem)
	f.cache[key] = glyphCacheEntry{mask, ok}
	return mask, ok
}

// rasterize is the uncached body of glyph.
func (f *vectorFace) rasterize(r rune, ppem int) (glyphMask, bool) {
	index, err := f.font.GlyphIndex(&f.buf, r)
	if err != nil || index == 0 {
		return glyphMask{}, false
	}

	size := fixed.I(ppem)
	segments, err := f.font.LoadGlyph(&f.buf, index, size, nil)
	if err != nil {
		return glyphMask{}, false
	}
	advanceFixed, err := f.font.GlyphAdvance(&f.buf, index, size, font.HintingNone)
	if err != nil {
		return glyphMask{}, false
	}
	ascentPx, height, err := f.lineMetrics(ppem)
	if err != nil {
		return glyphMask{}, false
	}

	width := max(1, roundHalfUp(float64(advanceFixed)/64))
	edges := flattenSegments(segments, ascentPx)
	coverage := rasterizeOutline(edges, width, height)

	return glyphMask{width: width, height: height, advance: width, coverage: coverage}, true
}

// lineMetrics is the font's ascent above the baseline, in pixels, and the
// line's total height (ascent + descent, rounded), both at ppem — the same
// for every glyph of that size, used to place each glyph's box and to size
// the panel of a line of text (lineHeight).
func (f *vectorFace) lineMetrics(ppem int) (ascentPx float64, height int, err error) {
	metrics, err := f.font.Metrics(&f.buf, fixed.I(ppem), font.HintingNone)
	if err != nil {
		return 0, 0, err
	}
	ascentPx = float64(metrics.Ascent) / 64
	descentPx := float64(metrics.Descent) / 64
	return ascentPx, max(1, roundHalfUp(ascentPx+descentPx)), nil
}

// lineHeight is the height, in pixels, of one line of text at ppem — the
// same value every glyph.height already carries, exposed on its own so a
// caller can size a line's panel without rasterizing any particular glyph.
func (f *vectorFace) lineHeight(ppem int) int {
	_, height, err := f.lineMetrics(ppem)
	if err != nil {
		return 0
	}
	return height
}

// textWidth is the width, in pixels, drawText would draw text at ppem: the
// sum of each rune's advance, skipping a rune the font does not have (it
// draws nothing, so it takes no width).
func (f *vectorFace) textWidth(text string, ppem int) int {
	width := 0
	for _, r := range text {
		if mask, ok := f.glyph(r, ppem); ok {
			width += mask.advance
		}
	}
	return width
}

// flattenSegments turns a glyph's curves into straight edges, in image-pixel
// coordinates: X unchanged, Y flipped and shifted so the baseline (font Y =
// 0) lands at row ascentPx, and the font's "up" (negative Y) lands above it
// (research.md item 4). Each contour (from one MoveTo to the next, or to the
// end) is implicitly closed back to its own start, as a glyph outline always
// is — nonzero winding depends on every contour being a closed loop.
func flattenSegments(segments sfnt.Segments, ascentPx float64) []edge {
	toPoint := func(p fixed.Point26_6) point {
		return point{x: float64(p.X) / 64, y: ascentPx + float64(p.Y)/64}
	}

	var edges []edge
	var current, start point
	open := false

	moveTo := func(p point) {
		if open && current != start {
			edges = append(edges, edge{current.x, current.y, start.x, start.y})
		}
		current, start, open = p, p, true
	}
	lineTo := func(p point) {
		edges = append(edges, edge{current.x, current.y, p.x, p.y})
		current = p
	}

	for _, s := range segments {
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			moveTo(toPoint(s.Args[0]))
		case sfnt.SegmentOpLineTo:
			lineTo(toPoint(s.Args[0]))
		case sfnt.SegmentOpQuadTo:
			flattenQuad(current, toPoint(s.Args[0]), toPoint(s.Args[1]), lineTo)
		case sfnt.SegmentOpCubeTo:
			flattenCube(current, toPoint(s.Args[0]), toPoint(s.Args[1]), toPoint(s.Args[2]), lineTo)
		}
	}
	if open && current != start {
		edges = append(edges, edge{current.x, current.y, start.x, start.y})
	}
	return edges
}

// flattenQuad emits the quadFlattenSteps points of the quadratic Bézier
// curve from p0 (not emitted; it is already the current position) through
// control point c to p1, at t = 1/n, 2/n, ..., n/n — the last point is
// always exactly p1.
func flattenQuad(p0, c, p1 point, emit func(point)) {
	for i := 1; i <= quadFlattenSteps; i++ {
		t := float64(i) / float64(quadFlattenSteps)
		u := 1 - t
		x := u*u*p0.x + 2*u*t*c.x + t*t*p1.x
		y := u*u*p0.y + 2*u*t*c.y + t*t*p1.y
		emit(point{x, y})
	}
}

// flattenCube is flattenQuad's cubic counterpart: the curve from p0 (not
// emitted) through control points c1, c2 to p1.
func flattenCube(p0, c1, c2, p1 point, emit func(point)) {
	for i := 1; i <= cubeFlattenSteps; i++ {
		t := float64(i) / float64(cubeFlattenSteps)
		u := 1 - t
		x := u*u*u*p0.x + 3*u*u*t*c1.x + 3*u*t*t*c2.x + t*t*t*p1.x
		y := u*u*u*p0.y + 3*u*u*t*c1.y + 3*u*t*t*c2.y + t*t*t*p1.y
		emit(point{x, y})
	}
}

// rasterizeOutline computes the antialiased coverage (0 to 255) of every
// pixel of a width x height grid against the shape closed edges describes,
// by the nonzero winding rule, supersampled on a fixed glyphSupersample x
// glyphSupersample grid per pixel — only "+ − × ÷", comparison and rounding,
// never a function with a per-architecture implementation (research.md item
// 1, 4). The result is row-major, row 0 first.
func rasterizeOutline(edges []edge, width, height int) []uint8 {
	coverage := make([]uint8, width*height)
	if len(edges) == 0 {
		return coverage
	}

	const samples = glyphSupersample
	const maxSamples = samples * samples

	for py := 0; py < height; py++ {
		for px := 0; px < width; px++ {
			covered := 0
			for sy := 0; sy < samples; sy++ {
				qy := float64(py) + (float64(sy)+0.5)/float64(samples)
				for sx := 0; sx < samples; sx++ {
					qx := float64(px) + (float64(sx)+0.5)/float64(samples)
					if windingNumber(edges, qx, qy) != 0 {
						covered++
					}
				}
			}
			coverage[py*width+px] = uint8(roundHalfUp(255 * float64(covered) / float64(maxSamples)))
		}
	}
	return coverage
}

// windingNumber is the nonzero-winding count of point (qx, qy) against
// edges: for every edge that straddles the horizontal line y = qy, whether
// it crosses to the right of qx, signed by the edge's own direction.
func windingNumber(edges []edge, qx, qy float64) int {
	winding := 0
	for _, e := range edges {
		rising := e.y1 > e.y0
		straddles := (e.y0 <= qy && e.y1 > qy) || (e.y1 <= qy && e.y0 > qy)
		if !straddles {
			continue
		}

		t := (qy - e.y0) / (e.y1 - e.y0)
		xCross := e.x0 + t*(e.x1-e.x0)
		if xCross > qx {
			if rising {
				winding++
			} else {
				winding--
			}
		}
	}
	return winding
}
