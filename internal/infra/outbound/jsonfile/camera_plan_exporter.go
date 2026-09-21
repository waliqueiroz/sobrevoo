package jsonfile

import (
	"errors"
	"fmt"
	"io"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/atomicfile"
)

// CameraPlanExporter implements domain.CameraPlanExporter, writing a camera
// plan as a JSON file (specs/003-camera-path-planning/contracts/plan-file.md).
type CameraPlanExporter struct{}

// NewCameraPlanExporter creates a CameraPlanExporter.
func NewCameraPlanExporter() CameraPlanExporter {
	return CameraPlanExporter{}
}

// Export writes plan to path, atomically: a failure never leaves a partial
// file, and without overwrite an existing path is refused.
func (CameraPlanExporter) Export(plan domain.CameraPlan, path string, overwrite bool) error {
	data, err := encodePlan(plan)
	if err != nil {
		return fmt.Errorf("encoding the plan: %w", err)
	}

	err = atomicfile.Publish(path, overwrite, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})

	switch {
	case err == nil:
		return nil
	case errors.Is(err, atomicfile.ErrExists):
		return fmt.Errorf("%w: %s (use --overwrite to replace it)", domain.ErrPlanDestinationExists, path)
	default:
		return fmt.Errorf("%w: %s: %w", domain.ErrPlanDestinationInvalid, path, err)
	}
}
