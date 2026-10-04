package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ScreenOverlay_DrawText(t *testing.T) {
	face := newVectorFace()

	t.Run("should leave pixels far from the text untouched", func(t *testing.T) {
		// given
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		img := NewFrameImage(Resolution{Width: 100, Height: 60}, background)
		overlay := screenOverlay{image: img, face: face}

		// when
		overlay.drawText("0", 5, 5, 20, RGB{R: 0xFF, G: 0xFF, B: 0xFF})

		// then
		assert.Equal(t, background, img.At(90, 50))
	})

	t.Run("should paint a fully covered pixel of the glyph's own mask exactly in the text color", func(t *testing.T) {
		// given
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		img := NewFrameImage(Resolution{Width: 60, Height: 60}, background)
		overlay := screenOverlay{image: img, face: face}
		mask, ok := face.glyph('0', 20)
		require.True(t, ok)

		// when
		overlay.drawText("0", 5, 5, 20, RGB{R: 0xFF, G: 0xFF, B: 0xFF})

		// then: find a pixel the mask says is fully covered (coverage 255
		// blends to exactly the painted color, never a mix with the
		// background) — the test does not hardcode which (gx, gy) that is,
		// only that the drawing follows the mask it is given
		found := false
		for gy := 0; gy < mask.height && !found; gy++ {
			for gx := 0; gx < mask.width && !found; gx++ {
				if mask.coverage[gy*mask.width+gx] == 255 {
					assert.Equal(t, RGB{R: 0xFF, G: 0xFF, B: 0xFF}, img.At(5+gx, 5+gy))
					found = true
				}
			}
		}
		require.True(t, found, "expected at least one fully covered pixel in '0' at ppem 20")
	})

	t.Run("should draw the same text twice into byte-identical images", func(t *testing.T) {
		// given
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		first := NewFrameImage(Resolution{Width: 200, Height: 60}, background)
		second := NewFrameImage(Resolution{Width: 200, Height: 60}, background)

		// when
		screenOverlay{image: first, face: face}.drawText("12.3 km", 5, 5, 20, RGB{R: 0xFF, G: 0xFF, B: 0xFF})
		screenOverlay{image: second, face: face}.drawText("12.3 km", 5, 5, 20, RGB{R: 0xFF, G: 0xFF, B: 0xFF})

		// then
		assert.Equal(t, first.Pix, second.Pix)
	})

	t.Run("should not draw anything, and not panic, for a rune outside the font", func(t *testing.T) {
		// given
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		img := NewFrameImage(Resolution{Width: 40, Height: 40}, background)
		overlay := screenOverlay{image: img, face: face}
		untouched := NewFrameImage(Resolution{Width: 40, Height: 40}, background)

		// when
		overlay.drawText("\u0001", 0, 0, 20, RGB{R: 0xFF, G: 0xFF, B: 0xFF})

		// then
		assert.Equal(t, untouched.Pix, img.Pix)
	})

	t.Run("should draw an outline around the glyph even when the background already matches the text color", func(t *testing.T) {
		// given: a background of exactly the text color, so the fill pass
		// alone would leave every pixel it touches unchanged — the only way
		// a pixel can differ afterwards is the outline pass (FR-004,
		// research.md item 6), drawn in OverlayTextOutlineColor before the
		// fill, independent of what is underneath
		color := OverlayTextColor
		img := NewFrameImage(Resolution{Width: 60, Height: 60}, color)
		overlay := screenOverlay{image: img, face: face}
		mask, ok := face.glyph('0', 20)
		require.True(t, ok)

		topInk, inkX := -1, -1
		for gy := 0; gy < mask.height && topInk < 0; gy++ {
			for gx := 0; gx < mask.width; gx++ {
				if mask.coverage[gy*mask.width+gx] > 0 {
					topInk, inkX = gy, gx
					break
				}
			}
		}
		require.GreaterOrEqual(t, topInk, 1, "expected a zero-coverage row above the ink to use as the halo target")

		// when
		overlay.drawText("0", 5, 5, 20, color)

		// then: the row directly above the first inked row, which the
		// glyph's own mask never covers, still changed — the outline halo
		assert.NotEqual(t, color, img.At(5+inkX, 5+topInk-1))
	})

	t.Run("should never extend the outline beyond the documented radius from the glyph's own ink", func(t *testing.T) {
		// given: at this image height, the outline radius rounds down to
		// its floor, OverlayOutlineMinWidth (1 px) — two pixels above the
		// first inked row is outside that radius from it
		color := OverlayTextColor
		img := NewFrameImage(Resolution{Width: 60, Height: 60}, color)
		overlay := screenOverlay{image: img, face: face}
		mask, ok := face.glyph('0', 20)
		require.True(t, ok)

		topInk, inkX := -1, -1
		for gy := 0; gy < mask.height && topInk < 0; gy++ {
			for gx := 0; gx < mask.width; gx++ {
				if mask.coverage[gy*mask.width+gx] > 0 {
					topInk, inkX = gy, gx
					break
				}
			}
		}
		require.GreaterOrEqual(t, topInk, 2, "expected at least two zero-coverage rows above the ink")

		// when
		overlay.drawText("0", 5, 5, 20, color)

		// then
		assert.Equal(t, color, img.At(5+inkX, 5+topInk-2))
	})
}

