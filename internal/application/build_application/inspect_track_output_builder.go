package build_application

import (
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/application"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

type InspectTrackOutputBuilder struct {
	output application.InspectTrackOutput
}

func NewInspectTrackOutputBuilder() *InspectTrackOutputBuilder {
	elevationGain := 50.0
	duration := 10 * time.Minute

	return &InspectTrackOutputBuilder{
		output: application.InspectTrackOutput{
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

func (b *InspectTrackOutputBuilder) WithFormat(format domain.Format) *InspectTrackOutputBuilder {
	b.output.Format = format
	return b
}

func (b *InspectTrackOutputBuilder) WithPointCountOriginal(count int) *InspectTrackOutputBuilder {
	b.output.PointCountOriginal = count
	return b
}

func (b *InspectTrackOutputBuilder) WithPointCountTreated(count int) *InspectTrackOutputBuilder {
	b.output.PointCountTreated = count
	return b
}

func (b *InspectTrackOutputBuilder) WithTotalDistanceMeters(meters float64) *InspectTrackOutputBuilder {
	b.output.TotalDistanceMeters = meters
	return b
}

func (b *InspectTrackOutputBuilder) WithElevationGainMeters(meters float64) *InspectTrackOutputBuilder {
	b.output.ElevationGainMeters = &meters
	return b
}

func (b *InspectTrackOutputBuilder) WithoutElevationGainMeters() *InspectTrackOutputBuilder {
	b.output.ElevationGainMeters = nil
	return b
}

func (b *InspectTrackOutputBuilder) WithDuration(duration time.Duration) *InspectTrackOutputBuilder {
	b.output.Duration = &duration
	return b
}

func (b *InspectTrackOutputBuilder) WithoutDuration() *InspectTrackOutputBuilder {
	b.output.Duration = nil
	return b
}

func (b *InspectTrackOutputBuilder) WithBoundingBox(boundingBox domain.BoundingBox) *InspectTrackOutputBuilder {
	b.output.BoundingBox = boundingBox
	return b
}

func (b *InspectTrackOutputBuilder) WithDiscarded(discarded domain.DiscardStats) *InspectTrackOutputBuilder {
	b.output.Discarded = discarded
	return b
}

func (b *InspectTrackOutputBuilder) Build() application.InspectTrackOutput {
	return b.output
}
