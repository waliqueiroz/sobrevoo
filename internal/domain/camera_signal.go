package domain

import (
	"math"
	"time"
)

const (
	// Spans are detected against these tolerances (research.md item 5).
	headingSpanToleranceDegrees = 0.05
	tiltSpanToleranceDegrees    = 0.05
	zoomSpanTolerance           = 0.001
	targetSpanToleranceRatio    = 0.01
)

// Signal is a quantity sampled once per frame of the video (a heading, a
// tilt, a zoom): the camera's motion is planned as a handful of signals.
type Signal []float64

// Unwrap removes the 360° jumps of a signal of angles in degrees, so
// consecutive values differ by the shortest arc. A difference of exactly 180°
// resolves counter-clockwise (decreasing angle).
func (s Signal) Unwrap() Signal {
	unwrapped := make(Signal, len(s))
	for i, angle := range s {
		if i == 0 {
			unwrapped[i] = angle
			continue
		}

		delta := math.Mod(angle-unwrapped[i-1]+180, 360)
		if delta < 0 {
			delta += 360
		}
		unwrapped[i] = unwrapped[i-1] + delta - 180
	}
	return unwrapped
}

// LimitRate returns the signal with each step limited: |out[k] - out[k-1]| is
// at most maxSteps[k] for every k >= 1 (maxSteps[0] is ignored; it must have
// as many entries as the signal). A forward pass followed by a backward pass
// makes the limit hold at every pair without accumulating lag; a signal that
// already respects the limits comes back unchanged.
func (s Signal) LimitRate(maxSteps []float64) Signal {
	limited := append(Signal{}, s...)

	for k := 1; k < len(limited); k++ {
		limited[k] = clamp(limited[k], limited[k-1]-maxSteps[k], limited[k-1]+maxSteps[k])
	}
	for k := len(limited) - 2; k >= 0; k-- {
		limited[k] = clamp(limited[k], limited[k+1]-maxSteps[k+1], limited[k+1]+maxSteps[k+1])
	}

	return limited
}

// Smooth smooths the signal with a Gaussian kernel of standard deviation
// sigma samples, truncated at three sigmas. Beyond the ends the signal is
// extended by odd reflection around the end value, which keeps the end values
// exactly and preserves linear trends. Convolving a signal whose steps are
// bounded with a normalized kernel never increases that bound.
func (s Signal) Smooth(sigma float64) Signal {
	if sigma <= 0 || len(s) == 0 {
		return append(Signal{}, s...)
	}

	radius := int(math.Ceil(3 * sigma))
	kernel := make([]float64, 2*radius+1)
	var total float64
	for i := range kernel {
		x := float64(i - radius)
		kernel[i] = math.Exp(-x * x / (2 * sigma * sigma))
		total += kernel[i]
	}
	for i := range kernel {
		kernel[i] /= total
	}

	last := len(s) - 1
	at := func(index int) float64 {
		switch {
		case index < 0:
			return 2*s[0] - s[min(-index, last)]
		case index > last:
			return 2*s[last] - s[max(2*last-index, 0)]
		}
		return s[index]
	}

	smoothed := make(Signal, len(s))
	for k := range s {
		var sum float64
		for i, weight := range kernel {
			sum += weight * at(k+i-radius)
		}
		smoothed[k] = sum
	}
	return smoothed
}

// SmoothedSpans returns the maximal runs of frames in which limited differs
// from the signal (the desired one) by more than tolerance — the stretches
// where a smoothness limit had to act. Times are frame index divided by
// frameRate.
func (s Signal) SmoothedSpans(limited Signal, tolerance float64, quantity SmoothedQuantity, frameRate float64) []SmoothedSpan {
	var spans []SmoothedSpan

	start := -1
	closeSpan := func(end int) {
		spans = append(spans, SmoothedSpan{
			Start:    frameTime(start, frameRate),
			End:      frameTime(end, frameRate),
			Quantity: quantity,
		})
		start = -1
	}

	for k := range s {
		if math.Abs(s[k]-limited[k]) > tolerance {
			if start < 0 {
				start = k
			}
			continue
		}
		if start >= 0 {
			closeSpan(k - 1)
		}
	}
	if start >= 0 {
		closeSpan(len(s) - 1)
	}

	return spans
}

func frameTime(index int, frameRate float64) time.Duration {
	return time.Duration(math.Round(float64(index) / frameRate * float64(time.Second)))
}

func normalizeDegrees(degrees float64) float64 {
	wrapped := math.Mod(degrees, 360)
	if wrapped < 0 {
		wrapped += 360
	}
	return wrapped
}

// quantize rounds v to a multiple of step; the plan's values are defined at
// this precision so results coincide across platforms (research.md item 9).
func quantize(v, step float64) float64 {
	return math.Round(v/step) * step
}
