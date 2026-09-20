package domain

import (
	"cmp"
	"fmt"
	"math"
	"sort"
)

const (
	coordinateStep = 1e-7
	lengthStep     = 1e-3
	angleStep      = 1e-3
)

// PlanCamera plans the camera flight over a treated track: for every frame
// of the video, where the camera is, where it points and where the activity
// marker is. The camera follows the treated route (simplified and smoothed);
// the marker's pace comes from the cleaned route, which still shows every
// long stop the simplification would have merged into a segment.
//
// The plan is deterministic: the same track and parameters always produce the
// same plan (Constitution Principle IV — and nothing here reads the clock,
// the environment or a random source).
func (t TreatedTrack) PlanCamera(parameters PlanParameters, tuning CameraTuning) (CameraPlan, error) {
	if err := parameters.Validate(); err != nil {
		return CameraPlan{}, err
	}

	plane := NewLocalPlane(t.Route)
	route := plane.ProjectRoute(t.Route)

	if length := route.Length(); length < tuning.MinTrackLengthMeters {
		return CameraPlan{}, fmt.Errorf("%w: length is %.1f m, minimum is %.1f m", ErrTrackTooShort, length, tuning.MinTrackLengthMeters)
	}
	if span := route.Span(); span > tuning.MaxTrackSpanMeters {
		return CameraPlan{}, fmt.Errorf("%w: span is %.1f km, maximum is %.1f km", ErrTrackTooLarge, span/1000, tuning.MaxTrackSpanMeters/1000)
	}

	duration, mode, err := parameters.resolveDuration(route.Length(), parameters.minimumDuration(route, tuning), tuning)
	if err != nil {
		return CameraPlan{}, err
	}
	parameters.Duration = &duration

	frameCount := parameters.FrameCount(duration)
	openingCount := roundHalfUp(float64(frameCount) * tuning.OpeningFraction)
	closingCount := roundHalfUp(float64(frameCount) * tuning.ClosingFraction)
	followCount := frameCount - openingCount - closingCount
	if followCount < 1 {
		return CameraPlan{}, fmt.Errorf("%w: %s leaves no frames to follow the track", ErrDurationTooShort, duration)
	}

	planner := cameraPlanner{plane: plane, route: route, parameters: parameters, tuning: tuning}

	// The marker's pace is planned on the cleaned route; its position is
	// then placed at the same fraction of the treated route's length.
	timeline := NewMarkerTimeline(t.Cleaned, tuning)
	scale := route.Length() / math.Max(timeline.Total(), 1e-9)

	// The following phase: the marker's progress, then a camera view per frame.
	distances := make([]float64, followCount)
	markers := make([]PlanePoint, followCount)
	for k := range distances {
		fraction := 0.0
		if followCount > 1 {
			fraction = float64(k) / float64(followCount-1)
		}
		distances[k] = timeline.DistanceAt(fraction) * scale
		markers[k] = route.PointAt(distances[k])
	}
	follow := planner.followViews(distances, markers)

	// All frames: opening, following, closing.
	views := make([]CameraView, 0, frameCount)
	overviewFirst := route.OverviewView(follow[0].Heading, tuning.OverviewMinDistanceFactor*follow[0].Distance, tuning)
	overviewLast := route.OverviewView(follow[followCount-1].Heading, tuning.OverviewMinDistanceFactor*follow[followCount-1].Distance, tuning)
	for i := 0; i < openingCount; i++ {
		views = append(views, overviewFirst.Blend(follow[0], float64(i)/float64(openingCount)))
	}
	views = append(views, follow...)
	for j := 0; j < closingCount; j++ {
		views = append(views, follow[followCount-1].Blend(overviewLast, float64(j+1)/float64(closingCount)))
	}

	views, spans := planner.limitAndSmooth(views)

	frames := make([]CameraFrame, frameCount)
	for i, view := range views {
		phase, marker, distance := PhaseFollowing, PlanePoint{}, 0.0
		switch {
		case i < openingCount:
			phase, marker, distance = PhaseOpening, route.Points[0], 0
		case i >= openingCount+followCount:
			phase, marker, distance = PhaseClosing, route.Points[len(route.Points)-1], route.Length()
		default:
			k := i - openingCount
			marker, distance = markers[k], distances[k]
		}
		frames[i] = planner.frame(i, phase, view, marker, distance)
	}

	return NewCameraPlan(parameters, mode, timeline.Reference, timeline.FallbackReason, frames, spans), nil
}

