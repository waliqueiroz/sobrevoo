package domain

import (
	"fmt"
	"math"
	"time"
)

// screenOverlay draws the fixed, on-screen blocks (distance, elevation and
// gain, elapsed time, elevation profile) on top of an already-drawn frame —
// never affected by depth, always on top, unlike overlay (frame_overlay.go),
// which draws the trail and the marker draped on the terrain
// (009-frame-overlays research.md items 1, 4, 5, 6, 11, 12).
type screenOverlay struct {
	image  FrameImage
	config OverlayConfig

	// appearance is only used for its MarkerColor, so the elevation
	// profile's dot reads as the same marker drawn on the terrain, and
	// never blends into the profile's own line (OverlayTextColor).
	appearance Appearance

	// face rasterizes every glyph this block draws, from the vector font
	// embedded in the binary — never a bitmap font, never a font installed
	// on the machine (011-overlay-polish FR-001/FR-002, research.md items
	// 1, 3, 4). Built once per Scene (frame_scene.go) and reused by every
	// frame of the same render.
	face *vectorFace
}

// glyphHeightRatio is about how tall, as a share of the frame's height, one
// line of overlay text is — the pixel size (ppem, "pixels per em") a glyph
// is rasterized at, computed once per draw call, not per glyph.
const glyphHeightRatio = 0.028

// overlayLinePadding is the gap kept around a line of text within its
// panel, and between consecutive blocks, in unscaled glyph heights.
const overlayLinePadding = 0.4

// overlayProfileHeightRatio is the height of the elevation profile's panel,
// as a share of the frame's height.
const overlayProfileHeightRatio = 0.12

// draw draws every block plan.Frames[index] and s.config call for, in a
// fixed position that does not move between frames. It does nothing when
// s.config is not enabled, and it silently skips a block whose data the
// plan does not have (FR-016): the time block when the plan has no clock
// reference, and the elevation and profile blocks when the plan has no
// elevation.
func (s screenOverlay) draw(plan CameraPlan, index int) {
	if !s.config.Enabled {
		return
	}
	frame := plan.Frames[index]

	width, height := s.image.Resolution.Width, s.image.Resolution.Height
	marginTop := roundHalfUp(OverlayTopMarginRatio * float64(height))
	marginSide := roundHalfUp(OverlaySideMarginRatio * float64(width))
	marginBottom := roundHalfUp(OverlayBottomMarginRatio * float64(height))
	ppem := max(1, roundHalfUp(glyphHeightRatio*float64(height)))
	pad := roundHalfUp(overlayLinePadding * float64(s.face.lineHeight(ppem)))
	lineHeight := s.face.lineHeight(ppem) + 2*pad

	var distanceText, elevationText, timeText string
	showDistance := s.config.Distance
	showElevation := s.config.Elevation && plan.ElevationAvailable
	showTime := s.config.Time && plan.TimeReference == TimeReferenceClock

	if showDistance {
		distanceText = "DIST " + formatOverlayDistance(frame.MarkerDistance)
	}
	if showElevation {
		elevationText = "ELEV " + formatOverlayElevation(frame.TrackElevation) + "   GAIN " + formatOverlayGain(frame.TrackElevationGain)
	}
	if showTime {
		timeText = "TIME " + formatOverlayElapsed(frame.ActivityElapsed)
	}
	panelWidth := numericPanelWidth(s.face, ppem, distanceText, elevationText, timeText)

	y := marginTop
	if showDistance {
		s.drawLine(marginSide, y, ppem, pad, panelWidth, distanceText)
		y += lineHeight + pad
	}
	if showElevation {
		s.drawLine(marginSide, y, ppem, pad, panelWidth, elevationText)
		y += lineHeight + pad
	}
	if showTime {
		s.drawLine(marginSide, y, ppem, pad, panelWidth, timeText)
	}
	if s.config.Profile && plan.ElevationAvailable {
		s.drawProfile(plan, index, marginSide, marginBottom, ppem)
	}
}

