package domain

import (
	"cmp"
	"fmt"
	"math"
	"sort"
	"time"
)

// smoothstepPeakSlope is the steepest slope of a smoothstep (3u² - 2u³): the
// opening and closing need that many times the average rate of change.
const smoothstepPeakSlope = 1.5

// PlanCamera plans the camera flight over a treated track: for every frame
// of the video, where the camera is, where it points and where the activity
// marker is. The camera follows the treated route (simplified and smoothed);
// the marker's pace comes from the cleaned points, which still show every
// long stop the simplification would have merged into a segment.
//
// The plan is deterministic: the same route and parameters always produce the
// same plan (Constitution Principle IV — and nothing here reads the clock,
// the environment or a random source).
func PlanCamera(treated TreatedTrack, parameters PlanParameters, tuning CameraTuning) (CameraPlan, error) {
	if err := parameters.Validate(); err != nil {
		return CameraPlan{}, err
	}

	points := treated.Route.Points
	plane := NewLocalPlane(points)
	route := projectAll(plane, points)

	length := TotalDistance(points)
	if length < tuning.MinTrackLengthMeters {
		return CameraPlan{}, fmt.Errorf("%w: length is %.1f m, minimum is %.1f m", ErrTrackTooShort, length, tuning.MinTrackLengthMeters)
	}
	if span := trackSpan(route); span > tuning.MaxTrackSpanMeters {
		return CameraPlan{}, fmt.Errorf("%w: span is %.1f km, maximum is %.1f km", ErrTrackTooLarge, span/1000, tuning.MaxTrackSpanMeters/1000)
	}

	minimum := MinimumDuration(points, parameters.FrameRate, parameters.Distance, parameters.Tilt, tuning)

	duration, mode := DefaultDuration(points, parameters.FrameRate, parameters.Distance, parameters.Tilt, tuning), DurationModeAutomatic
	if parameters.Duration != nil {
		if *parameters.Duration < minimum {
			return CameraPlan{}, fmt.Errorf("%w: %s requested, minimum for this track is %.2f s", ErrDurationTooShort, *parameters.Duration, roundUpToCentiseconds(minimum))
		}
		duration, mode = *parameters.Duration, DurationModeExplicit
	}
	parameters.Duration = &duration

	frameCount := parameters.FrameCount(duration)
	openingCount := roundHalfUp(float64(frameCount) * tuning.OpeningFraction)
	closingCount := roundHalfUp(float64(frameCount) * tuning.ClosingFraction)
	followCount := frameCount - openingCount - closingCount
	if followCount < 1 {
		return CameraPlan{}, fmt.Errorf("%w: %s leaves no frames to follow the track", ErrDurationTooShort, duration)
	}

	// The marker's pace is planned on the cleaned points; its position is
	// then placed at the same fraction of the treated route's length.
	timeline := BuildMarkerTimeline(treated.CleanedPoints, tuning)
	cumulative := routeDistances(points)
	total := cumulative[len(cumulative)-1]
	scale := total / math.Max(timeline.Total(), 1e-9)

	// The following phase: the marker's progress, then a camera view per frame.
	distances := make([]float64, followCount)
	markers := make([]PlanePoint, followCount)
	for k := range distances {
		fraction := 0.0
		if followCount > 1 {
			fraction = float64(k) / float64(followCount-1)
		}
		distances[k] = timeline.DistanceAt(fraction) * scale
		markers[k] = pointAt(route, cumulative, distances[k])
	}

	fps := parameters.FrameRate
	follow := followViews(route, cumulative, distances, markers, parameters, tuning, fps)

	// All frames: opening, following, closing.
	views := make([]CameraView, 0, frameCount)
	overviewFirst := OverviewView(route, follow[0].Heading, tuning.OverviewMinDistanceFactor*follow[0].Distance, tuning)
	overviewLast := OverviewView(route, follow[followCount-1].Heading, tuning.OverviewMinDistanceFactor*follow[followCount-1].Distance, tuning)
	for i := 0; i < openingCount; i++ {
		views = append(views, BlendView(overviewFirst, follow[0], float64(i)/float64(openingCount)))
	}
	views = append(views, follow...)
	for j := 0; j < closingCount; j++ {
		views = append(views, BlendView(follow[followCount-1], overviewLast, float64(j+1)/float64(closingCount)))
	}

	views, spans := limitAndSmooth(views, tuning, fps)

	frames := make([]CameraFrame, frameCount)
	for i, view := range views {
		phase, marker, distance := PhaseFollowing, PlanePoint{}, 0.0
		switch {
		case i < openingCount:
			phase, marker, distance = PhaseOpening, route[0], 0
		case i >= openingCount+followCount:
			phase, marker, distance = PhaseClosing, route[len(route)-1], total
		default:
			k := i - openingCount
			marker, distance = markers[k], distances[k]
		}
		frames[i] = buildFrame(i, phase, view, marker, distance, plane, fps)
	}

	return NewCameraPlan(parameters, mode, timeline.Reference, timeline.FallbackReason, frames, spans), nil
}

