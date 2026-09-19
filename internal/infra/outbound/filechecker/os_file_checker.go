// Package filechecker implements the domain.FileChecker port using
// os.Stat.
package filechecker

import "os"

// OS implements domain.FileChecker using os.Stat.
type OS struct{}

// NewOS creates an OS file checker.
func NewOS() OS {
	return OS{}
}

// Exists reports whether a file (or directory) is currently present at
// path.
func (OS) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