// numericPanelWidth is the width every present numeric block's panel
// shares, in pixels at ppem: the widest of texts, skipping an empty one —
// empty is how draw marks a block that is not shown this frame
// (011-overlay-polish FR-005, research.md item 7).
func numericPanelWidth(face *vectorFace, ppem int, texts ...string) int {
	width := 0
	for _, text := range texts {
		if text == "" {
			continue
		}
		width = max(width, face.textWidth(text, ppem))
	}
	return width
}

// drawLine draws one block: a panel panelWidth pixels wide plus pad on
// every side, its top-left corner at (x, y), with text over it in
// OverlayTextColor. panelWidth is shared by every numeric block present in
// the same draw call (numericPanelWidth), never text's own width alone —
// the three numeric panels line up at the same width (FR-005).
func (s screenOverlay) drawLine(x, y, ppem, pad, panelWidth int, text string) {
	h := s.face.lineHeight(ppem)
	drawPanel(s.image, x, y, x+panelWidth+2*pad, y+h+2*pad)
	s.drawText(text, x+pad, y+pad, ppem, OverlayTextColor)
}

// drawProfile draws the elevation profile block: a panel spanning the
// width between the two side margins, above the bottom margin, with the
// whole track's elevation profile as a line — from the (distance,
// elevation) of every PhaseFollowing frame of plan, recomputed fresh every
// call, never cached between frames (the same choice overlay.drawTrail
// already made, research.md item 11) — and a dot at index's own position on
// it, which is what moves from frame to frame.
func (s screenOverlay) drawProfile(plan CameraPlan, index, marginSide, marginBottom, ppem int) {
	width, height := s.image.Resolution.Width, s.image.Resolution.Height
	x0, x1 := marginSide, width-marginSide
	y1 := height - marginBottom
	y0 := y1 - roundHalfUp(overlayProfileHeightRatio*float64(height))
	if x1 <= x0 || y1 <= y0 {
		return
	}
	drawPanel(s.image, x0, y0, x1, y1)

	var distances, elevations []float64
	for _, f := range plan.Frames {
		if f.Phase == PhaseFollowing {
			distances = append(distances, f.MarkerDistance)
			elevations = append(elevations, f.TrackElevation)
		}
	}
	if len(distances) == 0 {
		return
	}

	maxDistance := distances[len(distances)-1]
	if maxDistance <= 0 {
		maxDistance = 1
	}
	minElevation, maxElevation := elevations[0], elevations[0]
	for _, e := range elevations {
		minElevation = math.Min(minElevation, e)
		maxElevation = math.Max(maxElevation, e)
	}
	elevationSpan := maxElevation - minElevation
	if elevationSpan <= 0 {
		elevationSpan = 1
	}

	pad := roundHalfUp(overlayLinePadding * float64(s.face.lineHeight(ppem)))
	innerX0, innerX1 := x0+pad, x1-pad
	innerY0, innerY1 := y0+pad, y1-pad
	if innerX1 <= innerX0 || innerY1 <= innerY0 {
		innerX0, innerX1, innerY0, innerY1 = x0, x1, y0, y1
	}

	point := func(distance, elevation float64) (int, int) {
		px := innerX0 + roundHalfUp((distance/maxDistance)*float64(innerX1-innerX0))
		py := innerY1 - roundHalfUp(((elevation-minElevation)/elevationSpan)*float64(innerY1-innerY0))
		return px, py
	}

	radius := profileMarkerRadius(height)
	thickness := max(1, radius/2)
	for i := 1; i < len(distances); i++ {
		fromX, fromY := point(distances[i-1], elevations[i-1])
		toX, toY := point(distances[i], elevations[i])
		s.drawSegment(fromX, fromY, toX, toY, thickness)
	}

	frame := plan.Frames[index]
	dotX, dotY := point(frame.MarkerDistance, frame.TrackElevation)
	s.drawDot(dotX, dotY, radius, s.appearance.MarkerColor)
}

