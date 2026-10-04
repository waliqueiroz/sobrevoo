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

func Test_DrawPanel(t *testing.T) {
	t.Run("should blend the panel color at exactly OverlayPanelOpacity over what was there", func(t *testing.T) {
		// given
		background := RGB{R: 0xFF, G: 0xFF, B: 0xFF}
		img := NewFrameImage(Resolution{Width: 20, Height: 20}, background)

		// when
		drawPanel(img, 5, 5, 15, 15)

		// then: white * (1 - opacity) + panel * opacity, rounded, for each channel
		expect := func(bg, panel uint8) uint8 {
			return rounded(float64(bg)*(1-OverlayPanelOpacity) + float64(panel)*OverlayPanelOpacity)
		}
		got := img.At(10, 10)
		assert.Equal(t, expect(background.R, OverlayPanelColor.R), got.R)
		assert.Equal(t, expect(background.G, OverlayPanelColor.G), got.G)
		assert.Equal(t, expect(background.B, OverlayPanelColor.B), got.B)
	})

	t.Run("should leave pixels outside the rectangle untouched", func(t *testing.T) {
		// given
		background := RGB{R: 0xFF, G: 0xFF, B: 0xFF}
		img := NewFrameImage(Resolution{Width: 20, Height: 20}, background)

		// when
		drawPanel(img, 5, 5, 15, 15)

		// then
		assert.Equal(t, background, img.At(0, 0))
		assert.Equal(t, background, img.At(19, 19))
	})
}

func Test_FormatOverlayDistance(t *testing.T) {
	t.Run("should show whole meters below 1 km", func(t *testing.T) {
		assert.Equal(t, "850 m", formatOverlayDistance(850))
		assert.Equal(t, "0 m", formatOverlayDistance(0))
		assert.Equal(t, "999 m", formatOverlayDistance(999.4))
	})

	t.Run("should show kilometers to one decimal from 1 km", func(t *testing.T) {
		assert.Equal(t, "1.0 km", formatOverlayDistance(1000))
		assert.Equal(t, "12.3 km", formatOverlayDistance(12345))
	})
}

func Test_FormatOverlayElevation(t *testing.T) {
	t.Run("should show whole meters, unsigned", func(t *testing.T) {
		assert.Equal(t, "1234 m", formatOverlayElevation(1234.4))
		assert.Equal(t, "0 m", formatOverlayElevation(0))
	})
}

func Test_FormatOverlayGain(t *testing.T) {
	t.Run("should show whole meters, signed with a leading plus", func(t *testing.T) {
		assert.Equal(t, "+567 m", formatOverlayGain(567.4))
		assert.Equal(t, "+0 m", formatOverlayGain(0))
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
		assert.Equal(t, "10.0 km/h", formatOverlaySpeed(10.0/3.6))
		assert.Equal(t, "0.0 km/h", formatOverlaySpeed(0))
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

	t.Run("should write the distance label in Portuguese", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, "DIST "+formatOverlayDistance(frame.MarkerDistance), distanceBlockText(frame))
	})

	t.Run("should write the elevation and gain labels in Portuguese", func(t *testing.T) {
		// given
		want := "ELEV " + formatOverlayElevation(frame.TrackElevation) + "   GANHO " + formatOverlayGain(frame.TrackElevationGain)

		// when / then
		assert.Equal(t, want, elevationBlockText(frame))
	})

	t.Run("should write the time label in Portuguese", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, "TEMPO "+formatOverlayElapsed(frame.ActivityElapsed), timeBlockText(frame))
	})

	t.Run("should write the speed label in Portuguese", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, "VEL "+formatOverlaySpeed(frame.MarkerSpeed), speedBlockText(frame))
	})

	t.Run("should never write the old English labels", func(t *testing.T) {
		// given / when
		texts := []string{distanceBlockText(frame), elevationBlockText(frame), timeBlockText(frame)}

		// then
		for _, text := range texts {
			assert.NotContains(t, text, "GAIN")
			assert.NotContains(t, text, "TIME")
		}
	})
}

