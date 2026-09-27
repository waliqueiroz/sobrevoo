// Package atomicfile publishes a file so that it either appears complete or
// does not appear at all. It is shared by the outbound adapters that write a
// file the user asked for (the camera plan and the geo data slice).
package atomicfile

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

var (
	// ErrExists reports a destination that already exists, when overwriting
	// was not requested.
	ErrExists = errors.New("destination already exists")

	// ErrInvalid reports a destination that cannot be written (missing
	// directory, no permission, ...) or content that could not be produced.
	ErrInvalid = errors.New("destination is not writable")
)

// link is os.Link; a variable only so tests can simulate a file system that
// cannot make hard links.
var link = os.Link

// syncFile flushes a file to stable storage; a variable only so tests can
// simulate a failing disk.
var syncFile = func(f *os.File) error { return f.Sync() }

// Publish writes the content produced by write to path. The content goes to
// a temporary file in the same directory first and is published only once
// complete, so a failure never leaves a partial file. Without overwrite, an
// existing path is refused atomically (there is no window between checking
// and creating): the error wraps ErrExists. Any other failure to write wraps
// ErrInvalid; an error returned by write itself is returned as it is.
func Publish(path string, overwrite bool, write func(io.Writer) error) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".sobrevoo-*.tmp")
	if err != nil {
		return invalid(path, err)
	}
	defer os.Remove(temporary.Name())

	if err := write(temporary); err != nil {
		temporary.Close()
		return err
	}
	// Flush the content before the rename makes the file visible: after a
	// power failure the final name must never hold an empty file.
	if err := syncFile(temporary); err != nil {
		temporary.Close()
		return invalid(path, err)
	}
	if err := temporary.Close(); err != nil {
		return invalid(path, err)
	}
	if err := os.Chmod(temporary.Name(), 0o644); err != nil {
		return invalid(path, err)
	}

	if overwrite {
		if err := os.Rename(temporary.Name(), path); err != nil {
			return invalid(path, err)
		}
		return nil
	}

	return publishExclusive(temporary.Name(), path)
}

// publishExclusive makes the temporary file appear at path, failing with
// ErrExists if path already exists. A hard link does this atomically; on file
// systems that cannot link, it falls back to creating the destination
// exclusively and copying the content into it, removing it again if the copy
// fails.
func publishExclusive(temporary, path string) error {
	err := link(temporary, path)
	if err == nil {
		return nil
	}
	if errors.Is(err, fs.ErrExist) {
		return exists(path)
	}

	// Linking failed for another reason: create the destination directly,
	// still exclusively.
	destination, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if openErr != nil {
		if errors.Is(openErr, fs.ErrExist) {
			return exists(path)
		}
		return invalid(path, openErr)
	}

	source, openErr := os.Open(temporary)
	if openErr != nil {
		destination.Close()
		os.Remove(path)
		return invalid(path, openErr)
	}
	defer source.Close()

	if _, copyErr := io.Copy(destination, source); copyErr != nil {
		destination.Close()
		os.Remove(path)
		return invalid(path, copyErr)
	}
	if closeErr := destination.Close(); closeErr != nil {
		os.Remove(path)
		return invalid(path, closeErr)
	}
	return nil
}

func exists(path string) error {
	return fmt.Errorf("%w: %s", ErrExists, path)
}

func invalid(_ string, cause error) error {
	return fmt.Errorf("%w: %w", ErrInvalid, cause)
}
