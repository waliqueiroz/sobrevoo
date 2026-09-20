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
}