func Test_StablePanelWidth(t *testing.T) {
	face := newVectorFace()
	const ppem = 20

	short := CameraFrame{Index: 0, Phase: PhaseFollowing, MarkerDistance: 0, TrackElevation: 80, TrackElevationGain: 6, ActivityElapsed: 0}
	long := CameraFrame{Index: 1, Phase: PhaseFollowing, MarkerDistance: 123456, TrackElevation: 9999, TrackElevationGain: 88888, ActivityElapsed: 5*time.Hour + 3*time.Minute + 2*time.Second}

	t.Run("should be the width of the widest text found in any frame of the plan, not just one", func(t *testing.T) {
		// given: the long frame's elevation+gain text is the widest of the
		// six (three blocks x two frames) — the function must find it even
		// though it is never the frame asked about by itself (it has no
		// notion of "the frame asked about": it always looks at every frame)
		full, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockElevation, OverlayBlockTime})
		plan := NewCameraPlan(PlanParameters{FrameRate: 30}, DurationModeExplicit, TimeReferenceClock, "", []CameraFrame{short, long}, nil, true)

		// when
		width := stablePanelWidth(face, plan, full, ppem)

		// then
		assert.Equal(t, face.textWidth(elevationBlockText(long), ppem), width)
	})

	t.Run("should be zero when no numeric block is shown", func(t *testing.T) {
		// given
		noBlocks, _ := NewOverlayConfig(true, nil)
		plan := NewCameraPlan(PlanParameters{FrameRate: 30}, DurationModeExplicit, TimeReferenceClock, "", []CameraFrame{short, long}, nil, true)

		// when
		width := stablePanelWidth(face, plan, noBlocks, ppem)

		// then
		assert.Equal(t, 0, width)
	})

	t.Run("should ignore a block the plan has no data for, even if config requests it", func(t *testing.T) {
		// given: no clock reference and no elevation — only distance is
		// actually shown, whatever config asks for
		full, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockElevation, OverlayBlockTime})
		plan := NewCameraPlan(PlanParameters{FrameRate: 30}, DurationModeExplicit, TimeReferenceDistance, "", []CameraFrame{short, long}, nil, false)

		// when
		width := stablePanelWidth(face, plan, full, ppem)

		// then
		assert.Equal(t, face.textWidth(distanceBlockText(long), ppem), width)
	})

	t.Run("should include the speed block among the widest candidates when requested", func(t *testing.T) {
		// given
		speedOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockSpeed})
		longSpeed := CameraFrame{Index: 2, Phase: PhaseFollowing, MarkerSpeed: 123.4}
		plan := NewCameraPlan(PlanParameters{FrameRate: 30}, DurationModeExplicit, TimeReferenceClock, "", []CameraFrame{short, long, longSpeed}, nil, true)

		// when
		width := stablePanelWidth(face, plan, speedOnly, ppem)

		// then
		assert.Equal(t, face.textWidth(speedBlockText(longSpeed), ppem), width)
	})

	t.Run("should ignore the speed block when the plan has no clock reference, even if requested", func(t *testing.T) {
		// given
		speedOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockSpeed})
		plan := NewCameraPlan(PlanParameters{FrameRate: 30}, DurationModeExplicit, TimeReferenceDistance, "", []CameraFrame{short, long}, nil, true)

		// when
		width := stablePanelWidth(face, plan, speedOnly, ppem)

		// then
		assert.Equal(t, 0, width)
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
	// panelWidthFor mirrors what Scene.Render now does before constructing
	// screenOverlay (frame_scene.go): compute the stable width once, from
	// the whole plan, and pass it in — draw itself no longer computes it.
	panelWidthFor := func(plan CameraPlan, config OverlayConfig, height int) int {
		return stablePanelWidth(face, plan, config, overlayPpem(height))
	}

	t.Run("should draw nothing when disabled", func(t *testing.T) {
		// given
		full, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockElevation, OverlayBlockTime, OverlayBlockProfile})
		disabled, _ := NewOverlayConfig(false, nil)
		plan := planWith(true, TimeReferenceClock)
		panelWidth := panelWidthFor(plan, full, 640)

		off, on := blank(), blank()

		// when
		screenOverlay{image: off, config: disabled, face: face}.draw(plan, 1)
		screenOverlay{image: on, config: full, face: face, panelWidth: panelWidth}.draw(plan, 1)

		// then
		assert.Equal(t, blank().Pix, off.Pix)
		assert.NotEqual(t, blank().Pix, on.Pix)
	})

	t.Run("should draw different pixels than the old English labels would", func(t *testing.T) {
		// given
		full, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockElevation, OverlayBlockTime})
		plan := planWith(true, TimeReferenceClock)
		frame := plan.Frames[1]

		got := blank()

		// when
		screenOverlay{image: got, config: full, face: face, panelWidth: panelWidthFor(plan, full, 640)}.draw(plan, 1)

		// then: build, by hand, through the same low-level primitives draw
		// uses, what the old English labels would have drawn at the same
		// layout — the real output must no longer match it
		old := blank()
		oldOverlay := screenOverlay{image: old, face: face}
		height, width := old.Resolution.Height, old.Resolution.Width
		marginTop := roundHalfUp(OverlayTopMarginRatio * float64(height))
		marginSide := roundHalfUp(OverlaySideMarginRatio * float64(width))
		ppem := overlayPpem(height)
		pad := roundHalfUp(overlayLinePadding * float64(face.lineHeight(ppem)))
		lineHeight := face.lineHeight(ppem) + 2*pad

		distanceOld := "DIST " + formatOverlayDistance(frame.MarkerDistance)
		elevationOld := "ELEV " + formatOverlayElevation(frame.TrackElevation) + "   GAIN " + formatOverlayGain(frame.TrackElevationGain)
		timeOld := "TIME " + formatOverlayElapsed(frame.ActivityElapsed)
		panelWidth := max(face.textWidth(distanceOld, ppem), face.textWidth(elevationOld, ppem), face.textWidth(timeOld, ppem))

		y := marginTop
		oldOverlay.drawLine(marginSide, y, ppem, pad, panelWidth, distanceOld)
		y += lineHeight + pad
		oldOverlay.drawLine(marginSide, y, ppem, pad, panelWidth, elevationOld)
		y += lineHeight + pad
		oldOverlay.drawLine(marginSide, y, ppem, pad, panelWidth, timeOld)

		assert.NotEqual(t, old.Pix, got.Pix)
	})

	t.Run("should draw only the requested blocks", func(t *testing.T) {
		// given
		distanceOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance})
		distanceAndTime, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockTime})
		plan := planWith(true, TimeReferenceClock)

		a, b := blank(), blank()

		// when
		screenOverlay{image: a, config: distanceOnly, face: face, panelWidth: panelWidthFor(plan, distanceOnly, 640)}.draw(plan, 1)
		screenOverlay{image: b, config: distanceAndTime, face: face, panelWidth: panelWidthFor(plan, distanceAndTime, 640)}.draw(plan, 1)

		// then: adding the time block changes the image further
		assert.NotEqual(t, a.Pix, b.Pix)
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
		screenOverlay{image: withClock, config: timeOnly, face: face, panelWidth: panelWidthFor(clockPlan, timeOnly, 640)}.draw(clockPlan, 1)
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
		screenOverlay{image: withSpeed, config: speedOnly, face: face, panelWidth: panelWidthFor(clockPlan, speedOnly, 640)}.draw(clockPlan, 1)
		screenOverlay{image: baseline, config: noBlocks, face: face}.draw(clockPlan, 1)

		// then
		assert.NotEqual(t, baseline.Pix, withSpeed.Pix)
	})

	t.Run("should draw nothing for the elevation and profile blocks when the plan has no elevation", func(t *testing.T) {
		// given
		elevationAndProfile, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockElevation, OverlayBlockProfile})
		noBlocks, _ := NewOverlayConfig(true, nil)
		plan := planWith(false, TimeReferenceClock)

		requested, baseline := blank(), blank()

		// when
		screenOverlay{image: requested, config: elevationAndProfile, face: face}.draw(plan, 1)
		screenOverlay{image: baseline, config: noBlocks, face: face}.draw(plan, 1)

		// then
		assert.Equal(t, baseline.Pix, requested.Pix)
	})

	t.Run("should draw the same plan and frame twice into byte-identical images", func(t *testing.T) {
		// given
		full, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockElevation, OverlayBlockTime, OverlayBlockProfile})
		plan := planWith(true, TimeReferenceClock)
		panelWidth := panelWidthFor(plan, full, 640)

		first, second := blank(), blank()

		// when
		screenOverlay{image: first, config: full, face: face, panelWidth: panelWidth}.draw(plan, 1)
		screenOverlay{image: second, config: full, face: face, panelWidth: panelWidth}.draw(plan, 1)

		// then
		assert.Equal(t, first.Pix, second.Pix)
	})

	t.Run("should give the distance panel the width the longer elevation text needs, when both are shown", func(t *testing.T) {
		// given
		full, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockElevation})
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}

		shortPlan := NewCameraPlan(PlanParameters{FrameRate: 30}, DurationModeExplicit, TimeReferenceClock, "",
			[]CameraFrame{{Index: 0, Phase: PhaseFollowing, MarkerDistance: 0, TrackElevation: 100, TrackElevationGain: 0}}, nil, true)
		longPlan := NewCameraPlan(PlanParameters{FrameRate: 30}, DurationModeExplicit, TimeReferenceClock, "",
			[]CameraFrame{{Index: 0, Phase: PhaseFollowing, MarkerDistance: 0, TrackElevation: 9999, TrackElevationGain: 88888}}, nil, true)

		shortImg := NewFrameImage(Resolution{Width: 1080, Height: 1920}, background)
		longImg := NewFrameImage(Resolution{Width: 1080, Height: 1920}, background)

		// when
		screenOverlay{image: shortImg, config: full, face: face, panelWidth: panelWidthFor(shortPlan, full, 1920)}.draw(shortPlan, 0)
		screenOverlay{image: longImg, config: full, face: face, panelWidth: panelWidthFor(longPlan, full, 1920)}.draw(longPlan, 0)

		// then: the distance panel (the first line) is measured at a row
		// just below its own top edge — inside the padding, so only the
		// panel's own background can be there, never glyph ink — and it
		// reaches further right in the plan whose elevation text is longer
		marginSide := roundHalfUp(OverlaySideMarginRatio * 1080.0)
		marginTop := roundHalfUp(OverlayTopMarginRatio * 1920.0)
		panelRight := func(img FrameImage) int {
			x := marginSide
			for img.At(x, marginTop+1) != background {
				x++
			}
			return x
		}

		assert.Less(t, panelRight(shortImg), panelRight(longImg))
	})

	t.Run("should give the distance panel only its own width when it is the only numeric block shown", func(t *testing.T) {
		// given
		distanceOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance})
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		plan := NewCameraPlan(PlanParameters{FrameRate: 30}, DurationModeExplicit, TimeReferenceClock, "",
			[]CameraFrame{{Index: 0, Phase: PhaseFollowing, MarkerDistance: 0}}, nil, true)
		img := NewFrameImage(Resolution{Width: 1080, Height: 1920}, background)

		// when
		screenOverlay{image: img, config: distanceOnly, face: face, panelWidth: panelWidthFor(plan, distanceOnly, 1920)}.draw(plan, 0)

		// then
		marginSide := roundHalfUp(OverlaySideMarginRatio * 1080.0)
		marginTop := roundHalfUp(OverlayTopMarginRatio * 1920.0)
		ppem := overlayPpem(1920)
		pad := roundHalfUp(overlayLinePadding * float64(face.lineHeight(ppem)))
		want := marginSide + face.textWidth("DIST 0 m", ppem) + 2*pad

		assert.NotEqual(t, background, img.At(want-1, marginTop+1))
		assert.Equal(t, background, img.At(want, marginTop+1))
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

	t.Run("should draw a visibly larger profile marker at a larger resolution of the same aspect ratio", func(t *testing.T) {
		// given: the dot is drawn at full opacity in appearance.MarkerColor
		// (the zero value, RGB{0,0,0}, since appearance is not set here),
		// never blended with the line (OverlayTextColor, white) or the
		// panel — so counting exactly black pixels counts the dot's own
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

	t.Run("should start the first block exactly at the top and side margins, nothing drawn before them", func(t *testing.T) {
		// given
		distanceOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance})
		plan := planWith(true, TimeReferenceClock)
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		img := NewFrameImage(Resolution{Width: 1080, Height: 1920}, background)

		// when
		screenOverlay{image: img, config: distanceOnly, face: face, panelWidth: panelWidthFor(plan, distanceOnly, 1920)}.draw(plan, 0)

		// then
		marginTop := roundHalfUp(OverlayTopMarginRatio * 1920.0)
		marginSide := roundHalfUp(OverlaySideMarginRatio * 1080.0)

		assert.Equal(t, background, img.At(marginSide, marginTop-1), "above the top margin")
		assert.Equal(t, background, img.At(marginSide-1, marginTop), "left of the side margin")
		assert.NotEqual(t, background, img.At(marginSide, marginTop), "the panel's own top-left corner")
	})

	t.Run("should draw nothing at or below the bottom margin, which is larger than the top and side margins", func(t *testing.T) {
		// given
		full, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockElevation, OverlayBlockTime, OverlayBlockProfile})
		plan := planWith(true, TimeReferenceClock)
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		img := NewFrameImage(Resolution{Width: 1080, Height: 1920}, background)

		// when: frame 0, whose profile dot sits at the chart's own left
		// edge (distance 0), far from the column checked below
		screenOverlay{image: img, config: full, face: face, panelWidth: panelWidthFor(plan, full, 1920)}.draw(plan, 0)

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
