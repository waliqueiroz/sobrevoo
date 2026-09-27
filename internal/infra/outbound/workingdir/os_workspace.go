// Package workingdir implements the domain.Workspace port using the
// operating system's own temporary-directory facilities.
package workingdir

import "os"

// OS implements domain.Workspace using os.MkdirTemp/os.RemoveAll.
type OS struct{}

// NewOS creates an OS workspace.
func NewOS() OS {
	return OS{}
}

// NewTemporary creates a fresh, empty directory under the system's temporary
// directory, and a function that removes it, and everything inside it,
// afterward. Calling the removal function more than once is not an error
// (os.RemoveAll already is idempotent).
func (OS) NewTemporary() (string, func() error, error) {
	path, err := os.MkdirTemp("", "sobrevoo-fly-*")
	if err != nil {
		return "", nil, err
	}
	return path, func() error { return os.RemoveAll(path) }, nil
}

// EnsureDirectory creates path, and any missing parent, with os.MkdirAll,
// which already does nothing when path exists as a directory, and already
// fails when it exists as something else.
func (OS) EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0o755)
}
