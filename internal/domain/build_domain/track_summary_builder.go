package build_domain

import (
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

type TrackSummaryBuilder struct {
	summary domain.TrackSummary
}

func NewTrackSummaryBuilder() *TrackSummaryBuilder {
	elevationGain := 50.0
	duration := 10 * time.Minute

	return &TrackSummaryBuilder{
		summary: domain.TrackSummary{
			Format:              domain.FormatGPX,
			PointCountOriginal:  10,
			PointCountTreated:   8,
			TotalDistanceMeters: 1000,
			ElevationGainMeters: &elevationGain,
			Duration:            &duration,
			BoundingBox: domain.BoundingBox{
				MinLatitude:  40.0,
				MaxLatitude:  40.5,
				MinLongitude: -3.5,
				MaxLongitude: -3.0,
			},
		},
	}
}

func (b *TrackSummaryBuilder) WithFormat(format domain.Format) *TrackSummaryBuilder {
	b.summary.Format = format
	return b
}

func (b *TrackSummaryBuilder) WithPointCountOriginal(count int) *TrackSummaryBuilder {
	b.summary.PointCountOriginal = count
	return b
}

func (b *TrackSummaryBuilder) WithPointCountTreated(count int) *TrackSummaryBuilder {
	b.summary.PointCountTreated = count
	return b
}

func (b *TrackSummaryBuilder) WithTotalDistanceMeters(meters float64) *TrackSummaryBuilder {
	b.summary.TotalDistanceMeters = meters
	return b
}

func (b *TrackSummaryBuilder) WithElevationGainMeters(meters float64) *TrackSummaryBuilder {
	b.summary.ElevationGainMeters = &meters
	return b
}

func (b *TrackSummaryBuilder) WithoutElevationGainMeters() *TrackSummaryBuilder {
	b.summary.ElevationGainMeters = nil
	return b
}

func (b *TrackSummaryBuilder) WithDuration(duration time.Duration) *TrackSummaryBuilder {
	b.summary.Duration = &duration
	return b
}

func (b *TrackSummaryBuilder) WithoutDuration() *TrackSummaryBuilder {
	b.summary.Duration = nil
	return b
}

func (b *TrackSummaryBuilder) WithBoundingBox(boundingBox domain.BoundingBox) *TrackSummaryBuilder {
	b.summary.BoundingBox = boundingBox
	return b
}

func (b *TrackSummaryBuilder) WithDiscarded(discarded domain.DiscardStats) *TrackSummaryBuilder {
	b.summary.Discarded = discarded
	return b
}

func (b *TrackSummaryBuilder) Build() domain.TrackSummary {
	return b.summary
}
