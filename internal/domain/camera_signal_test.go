package domain_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_Signal_Unwrap(t *testing.T) {
	t.Run("should never jump across 360 degrees", func(t *testing.T) {
		// given
		angles := []float64{350, 355, 0, 5, 10}

		// when
		unwrapped := []float64(domain.Signal(angles).Unwrap())

		// then
		assert.Equal(t, []float64{350, 355, 360, 365, 370}, unwrapped)
	})

	t.Run("should unwrap a counter-clockwise crossing of zero", func(t *testing.T) {
		// given
		angles := []float64{10, 5, 0, 355}

		// when
		unwrapped := []float64(domain.Signal(angles).Unwrap())

		// then
		assert.Equal(t, []float64{10, 5, 0, -5}, unwrapped)
	})

	t.Run("should resolve a difference of exactly 180 degrees counter-clockwise", func(t *testing.T) {
		// given
		angles := []float64{0, 180}

		// when
		unwrapped := []float64(domain.Signal(angles).Unwrap())

		// then
		assert.Equal(t, []float64{0, -180}, unwrapped)
	})

	t.Run("should return an empty result for no angles", func(t *testing.T) {
		// when
		unwrapped := []float64(domain.Signal(nil).Unwrap())

		// then
		assert.Empty(t, unwrapped)
	})
}

func Test_Signal_LimitRate(t *testing.T) {
	steps := func(n int, limit float64) []float64 {
		s := make([]float64, n)
		for i := range s {
			s[i] = limit
		}
		return s
	}

	t.Run("should leave a signal that already respects the limit unchanged", func(t *testing.T) {
		// given
		values := []float64{0, 1, 2, 2.5, 2.5, 2}

		// when
		limited := []float64(domain.Signal(values).LimitRate(steps(len(values), 1)))

		// then
		assert.Equal(t, values, limited)
	})

	t.Run("should limit a jump so no consecutive pair exceeds the limit", func(t *testing.T) {
		// given
		values := []float64{0, 0, 0, 100, 100, 100, 100, 100}

		// when
		limited := []float64(domain.Signal(values).LimitRate(steps(len(values), 10)))

		// then
		for k := 1; k < len(limited); k++ {
			assert.LessOrEqual(t, math.Abs(limited[k]-limited[k-1]), 10.0+1e-12, "pair %d", k)
		}
		assert.InDelta(t, 0.0, limited[0], 1e-12)
	})

	t.Run("should honor different limits per step", func(t *testing.T) {
		// given
		values := []float64{0, 100, 100, 100, 100}
		maxSteps := []float64{0, 5, 5, 50, 50}

		// when
		limited := []float64(domain.Signal(values).LimitRate(maxSteps))

		// then
		for k := 1; k < len(limited); k++ {
			assert.LessOrEqual(t, math.Abs(limited[k]-limited[k-1]), maxSteps[k]+1e-12, "pair %d", k)
		}
	})

	t.Run("should not modify its input", func(t *testing.T) {
		// given
		values := []float64{0, 100}

		// when
		_ = []float64(domain.Signal(values).LimitRate(steps(2, 1)))

		// then
		assert.Equal(t, []float64{0, 100}, values)
	})

	t.Run("should handle empty and single-value signals", func(t *testing.T) {
		// when / then
		assert.Empty(t, []float64(domain.Signal(nil).LimitRate(nil)))
		assert.Equal(t, []float64{3}, []float64(domain.Signal([]float64{3}).LimitRate([]float64{1})))
	})
}

