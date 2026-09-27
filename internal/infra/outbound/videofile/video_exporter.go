// Package videofile implements the outbound adapter that keeps a video in a file
// the user chose: it publishes the file whole or not at all, and never spoils
// the one that was there.
package videofile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/atomicfile"
)

// VideoExporter implements domain.VideoExporter: one video, as a file.
type VideoExporter struct{}

// NewVideoExporter creates a VideoExporter.
func NewVideoExporter() VideoExporter {
	return VideoExporter{}
}

// Check says, before anything is encoded, whether the video can be written to
// path, and leaves nothing behind.
func (VideoExporter) Check(path string, overwrite bool) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return invalid(path, err)
	}

	info, err := os.Stat(absolute)
	switch {
	case err == nil && info.IsDir():
		return fmt.Errorf("%w: %s is a directory", domain.ErrVideoDestinationInvalid, path)
	case err == nil && !overwrite:
		return exists(path)
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return invalid(path, err)
	}

	folder := filepath.Dir(absolute)
	folderInfo, err := os.Stat(folder)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("%w: %s: the folder does not exist", domain.ErrVideoDestinationInvalid, path)
	case err != nil:
		return invalid(path, err)
	case !folderInfo.IsDir():
		return fmt.Errorf("%w: %s: the folder is not a directory", domain.ErrVideoDestinationInvalid, path)
	}

	// Whether the folder can be written to is found out by writing to it.
	probe, err := os.CreateTemp(folder, ".sobrevoo-*.tmp")
	if err != nil {
		return invalid(path, err)
	}
	probe.Close()
	if err := os.Remove(probe.Name()); err != nil {
		return invalid(path, err)
	}
	return nil
}

// Export gives produce the path of a temporary file next to the destination and
// publishes what produce wrote, all at once.
func (VideoExporter) Export(path string, overwrite bool, produce func(temporary string) error) (int64, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return 0, invalid(path, err)
	}

	err = atomicfile.PublishPath(absolute, overwrite, produce)
	switch {
	case err == nil:
	case errors.Is(err, atomicfile.ErrExists):
		return 0, exists(path)
	case errors.Is(err, atomicfile.ErrInvalid):
		return 0, invalid(path, err)
	default:
		return 0, err
	}

	info, err := os.Stat(absolute)
	if err != nil {
		return 0, invalid(path, err)
	}
	return info.Size(), nil
}

func exists(path string) error {
	return fmt.Errorf("%w: %s; use --overwrite to replace it", domain.ErrVideoDestinationExists, path)
}

func invalid(path string, cause error) error {
	return fmt.Errorf("%w: %s: %w", domain.ErrVideoDestinationInvalid, path, cause)
}
