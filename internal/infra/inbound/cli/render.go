package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// warnIfFramingCut tells, on out, that the images are narrower than the video
// the plan was made for, which cuts the sides of the opening and the closing. It
// is only a warning: drawing at such a resolution is allowed (a quick check, a
// different crop).
func warnIfFramingCut(out io.Writer, plan domain.CameraPlan, resolution domain.Resolution) {
	if !resolution.NarrowerThan(plan.Parameters.Aspect) {
		return
	}

	fmt.Fprintf(out, "Warning: the plan was made for a %s video, but the resolution is %s, which is narrower: the opening and the closing may be cut at the sides; plan again with --aspect, or draw at a resolution of that shape\n",
		plan.Parameters.Aspect, formatResolution(resolution))
}

// NewRenderCommand creates the "render" command, which groups the commands that
// draw the frames of a flight: "render frame" draws one, "render all" draws
// them all.
func NewRenderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Draw the frames of a flight from a camera plan and its geo data slice",
		// See root.go: error presentation and exit codes are handled
		// entirely by the composition root.
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	return cmd
}
