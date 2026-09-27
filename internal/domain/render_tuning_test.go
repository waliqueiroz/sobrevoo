package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func initialRenderTuning() domain.RenderTuning {
	return domain.RenderTuning{
		VerticalFOVDegrees:       45,
		MinCameraClearanceMeters: 2,
		MinTiltForTargetDegrees:  1,
		TrailLiftMeters:          0.3,
		DepthBiasMeters:          1,
		DepthBiasRatio:           0.002,
		TileCacheBytes:           256 << 20,
		Workers:                  8,
	}
}

func Test_RenderTuning_Fingerprint(t *testing.T) {
	t.Run("should be the same for two equal tunings, and stable", func(t *testing.T) {
		// given
		first, second := initialRenderTuning(), initialRenderTuning()

		// when / then
		assert.Equal(t, first.Fingerprint(), second.Fingerprint())
		assert.Equal(t, "45|2|1|0.3|1|0.002", first.Fingerprint())
	})

	t.Run("should change with every field that changes how a frame looks", func(t *testing.T) {
		// given
		base := initialRenderTuning().Fingerprint()

		// when
		fov := initialRenderTuning()
		fov.VerticalFOVDegrees = 50
		clearance := initialRenderTuning()
		clearance.MinCameraClearanceMeters = 3
		tilt := initialRenderTuning()
		tilt.MinTiltForTargetDegrees = 2
		lift := initialRenderTuning()
		lift.TrailLiftMeters = 0.5
		bias := initialRenderTuning()
		bias.DepthBiasMeters = 2
		ratio := initialRenderTuning()
		ratio.DepthBiasRatio = 0.004

		// then
		assert.NotEqual(t, base, fov.Fingerprint())
		assert.NotEqual(t, base, clearance.Fingerprint())
		assert.NotEqual(t, base, tilt.Fingerprint())
		assert.NotEqual(t, base, lift.Fingerprint())
		assert.NotEqual(t, base, bias.Fingerprint())
		assert.NotEqual(t, base, ratio.Fingerprint())
	})

	t.Run("should not change with what only affects speed and memory", func(t *testing.T) {
		// given
		base := initialRenderTuning().Fingerprint()
		other := initialRenderTuning()
		other.TileCacheBytes = 1 << 20
		other.Workers = 1

		// when / then
		assert.Equal(t, base, other.Fingerprint())
	})
}
