package domain_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// planarLine returns points along a straight line on the plane, one every
// stepMeters, along the direction (dx, dy), and their cumulative distances.
func planarLine(dx, dy, stepMeters float64, count int) ([]domain.PlanePoint, []float64) {
	norm := math.Hypot(dx, dy)
	route := make([]domain.PlanePoint, count)
	cumulative := make([]float64, count)
	for i := range route {
		cumulative[i] = float64(i) * stepMeters
		route[i] = domain.PlanePoint{X: dx / norm * cumulative[i], Y: dy / norm * cumulative[i]}
	}
	return route, cumulative
}

func Test_DesiredHeading(t *testing.T) {
	t.Run("should point east along a route going east", func(t *testing.T) {
		// given
		route, cumulative := planarLine(1, 0, 10, 100)

		// when
		heading, ok := domain.DesiredHeading(route, cumulative, 500, 100)

		// then
		require.True(t, ok)
		assert.InDelta(t, 90.0, heading, 1e-9)
	})

	t.Run("should point north along a route going north", func(t *testing.T) {
		// given
		route, cumulative := planarLine(0, 1, 10, 100)

		// when
		heading, ok := domain.DesiredHeading(route, cumulative, 500, 100)

		// then
		require.True(t, ok)
		assert.InDelta(t, 0.0, heading, 1e-9)
	})

	t.Run("should point south-west along a route going south-west", func(t *testing.T) {
		// given
		route, cumulative := planarLine(-1, -1, 10, 100)

		// when
		heading, ok := domain.DesiredHeading(route, cumulative, 500, 100)

		// then
		require.True(t, ok)
		assert.InDelta(t, 225.0, heading, 1e-9)
	})

	t.Run("should report no clear direction where the tangents cancel out in a turn back", func(t *testing.T) {
		// given: out 500 m east, then back the same way
		out, _ := planarLine(1, 0, 10, 51)
		route := append([]domain.PlanePoint{}, out...)
		for i := len(out) - 2; i >= 0; i-- {
			route = append(route, out[i])
		}
		cumulative := make([]float64, len(route))
		for i := 1; i < len(route); i++ {
			cumulative[i] = cumulative[i-1] + 10
		}

		// when
		_, ok := domain.DesiredHeading(route, cumulative, 500, 100)

		// then
		assert.False(t, ok)
	})

	t.Run("should report no clear direction for a loop tighter than the window", func(t *testing.T) {
		// given: a 20 m radius circle, many laps, seen through a 500 m window
		var route []domain.PlanePoint
		var cumulative []float64
		for i := 0; i < 600; i++ {
			angle := float64(i) * 2 * math.Pi / 50
			route = append(route, domain.PlanePoint{X: 20 * math.Cos(angle), Y: 20 * math.Sin(angle)})
			cumulative = append(cumulative, float64(i)*2*math.Pi*20/50)
		}

		// when
		_, ok := domain.DesiredHeading(route, cumulative, cumulative[300], 500)

		// then
		assert.False(t, ok)
	})

	t.Run("should report no clear direction for a route of fewer than two points or a non-positive sigma", func(t *testing.T) {
		// given
		route, cumulative := planarLine(1, 0, 10, 10)

		// when
		_, tooShort := domain.DesiredHeading(route[:1], cumulative[:1], 0, 100)
		_, noSigma := domain.DesiredHeading(route, cumulative, 50, 0)

		// then
		assert.False(t, tooShort)
		assert.False(t, noSigma)
	})

	t.Run("should report no clear direction when every segment in the window has zero length", func(t *testing.T) {
		// given
		route := []domain.PlanePoint{{X: 1, Y: 1}, {X: 1, Y: 1}, {X: 1, Y: 1}}
		cumulative := []float64{0, 0, 0}

		// when
		_, ok := domain.DesiredHeading(route, cumulative, 0, 100)

		// then
		assert.False(t, ok)
	})
}

