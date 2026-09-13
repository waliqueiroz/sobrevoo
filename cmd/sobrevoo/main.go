// Command sobrevoo is Sobrevoo's CLI entrypoint: the composition root that
// wires the concrete adapters (outbound and inbound) around the core use
// cases. This is the only place allowed to know about every concrete piece
// at once.
package main

import (
	"fmt"
	"os"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/config"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/simplifier/douglaspeucker"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/smoother/catmullrom"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/trackparser"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg := config.Load()

	parser := trackparser.NewGPXParser()
	simplifier := douglaspeucker.New()
	smoother := catmullrom.New()
	inspectTrackService := application.NewInspectTrackService(parser, simplifier, smoother, cfg.MinPoints, cfg.MaxPlausibleSpeedKmh)

	root := cli.NewRootCommand()
	root.AddCommand(cli.NewInspectCommand(inspectTrackService, cfg.DefaultLevel))

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitCode(err)
	}

	return 0
}