func Test_Signal_Smooth(t *testing.T) {
	t.Run("should not increase the largest step of a signal with bounded steps", func(t *testing.T) {
		// given: a zig-zag with steps of at most 1
		values := make([]float64, 200)
		for i := range values {
			if i%7 < 4 {
				values[i] = float64(i % 7)
			} else {
				values[i] = float64(7 - i%7)
			}
		}

		// when
		smoothed := []float64(domain.Signal(values).Smooth(5))

		// then
		for k := 1; k < len(smoothed); k++ {
			assert.LessOrEqual(t, math.Abs(smoothed[k]-smoothed[k-1]), 1.0+1e-9, "step %d", k)
		}
	})

	t.Run("should keep the end values exactly", func(t *testing.T) {
		// given
		values := []float64{5, 1, 9, 2, 8, 3, 7, 4, 6, 5, 0, 10, 3, 3, 3, 3, 3, 3, 3, 12}

		// when
		smoothed := []float64(domain.Signal(values).Smooth(2))

		// then
		assert.InDelta(t, values[0], smoothed[0], 1e-9)
		assert.InDelta(t, values[len(values)-1], smoothed[len(values)-1], 1e-9)
	})

	t.Run("should keep a linear trend unchanged, including near the ends", func(t *testing.T) {
		// given
		values := make([]float64, 50)
		for i := range values {
			values[i] = 3*float64(i) + 2
		}

		// when
		smoothed := []float64(domain.Signal(values).Smooth(4))

		// then
		for i := range values {
			assert.InDelta(t, values[i], smoothed[i], 1e-6, "index %d", i)
		}
	})

	t.Run("should return a copy unchanged for a non-positive sigma", func(t *testing.T) {
		// given
		values := []float64{1, 2, 3}

		// when
		smoothed := []float64(domain.Signal(values).Smooth(0))
		smoothed[0] = 99

		// then
		assert.Equal(t, []float64{1, 2, 3}, values)
	})

	t.Run("should handle a signal shorter than the kernel", func(t *testing.T) {
		// given
		values := []float64{4, 4}

		// when
		smoothed := []float64(domain.Signal(values).Smooth(10))

		// then
		assert.InDelta(t, 4.0, smoothed[0], 1e-9)
		assert.InDelta(t, 4.0, smoothed[1], 1e-9)
	})

	t.Run("should return nothing for an empty signal", func(t *testing.T) {
		// when / then
		assert.Empty(t, []float64(domain.Signal(nil).Smooth(3)))
	})
}

func Test_Signal_SmoothedSpans(t *testing.T) {
	t.Run("should report a run of frames where the limit changed the value", func(t *testing.T) {
		// given
		desired := []float64{0, 0, 10, 10, 10, 0, 0, 0}
		limited := []float64{0, 0, 1, 2, 3, 0, 0, 0}

		// when
		spans := domain.Signal(desired).SmoothedSpans(domain.Signal(limited), 0.05, domain.QuantityHeading, 10)

		// then
		assert.Equal(t, []domain.SmoothedSpan{{Start: 200 * time.Millisecond, End: 400 * time.Millisecond, Quantity: domain.QuantityHeading}}, spans)
	})

	t.Run("should keep separate runs as separate spans", func(t *testing.T) {
		// given
		desired := []float64{5, 0, 0, 5, 5, 0}
		limited := []float64{0, 0, 0, 0, 0, 0}

		// when
		spans := domain.Signal(desired).SmoothedSpans(domain.Signal(limited), 0.05, domain.QuantityTilt, 1)

		// then
		require.Len(t, spans, 2)
		assert.Equal(t, 0*time.Second, spans[0].Start)
		assert.Equal(t, 0*time.Second, spans[0].End)
		assert.Equal(t, 3*time.Second, spans[1].Start)
		assert.Equal(t, 4*time.Second, spans[1].End)
	})

	t.Run("should close a run that lasts until the last frame", func(t *testing.T) {
		// given
		desired := []float64{0, 0, 5, 5}
		limited := []float64{0, 0, 0, 0}

		// when
		spans := domain.Signal(desired).SmoothedSpans(domain.Signal(limited), 0.05, domain.QuantityZoom, 2)

		// then
		assert.Equal(t, []domain.SmoothedSpan{{Start: time.Second, End: 1500 * time.Millisecond, Quantity: domain.QuantityZoom}}, spans)
	})

	t.Run("should ignore differences within the tolerance", func(t *testing.T) {
		// given
		desired := []float64{0, 0.04, 0.05, 0}
		limited := []float64{0, 0, 0, 0}

		// when
		spans := domain.Signal(desired).SmoothedSpans(domain.Signal(limited), 0.05, domain.QuantityHeading, 1)

		// then
		assert.Empty(t, spans)
	})
}
