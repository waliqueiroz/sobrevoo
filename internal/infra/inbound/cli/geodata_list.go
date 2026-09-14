package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
)

// NewGeoDataListCommand creates the "geodata list" command, which exposes
// ListGeoDataService (FR-009, FR-010).
func NewGeoDataListCommand(listGeoDataService application.ListGeoDataService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List every registered geo data source",
		// See root.go: error presentation and exit codes are handled
		// entirely by the composition root.
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGeoDataList(cmd, listGeoDataService)
		},
	}

	return cmd
}

func runGeoDataList(cmd *cobra.Command, listGeoDataService application.ListGeoDataService) error {
	output, err := listGeoDataService.Execute()
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.OutOrStdout(), formatGeoDataList(output))

	return nil
}

// formatGeoDataList renders a ListGeoDataOutput as the English,
// human-readable listing described in contracts/cli.md (FR-009, FR-010).
func formatGeoDataList(output application.ListGeoDataOutput) string {
	if len(output.Sources) == 0 {
		return "No geo data registered.\n"
	}

	var b strings.Builder
	for _, summary := range output.Sources {
		fmt.Fprintf(&b, "%s (%s): %s", summary.Source.Name, summary.Source.Type, formatBoundingBox(summary.Source.BoundingBox))
		if !summary.Available {
			fmt.Fprint(&b, " (file not found)")
		}
		fmt.Fprintln(&b)
	}

	return b.String()
}
