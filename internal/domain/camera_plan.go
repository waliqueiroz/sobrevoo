package domain

//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/camera_plan_exporter.go -package mockdomain . CameraPlanExporter

import (
	"math"
	"time"
)

// CameraPlanExporter writes a camera plan outside the process (a structured
// file, for inspection and for later stages). Concrete implementations live
// in internal/infra/outbound.
type CameraPlanExporter interface {
	// Export writes plan to path. Unless overwrite is true it must refuse a
	// path that already holds a file (ErrPlanDestinationExists), and it must
	// never leave a partial file behind on failure.
	Export(plan CameraPlan, path string, overwrite bool) error
}

// Phase tells which part of the video a frame belongs to.
type Phase string

const (
	PhaseOpening   Phase = "opening"
	PhaseFollowing Phase = "following"
	PhaseClosing   Phase = "closing"
)

// TimeReference tells what drove the marker's progress along the track.
type TimeReference string

const (
	// TimeReferenceClock means the track's timestamps were used.
	TimeReferenceClock TimeReference = "clock"

	// TimeReferenceDistance means the distance travelled was used.
	TimeReferenceDistance TimeReference = "distance"
)

// DurationMode tells whether the plan's duration was chosen by the user or
// computed from the track.
type DurationMode string

const (
	DurationModeAutomatic DurationMode = "automatic"
	DurationModeExplicit  DurationMode = "explicit"
)

// SmoothedQuantity names the camera quantity whose change had to be limited.
type SmoothedQuantity string

const (
	QuantityHeading     SmoothedQuantity = "heading"
	QuantityTilt        SmoothedQuantity = "tilt"
	QuantityZoom        SmoothedQuantity = "zoom"
	QuantityTargetSpeed SmoothedQuantity = "target_speed"
)

// SmoothedSpan is a continuous stretch of the video in which the camera's
// natural motion would have exceeded a smoothness limit and was limited.
type SmoothedSpan struct {
	Start    time.Duration
	End      time.Duration
	Quantity SmoothedQuantity
}

// CameraFrame is one instant of the video.
type CameraFrame struct {
	Index int
	Time  time.Duration
	Phase Phase

	// Camera position: decimal degrees (longitude in [-180, 180)) and
	// altitude in meters above the point being observed (relief is not
	// known at this stage).
	CameraLatitude  float64
	CameraLongitude float64
	CameraAltitude  float64

	// Heading is where the camera points horizontally, in degrees clockwise
	// from north, in [0, 360). Tilt is the angle below the horizon, in
	// [0, 90].
	Heading float64
	Tilt    float64

	// The activity marker: position, and meters travelled along the track.
	MarkerLatitude  float64
	MarkerLongitude float64
	MarkerDistance  float64

	// CameraToMarkerDistance is the straight-line distance, in meters.
	CameraToMarkerDistance float64
}

// PlanSummary describes a plan at a glance. It is computed from the frames.
type PlanSummary struct {
	Duration      time.Duration
	DurationMode  DurationMode
	FrameRate     float64
	FrameCount    int
	TimeReference TimeReference

	MinCameraAltitude float64
	MaxCameraAltitude float64
	MinCameraDistance float64
	MaxCameraDistance float64

	SmoothedSpans []SmoothedSpan
}

// CameraPlan is the complete result of camera planning: where the camera is,
// where it points and where the marker is, for every frame of the video.
type CameraPlan struct {
	// Parameters are the ones effectively used: Duration is always set, to
	// the requested duration or to the one computed from the track.
	Parameters PlanParameters

	TimeReference      TimeReference
	TimeFallbackReason string
	Frames             []CameraFrame
	Summary            PlanSummary
}

// NewCameraPlan assembles a plan and computes its summary from the frames, so
// the summary can never disagree with them.
func NewCameraPlan(
	parameters PlanParameters,
	durationMode DurationMode,
	timeReference TimeReference,
	fallbackReason string,
	frames []CameraFrame,
	spans []SmoothedSpan,
) CameraPlan {
	summary := PlanSummary{
		DurationMode:  durationMode,
		FrameRate:     parameters.FrameRate,
		FrameCount:    len(frames),
		TimeReference: timeReference,
		SmoothedSpans: append([]SmoothedSpan{}, spans...),
	}
	if parameters.Duration != nil {
		summary.Duration = *parameters.Duration
	}

	for i, f := range frames {
		if i == 0 {
			summary.MinCameraAltitude, summary.MaxCameraAltitude = f.CameraAltitude, f.CameraAltitude
			summary.MinCameraDistance, summary.MaxCameraDistance = f.CameraToMarkerDistance, f.CameraToMarkerDistance
			continue
		}
		summary.MinCameraAltitude = math.Min(summary.MinCameraAltitude, f.CameraAltitude)
		summary.MaxCameraAltitude = math.Max(summary.MaxCameraAltitude, f.CameraAltitude)
		summary.MinCameraDistance = math.Min(summary.MinCameraDistance, f.CameraToMarkerDistance)
		summary.MaxCameraDistance = math.Max(summary.MaxCameraDistance, f.CameraToMarkerDistance)
	}

	return CameraPlan{
		Parameters:         parameters,
		TimeReference:      timeReference,
		TimeFallbackReason: fallbackReason,
		Frames:             frames,
		Summary:            summary,
	}
}
