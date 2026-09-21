package builddomain

import "github.com/waliqueiroz/sobrevoo/internal/domain"

// SliceTuningBuilder builds a SliceTuning. Its defaults are the initial
// values of specs/004-geo-data-slice/research.md, the same ones the
// configuration adapter provides.
type SliceTuningBuilder struct {
	tuning domain.SliceTuning
}

func NewSliceTuningBuilder() *SliceTuningBuilder {
	return &SliceTuningBuilder{
		tuning: domain.SliceTuning{
			MarginFactor:          1.0,
			ReferenceHeightPixels: 1080,
			TexelScreenRatio:      2.0,
			EstimatedTileBytes:    64 * 1024,
			MaxSizeBytes:          256 * 1024 * 1024,
		},
	}
}

func (b *SliceTuningBuilder) WithMarginFactor(v float64) *SliceTuningBuilder {
	b.tuning.MarginFactor = v
	return b
}

func (b *SliceTuningBuilder) WithReferenceHeightPixels(v float64) *SliceTuningBuilder {
	b.tuning.ReferenceHeightPixels = v
	return b
}

func (b *SliceTuningBuilder) WithTexelScreenRatio(v float64) *SliceTuningBuilder {
	b.tuning.TexelScreenRatio = v
	return b
}

func (b *SliceTuningBuilder) WithEstimatedTileBytes(v int64) *SliceTuningBuilder {
	b.tuning.EstimatedTileBytes = v
	return b
}

func (b *SliceTuningBuilder) WithMaxSizeBytes(v int64) *SliceTuningBuilder {
	b.tuning.MaxSizeBytes = v
	return b
}

func (b *SliceTuningBuilder) Build() domain.SliceTuning {
	return b.tuning
}