func Test_UnwrapAngles(t *testing.T) {
	t.Run("should never jump across 360 degrees", func(t *testing.T) {
		// given
		angles := []float64{350, 355, 0, 5, 10}

		// when
		unwrapped := domain.UnwrapAngles(angles)

		// then
		assert.Equal(t, []float64{350, 355, 360, 365, 370}, unwrapped)
	})

	t.Run("should unwrap a counter-clockwise crossing of zero", func(t *testing.T) {
		// given
		angles := []float64{10, 5, 0, 355}

		// when
		unwrapped := domain.UnwrapAngles(angles)

		// then
		assert.Equal(t, []float64{10, 5, 0, -5}, unwrapped)
	})

	t.Run("should resolve a difference of exactly 180 degrees counter-clockwise", func(t *testing.T) {
		// given
		angles := []float64{0, 180}

		// when
		unwrapped := domain.UnwrapAngles(angles)

		// then
		assert.Equal(t, []float64{0, -180}, unwrapped)
	})

	t.Run("should return an empty result for no angles", func(t *testing.T) {
		// when
		unwrapped := domain.UnwrapAngles(nil)

		// then
		assert.Empty(t, unwrapped)
	})
}

func Test_FollowDistance(t *testing.T) {
	t.Run("should add the distance covered in the look-ahead time to the base distance", func(t *testing.T) {
		// when
		distance := domain.FollowDistance(600, 4, 10)

		// then
		assert.Equal(t, 640.0, distance)
	})

	t.Run("should be larger for the high level than for the low level at the same speed", func(t *testing.T) {
		// given
		tuning := defaultTuning()

		// when
		low := domain.FollowDistance(tuning.BaseDistanceMeters[domain.LevelLow], tuning.LookAheadSeconds[domain.LevelLow], 20)
		high := domain.FollowDistance(tuning.BaseDistanceMeters[domain.LevelHigh], tuning.LookAheadSeconds[domain.LevelHigh], 20)

		// then
		assert.Greater(t, high, low)
	})
}

func Test_ComputeCameraPose(t *testing.T) {
	t.Run("should place the camera behind the target, along the heading, at the tilt's altitude", func(t *testing.T) {
		// given: heading east (90), tilt 45, distance 100
		view := domain.CameraView{Target: domain.PlanePoint{X: 10, Y: 20}, Heading: 90, TiltDegrees: 45, Distance: 100}

		// when
		pose := domain.ComputeCameraPose(view)

		// then: 70.7 m west of the target, 70.7 m up
		assert.InDelta(t, 10-70.7107, pose.Position.X, 1e-3)
		assert.InDelta(t, 20.0, pose.Position.Y, 1e-3)
		assert.InDelta(t, 70.7107, pose.Altitude, 1e-3)
	})

	t.Run("should put the camera straight above the target when looking straight down", func(t *testing.T) {
		// given
		view := domain.CameraView{Target: domain.PlanePoint{X: 5, Y: 5}, Heading: 30, TiltDegrees: 90, Distance: 100}

		// when
		pose := domain.ComputeCameraPose(view)

		// then
		assert.InDelta(t, 5.0, pose.Position.X, 1e-9)
		assert.InDelta(t, 5.0, pose.Position.Y, 1e-9)
		assert.InDelta(t, 100.0, pose.Altitude, 1e-9)
	})

	t.Run("should place the camera south of the target when heading north", func(t *testing.T) {
		// given
		view := domain.CameraView{Heading: 0, TiltDegrees: 0, Distance: 50}

		// when
		pose := domain.ComputeCameraPose(view)

		// then
		assert.InDelta(t, 0.0, pose.Position.X, 1e-9)
		assert.InDelta(t, -50.0, pose.Position.Y, 1e-9)
		assert.InDelta(t, 0.0, pose.Altitude, 1e-9)
	})
}

