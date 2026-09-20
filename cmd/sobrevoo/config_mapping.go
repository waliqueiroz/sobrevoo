package main

import (
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/config"
)

// The configuration adapter has types of its own and does not know the domain
// (Constitution Principle VIII); the composition root is what maps them to
// the domain's types, where the core needs them.

func domainLevel(level config.Level) domain.Level {
	switch level {
	case config.LevelLow:
		return domain.LevelLow
	case config.LevelHigh:
		return domain.LevelHigh
	default:
		return domain.LevelMedium
	}
}

func domainLevelValues(values config.LevelValues) [3]float64 {
	var table [3]float64
	table[domain.LevelLow] = values.Low
	table[domain.LevelMedium] = values.Medium
	table[domain.LevelHigh] = values.High
	return table
}

func domainPlanParameters(defaults config.PlanDefaults) domain.PlanParameters {
	return domain.PlanParameters{
		FrameRate: defaults.FrameRate,
		Distance:  domainLevel(defaults.Distance),
		Tilt:      domainLevel(defaults.Tilt),
	}
}

func domainCameraTuning(t config.CameraTuning) domain.CameraTuning {
	return domain.CameraTuning{
		OpeningFraction: t.OpeningFraction,
		ClosingFraction: t.ClosingFraction,

		StopSpeedMetersPerSecond: t.StopSpeedMetersPerSecond,
		StopMinDuration:          t.StopMinDuration,
		StopCappedDuration:       t.StopCappedDuration,
		StopMaxShareOfMovingTime: t.StopMaxShareOfMovingTime,

		MaxHeadingRateDegPerSecond:  t.MaxHeadingRateDegPerSecond,
		MaxTiltRateDegPerSecond:     t.MaxTiltRateDegPerSecond,
		MaxLogDistanceRatePerSecond: t.MaxLogDistanceRatePerSecond,
		MaxTargetSpeedInDistances:   t.MaxTargetSpeedInDistances,
		GaussianSigmaSeconds:        t.GaussianSigmaSeconds,

		OverviewTiltDegrees:        t.OverviewTiltDegrees,
		OverviewVerticalFOVDegrees: t.OverviewVerticalFOVDegrees,
		OverviewMargin:             t.OverviewMargin,
		OverviewMinDistanceFactor:  t.OverviewMinDistanceFactor,

		MinFollowDuration: t.MinFollowDuration,
		MinPhaseDuration:  t.MinPhaseDuration,

		MinTrackLengthMeters: t.MinTrackLengthMeters,
		MaxTrackSpanMeters:   t.MaxTrackSpanMeters,

		AutoDurationBase:      t.AutoDurationBase,
		AutoDurationPerSqrtKm: t.AutoDurationPerSqrtKm,
		AutoDurationMin:       t.AutoDurationMin,
		AutoDurationMax:       t.AutoDurationMax,

		BaseDistanceMeters: domainLevelValues(t.BaseDistanceMeters),
		LookAheadSeconds:   domainLevelValues(t.LookAheadSeconds),
		TiltDegrees:        domainLevelValues(t.TiltDegrees),
	}
}
