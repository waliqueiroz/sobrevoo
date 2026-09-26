package cli_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

func Test_ExitCode(t *testing.T) {
	t.Run("should keep the codes of the first and second stages", func(t *testing.T) {
		// given
		expected := map[error]int{
			domain.ErrEmptyFile:                       1,
			domain.ErrUnsupportedFormat:               2,
			domain.ErrInsufficientPoints:              3,
			domain.ErrInsufficientPointsAfterCleaning: 3,
			domain.ErrDataFileNotFound:                5,
			domain.ErrDataFileUnreadable:              6,
			domain.ErrUnsupportedDataFormat:           7,
			domain.ErrDataSourceNameAlreadyUsed:       8,
			domain.ErrDataSourceNotRegistered:         9,
		}

		for err, code := range expected {
			// when / then
			assert.Equal(t, code, cli.ExitCode(err), err.Error())
		}
	})

	t.Run("should return 0 for no error and 4 for an unclassified error", func(t *testing.T) {
		// when / then
		assert.Equal(t, 0, cli.ExitCode(nil))
		assert.Equal(t, 4, cli.ExitCode(errors.New("boom")))
	})

	t.Run("should map ErrInvalidDuration to 10", func(t *testing.T) {
		assert.Equal(t, 10, cli.ExitCode(domain.ErrInvalidDuration))
	})

	t.Run("should map ErrInvalidFrameRate to 11", func(t *testing.T) {
		assert.Equal(t, 11, cli.ExitCode(domain.ErrInvalidFrameRate))
	})

	t.Run("should map ErrDurationTooShort to 12", func(t *testing.T) {
		assert.Equal(t, 12, cli.ExitCode(domain.ErrDurationTooShort))
	})

	t.Run("should map ErrTrackTooShort to 13", func(t *testing.T) {
		assert.Equal(t, 13, cli.ExitCode(domain.ErrTrackTooShort))
	})

	t.Run("should map ErrTrackTooLarge to 14", func(t *testing.T) {
		assert.Equal(t, 14, cli.ExitCode(domain.ErrTrackTooLarge))
	})

	t.Run("should map ErrPlanDestinationExists to 15", func(t *testing.T) {
		assert.Equal(t, 15, cli.ExitCode(domain.ErrPlanDestinationExists))
	})

	t.Run("should map ErrPlanDestinationInvalid to 16", func(t *testing.T) {
		assert.Equal(t, 16, cli.ExitCode(domain.ErrPlanDestinationInvalid))
	})

	t.Run("should recognize a sentinel wrapped with context", func(t *testing.T) {
		// given
		err := fmt.Errorf("%w: 20 s requested, minimum for this track is 38.80 s", domain.ErrDurationTooShort)

		// when / then
		assert.Equal(t, 12, cli.ExitCode(err))
	})

	t.Run("should map ErrPlanFileInvalid to 17", func(t *testing.T) {
		assert.Equal(t, 17, cli.ExitCode(domain.ErrPlanFileInvalid))
	})

	t.Run("should map ErrPlanFormatVersionUnsupported to 18", func(t *testing.T) {
		assert.Equal(t, 18, cli.ExitCode(domain.ErrPlanFormatVersionUnsupported))
	})

	t.Run("should map ErrAreaNotCovered to 19", func(t *testing.T) {
		assert.Equal(t, 19, cli.ExitCode(domain.ErrAreaNotCovered))
	})

	t.Run("should map ErrSliceTooLarge to 20", func(t *testing.T) {
		assert.Equal(t, 20, cli.ExitCode(domain.ErrSliceTooLarge))
	})

	t.Run("should map ErrGeoDataContentUnreadable to 21", func(t *testing.T) {
		assert.Equal(t, 21, cli.ExitCode(domain.ErrGeoDataContentUnreadable))
	})

	t.Run("should map ErrElevationUnitUnsupported to 22", func(t *testing.T) {
		assert.Equal(t, 22, cli.ExitCode(domain.ErrElevationUnitUnsupported))
	})

	t.Run("should map ErrSliceDestinationExists to 23", func(t *testing.T) {
		assert.Equal(t, 23, cli.ExitCode(domain.ErrSliceDestinationExists))
	})

	t.Run("should map ErrSliceDestinationInvalid to 24", func(t *testing.T) {
		assert.Equal(t, 24, cli.ExitCode(domain.ErrSliceDestinationInvalid))
	})

	t.Run("should map ErrElevationNotCovered to 25", func(t *testing.T) {
		assert.Equal(t, 25, cli.ExitCode(domain.ErrElevationNotCovered))
	})

	t.Run("should map ErrInvalidCoordinate to 26", func(t *testing.T) {
		assert.Equal(t, 26, cli.ExitCode(domain.ErrInvalidCoordinate))
	})

	t.Run("should map a wrapped fourth-stage error to its code", func(t *testing.T) {
		// given
		err := fmt.Errorf("%w: 300 MiB is more than 256 MiB", domain.ErrSliceTooLarge)

		// when / then
		assert.Equal(t, 20, cli.ExitCode(err))
	})

	t.Run("should map an AreaNotCoveredError to 19", func(t *testing.T) {
		// given
		err := &domain.AreaNotCoveredError{Report: domain.CoverageReport{Status: domain.CoverageStatusPartial}}

		// when / then
		assert.Equal(t, 19, cli.ExitCode(err))
		assert.Equal(t, 19, cli.ExitCode(fmt.Errorf("slice: %w", err)))
	})
}
