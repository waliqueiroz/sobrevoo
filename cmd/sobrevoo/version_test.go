package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_pickVersion(t *testing.T) {
	t.Run("should return the module version when no build-time version is set", func(t *testing.T) {
		// given
		buildOverride := ""
		moduleVersion := "v0.1.0"

		// when
		version := pickVersion(buildOverride, moduleVersion)

		// then
		assert.Equal(t, "v0.1.0", version)
	})

	t.Run("should return the development version when neither a build-time version nor a module version is set", func(t *testing.T) {
		// given
		buildOverride := ""
		moduleVersion := ""

		// when
		version := pickVersion(buildOverride, moduleVersion)

		// then
		assert.Equal(t, devVersion, version)
	})

	t.Run("should prefer the build-time version over the module version when both are set", func(t *testing.T) {
		// given
		buildOverride := "v0.2.0"
		moduleVersion := "v0.1.0"

		// when
		version := pickVersion(buildOverride, moduleVersion)

		// then
		assert.Equal(t, "v0.2.0", version)
	})
}
