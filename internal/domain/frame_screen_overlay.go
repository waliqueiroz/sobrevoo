package domain

import (
	"fmt"
	"image"
	"math"
	"time"

	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/inconsolata"
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
}

// overlayFace is the font every screen overlay block is drawn with: a
// bitmap embedded in the binary as Go source (golang.org/x/image), never a
// font installed on the machine (FR-008). Its glyph mask is fixed data,
// computed once when the module was built — reading it at render time adds
// no floating-point operation beyond the alpha blend every other overlay
// (overlay.blend) already does.
var overlayFace = inconsolata.Bold8x16

// glyphHeightRatio is about how tall, as a share of the frame's height,
// one line of overlay text is.
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
	margin := roundHalfUp(OverlayMarginRatio * float64(min(width, height)))
	scale := textScale(height)
	glyphHeight := overlayFace.Ascent + overlayFace.Descent
	pad := roundHalfUp(overlayLinePadding * float64(glyphHeight) * float64(scale))
	lineHeight := glyphHeight*scale + 2*pad

	y := margin
	if s.config.Distance {
		s.drawLine(margin, y, scale, pad, "DIST "+formatOverlayDistance(frame.MarkerDistance))
		y += lineHeight + pad
	}
	if s.config.Elevation && plan.ElevationAvailable {
		text := "ELEV " + formatOverlayElevation(frame.TrackElevation) + "   GAIN " + formatOverlayGain(frame.TrackElevationGain)
		s.drawLine(margin, y, scale, pad, text)
		y += lineHeight + pad
	}
	if s.config.Time && plan.TimeReference == TimeReferenceClock {
		s.drawLine(margin, y, scale, pad, "TIME "+formatOverlayElapsed(frame.ActivityElapsed))
	}
	if s.config.Profile && plan.ElevationAvailable {
		s.drawProfile(plan, index, margin, scale)
	}
}

// drawLine draws one block: a panel sized to fit text at scale plus pad on
// every side, its top-left corner at (x, y), with the text over it in
// OverlayTextColor.
func (s screenOverlay) drawLine(x, y, scale, pad int, text string) {
	glyphHeight := overlayFace.Ascent + overlayFace.Descent
	w, h := textWidth(text, scale), glyphHeight*scale
	drawPanel(s.image, x, y, x+w+2*pad, y+h+2*pad)
	drawText(s.image, text, x+pad, y+pad, scale, OverlayTextColor)
}

// drawProfile draws the elevation profile block: a panel spanning the
// width between the two side margins, above the bottom margin, with the
// whole track's elevation profile as a line — from the (distance,
// elevation) of every PhaseFollowing frame of plan, recomputed fresh every
// call, never cached between frames (the same choice overlay.drawTrail
// already made, research.md item 11) — and a dot at index's own position on
// it, which is what moves from frame to frame.
func (s screenOverlay) drawProfile(plan CameraPlan, index, margin, scale int) {
	width, height := s.image.Resolution.Width, s.image.Resolution.Height
	x0, x1 := margin, width-margin
	y1 := height - margin
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

	pad := roundHalfUp(overlayLinePadding * float64(overlayFace.Ascent+overlayFace.Descent) * float64(scale))
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

	thickness := max(1, scale/2)
	for i := 1; i < len(distances); i++ {
		fromX, fromY := point(distances[i-1], elevations[i-1])
		toX, toY := point(distances[i], elevations[i])
		s.drawSegment(fromX, fromY, toX, toY, thickness)
	}

	frame := plan.Frames[index]
	dotX, dotY := point(frame.MarkerDistance, frame.TrackElevation)
	s.drawDot(dotX, dotY, max(2, scale), s.appearance.MarkerColor)
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

// textScale is the integer factor a glyph is scaled by — nearest-neighbor
// pixel replication, never interpolation, so no floating point enters the
// per-pixel loop (research.md item 4).
func textScale(frameHeight int) int {
	glyphHeight := overlayFace.Ascent + overlayFace.Descent
	return max(1, roundHalfUp(glyphHeightRatio*float64(frameHeight)/float64(glyphHeight)))
}

// textWidth is the width, in pixels, drawText would draw text at scale.
func textWidth(text string, scale int) int {
	return len(text) * overlayFace.Advance * scale
}

// glyphOffset returns the row, within face's Mask, where r's glyph starts;
// ok is false for a rune the font does not have.
func glyphOffset(face *basicfont.Face, r rune) (offset int, ok bool) {
	glyphHeight := face.Ascent + face.Descent
	for _, rng := range face.Ranges {
		if r >= rng.Low && r < rng.High {
			return (int(r-rng.Low) + rng.Offset) * glyphHeight, true
		}
	}
	return 0, false
}

// drawText draws text in color, its top-left corner at (x, y), each glyph
// scaled by the integer factor scale (pixel replication) and blended over
// what is already there by its own alpha mask — the same mix formula
// overlay.blend already uses for the trail and the marker.
func drawText(img FrameImage, text string, x, y, scale int, color RGB) {
	mask, ok := overlayFace.Mask.(*image.Alpha)
	if !ok {
		return
	}
	glyphHeight := overlayFace.Ascent + overlayFace.Descent

	for i := 0; i < len(text); i++ {
		r := rune(text[i])
		left := x + i*overlayFace.Advance*scale
		offset, ok := glyphOffset(overlayFace, r)
		if !ok {
			continue
		}

		for gy := 0; gy < glyphHeight; gy++ {
			for gx := 0; gx < overlayFace.Width; gx++ {
				alpha := mask.Pix[(offset+gy)*mask.Stride+gx]
				if alpha == 0 {
					continue
				}
				coverage := float64(alpha) / 255

				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						blendPixel(img, left+gx*scale+sx, y+gy*scale+sy, color, coverage)
					}
				}
			}
		}
	}
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
