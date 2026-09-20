package builddomain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_SyntheticRouteBuilder_Build(t *testing.T) {
	t.Run("should build a line of the requested length at the equator", func(t *testing.T) {
		// given
		builder := builddomain.NewSyntheticRouteBuilder().WithOrigin(0, 0).WithLine(20000, 45)

		// when
		points := builder.Build()

		// then
		assert.InEpsilon(t, 20000.0, (domain.Route{Points: points}).Length(), 0.001)
	})

	t.Run("should build a line of the requested length at the antimeridian", func(t *testing.T) {
		// given
		builder := builddomain.NewSyntheticRouteBuilder().WithOrigin(10, 179.95).WithLine(20000, 45)

		// when
		points := builder.Build()

		// then
		assert.InEpsilon(t, 20000.0, (domain.Route{Points: points}).Length(), 0.001)
	})

	t.Run("should build a line of the requested length at a high latitude", func(t *testing.T) {
		// given
		builder := builddomain.NewSyntheticRouteBuilder().WithOrigin(85, 10).WithLine(20000, 45)

		// when
		points := builder.Build()

		// then
		assert.InEpsilon(t, 20000.0, (domain.Route{Points: points}).Length(), 0.001)
	})

	t.Run("should build an out-and-back route of twice the requested length at the equator", func(t *testing.T) {
		// given
		builder := builddomain.NewSyntheticRouteBuilder().WithOrigin(0, 0).WithOutAndBack(3000)

		// when
		points := builder.Build()

		// then
		assert.InEpsilon(t, 6000.0, (domain.Route{Points: points}).Length(), 0.001)
	})

	t.Run("should build an out-and-back route of twice the requested length at the antimeridian", func(t *testing.T) {
		// given
		builder := builddomain.NewSyntheticRouteBuilder().WithOrigin(10, 179.95).WithOutAndBack(3000)

		// when
		points := builder.Build()

		// then
		assert.InEpsilon(t, 6000.0, (domain.Route{Points: points}).Length(), 0.001)
	})

	t.Run("should build an out-and-back route of twice the requested length at a high latitude", func(t *testing.T) {
		// given
		builder := builddomain.NewSyntheticRouteBuilder().WithOrigin(85, 10).WithOutAndBack(3000)

		// when
		points := builder.Build()

		// then
		assert.InEpsilon(t, 6000.0, (domain.Route{Points: points}).Length(), 0.001)
	})

	t.Run("should build a circle of several laps whose length is the circumference times the laps", func(t *testing.T) {
		// given
		builder := builddomain.NewSyntheticRouteBuilder().WithCircle(150, 5)

		// when
		points := builder.Build()

		// then
		assert.InEpsilon(t, 2*3.141592653589793*150*5, (domain.Route{Points: points}).Length(), 0.001)
	})

	t.Run("should build a U-turn whose ends are on opposite legs", func(t *testing.T) {
		// given
		builder := builddomain.NewSyntheticRouteBuilder().WithUTurn(2000)

		// when
		points := builder.Build()

		// then
		first, last := points[0], points[len(points)-1]
		assert.InDelta(t, 400.0, first.DistanceTo(last), 10) // 2 × radius (a tenth of the length)
	})

	t.Run("should give every point a timestamp when a constant speed is set", func(t *testing.T) {
		// given
		builder := builddomain.NewSyntheticRouteBuilder().WithLine(1000, 90).WithConstantSpeed(5)

		// when
		points := builder.Build()

		// then
		duration, ok := (domain.Route{Points: points}).Duration()
		assert.True(t, ok)
		assert.InDelta(t, 200.0, duration.Seconds(), 0.5)
	})

	t.Run("should leave the points without timestamps by default and after WithoutTime", func(t *testing.T) {
		// given
		withoutTime := builddomain.NewSyntheticRouteBuilder().WithConstantSpeed(5).WithoutTime()

		// when
		defaultPoints := builddomain.NewSyntheticRouteBuilder().Build()
		clearedPoints := withoutTime.Build()

		// then
		assert.False(t, defaultPoints[0].HasTime())
		assert.False(t, clearedPoints[0].HasTime())
	})

	t.Run("should use the requested number of points", func(t *testing.T) {
		// given
		builder := builddomain.NewSyntheticRouteBuilder().WithPointCount(7)

		// when
		points := builder.Build()

		// then
		assert.Len(t, points, 7)
	})
}
