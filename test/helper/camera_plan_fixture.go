package helper

import (
	"encoding/json"
	"fmt"
	"math"
)

// PlanFrameSpec is one frame of a PlanFileSpec.
type PlanFrameSpec struct {
	Phase                           string
	CameraLat, CameraLon, CameraAlt float64
	Heading, Tilt                   float64
	MarkerLat, MarkerLon            float64
	MarkerDistance, CameraToMarker  float64

	// ActivityTime is activity_time_s; MarkerElevation and MarkerGain are
	// marker.elevation_m and marker.gain_m (009-frame-overlays);
	// MarkerSpeed is marker.speed_mps (014-speed-overlay-block).
	ActivityTime                float64
	MarkerElevation, MarkerGain float64
	MarkerSpeed                 float64
}

// PlanFileSpec describes a camera plan file fixture, in the format of
// specs/003-camera-path-planning/contracts/plan-file.md. It does not use the
// jsonfile adapter, so the tests of the reader stay independent of the writer.
type PlanFileSpec struct {
	Frames          []PlanFrameSpec
	DurationSeconds float64
	FrameRate       float64

	// AspectRatio is parameters.aspect_ratio; empty leaves the field out, as
	// in a plan made before it existed.
	AspectRatio string
}

// DefaultPlanFileSpec is a plan of three following frames, 0.1 s at 30 fps,
// flying north along longitude -46.6 from latitude -23.5.
func DefaultPlanFileSpec() PlanFileSpec {
	frame := func(lat, distance float64) PlanFrameSpec {
		return PlanFrameSpec{
			Phase:     "following",
			CameraLat: lat - 0.005, CameraLon: -46.6, CameraAlt: 400,
			Heading: 0, Tilt: 45,
			MarkerLat: lat, MarkerLon: -46.6,
			MarkerDistance: distance, CameraToMarker: 600,
			ActivityTime: distance / 5, MarkerElevation: 760 + distance/10, MarkerGain: distance / 10,
			MarkerSpeed: 5,
		}
	}
	return PlanFileSpec{
		Frames:          []PlanFrameSpec{frame(-23.5, 0), frame(-23.499, 111), frame(-23.498, 222)},
		DurationSeconds: 0.1,
		FrameRate:       30,
	}
}

func planDocument(spec PlanFileSpec) map[string]any {
	frames := make([]any, len(spec.Frames))
	minAlt, maxAlt := math.Inf(1), math.Inf(-1)
	minDistance, maxDistance := math.Inf(1), math.Inf(-1)
	for i, f := range spec.Frames {
		frames[i] = map[string]any{
			"index":           i,
			"time_s":          float64(i) / spec.FrameRate,
			"activity_time_s": f.ActivityTime,
			"phase":           f.Phase,
			"camera":          map[string]any{"lat": f.CameraLat, "lon": f.CameraLon, "altitude_m": f.CameraAlt},
			"heading_deg":     f.Heading,
			"tilt_deg":        f.Tilt,
			"marker": map[string]any{
				"lat": f.MarkerLat, "lon": f.MarkerLon, "distance_m": f.MarkerDistance,
				"elevation_m": f.MarkerElevation, "gain_m": f.MarkerGain, "speed_mps": f.MarkerSpeed,
			},
			"camera_to_marker_m": f.CameraToMarker,
		}
		minAlt, maxAlt = math.Min(minAlt, f.CameraAlt), math.Max(maxAlt, f.CameraAlt)
		minDistance, maxDistance = math.Min(minDistance, f.CameraToMarker), math.Max(maxDistance, f.CameraToMarker)
	}
	if len(spec.Frames) == 0 {
		minAlt, maxAlt, minDistance, maxDistance = 0, 0, 0, 0
	}

	parameters := map[string]any{
		"duration_s": spec.DurationSeconds, "frame_rate": spec.FrameRate,
		"distance": "medium", "tilt": "medium",
	}
	if spec.AspectRatio != "" {
		parameters["aspect_ratio"] = spec.AspectRatio
	}

	return map[string]any{
		"format_version": 3,
		"parameters":     parameters,
		"summary": map[string]any{
			"duration_s":           spec.DurationSeconds,
			"duration_mode":        "automatic",
			"frame_rate":           spec.FrameRate,
			"frame_count":          len(spec.Frames),
			"camera_altitude_m":    map[string]any{"min": minAlt, "max": maxAlt},
			"camera_distance_m":    map[string]any{"min": minDistance, "max": maxDistance},
			"time_reference":       "clock",
			"time_fallback_reason": "",
			"smoothed_spans":       []any{},
			"elevation_available":  true,
		},
		"frames": frames,
	}
}

