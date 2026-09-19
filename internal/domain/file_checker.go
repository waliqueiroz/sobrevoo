package domain

//go:generate go run go.uber.org/mock/mockgen -destination mock_domain/file_checker.go . FileChecker

// FileChecker reports whether a file still exists at a given path.
// Implemented by internal/infra/outbound/filechecker. Used to detect a
// registered source whose file has been moved or deleted (FR-010, FR-017)
// without requiring the core to import "os" directly (Constitution
// Principles I and II).
type FileChecker interface {
	Exists(path string) bool
}
