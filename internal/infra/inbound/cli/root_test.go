package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

func Test_NewRootCommand(t *testing.T) {
	t.Run("should expose a sobrevoo root command", func(t *testing.T) {
		// given
		root := cli.NewRootCommand("v0.0.0-test")

		// when
		use := root.Use

		// then
		assert.Equal(t, "sobrevoo", use)
	})

	t.Run("should print its description when run with --help", func(t *testing.T) {
		// given
		root := cli.NewRootCommand("v0.0.0-test")
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs([]string{"--help"})

		// when
		err := root.Execute()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "flyover")
	})

	t.Run("should print the program name and version, nothing else, and exit 0 when run with --version", func(t *testing.T) {
		// given
		root := cli.NewRootCommand("v0.1.0")
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs([]string{"--version"})

		// when
		err := root.Execute()

		// then
		require.NoError(t, err)
		assert.Equal(t, "sobrevoo v0.1.0\n", out.String())
	})
}
