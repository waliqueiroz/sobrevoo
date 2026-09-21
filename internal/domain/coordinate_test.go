package domain_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_NewCoordinate(t *testing.T) {
	t.Run("should keep a valid coordinate", func(t *testing.T) {
		// when
		coordinate, err := domain.NewCoordinate(-23.5505, -46.6333)

		// then
		require.NoError(t, err)
		assert.Equal(t, -23.5505, coordinate.Latitude)
		assert.Equal(t, -46.6333, coordinate.Longitude)
	})

	t.Run("should refuse a latitude above 90, saying the value and the range", func(t *testing.T) {
		// when
		_, err := domain.NewCoordinate(91, 0)

		// then
		require.ErrorIs(t, err, domain.ErrInvalidCoordinate)
		assert.ErrorContains(t, err, "latitude 91 is out of range, must be between -90 and 90")
	})

	t.Run("should refuse a latitude below -90", func(t *testing.T) {
		// when
		_, err := domain.NewCoordinate(-90.0001, 0)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidCoordinate)
	})

	t.Run("should refuse a longitude above 180, saying the value and the range", func(t *testing.T) {
		// when
		_, err := domain.NewCoordinate(0, 180.5)

		// then
		require.ErrorIs(t, err, domain.ErrInvalidCoordinate)
		assert.ErrorContains(t, err, "longitude 180.5 is out of range, must be between -180 and 180")
	})

	t.Run("should refuse a longitude below -180", func(t *testing.T) {
		// when
		_, err := domain.NewCoordinate(0, -181)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidCoordinate)
	})

	t.Run("should refuse a latitude that is not a number", func(t *testing.T) {
		// when
		_, err := domain.NewCoordinate(math.NaN(), 0)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidCoordinate)
	})

	t.Run("should refuse an infinite longitude", func(t *testing.T) {
		// when
		_, err := domain.NewCoordinate(0, math.Inf(1))

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidCoordinate)
	})

	t.Run("should accept the limits of the range", func(t *testing.T) {
		// when
		_, northErr := domain.NewCoordinate(90, 0)
		_, southErr := domain.NewCoordinate(-90, 0)
		_, westErr := domain.NewCoordinate(0, -180)
		_, eastErr := domain.NewCoordinate(0, 180)

		// then
		assert.NoError(t, northErr)
		assert.NoError(t, southErr)
		assert.NoError(t, westErr)
		assert.NoError(t, eastErr)
	})

	t.Run("should turn longitude 180 and -180 into the same position", func(t *testing.T) {
		// when
		east, _ := domain.NewCoordinate(0, 180)
		west, _ := domain.NewCoordinate(0, -180)

		// then
		assert.Equal(t, west, east)
		assert.Equal(t, -180.0, east.Longitude)
	})
}
