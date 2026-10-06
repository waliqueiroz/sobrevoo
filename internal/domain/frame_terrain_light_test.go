package domain

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_terrainLightDirection(t *testing.T) {
	t.Run("should be a unit vector", func(t *testing.T) {
		// given / when
		l := terrainLightDirection

		// then
		assert.InDelta(t, 1.0, l[0]*l[0]+l[1]*l[1]+l[2]*l[2], 1e-9)
	})

	t.Run("should have the fixed altitude as its vertical component", func(t *testing.T) {
		// given / when / then
		assert.InDelta(t, math.Sin(degreesToRadians(TerrainLightAltitudeDegrees)), terrainLightDirection[2], 1e-9)
	})
}

func Test_terrainLightFactor(t *testing.T) {
	// lightHorizontal is the light's direction, flattened to the horizontal
	// plane and renormalized to length 1 — the direction a slope faces when
	// it leans exactly toward the light.
	lx, ly := terrainLightDirection[0], terrainLightDirection[1]
	horizontal := math.Sqrt(lx*lx + ly*ly)
	lightHorizontal := func() (x, y float64) { return lx / horizontal, ly / horizontal }

	t.Run("should be exactly neutral for a flat surface, whatever the light's direction", func(t *testing.T) {
		// given / when
		factor := terrainLightFactor(0, 0, 1)

		// then: dot(N - up, L) is zero whenever N = up, by construction, not
		// because of the particular azimuth/altitude chosen today.
		assert.Equal(t, 1.0, factor)
	})

	t.Run("should brighten a slope whose horizontal lean faces the light", func(t *testing.T) {
		// given: a normal tilted toward the light's azimuth
		hx, hy := lightHorizontal()
		nx, ny, nz := 0.3*hx, 0.3*hy, math.Sqrt(1-0.3*0.3)

		// when
		factor := terrainLightFactor(nx, ny, nz)

		// then
		assert.Greater(t, factor, 1.0)
	})

	t.Run("should darken a slope whose horizontal lean faces away from the light", func(t *testing.T) {
		// given: a normal tilted exactly opposite the light's azimuth
		hx, hy := lightHorizontal()
		nx, ny, nz := -0.3*hx, -0.3*hy, math.Sqrt(1-0.3*0.3)

		// when
		factor := terrainLightFactor(nx, ny, nz)

		// then
		assert.Less(t, factor, 1.0)
	})

	t.Run("should never leave the fixed range for a near-vertical slope facing the light", func(t *testing.T) {
		// given: a normal almost in the horizontal plane, leaning toward the light
		hx, hy := lightHorizontal()
		nx, ny, nz := 0.999*hx, 0.999*hy, math.Sqrt(1-0.999*0.999)

		// when
		factor := terrainLightFactor(nx, ny, nz)

		// then
		assert.LessOrEqual(t, factor, TerrainLightMaxFactor)
		assert.GreaterOrEqual(t, factor, TerrainLightMinFactor)
	})

	t.Run("should never leave the fixed range for a near-vertical slope facing away from the light", func(t *testing.T) {
		// given: a normal almost in the horizontal plane, leaning away from the light
		hx, hy := lightHorizontal()
		nx, ny, nz := -0.999*hx, -0.999*hy, math.Sqrt(1-0.999*0.999)

		// when
		factor := terrainLightFactor(nx, ny, nz)

		// then
		assert.LessOrEqual(t, factor, TerrainLightMaxFactor)
		assert.GreaterOrEqual(t, factor, TerrainLightMinFactor)
	})

	t.Run("should actually clamp the lightest factor: a real 45° slope facing the light exactly reaches past it unclamped", func(t *testing.T) {
		// given: the exact slope that maximizes dot(N - up, L) for this light's 45° altitude —
		// a 45° slope (nz = √2/2) leaning exactly toward the light's azimuth. Unclamped, this
		// reaches 1 + cos(altitude)·(√2 − 1) ≈ 1.293 (worked out by hand and confirmed by this
		// test) — past TerrainLightMaxFactor, so the clamp genuinely binds here, unlike a looser
		// range where it never would (research.md item 8: the saturation this guards against).
		hx, hy := lightHorizontal()
		nz := math.Sqrt2 / 2
		horizontal := math.Sqrt(1 - nz*nz)

		// when
		factor := terrainLightFactor(horizontal*hx, horizontal*hy, nz)

		// then
		assert.Equal(t, TerrainLightMaxFactor, factor)
	})
}

// Test_TerrainLightFactor_Legibility is the one test of this file that
// checks History 2 (016-terrain-lighting): the fixed range keeps two
// distinguishable map colors distinguishable at both ends, and documents
// how far it protects a light background from saturating. apply mirrors
// exactly the math drawPixel uses to turn a base color and a factor into
// the final pixel.
func Test_TerrainLightFactor_Legibility(t *testing.T) {
	apply := func(c RGB, factor float64) RGB {
		return RGB{
			rounded(float64(float64(c.R) * factor)),
			rounded(float64(float64(c.G) * factor)),
			rounded(float64(float64(c.B) * factor)),
		}
	}

	// two typical, distinguishable map colors — neither already at an
	// extreme (0 or 255) of any channel
	a := RGB{R: 40, G: 80, B: 120}
	b := RGB{R: 60, G: 100, B: 140}

	t.Run("should keep two distinguishable, typical map colors distinguishable at the darkest factor of the fixed range", func(t *testing.T) {
		// given / when
		darkA, darkB := apply(a, TerrainLightMinFactor), apply(b, TerrainLightMinFactor)

		// then
		assert.NotEqual(t, darkA, darkB)
	})

	t.Run("should keep two distinguishable, typical map colors distinguishable at the lightest factor of the fixed range", func(t *testing.T) {
		// given / when
		lightA, lightB := apply(a, TerrainLightMaxFactor), apply(b, TerrainLightMaxFactor)

		// then
		assert.NotEqual(t, lightA, lightB)
	})

	t.Run("should no longer saturate a background around the floor the fixed range now protects", func(t *testing.T) {
		// given: the lightest background that TerrainLightMaxFactor can still multiply without
		// clipping any channel (255 / 1.15 ≈ 221.7) — a plausible, if light, map tone
		protected := RGB{R: 220, G: 220, B: 220}

		// when
		lightened := apply(protected, TerrainLightMaxFactor)

		// then
		assert.NotEqual(t, uint8(255), lightened.R)
	})

	t.Run("should document, not hide, the residual limit: a light OSM-style background can still saturate at the most extreme slopes", func(t *testing.T) {
		// given: a background around 242 — not an edge case, the common tone of light OpenStreetMap
		// themes (e.g. the default "Carto" style's cream background) — lit by the most extreme
		// slope this light can produce (a 45° slope facing it exactly, TerrainLightMaxFactor)
		typical := RGB{R: 242, G: 239, B: 233}

		// when
		lightened := apply(typical, TerrainLightMaxFactor)

		// then: still clips (242 × 1.15 ≈ 278), but far less than the 313 the original 1.4 would
		// have given — a known, documented residual limit (research.md item 8), only reached by
		// the steepest slopes exactly facing the light, not a defect this task claims to fix
		// entirely
		assert.Equal(t, uint8(255), lightened.R)
	})
}
