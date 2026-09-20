package jsonfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// CameraPlanExporter implements domain.CameraPlanExporter, writing a camera
// plan as a JSON file (specs/003-camera-path-planning/contracts/plan-file.md).
type CameraPlanExporter struct{}

// NewCameraPlanExporter creates a CameraPlanExporter.
func NewCameraPlanExporter() CameraPlanExporter {
	return CameraPlanExporter{}
}

// Export writes plan to path. The content goes to a temporary file in the
// same directory first and is published only once complete, so a failure
// never leaves a partial file. Without overwrite, an existing path is
// refused atomically (there is no window between checking and creating).
func (CameraPlanExporter) Export(plan domain.CameraPlan, path string, overwrite bool) error {
	data, err := encodePlan(plan)
	if err != nil {
		return fmt.Errorf("encoding the plan: %w", err)
	}

	temporary, err := os.CreateTemp(filepath.Dir(path), ".sobrevoo-plan-*.tmp")
	if err != nil {
		return invalidDestination(path, err)
	}
	defer os.Remove(temporary.Name())

	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return invalidDestination(path, err)
	}
	if err := temporary.Close(); err != nil {
		return invalidDestination(path, err)
	}
	if err := os.Chmod(temporary.Name(), 0o644); err != nil {
		return invalidDestination(path, err)
	}

	if overwrite {
		if err := os.Rename(temporary.Name(), path); err != nil {
			return invalidDestination(path, err)
		}
		return nil
	}

	return publishExclusive(temporary.Name(), path, data)
}

// publishExclusive makes the temporary file appear at path, failing with
// ErrPlanDestinationExists if path already exists. A hard link does this
// atomically; on file systems that cannot link, it falls back to creating the
// destination exclusively and writing the data into it, removing it again if
// the write fails.
func publishExclusive(temporary, path string, data []byte) error {
	err := os.Link(temporary, path)
	if err == nil {
		return nil
	}
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("%w: %s (use --overwrite to replace it)", domain.ErrPlanDestinationExists, path)
	}

	// Linking failed for another reason (typically an unsupported file
	// system): create the destination directly, still exclusively.
	file, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if openErr != nil {
		if errors.Is(openErr, fs.ErrExist) {
			return fmt.Errorf("%w: %s (use --overwrite to replace it)", domain.ErrPlanDestinationExists, path)
		}
		return invalidDestination(path, openErr)
	}
	if _, writeErr := file.Write(data); writeErr != nil {
		file.Close()
		os.Remove(path)
		return invalidDestination(path, writeErr)
	}
	if closeErr := file.Close(); closeErr != nil {
		os.Remove(path)
		return invalidDestination(path, closeErr)
	}
	return nil
}

func invalidDestination(path string, cause error) error {
	return fmt.Errorf("%w: %s: %w", domain.ErrPlanDestinationInvalid, path, cause)
}