func Test_DrawBlock(t *testing.T) {
	face := newVectorFace()

	t.Run("should draw up to three lines horizontally centered on centerX", func(t *testing.T) {
		// given: three lines of different width, all narrower than the
		// canvas, so each one's own centering can be checked independently
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		img := NewFrameImage(Resolution{Width: 400, Height: 200}, background)
		overlay := screenOverlay{image: img, face: face}
		text := overlayBlockText{label: "Distância", value: "12.3", unit: "km"}
		centerX, labelPpem, valuePpem := 200, 16, 32

		// when
		overlay.drawBlock(centerX, 10, labelPpem, valuePpem, text)

		// then: the leftmost and rightmost ink columns of each line sit
		// symmetrically around centerX
		inkBounds := func(y0, y1 int) (left, right int, found bool) {
			left, right = img.Resolution.Width, -1
			for y := y0; y < y1; y++ {
				for x := 0; x < img.Resolution.Width; x++ {
					if img.At(x, y) != background {
						found = true
						left = min(left, x)
						right = max(right, x)
					}
				}
			}
			return
		}

		labelPad := roundHalfUp(overlayLinePadding * float64(face.lineHeight(labelPpem)))
		valuePad := roundHalfUp(overlayLinePadding * float64(face.lineHeight(valuePpem)))
		labelY0 := 10
		labelY1 := labelY0 + face.lineHeight(labelPpem) + labelPad
		valueY1 := labelY1 + face.lineHeight(valuePpem) + valuePad

		labelLeft, labelRight, ok := inkBounds(labelY0, labelY1)
		require.True(t, ok, "expected ink in the label's own rows")
		assert.InDelta(t, centerX, (labelLeft+labelRight)/2, 1)

		valueLeft, valueRight, ok := inkBounds(labelY1, valueY1)
		require.True(t, ok, "expected ink in the value's own rows")
		assert.InDelta(t, centerX, (valueLeft+valueRight)/2, 1)
	})

	t.Run("should draw only two lines, with no gap, when unit is empty", func(t *testing.T) {
		// given: the same label/value as above, once with a unit and once
		// without — the version without must be strictly shorter (no third
		// line, no blank space reserved for it)
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		withUnit := NewFrameImage(Resolution{Width: 400, Height: 200}, background)
		withoutUnit := NewFrameImage(Resolution{Width: 400, Height: 200}, background)
		labelPpem, valuePpem := 16, 32

		// when
		screenOverlay{image: withUnit, face: face}.drawBlock(200, 10, labelPpem, valuePpem, overlayBlockText{label: "Tempo decorrido", value: "0:05:03", unit: "x"})
		screenOverlay{image: withoutUnit, face: face}.drawBlock(200, 10, labelPpem, valuePpem, overlayBlockText{label: "Tempo decorrido", value: "0:05:03", unit: ""})

		// then: the bottommost row with any ink is higher up without a unit
		bottomInk := func(img FrameImage) int {
			bottom := -1
			for y := 0; y < img.Resolution.Height; y++ {
				for x := 0; x < img.Resolution.Width; x++ {
					if img.At(x, y) != background {
						bottom = y
					}
				}
			}
			return bottom
		}
		assert.Less(t, bottomInk(withoutUnit), bottomInk(withUnit))
	})
}