func marshalPlan(document map[string]any) []byte {
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		panic(fmt.Errorf("encoding the plan file fixture: %w", err))
	}
	return data
}

// ValidPlanFile returns a plan file for spec.
func ValidPlanFile(spec PlanFileSpec) []byte {
	return marshalPlan(planDocument(spec))
}

// PlanFileWithVersion returns a valid plan file with another format_version.
func PlanFileWithVersion(version int) []byte {
	document := planDocument(DefaultPlanFileSpec())
	document["format_version"] = version
	return marshalPlan(document)
}

// PlanFileWithoutFormatVersion returns a plan file with no format_version.
func PlanFileWithoutFormatVersion() []byte {
	document := planDocument(DefaultPlanFileSpec())
	delete(document, "format_version")
	return marshalPlan(document)
}

// PlanFileWithoutFrames returns a plan file whose frames list is empty.
func PlanFileWithoutFrames() []byte {
	return ValidPlanFile(PlanFileSpec{DurationSeconds: 0, FrameRate: 30})
}

// PlanFileWithFrameCountMismatch returns a plan file whose summary says 7
// frames while it lists three.
func PlanFileWithFrameCountMismatch() []byte {
	document := planDocument(DefaultPlanFileSpec())
	document["summary"].(map[string]any)["frame_count"] = 7
	return marshalPlan(document)
}

// PlanFileWithLatitudeOutOfRange returns a plan file whose second frame has a
// marker at latitude 91.
func PlanFileWithLatitudeOutOfRange() []byte {
	spec := DefaultPlanFileSpec()
	spec.Frames[1].MarkerLat = 91
	return ValidPlanFile(spec)
}

// PlanFileWithoutFrameField returns a plan file whose second frame lacks the
// given key ("marker", "camera", "camera_to_marker_m" or "activity_time_s").
func PlanFileWithoutFrameField(field string) []byte {
	document := planDocument(DefaultPlanFileSpec())
	delete(document["frames"].([]any)[1].(map[string]any), field)
	return marshalPlan(document)
}

// PlanFileWithoutMarkerField returns a plan file whose second frame's marker
// lacks the given key ("elevation_m", "gain_m" or "speed_mps").
func PlanFileWithoutMarkerField(field string) []byte {
	document := planDocument(DefaultPlanFileSpec())
	marker := document["frames"].([]any)[1].(map[string]any)["marker"].(map[string]any)
	delete(marker, field)
	return marshalPlan(document)
}

// PlanFileWithoutSummaryField returns a plan file whose summary lacks the
// given key ("elevation_available").
func PlanFileWithoutSummaryField(field string) []byte {
	document := planDocument(DefaultPlanFileSpec())
	delete(document["summary"].(map[string]any), field)
	return marshalPlan(document)
}

// PlanFileWithoutSection returns a plan file without a top-level key
// ("parameters" or "summary").
func PlanFileWithoutSection(section string) []byte {
	document := planDocument(DefaultPlanFileSpec())
	delete(document, section)
	return marshalPlan(document)
}

// PlanFileWithoutFrameCount returns a plan file whose summary has no
// frame_count.
func PlanFileWithoutFrameCount() []byte {
	document := planDocument(DefaultPlanFileSpec())
	delete(document["summary"].(map[string]any), "frame_count")
	return marshalPlan(document)
}

// PlanFileWithUnknownField returns a valid plan file with extra fields that a
// reader must ignore.
func PlanFileWithUnknownField() []byte {
	document := planDocument(DefaultPlanFileSpec())
	document["generator"] = "someone else"
	document["frames"].([]any)[0].(map[string]any)["extra"] = 42
	return marshalPlan(document)
}

// TruncatedPlanFile returns half of a valid plan file.
func TruncatedPlanFile() []byte {
	data := ValidPlanFile(DefaultPlanFileSpec())
	return data[:len(data)/2]
}

// NotJSONContent returns bytes that are not JSON at all.
func NotJSONContent() []byte {
	return []byte("this is not a plan file")
}
