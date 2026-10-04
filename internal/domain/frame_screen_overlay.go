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

// glyphHeightRatio sizes the elevation profile's own inner padding — the
// only place left, after 015-overlay-redesign, that needs a single text
// size unrelated to a numeric block's label/value (profileFace draws no
// text of its own). The numeric row's text uses OverlayLabelHeightRatio/
// OverlayValueHeightRatio instead (overlayLabelPpem/overlayValuePpem).
const glyphHeightRatio = 0.028

// overlayLinePadding is the gap kept around a line of text within its
// block, and between consecutive lines, in unscaled glyph heights.
const overlayLinePadding = 0.4

// overlayProfileHeightRatio is the height of the elevation profile's panel,
// as a share of the frame's height.
const overlayProfileHeightRatio = 0.12

// overlayBlockText is what a numeric block draws, top to bottom: a
// spelled-out label, a larger value, and (when the value has one) a unit —
// 015-overlay-redesign FR-005/FR-006. unit is "" for a block whose value
// has none (elapsed time): drawBlock then skips that third line entirely,
// without reserving the space it would have taken.
type overlayBlockText struct {
	label, value, unit string
}

// draw draws every numeric block plan.Frames[index] and s.config call for,
// side by side in a single horizontal row at the top of the frame — never
// stacked, never in the order the user named them in (FR-002/FR-003/
// FR-004) — plus the elevation profile at the bottom. It does nothing when
// s.config is not enabled, and it silently skips a block whose data the
// plan does not have (FR-016): the time and speed blocks when the plan has
// no clock reference, and the elevation, gain and profile blocks when the
// plan has no elevation.
func (s screenOverlay) draw(plan CameraPlan, index int) {
	if !s.config.Enabled {
		return
	}
	frame := plan.Frames[index]

	width, height := s.image.Resolution.Width, s.image.Resolution.Height
	marginTop := roundHalfUp(OverlayTopMarginRatio * float64(height))
	marginSide := roundHalfUp(OverlaySideMarginRatio * float64(width))
	marginBottom := roundHalfUp(OverlayBottomMarginRatio * float64(height))

	var present []overlayBlockText
	for _, block := range overlayBlockOrder {
		switch block {
		case OverlayBlockDistance:
			if s.config.Distance {
				present = append(present, distanceBlockText(frame))
			}
		case OverlayBlockElevation:
			if s.config.Elevation && plan.ElevationAvailable {
				present = append(present, elevationBlockText(frame))
			}
		case OverlayBlockGain:
			if s.config.Gain && plan.ElevationAvailable {
				present = append(present, gainBlockText(frame))
			}
		case OverlayBlockTime:
			if s.config.Time && plan.TimeReference == TimeReferenceClock {
				present = append(present, timeBlockText(frame))
			}
		case OverlayBlockSpeed:
			if s.config.Speed && plan.TimeReference == TimeReferenceClock {
				present = append(present, speedBlockText(frame))
			}
		}
	}

	if usableWidth := width - 2*marginSide; len(present) > 0 && usableWidth > 0 {
		labelPpem, valuePpem := overlayLabelPpem(height), overlayValuePpem(height)
		colWidth := usableWidth / len(present)
		for i, text := range present {
			centerX := marginSide + colWidth*i + colWidth/2
			s.drawBlock(centerX, marginTop, labelPpem, valuePpem, text)
		}
	}

	if s.config.Profile && plan.ElevationAvailable {
		s.drawProfile(plan, index, marginSide, marginBottom, overlayPpem(height))
	}
}

// distanceBlockText, elevationBlockText, gainBlockText, timeBlockText and
// speedBlockText are the text of each numeric block, labeled in Brazilian
// Portuguese, spelled out with normal capitalization — this is a personal
// tool used in Portuguese, so every word the drawing writes is
// (015-overlay-redesign FR-005). The unit abbreviations ("km", "m",
// "km/h") and the formatting of the values are untouched (FR-013).
// elevationBlockText shows only the altitude; the accumulated gain, until
// this etapa a second number glued to it, is now gainBlockText's alone
// (FR-008/FR-009).
func distanceBlockText(frame CameraFrame) overlayBlockText {
	value, unit := formatOverlayDistance(frame.MarkerDistance)
	return overlayBlockText{label: "Distância", value: value, unit: unit}
}

