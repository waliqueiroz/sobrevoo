package domain

import (
	"fmt"
	"math"
	"time"
)

// smoothstepPeakSlope is the steepest slope of a smoothstep (3u² - 2u³): the
// opening and closing need that many times the average rate of change.
const smoothstepPeakSlope = 1.5

const (
	// MinFrameRate and MaxFrameRate bound the accepted frame rates, in
	// frames per second (inclusive).
	MinFrameRate = 1.0
	MaxFrameRate = 120.0

	// MaxExplicitDuration is the longest duration a user may request.
	MaxExplicitDuration = time.Hour

	levelCount = 3
)

// PlanParameters are the choices a user makes about a camera plan.
type PlanParameters struct {
	// Duration is the video duration. Nil means "automatic": it is derived
	// from the track's length.
	Duration *time.Duration

	// FrameRate is the number of frames per second.
	FrameRate float64

	// Distance and Tilt choose how far and how steep the camera flies.
	Distance Level
	Tilt     Level
}

// Validate checks the ranges of the parameters. The frame rate is always
// validated; the duration only when the user provided one.
func (p PlanParameters) Validate() error {
	if math.IsNaN(p.FrameRate) || math.IsInf(p.FrameRate, 0) || p.FrameRate < MinFrameRate || p.FrameRate > MaxFrameRate {
		return fmt.Errorf("%w: %g, must be between %g and %g frames per second", ErrInvalidFrameRate, p.FrameRate, MinFrameRate, MaxFrameRate)
	}

	if p.Duration != nil && (*p.Duration <= 0 || *p.Duration > MaxExplicitDuration) {
		return fmt.Errorf("%w: %s, must be greater than 0 s and at most %s", ErrInvalidDuration, *p.Duration, MaxExplicitDuration)
	}

	return nil
}

// FrameCount returns how many frames a video of the given duration has at the
// parameters' frame rate, rounding to the nearest whole frame (halves round
// up).
func (p PlanParameters) FrameCount(duration time.Duration) int {
	return int(math.Floor(duration.Seconds()*p.FrameRate + 0.5))
}

// CameraTuning holds the heuristic constants of the camera planning
// algorithm. They are injected (Constitution Principle VIII): the core
// defines the shape, an outbound configuration adapter provides the values.
// See specs/003-camera-path-planning/research.md for what each one means.
type CameraTuning struct {
	// OpeningFraction and ClosingFraction are the share of the video's
	// frames spent on the opening and on the closing.
	OpeningFraction float64
	ClosingFraction float64

	// A "long stop" is a run of segments slower than
	// StopSpeedMetersPerSecond lasting more than StopMinDuration. Its
	// duration in the video is capped at StopCappedDuration and at
	// StopMaxShareOfMovingTime of the moving time.
	StopSpeedMetersPerSecond float64
	StopMinDuration          time.Duration
	StopCappedDuration       time.Duration
	StopMaxShareOfMovingTime float64

	// Smoothness limits, all per second of video and relative to the
	// camera-to-target distance where a length is involved.
	MaxHeadingRateDegPerSecond  float64
	MaxTiltRateDegPerSecond     float64
	MaxLogDistanceRatePerSecond float64
	MaxTargetSpeedInDistances   float64
	GaussianSigmaSeconds        float64

	// The overview pose used by the opening and the closing.
	OverviewTiltDegrees        float64
	OverviewVerticalFOVDegrees float64
	OverviewMargin             float64
	OverviewMinDistanceFactor  float64

	// Minimum durations, in video time, of the following phase and of the
	// opening/closing phases.
	MinFollowDuration time.Duration
	MinPhaseDuration  time.Duration

	// Track limits: minimum length (distance travelled) and maximum span.
	MinTrackLengthMeters float64
	MaxTrackSpanMeters   float64

	// The automatic duration: AutoDurationBase + AutoDurationPerSqrtKm times
	// the square root of the track length in km, clamped to
	// [AutoDurationMin, AutoDurationMax].
	AutoDurationBase      time.Duration
	AutoDurationPerSqrtKm time.Duration
	AutoDurationMin       time.Duration
	AutoDurationMax       time.Duration

	// Per-level tables, indexed by Level.
	BaseDistanceMeters [levelCount]float64
	LookAheadSeconds   [levelCount]float64
	TiltDegrees        [levelCount]float64
}

