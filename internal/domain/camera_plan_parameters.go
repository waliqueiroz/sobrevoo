package domain

import (
	"fmt"
	"math"
	"time"
)

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