func elevationBlockText(frame CameraFrame) overlayBlockText {
	value, unit := formatOverlayElevation(frame.TrackElevation)
	return overlayBlockText{label: "Elevação", value: value, unit: unit}
}

func gainBlockText(frame CameraFrame) overlayBlockText {
	value, unit := formatOverlayGain(frame.TrackElevationGain)
	return overlayBlockText{label: "Ganho", value: value, unit: unit}
}

func timeBlockText(frame CameraFrame) overlayBlockText {
	return overlayBlockText{label: "Tempo decorrido", value: formatOverlayElapsed(frame.ActivityElapsed)}
}

func speedBlockText(frame CameraFrame) overlayBlockText {
	value, unit := formatOverlaySpeed(frame.MarkerSpeed)
	return overlayBlockText{label: "Velocidade", value: value, unit: unit}
}

// overlayPpem sizes the elevation profile's own inner padding (glyphHeightRatio)
// — the row of numeric blocks uses overlayLabelPpem/overlayValuePpem instead.
func overlayPpem(height int) int {
	return max(1, roundHalfUp(glyphHeightRatio*float64(height)))
}

// overlayLabelPpem and overlayValuePpem are the pixel sizes ("pixels per
// em") a numeric block's label/unit and value are rasterized at, for a
// frame height pixels tall (015-overlay-redesign FR-005, research.md item
// 3) — draw and drawBlock are the only callers, so the size text is drawn
// at and the size used to center it never drift apart.
func overlayLabelPpem(height int) int {
	return max(1, roundHalfUp(OverlayLabelHeightRatio*float64(height)))
}

func overlayValuePpem(height int) int {
	return max(1, roundHalfUp(OverlayValueHeightRatio*float64(height)))
}

// drawBlock draws text's up to three lines — label, value, and (when
// present) unit — each horizontally centered on centerX, top to bottom
// starting at topY, with no backing panel (FR-001): label and unit at
// labelPpem, value at valuePpem. A text.unit of "" skips the third line
// entirely, without leaving the gap it would have taken (FR-006).
func (s screenOverlay) drawBlock(centerX, topY, labelPpem, valuePpem int, text overlayBlockText) {
	labelPad := roundHalfUp(overlayLinePadding * float64(s.face.lineHeight(labelPpem)))
	valuePad := roundHalfUp(overlayLinePadding * float64(s.face.lineHeight(valuePpem)))

	y := topY
	s.drawCenteredLine(centerX, y, labelPpem, text.label)
	y += s.face.lineHeight(labelPpem) + labelPad
	s.drawCenteredLine(centerX, y, valuePpem, text.value)
	if text.unit == "" {
		return
	}
	y += s.face.lineHeight(valuePpem) + valuePad
	s.drawCenteredLine(centerX, y, labelPpem, text.unit)
}

// drawCenteredLine draws line, horizontally centered on centerX, its top
// edge at y, at ppem, in OverlayTextColor.
func (s screenOverlay) drawCenteredLine(centerX, y, ppem int, line string) {
	x := centerX - s.face.textWidth(line, ppem)/2
	s.drawText(line, x, y, ppem, OverlayTextColor)
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

	// The line and the marker are drawn twice each, casing before core —
	// the same technique TrailCasingColor already uses for the trail over
	// the terrain (frame_overlay.go, drawTrail) — so the profile stands
	// out from the terrain without a panel behind it (015-overlay-redesign
	// FR-012, research.md item 6).
	for i := 1; i < len(distances); i++ {
		fromX, fromY := point(distances[i-1], elevations[i-1])
		toX, toY := point(distances[i], elevations[i])
		s.drawSegment(fromX, fromY, toX, toY, thickness+profileCasingExtra, OverlayTextOutlineColor)
	}
	for i := 1; i < len(distances); i++ {
		fromX, fromY := point(distances[i-1], elevations[i-1])
		toX, toY := point(distances[i], elevations[i])
		s.drawSegment(fromX, fromY, toX, toY, thickness, OverlayTextColor)
	}

	frame := plan.Frames[index]
	dotX, dotY := point(frame.MarkerDistance, frame.TrackElevation)
	s.drawDot(dotX, dotY, radius+profileCasingExtra, OverlayTextOutlineColor)
	s.drawDot(dotX, dotY, radius, s.appearance.MarkerColor)
}