func Test_FormatOverlayDistance(t *testing.T) {
	t.Run("should show whole meters below 1 km", func(t *testing.T) {
		value, unit := formatOverlayDistance(850)
		assert.Equal(t, "850", value)
		assert.Equal(t, "m", unit)

		value, unit = formatOverlayDistance(0)
		assert.Equal(t, "0", value)
		assert.Equal(t, "m", unit)

		value, unit = formatOverlayDistance(999.4)
		assert.Equal(t, "999", value)
		assert.Equal(t, "m", unit)
	})

	t.Run("should show kilometers to one decimal from 1 km", func(t *testing.T) {
		value, unit := formatOverlayDistance(1000)
		assert.Equal(t, "1.0", value)
		assert.Equal(t, "km", unit)

		value, unit = formatOverlayDistance(12345)
		assert.Equal(t, "12.3", value)
		assert.Equal(t, "km", unit)
	})
}

func Test_FormatOverlayElevation(t *testing.T) {
	t.Run("should show whole meters, unsigned", func(t *testing.T) {
		value, unit := formatOverlayElevation(1234.4)
		assert.Equal(t, "1234", value)
		assert.Equal(t, "m", unit)

		value, unit = formatOverlayElevation(0)
		assert.Equal(t, "0", value)
		assert.Equal(t, "m", unit)
	})
}

func Test_FormatOverlayGain(t *testing.T) {
	t.Run("should show whole meters, signed with a leading plus", func(t *testing.T) {
		value, unit := formatOverlayGain(567.4)
		assert.Equal(t, "+567", value)
		assert.Equal(t, "m", unit)

		value, unit = formatOverlayGain(0)
		assert.Equal(t, "+0", value)
		assert.Equal(t, "m", unit)
	})
}

func Test_FormatOverlayElapsed(t *testing.T) {
	t.Run("should always show H:MM:SS, never omitting the hour", func(t *testing.T) {
		assert.Equal(t, "0:05:03", formatOverlayElapsed(5*time.Minute+3*time.Second))
		assert.Equal(t, "1:00:00", formatOverlayElapsed(time.Hour))
		assert.Equal(t, "0:00:00", formatOverlayElapsed(0))
	})
}

func Test_FormatOverlaySpeed(t *testing.T) {
	t.Run("should show km/h to one decimal, converted from meters per second", func(t *testing.T) {
		value, unit := formatOverlaySpeed(10.0 / 3.6)
		assert.Equal(t, "10.0", value)
		assert.Equal(t, "km/h", unit)

		value, unit = formatOverlaySpeed(0)
		assert.Equal(t, "0.0", value)
		assert.Equal(t, "km/h", unit)
	})
}

func Test_ProfileMarkerRadius(t *testing.T) {
	t.Run("should be the ratio of the frame's height when that is above the floor", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, roundHalfUp(ProfileMarkerRadiusRatio*2000), profileMarkerRadius(2000))
	})

	t.Run("should be the documented floor when the ratio would round below it", func(t *testing.T) {
		// given: at height 100, ProfileMarkerRadiusRatio*height is 1.2,
		// well under the floor
		// when / then
		assert.Equal(t, int(ProfileMarkerMinRadius), profileMarkerRadius(100))
	})

	t.Run("should scale proportionally between two heights, both above the floor", func(t *testing.T) {
		// given / when
		small := profileMarkerRadius(1000)
		large := profileMarkerRadius(2000)

		// then
		assert.Equal(t, 2*small, large)
	})
}

