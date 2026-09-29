package jsonfile

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// CameraPlanReader implements domain.CameraPlanReader, reading a plan file as
// CameraPlanExporter writes it (specs/003-camera-path-planning/contracts/
// plan-file.md).
type CameraPlanReader struct{}

// NewCameraPlanReader creates a CameraPlanReader.
func NewCameraPlanReader() CameraPlanReader {
	return CameraPlanReader{}
}

// acceptedPlanFormatVersions are the format versions Read understands, as the
// error message lists them.
var acceptedPlanFormatVersions = strconv.Itoa(planFormatVersion)

// The pointers tell a field that is absent from one that is zero. Unknown
// fields are ignored, as the format's contract says a consumer must.
type readParameters struct {
	DurationSeconds *float64 `json:"duration_s"`
	FrameRate       *float64 `json:"frame_rate"`
	Distance        string   `json:"distance"`
	Tilt            string   `json:"tilt"`
	AspectRatio     *string  `json:"aspect_ratio"`
}

type readSpan struct {
	StartSeconds float64 `json:"start_s"`
	EndSeconds   float64 `json:"end_s"`
	Quantity     string  `json:"quantity"`
}

type readSummary struct {
	DurationMode       string     `json:"duration_mode"`
	FrameCount         *int       `json:"frame_count"`
	TimeReference      string     `json:"time_reference"`
	TimeFallbackReason string     `json:"time_fallback_reason"`
	SmoothedSpans      []readSpan `json:"smoothed_spans"`
	ElevationAvailable *bool      `json:"elevation_available"`
}

type readCamera struct {
	Latitude  *float64 `json:"lat"`
	Longitude *float64 `json:"lon"`
	Altitude  float64  `json:"altitude_m"`
}

type readMarker struct {
	Latitude  *float64 `json:"lat"`
	Longitude *float64 `json:"lon"`
	Distance  float64  `json:"distance_m"`
	Elevation *float64 `json:"elevation_m"`
	Gain      *float64 `json:"gain_m"`
}

type readFrame struct {
	Index               int         `json:"index"`
	TimeSeconds         float64     `json:"time_s"`
	ActivityTimeSeconds *float64    `json:"activity_time_s"`
	Phase               string      `json:"phase"`
	Camera              *readCamera `json:"camera"`
	Heading             float64     `json:"heading_deg"`
	Tilt                float64     `json:"tilt_deg"`
	Marker              *readMarker `json:"marker"`
	CameraToMarker      *float64    `json:"camera_to_marker_m"`
}

type readPlan struct {
	FormatVersion *int            `json:"format_version"`
	Parameters    *readParameters `json:"parameters"`
	Summary       *readSummary    `json:"summary"`
	Frames        []readFrame     `json:"frames"`
}

