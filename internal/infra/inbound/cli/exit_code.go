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
	default:
		// Anything else (file-open I/O errors, malformed-but-recognized
		// content, CLI usage errors not already handled by Cobra itself) is
		// a generic failure not tied to a domain sentinel.
		return 4
	}
}