// profileMarkerRadius is the radius, in pixels, of the dot that marks the
// elevation profile's current position: a fraction of the frame's height,
// never smaller than the documented floor — the same ratio/floor pattern
// MarkerRadiusRatio/MarkerMinRadius already use for the marker drawn on the
// terrain, but with its own fixed constants (ProfileMarkerRadiusRatio/
// ProfileMarkerMinRadius), since the profile's marker is not a style choice
// and must stay visible whatever radius the user picked for the terrain
// marker (FR-006, research.md item 8).
func profileMarkerRadius(height int) int {
	return max(int(ProfileMarkerMinRadius), roundHalfUp(ProfileMarkerRadiusRatio*float64(height)))
}

// drawSegment draws a straight line from (x0, y0) to (x1, y1), thickness
// pixels wide, by stepping along its longer axis — no need for the
// symmetry a capsule in camera space needs (frame_overlay.go), since this
// is a small chart, not the trail over the terrain.
func (s screenOverlay) drawSegment(x0, y0, x1, y1, thickness int) {
	steps := max(absInt(x1-x0), absInt(y1-y0))
	if steps == 0 {
		s.plotSquare(x0, y0, thickness)
		return
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		s.plotSquare(x0+roundHalfUp(t*float64(x1-x0)), y0+roundHalfUp(t*float64(y1-y0)), thickness)
	}
}

// plotSquare paints a thickness x thickness square of OverlayTextColor
// centered on (cx, cy).
func (s screenOverlay) plotSquare(cx, cy, thickness int) {
	half := thickness / 2
	for oy := -half; oy <= half; oy++ {
		for ox := -half; ox <= half; ox++ {
			blendPixel(s.image, cx+ox, cy+oy, OverlayTextColor, 1)
		}
	}
}

// drawDot paints a filled circle of color, radius pixels, centered on
// (cx, cy) — the profile's marker of the current frame.
func (s screenOverlay) drawDot(cx, cy, radius int, color RGB) {
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx, dy := float64(x-cx), float64(y-cy)
			if math.Sqrt(dx*dx+dy*dy) <= float64(radius) {
				blendPixel(s.image, x, y, color, 1)
			}
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// drawText draws text in color, its top-left corner at (x, y), each glyph
// rasterized by s.face at the given ppem. Every glyph's outline is drawn
// first, across the whole text, then every glyph's fill, also across the
// whole text — never outline-then-fill one glyph at a time, so one glyph's
// outline is never painted back over a neighbor's fill (FR-004, research.md
// item 6). A rune s.face has no glyph for draws nothing and advances by
// nothing.
func (s screenOverlay) drawText(text string, x, y, ppem int, color RGB) {
	outlineRadius := max(int(OverlayOutlineMinWidth), roundHalfUp(OverlayOutlineRatio*float64(s.image.Resolution.Height)))

	penX := x
	for _, r := range text {
		mask, ok := s.face.glyph(r, ppem)
		if ok {
			s.drawGlyphOutline(mask, penX, y, outlineRadius)
		}
		penX += glyphAdvance(mask, ok)
	}

	penX = x
	for _, r := range text {
		mask, ok := s.face.glyph(r, ppem)
		if ok {
			s.drawGlyphFill(mask, penX, y, color)
		}
		penX += glyphAdvance(mask, ok)
	}
}

// glyphAdvance is mask.advance, or 0 when ok is false (the rune s.face has
// no glyph for) — shared by drawText's two passes so both advance the pen
// by exactly the same amount.
func glyphAdvance(mask glyphMask, ok bool) int {
	if !ok {
		return 0
	}
	return mask.advance
}

// drawGlyphFill blends mask's own coverage, in color, at (x, y) — the same
// mix formula overlay.blend already uses for the trail and the marker.
func (s screenOverlay) drawGlyphFill(mask glyphMask, x, y int, color RGB) {
	for gy := 0; gy < mask.height; gy++ {
		for gx := 0; gx < mask.width; gx++ {
			if alpha := mask.coverage[gy*mask.width+gx]; alpha > 0 {
				blendPixel(s.image, x+gx, y+gy, color, float64(alpha)/255)
			}
		}
	}
}

// drawGlyphOutline blends, in OverlayTextOutlineColor, the coverage of
// mask dilated by radius pixels — the maximum coverage within radius of
// each point, which is positive beyond mask's own box exactly where the
// dilation reaches past an edge pixel of the ink (FR-004, research.md item
// 6: the same casing-before-core technique TrailCasingColor/MarkerRingColor
// already use, applied to text).
func (s screenOverlay) drawGlyphOutline(mask glyphMask, x, y, radius int) {
	for gy := -radius; gy < mask.height+radius; gy++ {
		for gx := -radius; gx < mask.width+radius; gx++ {
			if alpha := mask.dilatedAt(gx, gy, radius); alpha > 0 {
				blendPixel(s.image, x+gx, y+gy, OverlayTextOutlineColor, float64(alpha)/255)
			}
		}
	}
}

// dilatedAt is the maximum coverage m has within radius pixels of (gx, gy)
// — gx, gy may fall outside m's own box, which lets an outline extend past
// it; a neighbor outside the box counts as zero coverage.
func (m glyphMask) dilatedAt(gx, gy, radius int) uint8 {
	var highest uint8
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			x, y := gx+dx, gy+dy
			if x < 0 || x >= m.width || y < 0 || y >= m.height {
				continue
			}
			if c := m.coverage[y*m.width+x]; c > highest {
				highest = c
			}
		}
	}
	return highest
}

