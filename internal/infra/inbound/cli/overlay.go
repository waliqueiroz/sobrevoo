package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// The help text of the two overlay flags shared by "render frame", "render
// all" and "fly" (009-frame-overlays): same names, same values accepted,
// same defaults, wherever they appear.
const (
	overlaysUsage      = "Whether the screen overlays (distance, elevation, time, elevation profile) are drawn at all"
	overlayBlocksUsage = "Comma-separated overlay blocks to show, when --overlays is not false: distance, elevation, time, profile"
)

// parseOverlay reads the two overlay flags, falling back to defaults for the
// ones the user did not change. An unknown block name in --overlay-blocks is
// ErrInvalidOverlayBlock directly (like a malformed --trail-color already is
// for ErrInvalidColor); --overlays itself is a plain boolean flag, so Cobra
// already rejects a value that is not true/false as a usage error before
// this ever runs.
func parseOverlay(cmd *cobra.Command, overlaysFlag bool, blocksFlag string, defaults domain.OverlayConfig) (domain.OverlayConfig, error) {
	enabled := defaults.Enabled
	if cmd.Flags().Changed("overlays") {
		enabled = overlaysFlag
	}

	blocks := overlayBlocksOf(defaults)
	if cmd.Flags().Changed("overlay-blocks") {
		blocks = nil
		for _, name := range strings.Split(blocksFlag, ",") {
			blocks = append(blocks, domain.OverlayBlock(strings.TrimSpace(name)))
		}
	}

	return domain.NewOverlayConfig(enabled, blocks)
}

// overlayBlocksOf lists the blocks config has on, in the fixed order
// formatOverlayBlocks displays them.
func overlayBlocksOf(config domain.OverlayConfig) []domain.OverlayBlock {
	var blocks []domain.OverlayBlock
	if config.Distance {
		blocks = append(blocks, domain.OverlayBlockDistance)
	}
	if config.Elevation {
		blocks = append(blocks, domain.OverlayBlockElevation)
	}
	if config.Time {
		blocks = append(blocks, domain.OverlayBlockTime)
	}
	if config.Profile {
		blocks = append(blocks, domain.OverlayBlockProfile)
	}
	return blocks
}

// formatOverlayBlocks renders config's blocks as a comma-separated list —
// the format --overlay-blocks accepts back — used to show the default in
// --help.
func formatOverlayBlocks(config domain.OverlayConfig) string {
	blocks := overlayBlocksOf(config)
	names := make([]string, len(blocks))
	for i, b := range blocks {
		names[i] = string(b)
	}
	return strings.Join(names, ",")
}
