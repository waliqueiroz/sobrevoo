package jsonfile

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// planFormatVersion is the version of the exported plan file format
// (specs/003-camera-path-planning/contracts/plan-file.md). It changes only
// when a field is removed or changes meaning.
const planFormatVersion = 1

// number is a JSON number printed with a fixed maximum number of decimal
// places and no trailing zeros, so the same value always produces the same
// bytes.
type number struct {
	value  float64
	places int
}

func (n number) MarshalJSON() ([]byte, error) {
	text := strconv.FormatFloat(n.value, 'f', n.places, 64)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	}
	if text == "-0" {
		text = "0"
	}
	return []byte(text), nil
}

func coordinate(v float64) number { return number{v, 7} }
func measure(v float64) number    { return number{v, 3} }
func seconds(d time.Duration) number {
	return number{d.Seconds(), 9}
}

type parametersFile struct {
	DurationSeconds number `json:"duration_s"`
	FrameRate       number `json:"frame_rate"`
	Distance        string `json:"distance"`
	Tilt            string `json:"tilt"`
}

type rangeFile struct {
	Min number `json:"min"`
	Max number `json:"max"`
}

type spanFile struct {
	StartSeconds number `json:"start_s"`
	EndSeconds   number `json:"end_s"`
	Quantity     string `json:"quantity"`
}

type summaryFile struct {
	DurationSeconds    number     `json:"duration_s"`
	DurationMode       string     `json:"duration_mode"`
	FrameRate          number     `json:"frame_rate"`
	FrameCount         int        `json:"frame_count"`
	CameraAltitude     rangeFile  `json:"camera_altitude_m"`
	CameraDistance     rangeFile  `json:"camera_distance_m"`
	TimeReference      string     `json:"time_reference"`
	TimeFallbackReason string     `json:"time_fallback_reason"`
	SmoothedSpans      []spanFile `json:"smoothed_spans"`
}

type cameraFile struct {
	Latitude  number `json:"lat"`
	Longitude number `json:"lon"`
	Altitude  number `json:"altitude_m"`
}

type markerFile struct {
	Latitude  number `json:"lat"`
	Longitude number `json:"lon"`
	Distance  number `json:"distance_m"`
}

type frameFile struct {
	Index          int        `json:"index"`
	TimeSeconds    number     `json:"time_s"`
	Phase          string     `json:"phase"`
	Camera         cameraFile `json:"camera"`
	Heading        number     `json:"heading_deg"`
	Tilt           number     `json:"tilt_deg"`
	Marker         markerFile `json:"marker"`
	CameraToMarker number     `json:"camera_to_marker_m"`
}

// encodePlan renders plan as the plan file: the header (format version,
// parameters and summary) indented by two spaces, and each frame compact, on
// a single line, so the file is easy to inspect with head, grep and diff. It
// contains nothing that varies between runs (no timestamps, no paths), so a
// given plan always produces identical bytes.
func encodePlan(plan domain.CameraPlan) ([]byte, error) {
	duration := plan.Summary.Duration

	parameters, err := json.MarshalIndent(parametersFile{
		DurationSeconds: seconds(duration),
		FrameRate:       number{plan.Parameters.FrameRate, 6},
		Distance:        levelText(plan.Parameters.Distance),
		Tilt:            levelText(plan.Parameters.Tilt),
	}, "  ", "  ")
	if err != nil {
		return nil, err
	}

	spans := make([]spanFile, len(plan.Summary.SmoothedSpans))
	for i, span := range plan.Summary.SmoothedSpans {
		spans[i] = spanFile{seconds(span.Start), seconds(span.End), string(span.Quantity)}
	}
	summary, err := json.MarshalIndent(summaryFile{
		DurationSeconds:    seconds(duration),
		DurationMode:       string(plan.Summary.DurationMode),
		FrameRate:          number{plan.Summary.FrameRate, 6},
		FrameCount:         plan.Summary.FrameCount,
		CameraAltitude:     rangeFile{measure(plan.Summary.MinCameraAltitude), measure(plan.Summary.MaxCameraAltitude)},
		CameraDistance:     rangeFile{measure(plan.Summary.MinCameraDistance), measure(plan.Summary.MaxCameraDistance)},
		TimeReference:      string(plan.Summary.TimeReference),
		TimeFallbackReason: plan.TimeFallbackReason,
		SmoothedSpans:      spans,
	}, "  ", "  ")
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	out.WriteString("{\n  \"format_version\": " + strconv.Itoa(planFormatVersion) + ",\n")
	out.WriteString("  \"parameters\": " + string(parameters) + ",\n")
	out.WriteString("  \"summary\": " + string(summary) + ",\n")
	out.WriteString("  \"frames\": [")
	for i, frame := range plan.Frames {
		line, err := json.Marshal(frameFile{
			Index:          frame.Index,
			TimeSeconds:    seconds(frame.Time),
			Phase:          string(frame.Phase),
			Camera:         cameraFile{coordinate(frame.CameraLatitude), coordinate(frame.CameraLongitude), measure(frame.CameraAltitude)},
			Heading:        measure(frame.Heading),
			Tilt:           measure(frame.Tilt),
			Marker:         markerFile{coordinate(frame.MarkerLatitude), coordinate(frame.MarkerLongitude), measure(frame.MarkerDistance)},
			CameraToMarker: measure(frame.CameraToMarkerDistance),
		})
		if err != nil {
			return nil, err
		}

		if i > 0 {
			out.WriteString(",")
		}
		out.WriteString("\n    ")
		out.Write(line)
	}
	if len(plan.Frames) > 0 {
		out.WriteString("\n  ")
	}
	out.WriteString("]\n}\n")

	return out.Bytes(), nil
}

func levelText(level domain.Level) string {
	switch level {
	case domain.LevelLow:
		return "low"
	case domain.LevelHigh:
		return "high"
	default:
		return "medium"
	}
}
