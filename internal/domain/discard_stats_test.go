package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_DiscardStats_Total(t *testing.T) {
	t.Run("should sum the counts for every discard reason", func(t *testing.T) {
		// given
		stats := domain.DiscardStats{
			ImpossibleCoordinates: 1,
			ConsecutiveDuplicates: 2,
			ImplausibleJumps:      3,
		}

		// when
		total := stats.Total()

		// then
		assert.Equal(t, 6, total)
	})

	t.Run("should return zero when nothing was discarded", func(t *testing.T) {
		// given
		stats := domain.DiscardStats{}

		// when
		total := stats.Total()

		// then
		assert.Equal(t, 0, total)
	})
}
