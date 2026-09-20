package builddomain

import (
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

type PlanParametersBuilder struct {
	parameters domain.PlanParameters
}

func NewPlanParametersBuilder() *PlanParametersBuilder {
	return &PlanParametersBuilder{
		parameters: domain.PlanParameters{
			Duration:  new(60 * time.Second),
			FrameRate: 30,
			Distance:  domain.LevelMedium,
			Tilt:      domain.LevelMedium,
		},
	}
}

func (b *PlanParametersBuilder) WithDuration(duration time.Duration) *PlanParametersBuilder {
	b.parameters.Duration = &duration
	return b
}

func (b *PlanParametersBuilder) WithoutDuration() *PlanParametersBuilder {
	b.parameters.Duration = nil
	return b
}

func (b *PlanParametersBuilder) WithFrameRate(frameRate float64) *PlanParametersBuilder {
	b.parameters.FrameRate = frameRate
	return b
}

func (b *PlanParametersBuilder) WithDistance(level domain.Level) *PlanParametersBuilder {
	b.parameters.Distance = level
	return b
}

func (b *PlanParametersBuilder) WithTilt(level domain.Level) *PlanParametersBuilder {
	b.parameters.Tilt = level
	return b
}

func (b *PlanParametersBuilder) Build() domain.PlanParameters {
	return b.parameters
}