func Test_OutlineRadius(t *testing.T) {
	t.Run("should be the ratio of ppem when that is above the floor", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, roundHalfUp(OverlayOutlineRatio*200), outlineRadius(200))
	})

	t.Run("should be the documented floor when the ratio would round below it", func(t *testing.T) {
		// given: at ppem 10, OverlayOutlineRatio*ppem is 0.35, well under
		// the floor
		// when / then
		assert.Equal(t, int(OverlayOutlineMinWidth), outlineRadius(10))
	})

	t.Run("should scale proportionally between two ppem values, both above the floor", func(t *testing.T) {
		// given / when
		small := outlineRadius(200)
		large := outlineRadius(400)

		// then
		assert.Equal(t, 2*small, large)
	})
}

func Test_BlockText(t *testing.T) {
	frame := CameraFrame{
		MarkerDistance:     8600,
		TrackElevation:     80,
		TrackElevationGain: 6,
		ActivityElapsed:    28*time.Minute + 44*time.Second,
		MarkerSpeed:        5.5,
	}

	t.Run("should write the distance block with its full Portuguese label", func(t *testing.T) {
		// given / when
		text := distanceBlockText(frame)

		// then
		assert.Equal(t, "Distância", text.label)
		value, unit := formatOverlayDistance(frame.MarkerDistance)
		assert.Equal(t, value, text.value)
		assert.Equal(t, unit, text.unit)
	})

	t.Run("should write the elevation block with only the altitude, no gain", func(t *testing.T) {
		// given / when
		text := elevationBlockText(frame)

		// then
		assert.Equal(t, "Elevação", text.label)
		value, unit := formatOverlayElevation(frame.TrackElevation)
		assert.Equal(t, value, text.value)
		assert.Equal(t, unit, text.unit)
		assert.NotContains(t, text.label, "anho")
	})

	t.Run("should write the gain block on its own", func(t *testing.T) {
		// given / when
		text := gainBlockText(frame)

		// then
		assert.Equal(t, "Ganho", text.label)
		value, unit := formatOverlayGain(frame.TrackElevationGain)
		assert.Equal(t, value, text.value)
		assert.Equal(t, unit, text.unit)
	})

	t.Run("should write the time block without a unit", func(t *testing.T) {
		// given / when
		text := timeBlockText(frame)

		// then
		assert.Equal(t, "Tempo decorrido", text.label)
		assert.Equal(t, formatOverlayElapsed(frame.ActivityElapsed), text.value)
		assert.Equal(t, "", text.unit)
	})

	t.Run("should write the speed block", func(t *testing.T) {
		// given / when
		text := speedBlockText(frame)

		// then
		assert.Equal(t, "Velocidade", text.label)
		value, unit := formatOverlaySpeed(frame.MarkerSpeed)
		assert.Equal(t, value, text.value)
		assert.Equal(t, unit, text.unit)
	})

	t.Run("should never write the old abbreviated, upper-case labels", func(t *testing.T) {
		// given / when
		labels := []string{
			distanceBlockText(frame).label,
			elevationBlockText(frame).label,
			gainBlockText(frame).label,
			timeBlockText(frame).label,
			speedBlockText(frame).label,
		}

		// then
		for _, label := range labels {
			assert.NotContains(t, label, "DIST")
			assert.NotContains(t, label, "ELEV")
			assert.NotContains(t, label, "GANHO")
			assert.NotContains(t, label, "TEMPO")
			assert.NotContains(t, label, "VEL")
		}
	})
}

