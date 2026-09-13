// Package config implements the adapter responsible for resolving Sobrevoo's
// internal configuration values (Constitution Principle VIII — configuration
// is injected from an adapter, never read directly by the core). At this
// stage there is no external configuration source (env var or file); Load
// returns fixed defaults, but callers already receive them through this
// adapter so a future configuration source can be introduced without
// changing the application layer's signature.
package config

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// Config holds the internal thresholds used by the track treatment pipeline
// (research.md items 7 and 9).
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
	DefaultLevel domain.Level
}

// Load returns Sobrevoo's configuration.
func Load() Config {
	return Config{
		MinPoints:            2,
		MaxPlausibleSpeedKmh: 130,
		DefaultLevel:         domain.LevelMedium,
	}
}