// followViews builds the camera views of the following phase, one per frame:
// the camera looks at the (lightly smoothed) marker, from a distance that
// grows with the marker's speed, pointing along the direction of the track
// around the marker.
func followViews(route []PlanePoint, cumulative, distances []float64, markers []PlanePoint, parameters PlanParameters, tuning CameraTuning, fps float64) []CameraView {
	count := len(distances)

	speeds := make([]float64, count)
	for k := range speeds {
		from, to := max(k-1, 0), min(k+1, count-1)
		if to > from {
			speeds[k] = (distances[to] - distances[from]) / (float64(to-from) / fps)
		}
	}
	speeds = GaussianSmooth(speeds, 4*tuning.GaussianSigmaSeconds*fps)

	base := tuning.BaseDistanceMeters[levelIndex(parameters.Distance)]
	lookAhead := tuning.LookAheadSeconds[levelIndex(parameters.Distance)]
	tilt := tuning.TiltDegrees[levelIndex(parameters.Tilt)]

	targetX, targetY := make([]float64, count), make([]float64, count)
	for k, m := range markers {
		targetX[k], targetY[k] = m.X, m.Y
	}
	targetSigma := 2 * tuning.GaussianSigmaSeconds * fps
	targetX, targetY = GaussianSmooth(targetX, targetSigma), GaussianSmooth(targetY, targetSigma)

	last := route[len(route)-1]
	previous := normalizeDegrees(radiansToDegrees(math.Atan2(last.X-route[0].X, last.Y-route[0].Y)))

	headings := make([]float64, count)
	views := make([]CameraView, count)
	for k := range views {
		distance := FollowDistance(base, lookAhead, speeds[k])
		heading, ok := DesiredHeading(route, cumulative, distances[k], distance)
		if !ok {
			heading = previous
		}
		previous = heading
		headings[k] = heading

		views[k] = CameraView{Target: PlanePoint{X: targetX[k], Y: targetY[k]}, TiltDegrees: tilt, Distance: distance}
	}

	for k, heading := range UnwrapAngles(headings) {
		views[k].Heading = heading
	}
	return views
}

