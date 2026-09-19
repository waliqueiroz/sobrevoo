// Package filechecker implements the domain.FileChecker port using
// os.Stat.
package filechecker

import "os"

// OSFileChecker implements domain.FileChecker using os.Stat.
type OSFileChecker struct{}

// NewOSFileChecker creates an OSFileChecker.
func NewOSFileChecker() OSFileChecker {
	return OSFileChecker{}
}

// Exists reports whether a file (or directory) is currently present at
// path.
func (OSFileChecker) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
