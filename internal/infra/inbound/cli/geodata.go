package cli

import "github.com/spf13/cobra"

// NewGeoDataCommand creates the "geodata" command, which groups the
// register/list/remove/check subcommands (this feature's four use cases)
// under a single namespace, the same way "inspect" hangs off the root
// command today — keeping room for future stages (camera, rendering) to
// add their own command groups without colliding names.
func NewGeoDataCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "geodata",
		Short: "Manage the local geographic data registry (base maps and elevation)",
		// See root.go: error presentation and exit codes are handled
		// entirely by the composition root.
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	return cmd
}
