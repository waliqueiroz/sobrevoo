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

	t.Run("should map ErrInvalidAspectRatio to 39", func(t *testing.T) {
		assert.Equal(t, 39, cli.ExitCode(domain.ErrInvalidAspectRatio))
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

	t.Run("should map ErrSliceFileInvalid to 27", func(t *testing.T) {
		assert.Equal(t, 27, cli.ExitCode(domain.ErrSliceFileInvalid))
	})

	t.Run("should map ErrSliceFormatVersionUnsupported to 28", func(t *testing.T) {
		assert.Equal(t, 28, cli.ExitCode(domain.ErrSliceFormatVersionUnsupported))
	})

	t.Run("should map ErrSliceDoesNotMatchPlan to 29", func(t *testing.T) {
		assert.Equal(t, 29, cli.ExitCode(domain.ErrSliceDoesNotMatchPlan))
	})

	t.Run("should map ErrSliceDoesNotCoverPlan to 30", func(t *testing.T) {
		assert.Equal(t, 30, cli.ExitCode(domain.ErrSliceDoesNotCoverPlan))
	})

	t.Run("should map ErrTileFormatUnsupported to 31", func(t *testing.T) {
		assert.Equal(t, 31, cli.ExitCode(domain.ErrTileFormatUnsupported))
	})

	t.Run("should map ErrNoElevationData to 32", func(t *testing.T) {
		assert.Equal(t, 32, cli.ExitCode(domain.ErrNoElevationData))
	})

	t.Run("should map ErrFrameOutOfRange to 33", func(t *testing.T) {
		assert.Equal(t, 33, cli.ExitCode(domain.ErrFrameOutOfRange))
	})

	t.Run("should map ErrInvalidResolution to 34", func(t *testing.T) {
		assert.Equal(t, 34, cli.ExitCode(domain.ErrInvalidResolution))
	})

	t.Run("should map ErrFrameDestinationInvalid to 35", func(t *testing.T) {
		assert.Equal(t, 35, cli.ExitCode(domain.ErrFrameDestinationInvalid))
	})

	t.Run("should map ErrFrameDestinationExists to 36", func(t *testing.T) {
		assert.Equal(t, 36, cli.ExitCode(domain.ErrFrameDestinationExists))
	})

	t.Run("should map ErrFrameSetConflict to 37", func(t *testing.T) {
		assert.Equal(t, 37, cli.ExitCode(domain.ErrFrameSetConflict))
	})

	t.Run("should map ErrRenderInterrupted to 38", func(t *testing.T) {
		assert.Equal(t, 38, cli.ExitCode(domain.ErrRenderInterrupted))
	})

	t.Run("should map a wrapped fifth-stage error to its code", func(t *testing.T) {
		// given
		err := fmt.Errorf("%w: base map \"bbbike\" has vector tiles (pbf)", domain.ErrTileFormatUnsupported)

		// when / then
		assert.Equal(t, 31, cli.ExitCode(err))
	})

	t.Run("should map ErrFrameDirectoryInvalid to 40", func(t *testing.T) {
		assert.Equal(t, 40, cli.ExitCode(domain.ErrFrameDirectoryInvalid))
	})

	t.Run("should map ErrFrameSequenceInvalid to 41", func(t *testing.T) {
		assert.Equal(t, 41, cli.ExitCode(domain.ErrFrameSequenceInvalid))
	})

	t.Run("should map ErrFrameResolutionInvalid to 42", func(t *testing.T) {
		assert.Equal(t, 42, cli.ExitCode(domain.ErrFrameResolutionInvalid))
	})

	t.Run("should map ErrFramesDoNotMatchPlan to 43", func(t *testing.T) {
		assert.Equal(t, 43, cli.ExitCode(domain.ErrFramesDoNotMatchPlan))
	})

	t.Run("should map ErrFramesWithoutPlanID to 44", func(t *testing.T) {
		assert.Equal(t, 44, cli.ExitCode(domain.ErrFramesWithoutPlanID))
	})

	t.Run("should map ErrFrameFileInvalid to 45", func(t *testing.T) {
		assert.Equal(t, 45, cli.ExitCode(domain.ErrFrameFileInvalid))
	})

	t.Run("should map ErrEncoderUnavailable to 46", func(t *testing.T) {
		assert.Equal(t, 46, cli.ExitCode(domain.ErrEncoderUnavailable))
	})

	t.Run("should map ErrVideoDestinationExists to 47", func(t *testing.T) {
		assert.Equal(t, 47, cli.ExitCode(domain.ErrVideoDestinationExists))
	})

	t.Run("should map ErrVideoDestinationInvalid to 48", func(t *testing.T) {
		assert.Equal(t, 48, cli.ExitCode(domain.ErrVideoDestinationInvalid))
	})

	t.Run("should map ErrVideoInterrupted to 49", func(t *testing.T) {
		assert.Equal(t, 49, cli.ExitCode(domain.ErrVideoInterrupted))
	})

	t.Run("should map ErrVideoEncodingFailed to 50", func(t *testing.T) {
		assert.Equal(t, 50, cli.ExitCode(domain.ErrVideoEncodingFailed))
	})

	t.Run("should map ErrFlightInterrupted to 51", func(t *testing.T) {
		assert.Equal(t, 51, cli.ExitCode(domain.ErrFlightInterrupted))
	})

	t.Run("should not take the interruption of a single-command run for that of a stage", func(t *testing.T) {
		// given / when / then
		assert.NotEqual(t, cli.ExitCode(domain.ErrFlightInterrupted), cli.ExitCode(domain.ErrRenderInterrupted))
		assert.NotEqual(t, cli.ExitCode(domain.ErrFlightInterrupted), cli.ExitCode(domain.ErrVideoInterrupted))
	})

	t.Run("should map a wrapped sixth-stage error to its code", func(t *testing.T) {
		// given
		err := fmt.Errorf("%w: 4 missing (12-15)", domain.ErrFrameSequenceInvalid)

		// when / then
		assert.Equal(t, 41, cli.ExitCode(err))
	})

	t.Run("should not take the errors of the frames of a video for those of the stages before", func(t *testing.T) {
		// given / when / then
		assert.NotEqual(t, cli.ExitCode(domain.ErrSliceDoesNotMatchPlan), cli.ExitCode(domain.ErrFramesDoNotMatchPlan))
		assert.NotEqual(t, cli.ExitCode(domain.ErrFrameDestinationInvalid), cli.ExitCode(domain.ErrFrameDirectoryInvalid))
		assert.NotEqual(t, cli.ExitCode(domain.ErrRenderInterrupted), cli.ExitCode(domain.ErrVideoInterrupted))
	})

	t.Run("should keep the codes 1 to 26 of the earlier stages", func(t *testing.T) {
		// given
		expected := map[error]int{
			domain.ErrEmptyFile:                    1,
			domain.ErrInvalidDuration:              10,
			domain.ErrPlanDestinationInvalid:       16,
			domain.ErrPlanFileInvalid:              17,
			domain.ErrPlanFormatVersionUnsupported: 18,
			domain.ErrAreaNotCovered:               19,
			domain.ErrSliceTooLarge:                20,
			domain.ErrGeoDataContentUnreadable:     21,
			domain.ErrElevationUnitUnsupported:     22,
			domain.ErrSliceDestinationExists:       23,
			domain.ErrSliceDestinationInvalid:      24,
			domain.ErrElevationNotCovered:          25,
			domain.ErrInvalidCoordinate:            26,
		}

		for err, code := range expected {
			// when / then
			assert.Equal(t, code, cli.ExitCode(err), err.Error())
		}
	})
}
