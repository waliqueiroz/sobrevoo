package pngfile

import (
	"errors"
	"fmt"
	"io"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/atomicfile"
)

// FrameExporter implements domain.FrameExporter: one frame, as a PNG file.
type FrameExporter struct{}

// NewFrameExporter creates a FrameExporter.
func NewFrameExporter() FrameExporter {
	return FrameExporter{}
}

// Export writes the frame to path, atomically: a failure never leaves a partial
// file — nor spoils the one that was there —, and without overwrite an existing
// path is refused.
func (FrameExporter) Export(image domain.FrameImage, mark domain.FrameMark, path string, overwrite bool) error {
	data, err := encodeFrame(image, mark)
	if err != nil {
		return fmt.Errorf("encoding the frame: %w", err)
	}

	err = atomicfile.Publish(path, overwrite, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})

	switch {
	case err == nil:
		return nil
	case errors.Is(err, atomicfile.ErrExists):
		return fmt.Errorf("%w: %s (use --overwrite to replace it)", domain.ErrFrameDestinationExists, path)
	default:
		return fmt.Errorf("%w: %s: %w", domain.ErrFrameDestinationInvalid, path, err)
	}
}
