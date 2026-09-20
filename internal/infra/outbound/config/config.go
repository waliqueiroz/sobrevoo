// Package config implements the adapter responsible for resolving Sobrevoo's
// internal configuration values (Constitution Principle VIII — configuration
// is injected from an adapter, never read directly by the core). At this
// stage there is no external configuration source (env var or file); Load
// returns fixed defaults, but callers already receive them through this
// adapter so a future configuration source can be introduced without
// changing the application layer's signature.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// registryFileName is the geo data registry's file name inside the
// directory returned by os.UserHomeDir() (research.md item 5).
const registryDir = ".sobrevoo"
const registryFileName = "registry.json"

// Level is a low/medium/high choice, as the configuration expresses it. It
// belongs to this package, not to the domain: the composition root maps it
// to domain.Level where it is needed.
type Level string

const (
	LevelLow    Level = "low"
	LevelMedium Level = "medium"
	LevelHigh   Level = "high"
)

// LevelValues holds one value per Level.
type LevelValues struct {
	Low    float64
	Medium float64
	High   float64
}

// CameraTuning holds the heuristic constants of camera planning (the third
// stage): smoothness limits, opening/closing, stop compression, the automatic
// duration and the per-level tables. Initial values, meant to be adjusted once
// rendering shows how the flight really looks
// (specs/003-camera-path-planning/research.md, item 12). The composition root
// maps it to the domain's own CameraTuning.
type CameraTuning struct {
	OpeningFraction float64
	ClosingFraction float64

	StopSpeedMetersPerSecond float64
	StopMinDuration          time.Duration
	StopCappedDuration       time.Duration
	StopMaxShareOfMovingTime float64

	MaxHeadingRateDegPerSecond  float64
	MaxTiltRateDegPerSecond     float64
	MaxLogDistanceRatePerSecond float64
	MaxTargetSpeedInDistances   float64
	GaussianSigmaSeconds        float64

	OverviewTiltDegrees        float64
	OverviewVerticalFOVDegrees float64
	OverviewMargin             float64
	OverviewMinDistanceFactor  float64

	MinFollowDuration time.Duration
	MinPhaseDuration  time.Duration

	MinTrackLengthMeters float64
	MaxTrackSpanMeters   float64

	AutoDurationBase      time.Duration
	AutoDurationPerSqrtKm time.Duration
	AutoDurationMin       time.Duration
	AutoDurationMax       time.Duration

	BaseDistanceMeters LevelValues
	LookAheadSeconds   LevelValues
	TiltDegrees        LevelValues
}

// PlanDefaults are the camera plan parameters used when the user does not
// choose them: 30 frames per second and medium distance and tilt. There is no
// default duration: when the user gives none, it is computed from the track.
type PlanDefaults struct {
	FrameRate float64
	Distance  Level
	Tilt      Level
}

// Config holds the internal thresholds used by the track treatment pipeline
// (research.md items 7 and 9), plus the resolved locations Sobrevoo persists
// state to.
type Config struct {
	// MinPoints is the minimum number of points a track must have — both
	// before and after cleaning — to compute a summary (FR-006).
	MinPoints int

	// MaxPlausibleSpeedKmh is the maximum speed, in km/h, considered
	// physically plausible between two consecutive points for a running,
	// cycling or walking activity. Points implying a higher speed are
	// discarded as implausible jumps (FR-010).
	MaxPlausibleSpeedKmh float64

	// DefaultLevel is the simplification/smoothing level applied when the
	// user does not specify one explicitly (FR-016).
	DefaultLevel Level

	// RegistryPath is where the geo data registry is persisted: a single
	// JSON file at a fixed path — the same regardless of the current
	// working directory (FR-008) — under the user's home directory
	// (~/.sobrevoo/registry.json), independent of the host OS's own
	// configuration-directory convention (research.md item 5).
	RegistryPath string

	// CameraTuning holds the constants of camera planning.
	CameraTuning CameraTuning

	// PlanDefaults are the camera plan parameters used when the user does not
	// choose them. They are distinct from DefaultLevel, which is the
	// treatment level of the track.
	PlanDefaults PlanDefaults
}

// Load returns Sobrevoo's configuration. It fails only when the user's home
// directory cannot be resolved (os.UserHomeDir()), which RegistryPath
// depends on.
func Load() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("resolving home directory: %w", err)
	}

	return Config{
		MinPoints:            2,
		MaxPlausibleSpeedKmh: 130,
		DefaultLevel:         LevelMedium,
		RegistryPath:         filepath.Join(home, registryDir, registryFileName),
		CameraTuning:         cameraTuning(),
		PlanDefaults:         PlanDefaults{FrameRate: 30, Distance: LevelMedium, Tilt: LevelMedium},
	}, nil
}

func cameraTuning() CameraTuning {
	return CameraTuning{
		OpeningFraction: 0.10,
		ClosingFraction: 0.10,

		StopSpeedMetersPerSecond: 0.5,
		StopMinDuration:          30 * time.Second,
		StopCappedDuration:       2 * time.Second,
		StopMaxShareOfMovingTime: 0.05,

		MaxHeadingRateDegPerSecond:  45,
		MaxTiltRateDegPerSecond:     30,
		MaxLogDistanceRatePerSecond: 1.5,
		MaxTargetSpeedInDistances:   1.0,
		GaussianSigmaSeconds:        0.5,

		OverviewTiltDegrees:        60,
		OverviewVerticalFOVDegrees: 45,
		OverviewMargin:             1.2,
		OverviewMinDistanceFactor:  2,

		MinFollowDuration: 5 * time.Second,
		MinPhaseDuration:  2 * time.Second,

		MinTrackLengthMeters: 50,
		MaxTrackSpanMeters:   2_000_000,

		AutoDurationBase:      15 * time.Second,
		AutoDurationPerSqrtKm: 6 * time.Second,
		AutoDurationMin:       20 * time.Second,
		AutoDurationMax:       120 * time.Second,

		BaseDistanceMeters: LevelValues{Low: 300, Medium: 600, High: 1200},
		LookAheadSeconds:   LevelValues{Low: 2, Medium: 4, High: 8},
		TiltDegrees:        LevelValues{Low: 25, Medium: 45, High: 65},
	}
}
