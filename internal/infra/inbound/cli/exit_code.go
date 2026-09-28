package cli

import (
	"errors"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// ExitCode translates an error returned by a command into a process exit
// code (Constitution Principle VII — the CLI adapter is what maps domain
// sentinel errors to codes; the core never calls os.Exit or knows about
// exit codes at all), following the table in contracts/cli.md.
func ExitCode(err error) int {
	var usageErr *usageError

	switch {
	case err == nil:
		return 0
	case errors.Is(err, domain.ErrEmptyFile):
		return 1
	case errors.Is(err, domain.ErrUnsupportedFormat), errors.As(err, &usageErr):
		return 2
	case errors.Is(err, domain.ErrInsufficientPoints), errors.Is(err, domain.ErrInsufficientPointsAfterCleaning):
		return 3
	case errors.Is(err, domain.ErrDataFileNotFound):
		return 5
	case errors.Is(err, domain.ErrDataFileUnreadable):
		return 6
	case errors.Is(err, domain.ErrUnsupportedDataFormat):
		return 7
	case errors.Is(err, domain.ErrDataSourceNameAlreadyUsed):
		return 8
	case errors.Is(err, domain.ErrDataSourceNotRegistered):
		return 9
	case errors.Is(err, domain.ErrInvalidDuration):
		return 10
	case errors.Is(err, domain.ErrInvalidFrameRate):
		return 11
	case errors.Is(err, domain.ErrDurationTooShort):
		return 12
	case errors.Is(err, domain.ErrTrackTooShort):
		return 13
	case errors.Is(err, domain.ErrTrackTooLarge):
		return 14
	case errors.Is(err, domain.ErrPlanDestinationExists):
		return 15
	case errors.Is(err, domain.ErrPlanDestinationInvalid):
		return 16
	case errors.Is(err, domain.ErrPlanFileInvalid):
		return 17
	case errors.Is(err, domain.ErrPlanFormatVersionUnsupported):
		return 18
	case errors.Is(err, domain.ErrAreaNotCovered):
		return 19
	case errors.Is(err, domain.ErrSliceTooLarge):
		return 20
	case errors.Is(err, domain.ErrGeoDataContentUnreadable):
		return 21
	case errors.Is(err, domain.ErrElevationUnitUnsupported):
		return 22
	case errors.Is(err, domain.ErrSliceDestinationExists):
		return 23
	case errors.Is(err, domain.ErrSliceDestinationInvalid):
		return 24
	case errors.Is(err, domain.ErrElevationNotCovered):
		return 25
	case errors.Is(err, domain.ErrInvalidCoordinate):
		return 26
	case errors.Is(err, domain.ErrSliceFileInvalid):
		return 27
	case errors.Is(err, domain.ErrSliceFormatVersionUnsupported):
		return 28
	case errors.Is(err, domain.ErrSliceDoesNotMatchPlan):
		return 29
	case errors.Is(err, domain.ErrSliceDoesNotCoverPlan):
		return 30
	case errors.Is(err, domain.ErrTileFormatUnsupported):
		return 31
	case errors.Is(err, domain.ErrNoElevationData):
		return 32
	case errors.Is(err, domain.ErrFrameOutOfRange):
		return 33
	case errors.Is(err, domain.ErrInvalidResolution):
		return 34
	case errors.Is(err, domain.ErrFrameDestinationInvalid):
		return 35
	case errors.Is(err, domain.ErrFrameDestinationExists):
		return 36
	case errors.Is(err, domain.ErrFrameSetConflict):
		return 37
	case errors.Is(err, domain.ErrRenderInterrupted):
		return 38
	case errors.Is(err, domain.ErrInvalidAspectRatio):
		return 39
	case errors.Is(err, domain.ErrFrameDirectoryInvalid):
		return 40
	case errors.Is(err, domain.ErrFrameSequenceInvalid):
		return 41
	case errors.Is(err, domain.ErrFrameResolutionInvalid):
		return 42
	case errors.Is(err, domain.ErrFramesDoNotMatchPlan):
		return 43
	case errors.Is(err, domain.ErrFramesWithoutPlanID):
		return 44
	case errors.Is(err, domain.ErrFrameFileInvalid):
		return 45
	case errors.Is(err, domain.ErrEncoderUnavailable):
		return 46
	case errors.Is(err, domain.ErrVideoDestinationExists):
		return 47
	case errors.Is(err, domain.ErrVideoDestinationInvalid):
		return 48
	case errors.Is(err, domain.ErrVideoInterrupted):
		return 49
	case errors.Is(err, domain.ErrVideoEncodingFailed):
		return 50
	case errors.Is(err, domain.ErrFlightInterrupted):
		return 51
	case errors.Is(err, domain.ErrInvalidColor):
		return 52
	case errors.Is(err, domain.ErrInvalidTrailWidth):
		return 53
	case errors.Is(err, domain.ErrInvalidMarkerRadius):
		return 54
	default:
		// Anything else (file-open I/O errors, malformed-but-recognized
		// content, CLI usage errors not already handled by Cobra itself) is
		// a generic failure not tied to a domain sentinel.
		return 4
	}
}