func Test_ScreenOverlay_Draw(t *testing.T) {
	face := newVectorFace()

	blank := func() FrameImage {
		return NewFrameImage(Resolution{Width: 360, Height: 640}, RGB{R: 0x20, G: 0x26, B: 0x2E})
	}

	frames := func() []CameraFrame {
		return []CameraFrame{
			{Index: 0, Phase: PhaseFollowing, MarkerDistance: 0, ActivityElapsed: 0, TrackElevation: 100, TrackElevationGain: 0},
			{Index: 1, Phase: PhaseFollowing, MarkerDistance: 50, ActivityElapsed: 10 * time.Second, TrackElevation: 120, TrackElevationGain: 20},
		}
	}
	planWith := func(elevationAvailable bool, reference TimeReference) CameraPlan {
		return NewCameraPlan(PlanParameters{FrameRate: 30}, DurationModeExplicit, reference, "", frames(), nil, elevationAvailable)
	}

	t.Run("should draw nothing when disabled", func(t *testing.T) {
		// given
		full, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockElevation, OverlayBlockTime, OverlayBlockProfile})
		disabled, _ := NewOverlayConfig(false, nil)
		plan := planWith(true, TimeReferenceClock)

		off, on := blank(), blank()

		// when
		screenOverlay{image: off, config: disabled, face: face}.draw(plan, 1)
		screenOverlay{image: on, config: full, face: face}.draw(plan, 1)

		// then
		assert.Equal(t, blank().Pix, off.Pix)
		assert.NotEqual(t, blank().Pix, on.Pix)
	})

	t.Run("should draw only the requested blocks", func(t *testing.T) {
		// given
		distanceOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance})
		distanceAndTime, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockTime})
		plan := planWith(true, TimeReferenceClock)

		a, b := blank(), blank()

		// when
		screenOverlay{image: a, config: distanceOnly, face: face}.draw(plan, 1)
		screenOverlay{image: b, config: distanceAndTime, face: face}.draw(plan, 1)

		// then: adding the time block changes the image further
		assert.NotEqual(t, a.Pix, b.Pix)
	})

	t.Run("should draw blocks side by side, in columns, not stacked", func(t *testing.T) {
		// given: two blocks, one on each side of the fixed order
		// (distance, then time) — if they were still stacked vertically,
		// the whole image would be identical to the single-block one,
		// distanceOnly, except for extra rows below it; side by side, the
		// second column (to the right of the midline) must already differ
		// at the very first row of text, where distanceOnly draws nothing
		distanceOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance})
		distanceAndTime, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockTime})
		plan := planWith(true, TimeReferenceClock)

		a, b := blank(), blank()

		// when
		screenOverlay{image: a, config: distanceOnly, face: face}.draw(plan, 1)
		screenOverlay{image: b, config: distanceAndTime, face: face}.draw(plan, 1)

		// then: the right half of the top rows differs between the two —
		// proof there is drawing there in b that a (stacked, it would still
		// be empty on the right) does not have
		height := a.Resolution.Height
		marginTop := roundHalfUp(OverlayTopMarginRatio * float64(height))
		rightHalfDiffers := false
		for y := marginTop; y < marginTop+40 && !rightHalfDiffers; y++ {
			for x := a.Resolution.Width / 2; x < a.Resolution.Width; x++ {
				if a.At(x, y) != b.At(x, y) {
					rightHalfDiffers = true
					break
				}
			}
		}
		assert.True(t, rightHalfDiffers, "expected the second column (time) to draw something the single-column image does not have")
	})

	t.Run("should draw the same pixels whatever order the blocks were named in NewOverlayConfig", func(t *testing.T) {
		// given: screenOverlay.draw never reads the order blocks arrived
		// in — only OverlayConfig's bools, in the fixed overlayBlockOrder
		// — so two configs built from the same set, named in different
		// orders, must draw identically
		forward, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockSpeed, OverlayBlockTime})
		backward, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockTime, OverlayBlockSpeed, OverlayBlockDistance})
		plan := planWith(true, TimeReferenceClock)

		a, b := blank(), blank()

		// when
		screenOverlay{image: a, config: forward, face: face}.draw(plan, 1)
		screenOverlay{image: b, config: backward, face: face}.draw(plan, 1)

		// then
		assert.Equal(t, a.Pix, b.Pix)
	})

	t.Run("should not leave a gap for a block that is off, in the middle of the fixed order", func(t *testing.T) {
		// given: overlayBlockOrder is speed, elevation, distance, gain,
		// time — turning elevation off (in the middle) must not leave an
		// empty column between speed and distance: the two must land in
		// exactly the same place as when there were only two blocks to
		// begin with
		twoBlocks, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockSpeed, OverlayBlockDistance})
		withGapInTheMiddle, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockSpeed, OverlayBlockElevation, OverlayBlockDistance})
		plan := planWith(false, TimeReferenceClock) // no elevation available: elevation never shows either way

		a, b := blank(), blank()

		// when
		screenOverlay{image: a, config: twoBlocks, face: face}.draw(plan, 1)
		screenOverlay{image: b, config: withGapInTheMiddle, face: face}.draw(plan, 1)

		// then
		assert.Equal(t, a.Pix, b.Pix)
	})

	t.Run("should draw nothing for the time block when the plan has no clock reference", func(t *testing.T) {
		// given
		timeOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockTime})
		distancePlan := planWith(true, TimeReferenceDistance)

		withoutClock, baseline := blank(), blank()
		noBlocks, _ := NewOverlayConfig(true, nil)

		// when
		screenOverlay{image: withoutClock, config: timeOnly, face: face}.draw(distancePlan, 1)
		screenOverlay{image: baseline, config: noBlocks, face: face}.draw(distancePlan, 1)

		// then
		assert.Equal(t, baseline.Pix, withoutClock.Pix)
	})

	t.Run("should draw something for the time block when the plan does have a clock reference", func(t *testing.T) {
		// given
		timeOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockTime})
		clockPlan := planWith(true, TimeReferenceClock)

		withClock, baseline := blank(), blank()
		noBlocks, _ := NewOverlayConfig(true, nil)

		// when
		screenOverlay{image: withClock, config: timeOnly, face: face}.draw(clockPlan, 1)
		screenOverlay{image: baseline, config: noBlocks, face: face}.draw(clockPlan, 1)

		// then
		assert.NotEqual(t, baseline.Pix, withClock.Pix)
	})

	t.Run("should draw nothing for the speed block when the plan has no clock reference", func(t *testing.T) {
		// given
		speedOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockSpeed})
		distancePlan := planWith(true, TimeReferenceDistance)

		withoutClock, baseline := blank(), blank()
		noBlocks, _ := NewOverlayConfig(true, nil)

		// when
		screenOverlay{image: withoutClock, config: speedOnly, face: face}.draw(distancePlan, 1)
		screenOverlay{image: baseline, config: noBlocks, face: face}.draw(distancePlan, 1)

		// then
		assert.Equal(t, baseline.Pix, withoutClock.Pix)
	})

	t.Run("should draw the speed block when requested and the plan has a clock reference", func(t *testing.T) {
		// given
		speedOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockSpeed})
		clockPlan := planWith(true, TimeReferenceClock)

		withSpeed, baseline := blank(), blank()
		noBlocks, _ := NewOverlayConfig(true, nil)

		// when
		screenOverlay{image: withSpeed, config: speedOnly, face: face}.draw(clockPlan, 1)
		screenOverlay{image: baseline, config: noBlocks, face: face}.draw(clockPlan, 1)

		// then
		assert.NotEqual(t, baseline.Pix, withSpeed.Pix)
	})

	t.Run("should draw the elevation block without gain, and the gain block without elevation, when each is requested alone", func(t *testing.T) {
		// given
		elevationOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockElevation})
		gainOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockGain})
		plan := planWith(true, TimeReferenceClock)

		elevationImg, gainImg := blank(), blank()

		// when
		screenOverlay{image: elevationImg, config: elevationOnly, face: face}.draw(plan, 1)
		screenOverlay{image: gainImg, config: gainOnly, face: face}.draw(plan, 1)

		// then: each draws something, and they draw different things
		assert.NotEqual(t, blank().Pix, elevationImg.Pix)
		assert.NotEqual(t, blank().Pix, gainImg.Pix)
		assert.NotEqual(t, elevationImg.Pix, gainImg.Pix)
	})

	t.Run("should draw both as distinct blocks when elevation and gain are requested together", func(t *testing.T) {
		// given
		both, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockElevation, OverlayBlockGain})
		elevationOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockElevation})
		plan := planWith(true, TimeReferenceClock)

		bothImg, elevationImg := blank(), blank()

		// when
		screenOverlay{image: bothImg, config: both, face: face}.draw(plan, 1)
		screenOverlay{image: elevationImg, config: elevationOnly, face: face}.draw(plan, 1)

		// then: requesting gain too draws more than elevation alone
		assert.NotEqual(t, bothImg.Pix, elevationImg.Pix)
	})

	t.Run("should draw nothing for the elevation, gain and profile blocks when the plan has no elevation", func(t *testing.T) {
		// given
		elevationGainAndProfile, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockElevation, OverlayBlockGain, OverlayBlockProfile})
		noBlocks, _ := NewOverlayConfig(true, nil)
		plan := planWith(false, TimeReferenceClock)

		requested, baseline := blank(), blank()

		// when
		screenOverlay{image: requested, config: elevationGainAndProfile, face: face}.draw(plan, 1)
		screenOverlay{image: baseline, config: noBlocks, face: face}.draw(plan, 1)

		// then
		assert.Equal(t, baseline.Pix, requested.Pix)
	})

	t.Run("should draw the same plan and frame twice into byte-identical images", func(t *testing.T) {
		// given
		full, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockElevation, OverlayBlockGain, OverlayBlockTime, OverlayBlockSpeed, OverlayBlockProfile})
		plan := planWith(true, TimeReferenceClock)

		first, second := blank(), blank()

		// when
		screenOverlay{image: first, config: full, face: face}.draw(plan, 1)
		screenOverlay{image: second, config: full, face: face}.draw(plan, 1)

		// then
		assert.Equal(t, first.Pix, second.Pix)
	})

	t.Run("should move the profile's dot for a different frame while the line stays the same", func(t *testing.T) {
		// given
		profileOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockProfile})
		plan := planWith(true, TimeReferenceClock)

		firstFrame, secondFrame := blank(), blank()

		// when
		screenOverlay{image: firstFrame, config: profileOnly, face: face}.draw(plan, 0)
		screenOverlay{image: secondFrame, config: profileOnly, face: face}.draw(plan, 1)

		// then
		assert.NotEqual(t, firstFrame.Pix, secondFrame.Pix)
	})

	t.Run("should draw the profile's line and marker with an outline instead of a panel", func(t *testing.T) {
		// given: the exact blend the old panel painted — background mixed
		// with RGB{0,0,0} at 0.55 opacity (011-overlay-polish/
		// 009-frame-overlays) — hardcoded here, not read from a live
		// constant, because this etapa removes OverlayPanelColor/
		// OverlayPanelOpacity entirely: nothing should paint this color
		// anymore, anywhere in the frame
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		oldPanelBlend := func(bg uint8) uint8 { return rounded(float64(bg) * (1 - 0.55)) }
		oldPanelColor := RGB{oldPanelBlend(background.R), oldPanelBlend(background.G), oldPanelBlend(background.B)}

		profileOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockProfile})
		plan := planWith(true, TimeReferenceClock)
		img := blank()

		// when
		screenOverlay{image: img, config: profileOnly, face: face}.draw(plan, 1)

		// then: no pixel in the whole image is the old panel's blended
		// color — the panel is gone — yet the block still draws something
		// (the line/marker's outline and core)
		foundOldPanelColor, foundOutline := false, false
		for y := 0; y < img.Resolution.Height; y++ {
			for x := 0; x < img.Resolution.Width; x++ {
				switch img.At(x, y) {
				case oldPanelColor:
					foundOldPanelColor = true
				case OverlayTextOutlineColor:
					foundOutline = true
				}
			}
		}
		assert.False(t, foundOldPanelColor, "expected no pixel blended as the old panel")
		assert.True(t, foundOutline, "expected the line/marker's outline color somewhere in the profile area")
		assert.NotEqual(t, blank().Pix, img.Pix)
	})

	t.Run("should draw a visibly larger profile marker at a larger resolution of the same aspect ratio", func(t *testing.T) {
		// given: the dot is drawn at full opacity in appearance.MarkerColor
		// (the zero value, RGB{0,0,0}, since appearance is not set here),
		// never blended with the line (OverlayTextColor, white) — so
		// counting exactly black pixels counts the dot's own
		// area, nothing else
		profileOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockProfile})
		plan := planWith(true, TimeReferenceClock)
		countBlack := func(img FrameImage) int {
			count := 0
			for y := 0; y < img.Resolution.Height; y++ {
				for x := 0; x < img.Resolution.Width; x++ {
					if img.At(x, y) == (RGB{}) {
						count++
					}
				}
			}
			return count
		}

		small := NewFrameImage(Resolution{Width: 360, Height: 640}, RGB{R: 0x20, G: 0x26, B: 0x2E})
		large := NewFrameImage(Resolution{Width: 1080, Height: 1920}, RGB{R: 0x20, G: 0x26, B: 0x2E})

		// when
		screenOverlay{image: small, config: profileOnly, face: face}.draw(plan, 1)
		screenOverlay{image: large, config: profileOnly, face: face}.draw(plan, 1)

		// then
		assert.Greater(t, countBlack(large), countBlack(small))
	})

	t.Run("should draw nothing above the top margin or beside the side margins", func(t *testing.T) {
		// given: a single column spanning the whole usable width — its text
		// is centered within it (FR-003), so it may not touch the margins
		// exactly, but it must never draw past them
		distanceOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance})
		plan := planWith(true, TimeReferenceClock)
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		img := NewFrameImage(Resolution{Width: 1080, Height: 1920}, background)

		// when
		screenOverlay{image: img, config: distanceOnly, face: face}.draw(plan, 0)

		// then
		marginTop := roundHalfUp(OverlayTopMarginRatio * 1920.0)
		marginSide := roundHalfUp(OverlaySideMarginRatio * 1080.0)

		for x := 0; x < 1080; x += 37 {
			assert.Equal(t, background, img.At(x, marginTop-1), "row above the top margin, x=%d", x)
		}
		labelPpem := overlayLabelPpem(1920)
		for y := marginTop; y < marginTop+face.lineHeight(labelPpem); y++ {
			assert.Equal(t, background, img.At(marginSide-1, y), "column left of the side margin, y=%d", y)
			assert.Equal(t, background, img.At(1080-marginSide, y), "column right of the side margin, y=%d", y)
		}
	})

	t.Run("should draw nothing at or below the bottom margin, which is larger than the top and side margins", func(t *testing.T) {
		// given
		full, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockElevation, OverlayBlockTime, OverlayBlockProfile})
		plan := planWith(true, TimeReferenceClock)
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		img := NewFrameImage(Resolution{Width: 1080, Height: 1920}, background)

		// when: frame 0, whose profile dot sits at the chart's own left
		// edge (distance 0), far from the column checked below
		screenOverlay{image: img, config: full, face: face}.draw(plan, 0)

		// then: the profile block, the one closest to the bottom, never
		// reaches the bottom margin itself
		marginSide := roundHalfUp(OverlaySideMarginRatio * 1080.0)
		marginBottom := roundHalfUp(OverlayBottomMarginRatio * 1920.0)
		y1 := 1920 - marginBottom

		assert.Equal(t, background, img.At(1080-marginSide-5, y1))
		assert.Greater(t, OverlayBottomMarginRatio, OverlayTopMarginRatio)
		assert.Greater(t, OverlayBottomMarginRatio, OverlaySideMarginRatio)
	})
}