// Read reads the plan file at path. The format version is checked first, so a
// file of a future version is reported as such and not as malformed.
func (CameraPlanReader) Read(path string) (domain.CameraPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.CameraPlan{}, fmt.Errorf("reading the plan file: %w", err)
	}

	var file readPlan
	if err := json.Unmarshal(data, &file); err != nil {
		return domain.CameraPlan{}, invalidPlanFile("it is not valid JSON: %v", err)
	}

	if file.FormatVersion == nil {
		return domain.CameraPlan{}, invalidPlanFile("format_version is missing")
	}
	if *file.FormatVersion != planFormatVersion {
		return domain.CameraPlan{}, fmt.Errorf("%w: found %d, accepted: %s; generate the plan again with 'sobrevoo plan'", domain.ErrPlanFormatVersionUnsupported, *file.FormatVersion, acceptedPlanFormatVersions)
	}

	switch {
	case file.Parameters == nil:
		return domain.CameraPlan{}, invalidPlanFile("parameters is missing")
	case file.Summary == nil:
		return domain.CameraPlan{}, invalidPlanFile("summary is missing")
	case file.Parameters.DurationSeconds == nil:
		return domain.CameraPlan{}, invalidPlanFile("parameters.duration_s is missing")
	case file.Parameters.FrameRate == nil:
		return domain.CameraPlan{}, invalidPlanFile("parameters.frame_rate is missing")
	case file.Summary.FrameCount == nil:
		return domain.CameraPlan{}, invalidPlanFile("summary.frame_count is missing")
	case *file.Summary.FrameCount != len(file.Frames):
		return domain.CameraPlan{}, invalidPlanFile("summary.frame_count is %d but the file lists %d frames", *file.Summary.FrameCount, len(file.Frames))
	case file.Summary.ElevationAvailable == nil:
		return domain.CameraPlan{}, invalidPlanFile("summary.elevation_available is missing")
	}

	frames := make([]domain.CameraFrame, len(file.Frames))
	for i, f := range file.Frames {
		switch {
		case f.Camera == nil || f.Camera.Latitude == nil || f.Camera.Longitude == nil:
			return domain.CameraPlan{}, invalidPlanFile("frames[%d].camera is missing", i)
		case f.Marker == nil || f.Marker.Latitude == nil || f.Marker.Longitude == nil:
			return domain.CameraPlan{}, invalidPlanFile("frames[%d].marker is missing", i)
		case f.CameraToMarker == nil:
			return domain.CameraPlan{}, invalidPlanFile("frames[%d].camera_to_marker_m is missing", i)
		case f.ActivityTimeSeconds == nil:
			return domain.CameraPlan{}, invalidPlanFile("frames[%d].activity_time_s is missing", i)
		case f.Marker.Elevation == nil:
			return domain.CameraPlan{}, invalidPlanFile("frames[%d].marker.elevation_m is missing", i)
		case f.Marker.Gain == nil:
			return domain.CameraPlan{}, invalidPlanFile("frames[%d].marker.gain_m is missing", i)
		}

		frames[i] = domain.CameraFrame{
			Index:                  f.Index,
			Time:                   secondsToDuration(f.TimeSeconds),
			Phase:                  domain.Phase(f.Phase),
			CameraLatitude:         *f.Camera.Latitude,
			CameraLongitude:        *f.Camera.Longitude,
			CameraAltitude:         f.Camera.Altitude,
			Heading:                f.Heading,
			Tilt:                   f.Tilt,
			MarkerLatitude:         *f.Marker.Latitude,
			MarkerLongitude:        *f.Marker.Longitude,
			MarkerDistance:         f.Marker.Distance,
			CameraToMarkerDistance: *f.CameraToMarker,
			ActivityElapsed:        secondsToDuration(*f.ActivityTimeSeconds),
			TrackElevation:         *f.Marker.Elevation,
			TrackElevationGain:     *f.Marker.Gain,
		}
	}

	spans := make([]domain.SmoothedSpan, len(file.Summary.SmoothedSpans))
	for i, span := range file.Summary.SmoothedSpans {
		spans[i] = domain.SmoothedSpan{
			Start:    secondsToDuration(span.StartSeconds),
			End:      secondsToDuration(span.EndSeconds),
			Quantity: domain.SmoothedQuantity(span.Quantity),
		}
	}

	// A plan made before the aspect ratio existed framed the track by the
	// vertical field of view only, which is what a landscape video needs.
	aspect := domain.LandscapeAspectRatio
	if file.Parameters.AspectRatio != nil {
		var err error
		if aspect, err = domain.ParseAspectRatio(*file.Parameters.AspectRatio); err != nil {
			return domain.CameraPlan{}, invalidPlanFile("parameters.aspect_ratio: %v", err)
		}
	}

	parameters := domain.PlanParameters{
		Duration:  new(secondsToDuration(*file.Parameters.DurationSeconds)),
		FrameRate: *file.Parameters.FrameRate,
		Distance:  parseLevel(file.Parameters.Distance),
		Tilt:      parseLevel(file.Parameters.Tilt),
		Aspect:    aspect,
	}

	return domain.NewCameraPlan(
		parameters,
		domain.DurationMode(file.Summary.DurationMode),
		domain.TimeReference(file.Summary.TimeReference),
		file.Summary.TimeFallbackReason,
		frames,
		spans,
		*file.Summary.ElevationAvailable,
	), nil
}

func invalidPlanFile(format string, args ...any) error {
	return fmt.Errorf("%w: %s", domain.ErrPlanFileInvalid, fmt.Sprintf(format, args...))
}

// secondsToDuration converts a number of seconds to a duration, to the
// nanosecond.
func secondsToDuration(seconds float64) time.Duration {
	return time.Duration(math.Round(seconds * float64(time.Second)))
}

// parseLevel is levelText's inverse; an unknown text reads as medium, the
// default level.
func parseLevel(text string) domain.Level {
	switch text {
	case "low":
		return domain.LevelLow
	case "high":
		return domain.LevelHigh
	default:
		return domain.LevelMedium
	}
}