func Test_LimitRate(t *testing.T) {
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
		limited := domain.LimitRate(values, steps(len(values), 1))

		// then
		assert.Equal(t, values, limited)
	})

	t.Run("should limit a jump so no consecutive pair exceeds the limit", func(t *testing.T) {
		// given
		values := []float64{0, 0, 0, 100, 100, 100, 100, 100}

		// when
		limited := domain.LimitRate(values, steps(len(values), 10))

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
		limited := domain.LimitRate(values, maxSteps)

		// then
		for k := 1; k < len(limited); k++ {
			assert.LessOrEqual(t, math.Abs(limited[k]-limited[k-1]), maxSteps[k]+1e-12, "pair %d", k)
		}
	})

	t.Run("should not modify its input", func(t *testing.T) {
		// given
		values := []float64{0, 100}

		// when
		domain.LimitRate(values, steps(2, 1))

		// then
		assert.Equal(t, []float64{0, 100}, values)
	})

	t.Run("should handle empty and single-value signals", func(t *testing.T) {
		// when / then
		assert.Empty(t, domain.LimitRate(nil, nil))
		assert.Equal(t, []float64{3}, domain.LimitRate([]float64{3}, []float64{1}))
	})
}

func Test_GaussianSmooth(t *testing.T) {
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
		smoothed := domain.GaussianSmooth(values, 5)

		// then
		for k := 1; k < len(smoothed); k++ {
			assert.LessOrEqual(t, math.Abs(smoothed[k]-smoothed[k-1]), 1.0+1e-9, "step %d", k)
		}
	})

	t.Run("should keep the end values exactly", func(t *testing.T) {
		// given
		values := []float64{5, 1, 9, 2, 8, 3, 7, 4, 6, 5, 0, 10, 3, 3, 3, 3, 3, 3, 3, 12}

		// when
		smoothed := domain.GaussianSmooth(values, 2)

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
		smoothed := domain.GaussianSmooth(values, 4)

		// then
		for i := range values {
			assert.InDelta(t, values[i], smoothed[i], 1e-6, "index %d", i)
		}
	})

	t.Run("should return a copy unchanged for a non-positive sigma", func(t *testing.T) {
		// given
		values := []float64{1, 2, 3}

		// when
		smoothed := domain.GaussianSmooth(values, 0)
		smoothed[0] = 99

		// then
		assert.Equal(t, []float64{1, 2, 3}, values)
	})

	t.Run("should handle a signal shorter than the kernel", func(t *testing.T) {
		// given
		values := []float64{4, 4}

		// when
		smoothed := domain.GaussianSmooth(values, 10)

		// then
		assert.InDelta(t, 4.0, smoothed[0], 1e-9)
		assert.InDelta(t, 4.0, smoothed[1], 1e-9)
	})

	t.Run("should return nothing for an empty signal", func(t *testing.T) {
		// when / then
		assert.Empty(t, domain.GaussianSmooth(nil, 3))
	})
}

func Test_DetectSmoothedSpans(t *testing.T) {
	t.Run("should report a run of frames where the limit changed the value", func(t *testing.T) {
		// given
		desired := []float64{0, 0, 10, 10, 10, 0, 0, 0}
		limited := []float64{0, 0, 1, 2, 3, 0, 0, 0}

		// when
		spans := domain.DetectSmoothedSpans(desired, limited, 0.05, domain.QuantityHeading, 10)

		// then
		assert.Equal(t, []domain.SmoothedSpan{{Start: 200 * time.Millisecond, End: 400 * time.Millisecond, Quantity: domain.QuantityHeading}}, spans)
	})

	t.Run("should keep separate runs as separate spans", func(t *testing.T) {
		// given
		desired := []float64{5, 0, 0, 5, 5, 0}
		limited := []float64{0, 0, 0, 0, 0, 0}

		// when
		spans := domain.DetectSmoothedSpans(desired, limited, 0.05, domain.QuantityTilt, 1)

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
		spans := domain.DetectSmoothedSpans(desired, limited, 0.05, domain.QuantityZoom, 2)

		// then
		assert.Equal(t, []domain.SmoothedSpan{{Start: time.Second, End: 1500 * time.Millisecond, Quantity: domain.QuantityZoom}}, spans)
	})

	t.Run("should ignore differences within the tolerance", func(t *testing.T) {
		// given
		desired := []float64{0, 0.04, 0.05, 0}
		limited := []float64{0, 0, 0, 0}

		// when
		spans := domain.DetectSmoothedSpans(desired, limited, 0.05, domain.QuantityHeading, 1)

		// then
		assert.Empty(t, spans)
	})
}