// profileCasingExtra is how many pixels wider the elevation profile's
// casing (OverlayTextOutlineColor) is drawn than its line/marker's own
// thickness/radius — a fixed pixel increment, the same category as
// TrailCasingColor's own (core+1), not a ratio: the line and the marker
// already have their own proportional sizing (thickness, profileMarkerRadius);
// the casing only needs a few more pixels around them to read as an
// outline (015-overlay-redesign FR-012, research.md item 6).
const profileCasingExtra = 2

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
// pixels wide, in color, by stepping along its longer axis — no need for
// the symmetry a capsule in camera space needs (frame_overlay.go), since
// this is a small chart, not the trail over the terrain.
func (s screenOverlay) drawSegment(x0, y0, x1, y1, thickness int, color RGB) {
	steps := max(absInt(x1-x0), absInt(y1-y0))
	if steps == 0 {
		s.plotSquare(x0, y0, thickness, color)
		return
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		s.plotSquare(x0+roundHalfUp(t*float64(x1-x0)), y0+roundHalfUp(t*float64(y1-y0)), thickness, color)
	}
}

// plotSquare paints a thickness x thickness square of color centered on
// (cx, cy).
func (s screenOverlay) plotSquare(cx, cy, thickness int, color RGB) {
	half := thickness / 2
	for oy := -half; oy <= half; oy++ {
		for ox := -half; ox <= half; ox++ {
			blendPixel(s.image, cx+ox, cy+oy, color, 1)
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
	radius := outlineRadius(ppem)

	penX := x
	for _, r := range text {
		mask, ok := s.face.glyph(r, ppem)
		if ok {
			s.drawGlyphOutline(mask, penX, y, radius)
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

// outlineRadius is the dilation radius of the text's outline, in pixels: a
// fraction of ppem (the glyph's own size), never smaller than the
// documented floor — proportional to the letter being outlined, not to the
// frame it is drawn on, so the same proportion holds at any resolution
// (012-overlay-ptbr-readability FR-005, research.md item 2).
func outlineRadius(ppem int) int {
	return max(int(OverlayOutlineMinWidth), roundHalfUp(OverlayOutlineRatio*float64(ppem)))
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

// formatOverlayDistance formats meters as whole meters below 1 km, or
// kilometers to one decimal from 1 km (research.md item 12), as separate
// value and unit strings (015-overlay-redesign FR-005) — the same numbers
// and rounding as before this etapa (FR-013), just not concatenated.
func formatOverlayDistance(meters float64) (value, unit string) {
	if meters < 1000 {
		return fmt.Sprintf("%.0f", meters), "m"
	}
	return fmt.Sprintf("%.1f", meters/1000), "km"
}

// formatOverlayElevation formats meters as whole, unsigned meters.
func formatOverlayElevation(meters float64) (value, unit string) {
	return fmt.Sprintf("%.0f", meters), "m"
}

// formatOverlayGain formats meters as whole meters, always signed with a
// leading plus (TrackElevationGain never decreases, but a climb is still
// framed as a gain, not a bare measure).
func formatOverlayGain(meters float64) (value, unit string) {
	return fmt.Sprintf("+%.0f", meters), "m"
}

// formatOverlaySpeed formats meters per second as kilometers per hour, one
// decimal (014-speed-overlay-block) — the unit anyone who cycles or runs
// already reads speed in, the same metric convention the other blocks use.
func formatOverlaySpeed(metersPerSecond float64) (value, unit string) {
	return fmt.Sprintf("%.1f", metersPerSecond*3.6), "km/h"
}

// formatOverlayElapsed formats d as H:MM:SS, never omitting the hour, so
// the block's width never changes between frames. Unlike the other four,
// it has no unit (015-overlay-redesign FR-006): it keeps a single string.
func formatOverlayElapsed(d time.Duration) string {
	seconds := int(d.Round(time.Second) / time.Second)
	return fmt.Sprintf("%d:%02d:%02d", seconds/3600, seconds%3600/60, seconds%60)
}
