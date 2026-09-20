package builddomain

import (
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// CameraTuningBuilder builds a CameraTuning. Its defaults are the initial
// values of specs/003-camera-path-planning/research.md, the same ones the
// configuration adapter provides.
type CameraTuningBuilder struct {
	tuning domain.CameraTuning
}

func NewCameraTuningBuilder() *CameraTuningBuilder {
	return &CameraTuningBuilder{
		tuning: domain.CameraTuning{
			OpeningFraction: 0.10,
			ClosingFraction: 0.10,

			StopSpeedMetersPerSecond: 0.5,
			StopMinDuration:          30 * time.Second,
			StopCappedDuration:       2 * time.Second,
			StopMaxShareOfMovingTime: 0.05,

			MaxHeadingRateDegPerSecond:  45,
			MaxTiltRateDegPerSecond:     30,
			MaxLogDistanceRatePerSecond: 1.5,
			MaxTargetSpeedInDistances:   1.0,
			GaussianSigmaSeconds:        0.5,

			OverviewTiltDegrees:        60,
			OverviewVerticalFOVDegrees: 45,
			OverviewMargin:             1.2,
			OverviewMinDistanceFactor:  2,

			MinFollowDuration: 5 * time.Second,
			MinPhaseDuration:  2 * time.Second,

			MinTrackLengthMeters: 50,
			MaxTrackSpanMeters:   2_000_000,

			AutoDurationBase:      15 * time.Second,
			AutoDurationPerSqrtKm: 6 * time.Second,
			AutoDurationMin:       20 * time.Second,
			AutoDurationMax:       120 * time.Second,

			BaseDistanceMeters: [3]float64{300, 600, 1200},
			LookAheadSeconds:   [3]float64{2, 4, 8},
			TiltDegrees:        [3]float64{25, 45, 65},
		},
	}
}

func (b *CameraTuningBuilder) WithMaxHeadingRateDegPerSecond(rate float64) *CameraTuningBuilder {
	b.tuning.MaxHeadingRateDegPerSecond = rate
	return b
}

func (b *CameraTuningBuilder) WithMaxTiltRateDegPerSecond(rate float64) *CameraTuningBuilder {
	b.tuning.MaxTiltRateDegPerSecond = rate
	return b
}

func (b *CameraTuningBuilder) WithMaxLogDistanceRatePerSecond(rate float64) *CameraTuningBuilder {
	b.tuning.MaxLogDistanceRatePerSecond = rate
	return b
}

func (b *CameraTuningBuilder) WithAutoDurationMax(duration time.Duration) *CameraTuningBuilder {
	b.tuning.AutoDurationMax = duration
	return b
}

func (b *CameraTuningBuilder) Build() domain.CameraTuning {
	return b.tuning
}
