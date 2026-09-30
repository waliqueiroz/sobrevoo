package cli

import (
	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// The help text of the two source-selection flags shared by "geodata
// check", "geodata slice" and "fly" (010-geo-data-source-control): same
// names, same effect, wherever they appear.
const (
	baseMapNameUsage   = "Name of the registered base map source to use exclusively (default: automatic selection by area)"
	elevationNameUsage = "Name of the registered elevation source to use exclusively (default: automatic selection by area)"
)

// parseSourceSelection reads the two source-selection flags, leaving a field
// nil when its flag was not changed (automatic selection). Unlike
// parseAppearance/parseOverlay, it never fails: whether a requested name
// exists and is of the right type is validated later, against the
// registry, by domain.SourceSelection.Resolve.
func parseSourceSelection(cmd *cobra.Command, baseMapFlag, elevationFlag string) domain.SourceSelection {
	var selection domain.SourceSelection

	if cmd.Flags().Changed("base-map") {
		selection.BaseMapName = &baseMapFlag
	}
	if cmd.Flags().Changed("elevation") {
		selection.ElevationName = &elevationFlag
	}

	return selection
}