// limitAndSmooth applies the smoothness limits to a sequence of desired views
// (heading, tilt, zoom and target speed) and smooths the result, returning
// the final views and the stretches where a limit had to act.
func limitAndSmooth(desired []CameraView, tuning CameraTuning, fps float64) ([]CameraView, []SmoothedSpan) {
	count := len(desired)
	headings, tilts, zooms := make([]float64, count), make([]float64, count), make([]float64, count)
	xs, ys, targetSteps := make([]float64, count), make([]float64, count), make([]float64, count)
	headingSteps, tiltSteps, zoomSteps := make([]float64, count), make([]float64, count), make([]float64, count)

	for k, v := range desired {
		headings[k], tilts[k], zooms[k] = v.Heading, v.TiltDegrees, math.Log(v.Distance)
		xs[k], ys[k] = v.Target.X, v.Target.Y
		headingSteps[k] = tuning.MaxHeadingRateDegPerSecond / fps
		tiltSteps[k] = tuning.MaxTiltRateDegPerSecond / fps
		zoomSteps[k] = tuning.MaxLogDistanceRatePerSecond / fps
		// The step limit is per axis, so a step along both never exceeds the
		// limit on its length.
		targetSteps[k] = tuning.MaxTargetSpeedInDistances * v.Distance / fps / math.Sqrt2
	}

	limitedHeadings := LimitRate(headings, headingSteps)
	limitedTilts := LimitRate(tilts, tiltSteps)
	limitedZooms := LimitRate(zooms, zoomSteps)
	limitedXs, limitedYs := LimitRate(xs, targetSteps), LimitRate(ys, targetSteps)

	targetDeviation, zeros := make([]float64, count), make([]float64, count)
	for k := range desired {
		targetDeviation[k] = math.Hypot(xs[k]-limitedXs[k], ys[k]-limitedYs[k]) / desired[k].Distance
	}

	var spans []SmoothedSpan
	spans = append(spans, DetectSmoothedSpans(headings, limitedHeadings, headingSpanToleranceDegrees, QuantityHeading, fps)...)
	spans = append(spans, DetectSmoothedSpans(tilts, limitedTilts, tiltSpanToleranceDegrees, QuantityTilt, fps)...)
	spans = append(spans, DetectSmoothedSpans(zooms, limitedZooms, zoomSpanTolerance, QuantityZoom, fps)...)
	spans = append(spans, DetectSmoothedSpans(targetDeviation, zeros, targetSpanToleranceRatio, QuantityTargetSpeed, fps)...)
	sort.SliceStable(spans, func(a, b int) bool {
		return cmp.Or(cmp.Compare(spans[a].Start, spans[b].Start), cmp.Compare(spans[a].Quantity, spans[b].Quantity)) < 0
	})

	sigma := tuning.GaussianSigmaSeconds * fps
	smoothedHeadings := GaussianSmooth(limitedHeadings, sigma)
	smoothedTilts := GaussianSmooth(limitedTilts, sigma)
	smoothedZooms := GaussianSmooth(limitedZooms, sigma)
	smoothedXs, smoothedYs := GaussianSmooth(limitedXs, sigma), GaussianSmooth(limitedYs, sigma)

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

// buildFrame turns a camera view into a plan frame, quantizing every value.
func buildFrame(index int, phase Phase, view CameraView, marker PlanePoint, markerDistance float64, plane LocalPlane, fps float64) CameraFrame {
	pose := ComputeCameraPose(view)
	cameraLat, cameraLon := plane.Unproject(pose.Position)
	markerLat, markerLon := plane.Unproject(marker)
	toMarker := math.Sqrt(math.Pow(pose.Position.X-marker.X, 2) + math.Pow(pose.Position.Y-marker.Y, 2) + pose.Altitude*pose.Altitude)

	return CameraFrame{
		Index:                  index,
		Time:                   frameTime(index, fps),
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

const (
	coordinateStep = 1e-7
	lengthStep     = 1e-3
	angleStep      = 1e-3
)

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

// MinimumDuration is the shortest video duration over points that still lets
// the opening and the closing reach the follow pose within the smoothness
// limits, and leaves enough time to follow the track. It is computed from
// conservative bounds that do not depend on the duration itself, and rounded
// up to a whole frame.
func MinimumDuration(points []TrackPoint, frameRate float64, distance, tilt Level, tuning CameraTuning) time.Duration {
	route := projectAll(NewLocalPlane(points), points)

	base := tuning.BaseDistanceMeters[levelIndex(distance)]
	overview := OverviewView(route, 0, tuning.OverviewMinDistanceFactor*base, tuning)
	followTilt := tuning.TiltDegrees[levelIndex(tilt)]

	phase := math.Max(tuning.MinPhaseDuration.Seconds(), math.Max(
		smoothstepPeakSlope*math.Abs(overview.TiltDegrees-followTilt)/tuning.MaxTiltRateDegPerSecond,
		smoothstepPeakSlope*math.Abs(math.Log(overview.Distance/base))/tuning.MaxLogDistanceRatePerSecond,
	))

	followShare := 1 - tuning.OpeningFraction - tuning.ClosingFraction
	seconds := math.Max(
		math.Max(phase/tuning.OpeningFraction, phase/tuning.ClosingFraction),
		tuning.MinFollowDuration.Seconds()/followShare,
	)

	frames := math.Ceil(seconds*frameRate - 1e-9)
	return time.Duration(math.Ceil(frames / frameRate * float64(time.Second)))
}

// DefaultDuration is the video duration used when the user does not choose
// one: a base duration that grows with the square root of the track's length,
// clamped to the configured range, and never below MinimumDuration.
func DefaultDuration(points []TrackPoint, frameRate float64, distance, tilt Level, tuning CameraTuning) time.Duration {
	km := TotalDistance(points) / 1000
	seconds := tuning.AutoDurationBase.Seconds() + tuning.AutoDurationPerSqrtKm.Seconds()*math.Sqrt(km)
	seconds = clamp(seconds, tuning.AutoDurationMin.Seconds(), tuning.AutoDurationMax.Seconds())

	duration := time.Duration(math.Round(seconds)) * time.Second
	return max(duration, MinimumDuration(points, frameRate, distance, tilt, tuning))
}

// routeDistances returns the distance travelled, in meters, at each point.
func routeDistances(points []TrackPoint) []float64 {
	cumulative := make([]float64, len(points))
	for i := 1; i < len(points); i++ {
		cumulative[i] = cumulative[i-1] + Haversine(points[i-1], points[i])
	}
	return cumulative
}

func projectAll(plane LocalPlane, points []TrackPoint) []PlanePoint {
	route := make([]PlanePoint, len(points))
	for i, p := range points {
		route[i] = plane.Project(p.Latitude, p.Longitude)
	}
	return route
}

// trackSpan is twice the largest distance from the plane's center to a point
// of the route: roughly the largest distance between two points.
func trackSpan(route []PlanePoint) float64 {
	var radius float64
	for _, p := range route {
		radius = math.Max(radius, math.Hypot(p.X, p.Y))
	}
	return 2 * radius
}

// pointAt returns the position after travelling s meters along the route.
func pointAt(route []PlanePoint, cumulative []float64, s float64) PlanePoint {
	// i is the first point at or after s, so cumulative[i-1] < s <= cumulative[i]
	// and the segment has positive length (i is clamped for s beyond the end).
	i := min(sort.SearchFloat64s(cumulative, s), len(route)-1)
	if i == 0 {
		return route[0]
	}

	t := clamp((s-cumulative[i-1])/(cumulative[i]-cumulative[i-1]), 0, 1)
	return PlanePoint{
		X: route[i-1].X + t*(route[i].X-route[i-1].X),
		Y: route[i-1].Y + t*(route[i].Y-route[i-1].Y),
	}
}

func levelIndex(level Level) int {
	return min(max(int(level), 0), levelCount-1)
}

func roundHalfUp(v float64) int {
	return int(math.Floor(v + 0.5))
}

func roundUpToCentiseconds(d time.Duration) float64 {
	return math.Ceil(d.Seconds()*100-1e-9) / 100
}
