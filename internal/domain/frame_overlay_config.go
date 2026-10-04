package domain

import "fmt"

// OverlayBlock names one of the independently toggleable blocks a screen
// overlay can show (009-frame-overlays FR-004, Clarifications).
type OverlayBlock string

const (
	OverlayBlockDistance  OverlayBlock = "distance"
	OverlayBlockElevation OverlayBlock = "elevation"
	OverlayBlockTime      OverlayBlock = "time"
	OverlayBlockProfile   OverlayBlock = "profile"

	// OverlayBlockSpeed is the fifth block (014-speed-overlay-block),
	// different from the original four in one way: it is not among the
	// default blocks a plain "overlays on" turns on — it only appears when
	// named explicitly in the list.
	OverlayBlockSpeed OverlayBlock = "speed"

	// OverlayBlockGain is the sixth block (015-overlay-redesign): the
	// elevation gain accumulated up to the marker, split out of
	// OverlayBlockElevation (which now shows only the altitude) into its
	// own block — like OverlayBlockSpeed, it is not among the default
	// blocks; it only appears when named explicitly.
	OverlayBlockGain OverlayBlock = "gain"
)

// overlayBlockOrder is the fixed, documented left-to-right order the top
// row's numeric blocks are drawn in, whatever order the user named them in
// --overlay-blocks (015-overlay-redesign Clarifications, Session
// 2026-10-04). OverlayBlockProfile is not part of it: the elevation
// profile is drawn separately, at the bottom of the frame.
var overlayBlockOrder = []OverlayBlock{
	OverlayBlockSpeed,
	OverlayBlockElevation,
	OverlayBlockDistance,
	OverlayBlockGain,
	OverlayBlockTime,
}

// OverlayConfig is what the user chose about the screen overlays: whether
// they are drawn at all, and — when they are — which blocks show. It does
// not include the hatch of "no map" or the checkerboard of "no elevation"
// (fixed, part of RenderTuning), nor anything about the trail or the
// marker drawn on the terrain (Appearance) — it is only about what is
// drawn fixed on the screen, on top of everything else.
type OverlayConfig struct {
	Enabled bool

	Distance  bool
	Elevation bool
	Gain      bool
	Time      bool
	Profile   bool
	Speed     bool
}

// NewOverlayConfig checks every element of blocks is one of the documented
// names (ErrInvalidOverlayBlock, citing the one that is not) and turns on
// the corresponding field. When enabled is false every field is false
// regardless of blocks — turning the overlays off is turning them all off,
// whatever was asked.
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
		case OverlayBlockGain:
			config.Gain = true
		case OverlayBlockProfile:
			config.Profile = true
		case OverlayBlockSpeed:
			config.Speed = true
		default:
			return OverlayConfig{}, fmt.Errorf("%w: %q, expected one of distance, elevation, gain, time, profile, speed", ErrInvalidOverlayBlock, block)
		}
	}

	return config, nil
}

// Fingerprint is a canonical text of the seven fields, which takes part in
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
		flag(o.Profile) + "|" +
		flag(o.Speed) + "|" +
		flag(o.Gain)
}
