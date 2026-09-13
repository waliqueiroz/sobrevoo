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
		root := cli.NewRootCommand()

		// when
		use := root.Use

		// then
		assert.Equal(t, "sobrevoo", use)
	})

	t.Run("should print its description when run with --help", func(t *testing.T) {
		// given
		root := cli.NewRootCommand()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs([]string{"--help"})

		// when
		err := root.Execute()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "flyover")
	})
}
