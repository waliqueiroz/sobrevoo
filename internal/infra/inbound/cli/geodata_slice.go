package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// NewGeoDataSliceCommand creates the "geodata slice" command, which loads an
// exported camera plan (CameraPlanService.Load) and gathers the geo data the
// flight needs (GeoSliceService, FR-001 through FR-021 of the fourth stage).
func NewGeoDataSliceCommand(cameraPlanService application.CameraPlanService, geoSliceService application.GeoSliceService) *cobra.Command {
	var exportFlag string
	var overwriteFlag bool
	var baseMapFlag, elevationFlag string

	cmd := &cobra.Command{
		Use:   "slice <plan-file>",
		Short: "Gather the map tiles and elevation samples a camera plan needs from the registered geo data",
		// See root.go: error presentation and exit codes are handled
		// entirely by the composition root.
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return newUsageError(fmt.Errorf("accepts exactly one plan file argument, received %d", len(args)))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if overwriteFlag && exportFlag == "" {
				return newUsageError(fmt.Errorf("--overwrite requires --export"))
			}

			selection := parseSourceSelection(cmd, baseMapFlag, elevationFlag)
			return runGeoDataSlice(cmd, cameraPlanService, geoSliceService, args[0], exportFlag, overwriteFlag, selection)
		},
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return newUsageError(err) })

	cmd.Flags().StringVar(&exportFlag, "export", "", "Write the complete slice to this file, as a ZIP")
	cmd.Flags().BoolVar(&overwriteFlag, "overwrite", false, "With --export, replace the file if it already exists")
	cmd.Flags().StringVar(&baseMapFlag, "base-map", "", baseMapNameUsage)
	cmd.Flags().StringVar(&elevationFlag, "elevation", "", elevationNameUsage)

	return cmd
}

func runGeoDataSlice(cmd *cobra.Command, cameraPlanService application.CameraPlanService, geoSliceService application.GeoSliceService, planPath, exportPath string, overwrite bool, selection domain.SourceSelection) error {
	plan, err := cameraPlanService.Load(planPath)
	if err != nil {
		return err
	}

	slice, err := geoSliceService.Generate(plan, selection)
	if err != nil {
		return err
	}

	// Export first: on failure nothing is printed, so an error never comes
	// with a summary (contracts/cli.md).
	if exportPath != "" {
		if err := geoSliceService.Export(slice, exportPath, overwrite); err != nil {
			return err
		}
	}

	fmt.Fprint(cmd.OutOrStdout(), formatSliceSummary(slice))
	if exportPath != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Slice written to %s\n", exportPath)
	}

	return nil
}

// maxListedMissingTiles is how many missing tiles the summary lists; the rest
// are only counted (the exported file lists them all).
const maxListedMissingTiles = 20

// formatSliceSummary renders a slice's summary as the English, human-readable
// text described in contracts/cli.md (FR-012).
func formatSliceSummary(slice domain.GeoSlice) string {
	summary := slice.Summary

	var b strings.Builder
	fmt.Fprintf(&b, "Area: %s\n", formatArea(summary.Area))

	for _, tileSet := range slice.TileSets {
		detail := tileSet.Detail
		fmt.Fprintf(&b, "Base map detail (%s): level %d (ideal %d, source offers %d-%d; %s)\n",
			tileSet.Source.Name, detail.Chosen, detail.Ideal, detail.Min, detail.Max, detail.Reason)
		fmt.Fprintf(&b, "  %s\n", detail.Explanation)
	}

	switch {
	case summary.TileCount == 0 && summary.MissingTileCount > 0:
		fmt.Fprintf(&b, "Map tiles: 0 present, %d missing (no imagery in this slice)\n", summary.MissingTileCount)
	case summary.MissingTileCount > 0:
		fmt.Fprintf(&b, "Map tiles: %d present, %d missing\n", summary.TileCount, summary.MissingTileCount)
	default:
		fmt.Fprintf(&b, "Map tiles: %d present\n", summary.TileCount)
	}
	b.WriteString(formatMissingTiles(slice))

	if summary.NoValueSampleCount > 0 {
		fmt.Fprintf(&b, "Elevation samples: %d (%d without value)\n", summary.SampleCount, summary.NoValueSampleCount)
	} else {
		fmt.Fprintf(&b, "Elevation samples: %d\n", summary.SampleCount)
	}
	if summary.HasElevationRange {
		fmt.Fprintf(&b, "Elevation range: %.1f m - %.1f m\n", summary.MinElevation, summary.MaxElevation)
	} else {
		b.WriteString("Elevation range: none (no sample has a value)\n")
	}

	b.WriteString("Sources:\n")
	for _, use := range summary.Sources {
		fmt.Fprintf(&b, "  %s (%s, %s)\n", use.Source.Name, use.Source.Type, use.Source.Format)
	}

	fmt.Fprintf(&b, "Size: %s\n", formatBytes(summary.SizeBytes))

	return b.String()
}

func formatArea(area domain.BoundingBox) string {
	text := fmt.Sprintf("lat %.4f to %.4f, lon %.4f to %.4f", area.MinLatitude, area.MaxLatitude, area.MinLongitude, area.MaxLongitude)
	if area.CrossesAntimeridian {
		text += " (crosses the antimeridian)"
	}
	return text
}

// formatMissingTiles lists the first missing tiles, saying which map each
// belongs to and where it is, and counts the rest.
func formatMissingTiles(slice domain.GeoSlice) string {
	var b strings.Builder

	listed := 0
	for _, tileSet := range slice.TileSets {
		for _, id := range tileSet.Missing {
			if listed == maxListedMissingTiles {
				break
			}
			fmt.Fprintf(&b, "  missing: %s level %d x=%d y=%d\n", tileSet.Source.Name, id.Level, id.X, id.Y)
			listed++
		}
	}

	if rest := slice.Summary.MissingTileCount - listed; rest > 0 {
		fmt.Fprintf(&b, "  ... and %d more (all listed in the exported file)\n", rest)
	}
	return b.String()
}

func formatBytes(size int64) string {
	const kib = 1024
	switch {
	case size < kib:
		return fmt.Sprintf("%d B", size)
	case size < kib*kib:
		return fmt.Sprintf("%.1f KiB", float64(size)/kib)
	case size < kib*kib*kib:
		return fmt.Sprintf("%.1f MiB", float64(size)/(kib*kib))
	default:
		return fmt.Sprintf("%.1f GiB", float64(size)/(kib*kib*kib))
	}
}
