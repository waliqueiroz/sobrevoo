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
		Aspect:    domain.AspectRatio{Width: defaults.AspectWidth, Height: defaults.AspectHeight},
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

func domainSliceTuning(t config.SliceTuning) domain.SliceTuning {
	return domain.SliceTuning{
		MarginFactor:          t.MarginFactor,
		ReferenceHeightPixels: t.ReferenceHeightPixels,
		TexelScreenRatio:      t.TexelScreenRatio,
		EstimatedTileBytes:    t.EstimatedTileBytes,
		MaxSizeBytes:          t.MaxSizeBytes,
	}
}

func domainRenderTuning(t config.RenderTuning) domain.RenderTuning {
	return domain.RenderTuning{
		VerticalFOVDegrees:       t.VerticalFOVDegrees,
		MinCameraClearanceMeters: t.MinCameraClearanceMeters,
		MinTiltForTargetDegrees:  t.MinTiltForTargetDegrees,
		TrailLiftMeters:          t.TrailLiftMeters,
		DepthBiasMeters:          t.DepthBiasMeters,
		DepthBiasRatio:           t.DepthBiasRatio,
		TileCacheBytes:           t.TileCacheBytes,
		Workers:                  t.Workers,
	}
}

// domainRenderResolution is the resolution of the images when the user
// chooses none, checked as any resolution is.
func domainRenderResolution(d config.RenderDefaults) (domain.Resolution, error) {
	return domain.NewResolution(d.Width, d.Height)
}

// domainAppearance is the appearance a frame is drawn with when the user
// chooses none, parsed and checked as any appearance is (008-frame-appearance).
func domainAppearance(d config.RenderDefaults) (domain.Appearance, error) {
	trailColor, err := domain.ParseColor(d.TrailColor)
	if err != nil {
		return domain.Appearance{}, err
	}
	markerColor, err := domain.ParseColor(d.MarkerColor)
	if err != nil {
		return domain.Appearance{}, err
	}
	backgroundColor, err := domain.ParseColor(d.BackgroundColor)
	if err != nil {
		return domain.Appearance{}, err
	}

	return domain.NewAppearance(trailColor, d.TrailWidthRatio, markerColor, d.MarkerRadiusRatio, backgroundColor)
}

// domainVideoQuality is the quality of the video the user gets when they choose
// none.
func domainVideoQuality(level config.Level) domain.VideoQuality {
	switch level {
	case config.LevelLow:
		return domain.VideoQualityLow
	case config.LevelHigh:
		return domain.VideoQualityHigh
	default:
		return domain.VideoQualityMedium
	}
}
