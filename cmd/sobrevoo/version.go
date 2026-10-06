package main

import "runtime/debug"

// version is empty by default. A release build published outside the
// `go install module@vX.Y.Z` flow can fix it explicitly at build time with
// `-ldflags "-X main.version=vX.Y.Z"`; when set, it takes precedence over
// the module version Go's toolchain already embeds in the binary.
var version string

// devVersion is reported when build info isn't available at all — e.g. a
// binary built with `-buildvcs=false`, without git installed, or without
// module support — so it never pretends to be a numbered release. A plain
// local build from a git checkout doesn't usually hit this path: Go's own
// VCS stamping already fills info.Main.Version with a pseudo-version
// (v0.0.0-<timestamp>-<commit>[+dirty]) in that case, which readModuleVersion
// reports as-is — just as honestly "not a tagged release" as this literal.
const devVersion = "(devel)"

// resolveVersion determines the version string `sobrevoo --version`
// reports: the build-time override, if any, otherwise the module version
// Go recorded in the binary, otherwise devVersion.
func resolveVersion() string {
	return pickVersion(version, readModuleVersion())
}

// readModuleVersion reads the version of the main module that Go's
// toolchain embeds in the binary — the tag used by `go install
// module@vX.Y.Z`, or, for a binary built locally from a git checkout
// without one, a VCS pseudo-version ("(devel)" only when VCS stamping
// itself is unavailable). It returns "" when build info isn't available
// at all.
func readModuleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	return info.Main.Version
}

// pickVersion decides which of the two candidate version strings prevails:
// buildOverride (fixed at build time) wins when set; otherwise
// moduleVersion; otherwise devVersion.
func pickVersion(buildOverride, moduleVersion string) string {
	if buildOverride != "" {
		return buildOverride
	}

	if moduleVersion != "" {
		return moduleVersion
	}

	return devVersion
}
