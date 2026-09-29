package domain

import "fmt"

// OverlayBlock names one of the four independently toggleable blocks a
// screen overlay can show (009-frame-overlays FR-004, Clarifications).
type OverlayBlock string

const (
	OverlayBlockDistance  OverlayBlock = "distance"
	OverlayBlockElevation OverlayBlock = "elevation"
	OverlayBlockTime      OverlayBlock = "time"
	OverlayBlockProfile   OverlayBlock = "profile"
)

// OverlayConfig is what the user chose about the screen overlays: whether
// they are drawn at all, and — when they are — which of the four blocks
// show. It does not include the hatch of "no map" or the checkerboard of
// "no elevation" (fixed, part of RenderTuning), nor anything about the
// trail or the marker drawn on the terrain (Appearance) — it is only about
// what is drawn fixed on the screen, on top of everything else.
type OverlayConfig struct {
	Enabled bool

	Distance  bool
	Elevation bool
	Time      bool
	Profile   bool
}

// NewOverlayConfig checks every element of blocks is one of the four
// documented names (ErrInvalidOverlayBlock, citing the one that is not) and
// turns on the corresponding field. When enabled is false the four fields
// are all false regardless of blocks — turning the overlays off is turning
// them all off, whatever was asked.
func NewOverlayConfig(enabled bool, blocks []OverlayBlock) (OverlayConfig, error) {
	config := OverlayConfig{Enabled: enabled}
	if !enabled {
		return config, nil
	}

	for _, block := range blocks {
		switch block {
		case OverlayBlockDistance:
			config.Distance = true
		case OverlayBlockElevation:
			config.Elevation = true
		case OverlayBlockTime:
			config.Time = true
		case OverlayBlockProfile:
			config.Profile = true
		default:
			return OverlayConfig{}, fmt.Errorf("%w: %q, expected one of distance, elevation, time, profile", ErrInvalidOverlayBlock, block)
		}
	}

	return config, nil
}

// Fingerprint is a canonical text of the five fields, which takes part in
// the identification of a set of frames (FrameSetID), the same way
// Appearance.Fingerprint and RenderTuning.Fingerprint already do.
func (o OverlayConfig) Fingerprint() string {
	flag := func(v bool) string {
		if v {
			return "1"
		}
		return "0"
	}

	return flag(o.Enabled) + "|" +
		flag(o.Distance) + "|" +
		flag(o.Elevation) + "|" +
		flag(o.Time) + "|" +
		flag(o.Profile)
}
