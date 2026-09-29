package domain

import (
	"image"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/image/font/inconsolata"
)

// alphaAt reads the embedded font's own alpha mask for r at (gx, gy) within
// its glyph box, the same way drawText must — so the test does not hardcode
// the font's byte table.
func alphaAt(r rune, gx, gy int) uint8 {
	face := inconsolata.Bold8x16
	glyphHeight := face.Ascent + face.Descent
	for _, rng := range face.Ranges {
		if r >= rng.Low && r < rng.High {
			y0 := (int(r-rng.Low) + rng.Offset) * glyphHeight
			mask := face.Mask.(*image.Alpha)
			return mask.Pix[(y0+gy)*mask.Stride+gx]
		}
	}
	return 0
}

func Test_DrawText(t *testing.T) {
	face := inconsolata.Bold8x16
	glyphWidth, glyphHeight := face.Width, face.Ascent+face.Descent

	t.Run("should light up exactly the glyph's own mask, at scale 1, over a black background with white text", func(t *testing.T) {
		// given
		img := NewFrameImage(Resolution{Width: 40, Height: 40}, RGB{})

		// when
		drawText(img, "0", 0, 0, 1, RGB{R: 0xFF, G: 0xFF, B: 0xFF})

		// then: black (0) mixed with white (255) at coverage alpha/255 is
		// exactly `alpha` itself — the mask's own byte, unrounded
		for gy := 0; gy < glyphHeight; gy++ {
			for gx := 0; gx < glyphWidth; gx++ {
				want := alphaAt('0', gx, gy)
				got := img.At(gx, gy)
				assert.Equal(t, want, got.R, "pixel (%d,%d)", gx, gy)
				assert.Equal(t, want, got.G, "pixel (%d,%d)", gx, gy)
				assert.Equal(t, want, got.B, "pixel (%d,%d)", gx, gy)
			}
		}
	})

	t.Run("should leave pixels far from the text untouched", func(t *testing.T) {
		// given
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		img := NewFrameImage(Resolution{Width: 60, Height: 60}, background)

		// when
		drawText(img, "0", 0, 0, 1, RGB{R: 0xFF, G: 0xFF, B: 0xFF})

		// then
		assert.Equal(t, background, img.At(50, 50))
	})

	t.Run("should replicate each glyph pixel into a scale x scale block", func(t *testing.T) {
		// given
		const scale = 3
		img := NewFrameImage(Resolution{Width: 200, Height: 200}, RGB{})

		// when
		drawText(img, "0", 0, 0, scale, RGB{R: 0xFF, G: 0xFF, B: 0xFF})

		// then
		for gy := 0; gy < glyphHeight; gy++ {
			for gx := 0; gx < glyphWidth; gx++ {
				want := alphaAt('0', gx, gy)
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						got := img.At(gx*scale+sx, gy*scale+sy)
						assert.Equal(t, want, got.R, "block of glyph pixel (%d,%d)", gx, gy)
					}
				}
			}
		}
	})

	t.Run("should draw the same text twice into byte-identical images", func(t *testing.T) {
		// given
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		first := NewFrameImage(Resolution{Width: 200, Height: 60}, background)
		second := NewFrameImage(Resolution{Width: 200, Height: 60}, background)

		// when
		drawText(first, "12.3 km", 5, 5, 2, RGB{R: 0xFF, G: 0xFF, B: 0xFF})
		drawText(second, "12.3 km", 5, 5, 2, RGB{R: 0xFF, G: 0xFF, B: 0xFF})

		// then
		assert.Equal(t, first.Pix, second.Pix)
	})

	t.Run("should not draw a rune outside the font's ranges", func(t *testing.T) {
		// given
		background := RGB{R: 0x20, G: 0x26, B: 0x2E}
		img := NewFrameImage(Resolution{Width: 40, Height: 40}, background)

		// when
		drawText(img, "\u0001", 0, 0, 1, RGB{R: 0xFF, G: 0xFF, B: 0xFF})

		// then
		assert.Equal(t, background, img.At(0, 0))
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

func Test_ScreenOverlay_Draw(t *testing.T) {
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
		screenOverlay{image: off, config: disabled}.draw(plan, 1)
		screenOverlay{image: on, config: full}.draw(plan, 1)

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
		screenOverlay{image: a, config: distanceOnly}.draw(plan, 1)
		screenOverlay{image: b, config: distanceAndTime}.draw(plan, 1)

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
		screenOverlay{image: withoutClock, config: timeOnly}.draw(distancePlan, 1)
		screenOverlay{image: baseline, config: noBlocks}.draw(distancePlan, 1)

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
		screenOverlay{image: withClock, config: timeOnly}.draw(clockPlan, 1)
		screenOverlay{image: baseline, config: noBlocks}.draw(clockPlan, 1)

		// then
		assert.NotEqual(t, baseline.Pix, withClock.Pix)
	})

	t.Run("should draw nothing for the elevation and profile blocks when the plan has no elevation", func(t *testing.T) {
		// given
		elevationAndProfile, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockElevation, OverlayBlockProfile})
		noBlocks, _ := NewOverlayConfig(true, nil)
		plan := planWith(false, TimeReferenceClock)

		requested, baseline := blank(), blank()

		// when
		screenOverlay{image: requested, config: elevationAndProfile}.draw(plan, 1)
		screenOverlay{image: baseline, config: noBlocks}.draw(plan, 1)

		// then
		assert.Equal(t, baseline.Pix, requested.Pix)
	})

	t.Run("should draw the same plan and frame twice into byte-identical images", func(t *testing.T) {
		// given
		full, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockDistance, OverlayBlockElevation, OverlayBlockTime, OverlayBlockProfile})
		plan := planWith(true, TimeReferenceClock)

		first, second := blank(), blank()

		// when
		screenOverlay{image: first, config: full}.draw(plan, 1)
		screenOverlay{image: second, config: full}.draw(plan, 1)

		// then
		assert.Equal(t, first.Pix, second.Pix)
	})

	t.Run("should move the profile's dot for a different frame while the line stays the same", func(t *testing.T) {
		// given
		profileOnly, _ := NewOverlayConfig(true, []OverlayBlock{OverlayBlockProfile})
		plan := planWith(true, TimeReferenceClock)

		firstFrame, secondFrame := blank(), blank()

		// when
		screenOverlay{image: firstFrame, config: profileOnly}.draw(plan, 0)
		screenOverlay{image: secondFrame, config: profileOnly}.draw(plan, 1)

		// then
		assert.NotEqual(t, firstFrame.Pix, secondFrame.Pix)
	})
}
