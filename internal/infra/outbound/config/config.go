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

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// registryFileName is the geo data registry's file name inside the
// directory returned by os.UserHomeDir() (research.md item 5).
const registryDir = ".sobrevoo"
const registryFileName = "registry.json"

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
	DefaultLevel domain.Level

	// RegistryPath is where the geo data registry is persisted: a single
	// JSON file at a fixed path — the same regardless of the current
	// working directory (FR-008) — under the user's home directory
	// (~/.sobrevoo/registry.json), independent of the host OS's own
	// configuration-directory convention (research.md item 5).
	RegistryPath string
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
		DefaultLevel:         domain.LevelMedium,
		RegistryPath:         filepath.Join(home, registryDir, registryFileName),
	}, nil
}
