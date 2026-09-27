package domain

//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/workspace.go -package mockdomain . Workspace

// Workspace provides the directory a single-command run without a kept
// intermediates directory draws its frames into: FrameService and VideoService
// need a real directory on disk (the encoder reads the frames from it), but the
// plan and the geo data slice, passed between services as values, never do. It
// does not belong to a single entity, so it gets its own file instead of being
// declared alongside one.
type Workspace interface {
	// NewTemporary creates a fresh, empty directory for this run's exclusive
	// use, and the function that removes it, and everything inside it,
	// afterward. remove MUST be called exactly once, however the run ends.
	NewTemporary() (path string, remove func() error, err error)

	// EnsureDirectory creates path, and any missing parent directory, unless
	// it already exists as a directory — the same way a kept intermediates
	// directory the user has not used before needs to come into being before
	// anything can be written under it.
	EnsureDirectory(path string) error
}