// FollowDistance is how far the camera flies from the marker while following
// it at the given distance level: a base distance plus the distance the
// marker covers, in the video, in the level's look-ahead time — so
// fast-moving markers stay in view.
func (t CameraTuning) FollowDistance(level Level, markerSpeed float64) float64 {
	return t.BaseDistanceMeters[level.index()] + t.LookAheadSeconds[level.index()]*markerSpeed
}

// MinimumDuration is the shortest video duration over route that still lets
// the opening and the closing reach the follow pose within the smoothness
// limits, and leaves enough time to follow the route. It is computed from
// conservative bounds that do not depend on the duration itself, and rounded
// up to a whole frame.
func (p PlanParameters) MinimumDuration(route Route, tuning CameraTuning) time.Duration {
	return p.minimumDuration(NewLocalPlane(route).ProjectRoute(route), tuning)
}

func (p PlanParameters) minimumDuration(route PlanarRoute, tuning CameraTuning) time.Duration {
	base := tuning.BaseDistanceMeters[p.Distance.index()]
	overview := route.OverviewView(0, tuning.OverviewMinDistanceFactor*base, tuning)
	followTilt := tuning.TiltDegrees[p.Tilt.index()]

	phase := math.Max(tuning.MinPhaseDuration.Seconds(), math.Max(
		smoothstepPeakSlope*math.Abs(overview.TiltDegrees-followTilt)/tuning.MaxTiltRateDegPerSecond,
		smoothstepPeakSlope*math.Abs(math.Log(overview.Distance/base))/tuning.MaxLogDistanceRatePerSecond,
	))

	followShare := 1 - tuning.OpeningFraction - tuning.ClosingFraction
	seconds := math.Max(
		math.Max(phase/tuning.OpeningFraction, phase/tuning.ClosingFraction),
		tuning.MinFollowDuration.Seconds()/followShare,
	)

	frames := math.Ceil(seconds*p.FrameRate - 1e-9)
	return time.Duration(math.Ceil(frames / p.FrameRate * float64(time.Second)))
}

// DefaultDuration is the video duration used when the user does not choose
// one: a base duration that grows with the square root of the route's length,
// clamped to the configured range, and never below MinimumDuration.
func (p PlanParameters) DefaultDuration(route Route, tuning CameraTuning) time.Duration {
	return p.defaultDuration(route.Length(), p.MinimumDuration(route, tuning), tuning)
}

func (p PlanParameters) defaultDuration(length float64, minimum time.Duration, tuning CameraTuning) time.Duration {
	km := length / 1000
	seconds := tuning.AutoDurationBase.Seconds() + tuning.AutoDurationPerSqrtKm.Seconds()*math.Sqrt(km)
	seconds = clamp(seconds, tuning.AutoDurationMin.Seconds(), tuning.AutoDurationMax.Seconds())

	return max(time.Duration(math.Round(seconds))*time.Second, minimum)
}

// resolveDuration returns the duration of the video and whether the user
// chose it: the requested one, if it is at least the minimum for the route,
// or else the default one.
func (p PlanParameters) resolveDuration(length float64, minimum time.Duration, tuning CameraTuning) (time.Duration, DurationMode, error) {
	if p.Duration == nil {
		return p.defaultDuration(length, minimum, tuning), DurationModeAutomatic, nil
	}

	if *p.Duration < minimum {
		return 0, "", fmt.Errorf("%w: %s requested, minimum for this track is %.2f s", ErrDurationTooShort, *p.Duration, roundUpToCentiseconds(minimum))
	}
	return *p.Duration, DurationModeExplicit, nil
}

func roundUpToCentiseconds(d time.Duration) float64 {
	return math.Ceil(d.Seconds()*100-1e-9) / 100
}