// cameraPlanner holds what planning a camera flight needs while it runs.
type cameraPlanner struct {
	plane      LocalPlane
	route      PlanarRoute
	parameters PlanParameters
	tuning     CameraTuning
}

// followViews builds the camera views of the following phase, one per frame:
// the camera looks at the (lightly smoothed) marker, from a distance that
// grows with the marker's speed, pointing along the direction of the route
// around the marker.
func (c cameraPlanner) followViews(distances []float64, markers []PlanePoint) []CameraView {
	fps := c.parameters.FrameRate
	count := len(distances)

	speeds := make(Signal, count)
	for k := range speeds {
		from, to := max(k-1, 0), min(k+1, count-1)
		if to > from {
			speeds[k] = (distances[to] - distances[from]) / (float64(to-from) / fps)
		}
	}
	speeds = speeds.Smooth(4 * c.tuning.GaussianSigmaSeconds * fps)

	tilt := c.tuning.TiltDegrees[c.parameters.Tilt.index()]

	targetX, targetY := make(Signal, count), make(Signal, count)
	for k, m := range markers {
		targetX[k], targetY[k] = m.X, m.Y
	}
	targetSigma := 2 * c.tuning.GaussianSigmaSeconds * fps
	targetX, targetY = targetX.Smooth(targetSigma), targetY.Smooth(targetSigma)

	previous := c.route.ChordHeading()

	headings := make(Signal, count)
	views := make([]CameraView, count)
	for k := range views {
		distance := c.tuning.FollowDistance(c.parameters.Distance, speeds[k])
		heading, ok := c.route.HeadingAt(distances[k], distance)
		if !ok {
			heading = previous
		}
		previous = heading
		headings[k] = heading

		views[k] = CameraView{Target: PlanePoint{X: targetX[k], Y: targetY[k]}, TiltDegrees: tilt, Distance: distance}
	}

	for k, heading := range headings.Unwrap() {
		views[k].Heading = heading
	}
	return views
}

