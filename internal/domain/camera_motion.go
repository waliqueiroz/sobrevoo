package domain

import (
	"math"
	"sort"
	"time"
)

const (
	// headingHoldRatio is the resultant/total-weight ratio of the tangent
	// window below which the track has no clear direction there (a turn
	// back, or a loop tighter than the window), so the camera keeps its
	// heading instead of guessing.
	headingHoldRatio = 0.05

	// spans are detected against these tolerances (see research.md item 5).
	headingSpanToleranceDegrees = 0.05
	tiltSpanToleranceDegrees    = 0.05
	zoomSpanTolerance           = 0.001
	targetSpanToleranceRatio    = 0.01
)

// CameraPose is where a camera is, as derived from what it looks at.
type CameraPose struct {
	// Position is the camera's horizontal position on the local plane.
	Position PlanePoint

	// Altitude is the camera's height above the observed point, in meters.
	Altitude float64
}

// CameraView is what a camera looks at and how: the point it observes, the
// horizontal direction it points (degrees clockwise from north), its tilt
// below the horizon (degrees) and its straight-line distance to the point.
type CameraView struct {
	Target      PlanePoint
	Heading     float64
	TiltDegrees float64
	Distance    float64
}

// ComputeCameraPose places the camera behind the target, along the heading:
// its horizontal offset is Distance·cos(tilt) and its altitude
// Distance·sin(tilt).
func ComputeCameraPose(view CameraView) CameraPose {
	heading := degreesToRadians(view.Heading)
	tilt := degreesToRadians(view.TiltDegrees)
	horizontal := view.Distance * math.Cos(tilt)

	return CameraPose{
		Position: PlanePoint{
			X: view.Target.X - horizontal*math.Sin(heading),
			Y: view.Target.Y - horizontal*math.Cos(heading),
		},
		Altitude: view.Distance * math.Sin(tilt),
	}
}

// FollowDistance is how far the camera flies from the marker while following
// it: a base distance plus the distance the marker covers, in the video, in
// lookAheadSeconds — so fast-moving markers stay in view.
func FollowDistance(baseMeters, lookAheadSeconds, markerSpeed float64) float64 {
	return baseMeters + lookAheadSeconds*markerSpeed
}

// DesiredHeading returns the direction, in degrees clockwise from north in
// [0, 360), the camera should point at arc position s (meters travelled) of
// route: the direction of the Gaussian-weighted sum of the route's tangents
// around s, with standard deviation sigma meters. ok is false when the
// tangents cancel out (a turn back, a tight loop) and there is no clear
// direction.
func DesiredHeading(route []PlanePoint, cumulative []float64, s, sigma float64) (heading float64, ok bool) {
	if len(route) < 2 || sigma <= 0 {
		return 0, false
	}

	lo := sort.SearchFloat64s(cumulative, s-3*sigma)
	hi := sort.SearchFloat64s(cumulative, s+3*sigma)
	lo = max(lo-1, 0)
	hi = min(hi+1, len(route)-1)

	var sumX, sumY, sumWeight float64
	for i := lo; i < hi; i++ {
		dx, dy := route[i+1].X-route[i].X, route[i+1].Y-route[i].Y
		length := math.Hypot(dx, dy)
		if length < 1e-9 {
			continue
		}

		offset := (cumulative[i]+cumulative[i+1])/2 - s
		weight := length * math.Exp(-offset*offset/(2*sigma*sigma))
		sumX += weight * dx / length
		sumY += weight * dy / length
		sumWeight += weight
	}

	if sumWeight == 0 || math.Hypot(sumX, sumY) < headingHoldRatio*sumWeight {
		return 0, false
	}

	return normalizeDegrees(radiansToDegrees(math.Atan2(sumX, sumY))), true
}

// UnwrapAngles removes the 360° jumps of a sequence of angles in degrees, so
// consecutive values differ by the shortest arc. A difference of exactly 180°
// resolves counter-clockwise (decreasing angle).
func UnwrapAngles(angles []float64) []float64 {
	unwrapped := make([]float64, len(angles))
	for i, angle := range angles {
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

// LimitRate returns values with each step limited: |out[k] - out[k-1]| is at
// most maxSteps[k] for every k >= 1 (maxSteps[0] is ignored; it must have as
// many entries as values). A forward pass followed by a backward pass makes
// the limit hold at every pair without accumulating lag; a signal that
// already respects the limits comes back unchanged.
func LimitRate(values, maxSteps []float64) []float64 {
	limited := append([]float64{}, values...)

	for k := 1; k < len(limited); k++ {
		limited[k] = clamp(limited[k], limited[k-1]-maxSteps[k], limited[k-1]+maxSteps[k])
	}
	for k := len(limited) - 2; k >= 0; k-- {
		limited[k] = clamp(limited[k], limited[k+1]-maxSteps[k+1], limited[k+1]+maxSteps[k+1])
	}

	return limited
}

// GaussianSmooth smooths values with a Gaussian kernel of standard deviation
// sigma samples, truncated at three sigmas. Beyond the ends the signal is
// extended by odd reflection around the end value, which keeps the end values
// exactly and preserves linear trends. Convolving a signal whose steps are
// bounded with a normalized kernel never increases that bound.
func GaussianSmooth(values []float64, sigma float64) []float64 {
	if sigma <= 0 || len(values) == 0 {
		return append([]float64{}, values...)
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

	last := len(values) - 1
	at := func(index int) float64 {
		switch {
		case index < 0:
			return 2*values[0] - values[min(-index, last)]
		case index > last:
			return 2*values[last] - values[max(2*last-index, 0)]
		}
		return values[index]
	}

	smoothed := make([]float64, len(values))
	for k := range values {
		var sum float64
		for i, weight := range kernel {
			sum += weight * at(k+i-radius)
		}
		smoothed[k] = sum
	}
	return smoothed
}

// DetectSmoothedSpans returns the maximal runs of frames in which limited
// differs from desired by more than tolerance — the stretches where a
// smoothness limit had to act. Times are frame index divided by frameRate.
func DetectSmoothedSpans(desired, limited []float64, tolerance float64, quantity SmoothedQuantity, frameRate float64) []SmoothedSpan {
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

	for k := range desired {
		if math.Abs(desired[k]-limited[k]) > tolerance {
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
		closeSpan(len(desired) - 1)
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
