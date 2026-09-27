package builddomain

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// RenderTuningBuilder builds a RenderTuning with the initial values of the
// configuration.
type RenderTuningBuilder struct {
	tuning domain.RenderTuning
}

func NewRenderTuningBuilder() *RenderTuningBuilder {
	return &RenderTuningBuilder{
		tuning: domain.RenderTuning{
			VerticalFOVDegrees:       45,
			MinCameraClearanceMeters: 2,
			MinTiltForTargetDegrees:  1,
			TrailLiftMeters:          0.3,
			DepthBiasMeters:          1,
			DepthBiasRatio:           0.002,
			TileCacheBytes:           256 * 1024 * 1024,
			Workers:                  4,
		},
	}
}

func (b *RenderTuningBuilder) WithVerticalFOVDegrees(degrees float64) *RenderTuningBuilder {
	b.tuning.VerticalFOVDegrees = degrees
	return b
}

func (b *RenderTuningBuilder) WithMinCameraClearanceMeters(meters float64) *RenderTuningBuilder {
	b.tuning.MinCameraClearanceMeters = meters
	return b
}

func (b *RenderTuningBuilder) WithMinTiltForTargetDegrees(degrees float64) *RenderTuningBuilder {
	b.tuning.MinTiltForTargetDegrees = degrees
	return b
}

func (b *RenderTuningBuilder) WithTrailLiftMeters(meters float64) *RenderTuningBuilder {
	b.tuning.TrailLiftMeters = meters
	return b
}

func (b *RenderTuningBuilder) WithDepthBias(meters, ratio float64) *RenderTuningBuilder {
	b.tuning.DepthBiasMeters, b.tuning.DepthBiasRatio = meters, ratio
	return b
}

func (b *RenderTuningBuilder) WithTileCacheBytes(bytes int64) *RenderTuningBuilder {
	b.tuning.TileCacheBytes = bytes
	return b
}

func (b *RenderTuningBuilder) WithWorkers(workers int) *RenderTuningBuilder {
	b.tuning.Workers = workers
	return b
}

func (b *RenderTuningBuilder) Build() domain.RenderTuning {
	return b.tuning
}