// limitAndSmooth applies the smoothness limits to a sequence of desired views
// (heading, tilt, zoom and target speed) and smooths the result, returning
// the final views and the stretches where a limit had to act.
func (c cameraPlanner) limitAndSmooth(desired []CameraView) ([]CameraView, []SmoothedSpan) {
	fps := c.parameters.FrameRate
	count := len(desired)
	headings, tilts, zooms := make(Signal, count), make(Signal, count), make(Signal, count)
	xs, ys, targetSteps := make(Signal, count), make(Signal, count), make([]float64, count)
	headingSteps, tiltSteps, zoomSteps := make([]float64, count), make([]float64, count), make([]float64, count)

	for k, v := range desired {
		headings[k], tilts[k], zooms[k] = v.Heading, v.TiltDegrees, math.Log(v.Distance)
		xs[k], ys[k] = v.Target.X, v.Target.Y
		headingSteps[k] = c.tuning.MaxHeadingRateDegPerSecond / fps
		tiltSteps[k] = c.tuning.MaxTiltRateDegPerSecond / fps
		zoomSteps[k] = c.tuning.MaxLogDistanceRatePerSecond / fps
		// The step limit is per axis, so a step along both never exceeds the
		// limit on its length.
		targetSteps[k] = c.tuning.MaxTargetSpeedInDistances * v.Distance / fps / math.Sqrt2
	}

	limitedHeadings := headings.LimitRate(headingSteps)
	limitedTilts := tilts.LimitRate(tiltSteps)
	limitedZooms := zooms.LimitRate(zoomSteps)
	limitedXs, limitedYs := xs.LimitRate(targetSteps), ys.LimitRate(targetSteps)

	targetDeviation, zeros := make(Signal, count), make(Signal, count)
	for k := range desired {
		targetDeviation[k] = math.Hypot(xs[k]-limitedXs[k], ys[k]-limitedYs[k]) / desired[k].Distance
	}

	var spans []SmoothedSpan
	spans = append(spans, headings.SmoothedSpans(limitedHeadings, headingSpanToleranceDegrees, QuantityHeading, fps)...)
	spans = append(spans, tilts.SmoothedSpans(limitedTilts, tiltSpanToleranceDegrees, QuantityTilt, fps)...)
	spans = append(spans, zooms.SmoothedSpans(limitedZooms, zoomSpanTolerance, QuantityZoom, fps)...)
	spans = append(spans, targetDeviation.SmoothedSpans(zeros, targetSpanToleranceRatio, QuantityTargetSpeed, fps)...)
	sort.SliceStable(spans, func(a, b int) bool {
		return cmp.Or(cmp.Compare(spans[a].Start, spans[b].Start), cmp.Compare(spans[a].Quantity, spans[b].Quantity)) < 0
	})

	sigma := c.tuning.GaussianSigmaSeconds * fps
	smoothedHeadings := limitedHeadings.Smooth(sigma)
	smoothedTilts := limitedTilts.Smooth(sigma)
	smoothedZooms := limitedZooms.Smooth(sigma)
	smoothedXs, smoothedYs := limitedXs.Smooth(sigma), limitedYs.Smooth(sigma)

	views := make([]CameraView, count)
	for k := range views {
		views[k] = CameraView{
			Target:      PlanePoint{X: smoothedXs[k], Y: smoothedYs[k]},
			Heading:     normalizeDegrees(smoothedHeadings[k]),
			TiltDegrees: clamp(smoothedTilts[k], 0, 90),
			Distance:    math.Exp(smoothedZooms[k]),
		}
	}
	return views, spans
}

// frame turns a camera view into a plan frame, quantizing every value.
func (c cameraPlanner) frame(index int, phase Phase, view CameraView, marker PlanePoint, markerDistance float64) CameraFrame {
	pose := view.Pose()
	cameraLat, cameraLon := c.plane.Unproject(pose.Position)
	markerLat, markerLon := c.plane.Unproject(marker)
	toMarker := math.Sqrt(math.Pow(pose.Position.X-marker.X, 2) + math.Pow(pose.Position.Y-marker.Y, 2) + pose.Altitude*pose.Altitude)

	return CameraFrame{
		Index:                  index,
		Time:                   frameTime(index, c.parameters.FrameRate),
		Phase:                  phase,
		CameraLatitude:         quantize(cameraLat, coordinateStep),
		CameraLongitude:        quantizeLongitude(cameraLon),
		CameraAltitude:         quantize(pose.Altitude, lengthStep),
		Heading:                quantizeHeading(view.Heading),
		Tilt:                   quantize(view.TiltDegrees, angleStep),
		MarkerLatitude:         quantize(markerLat, coordinateStep),
		MarkerLongitude:        quantizeLongitude(markerLon),
		MarkerDistance:         quantize(markerDistance, lengthStep),
		CameraToMarkerDistance: quantize(toMarker, lengthStep),
	}
}

// quantizeLongitude quantizes lon and wraps it back into [-180, 180), since
// rounding can push a value just below 180 up to 180.
func quantizeLongitude(lon float64) float64 {
	return normalizeLongitude(quantize(lon, coordinateStep))
}

func quantizeHeading(heading float64) float64 {
	q := quantize(heading, angleStep)
	if q >= 360 {
		q -= 360
	}
	return q
}

func roundHalfUp(v float64) int {
	return int(math.Floor(v + 0.5))
}