// blendPixel paints (x, y) with c over what img already has there, by a
// coverage from 0 to 1 — the same formula overlay.blend uses, kept separate
// since a screen overlay has no notion of camera depth to test against.
// Out-of-bounds coordinates are ignored.
func blendPixel(img FrameImage, x, y int, c RGB, coverage float64) {
	if x < 0 || x >= img.Resolution.Width || y < 0 || y >= img.Resolution.Height {
		return
	}
	current := img.At(x, y)
	mix := func(old, painted uint8) uint8 {
		return rounded(float64(float64(old)*(1-coverage)) + float64(float64(painted)*coverage))
	}
	img.Set(x, y, RGB{mix(current.R, c.R), mix(current.G, c.G), mix(current.B, c.B)})
}

// drawPanel blends OverlayPanelColor at OverlayPanelOpacity over the
// rectangle [x0, x1) x [y0, y1), clamped to img's bounds — the backing
// plate that keeps every block legible over any background (FR-010),
// without sampling what is under it.
func drawPanel(img FrameImage, x0, y0, x1, y1 int) {
	x0, y0 = max(x0, 0), max(y0, 0)
	x1, y1 = min(x1, img.Resolution.Width), min(y1, img.Resolution.Height)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			blendPixel(img, x, y, OverlayPanelColor, OverlayPanelOpacity)
		}
	}
}

// formatOverlayDistance formats meters as whole meters below 1 km, or
// kilometers to one decimal from 1 km (research.md item 12).
func formatOverlayDistance(meters float64) string {
	if meters < 1000 {
		return fmt.Sprintf("%.0f m", meters)
	}
	return fmt.Sprintf("%.1f km", meters/1000)
}

// formatOverlayElevation formats meters as whole, unsigned meters.
func formatOverlayElevation(meters float64) string {
	return fmt.Sprintf("%.0f m", meters)
}

// formatOverlayGain formats meters as whole meters, always signed with a
// leading plus (TrackElevationGain never decreases, but a climb is still
// framed as a gain, not a bare measure).
func formatOverlayGain(meters float64) string {
	return fmt.Sprintf("+%.0f m", meters)
}

// formatOverlayElapsed formats d as H:MM:SS, never omitting the hour, so
// the block's width never changes between frames.
func formatOverlayElapsed(d time.Duration) string {
	seconds := int(d.Round(time.Second) / time.Second)
	return fmt.Sprintf("%d:%02d:%02d", seconds/3600, seconds%3600/60, seconds%60)
}
