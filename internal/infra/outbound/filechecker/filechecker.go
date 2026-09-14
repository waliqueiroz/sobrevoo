// Package filechecker implements the domain.FileChecker port using
// os.Stat.
package filechecker

import "os"

// FileChecker implements domain.FileChecker using os.Stat.
type FileChecker struct{}

// New creates a FileChecker.
func New() FileChecker {
	return FileChecker{}
}

// Exists reports whether a file (or directory) is currently present at
// path.
func (FileChecker) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
