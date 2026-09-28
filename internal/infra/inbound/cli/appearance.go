package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// The help text of the five appearance flags shared by "render frame",
// "render all" and "fly" (008-frame-appearance): same names, same values
// accepted, same defaults, wherever they appear.
const (
	trailColorUsage      = "Color of the trail, as #RRGGBB"
	trailWidthUsage      = "Width of the trail, as a share of the frame's height, from 0.0005 to 0.05"
	markerColorUsage     = "Color of the marker, as #RRGGBB"
	markerRadiusUsage    = "Radius of the marker, as a share of the frame's height, from 0.001 to 0.1"
	backgroundColorUsage = "Color of the background, as #RRGGBB"
)

// parseAppearance reads the five appearance flags, falling back to defaults
// for the ones the user did not change. A malformed color is ErrInvalidColor
// directly (like --aspect already is for ErrInvalidAspectRatio); a
// --trail-width/--marker-radius that is not a finite number is a usage error
// (like --fps/--duration already are), and one that is a number outside the
// documented range is ErrInvalidTrailWidth/ErrInvalidMarkerRadius.
func parseAppearance(cmd *cobra.Command, trailColorFlag, trailWidthFlag, markerColorFlag, markerRadiusFlag, backgroundColorFlag string, defaults domain.Appearance) (domain.Appearance, error) {
	trailColor := defaults.TrailColor
	if cmd.Flags().Changed("trail-color") {
		var err error
		if trailColor, err = domain.ParseColor(trailColorFlag); err != nil {
			return domain.Appearance{}, err
		}
	}

	trailWidth := defaults.TrailWidthRatio
	if cmd.Flags().Changed("trail-width") {
		value, err := parseFiniteNumber(trailWidthFlag)
		if err != nil {
			return domain.Appearance{}, newUsageError(fmt.Errorf("--trail-width: %w", err))
		}
		trailWidth = value
	}

	markerColor := defaults.MarkerColor
	if cmd.Flags().Changed("marker-color") {
		var err error
		if markerColor, err = domain.ParseColor(markerColorFlag); err != nil {
			return domain.Appearance{}, err
		}
	}

	markerRadius := defaults.MarkerRadiusRatio
	if cmd.Flags().Changed("marker-radius") {
		value, err := parseFiniteNumber(markerRadiusFlag)
		if err != nil {
			return domain.Appearance{}, newUsageError(fmt.Errorf("--marker-radius: %w", err))
		}
		markerRadius = value
	}

	backgroundColor := defaults.BackgroundColor
	if cmd.Flags().Changed("background-color") {
		var err error
		if backgroundColor, err = domain.ParseColor(backgroundColorFlag); err != nil {
			return domain.Appearance{}, err
		}
	}

	return domain.NewAppearance(trailColor, trailWidth, markerColor, markerRadius, backgroundColor)
}

// formatColor renders a color as #RRGGBB, the format the five appearance
// flags accept back — used to show the default of each in --help.
func formatColor(c domain.RGB) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}
