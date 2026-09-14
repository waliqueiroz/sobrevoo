package clock_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/clock"
)

func Test_Clock_Now(t *testing.T) {
	t.Run("should return the current time", func(t *testing.T) {
		// given
		c := clock.New()
		before := time.Now()

		// when
		now := c.Now()

		// then
		after := time.Now()
		assert.False(t, now.Before(before))
		assert.False(t, now.After(after))
	})
}
