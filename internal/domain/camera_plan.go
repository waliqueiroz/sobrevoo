package domain

//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/camera_plan_exporter.go -package mockdomain . CameraPlanExporter
//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/camera_plan_reader.go -package mockdomain . CameraPlanReader

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
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

// CameraPlanReader reads back a camera plan a CameraPlanExporter wrote.
// Concrete implementations live in internal/infra/outbound.
type CameraPlanReader interface {
	// Read reads the plan file at path. It fails with ErrPlanFileInvalid
	// when the file is not a plan or is malformed, and with
	// ErrPlanFormatVersionUnsupported for a format version it does not
	// know; an I/O error opening or reading the file is returned wrapped,
	// with no sentinel.
	Read(path string) (CameraPlan, error)
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

// Validate checks that a plan is coherent with itself, so a plan read from a
// file can be trusted: it has frames, their number is the duration times the
// frame rate, and every frame is in order, in a known phase, at valid
// coordinates and at a finite, non-negative distance from the camera to the
// marker. The error is ErrPlanFileInvalid and always names the field.
func (c CameraPlan) Validate() error {
	invalid := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrPlanFileInvalid, fmt.Sprintf(format, args...))
	}

	if len(c.Frames) == 0 {
		return invalid("the plan has no frames")
	}
	if c.Parameters.Duration == nil {
		return invalid("the duration is missing")
	}
	if math.IsNaN(c.Parameters.FrameRate) || math.IsInf(c.Parameters.FrameRate, 0) || c.Parameters.FrameRate <= 0 {
		return invalid("frame rate is %g, must be a positive number", c.Parameters.FrameRate)
	}
	if expected := c.Parameters.FrameCount(*c.Parameters.Duration); expected != len(c.Frames) {
		return invalid("the plan has %d frames but duration × frame rate is %d", len(c.Frames), expected)
	}

	for i, f := range c.Frames {
		switch {
		case f.Index != i:
			return invalid("frames[%d].index is %d, must be %d", i, f.Index, i)
		case f.Phase != PhaseOpening && f.Phase != PhaseFollowing && f.Phase != PhaseClosing:
			return invalid("frames[%d].phase is %q, must be opening, following or closing", i, f.Phase)
		case !validLatitude(f.CameraLatitude):
			return invalid("frames[%d].camera.lat is %g, must be between -90 and 90", i, f.CameraLatitude)
		case !validLongitude(f.CameraLongitude):
			return invalid("frames[%d].camera.lon is %g, must be between -180 and 180", i, f.CameraLongitude)
		case !validLatitude(f.MarkerLatitude):
			return invalid("frames[%d].marker.lat is %g, must be between -90 and 90", i, f.MarkerLatitude)
		case !validLongitude(f.MarkerLongitude):
			return invalid("frames[%d].marker.lon is %g, must be between -180 and 180", i, f.MarkerLongitude)
		case math.IsNaN(f.CameraToMarkerDistance) || math.IsInf(f.CameraToMarkerDistance, 0) || f.CameraToMarkerDistance < 0:
			return invalid("frames[%d].camera_to_marker_m is %g, must be a finite number of at least 0", i, f.CameraToMarkerDistance)
		}
	}

	return nil
}

func validLatitude(degrees float64) bool {
	return !math.IsNaN(degrees) && degrees >= -90 && degrees <= 90
}

func validLongitude(degrees float64) bool {
	return !math.IsNaN(degrees) && degrees >= -180 && degrees <= 180
}

// minimumCosLatitude keeps the width of an area, in degrees of longitude,
// finite at the poles, where a meter spans ever more degrees.
const minimumCosLatitude = 1e-6

// AreaOfInterest is the geographic area the flight needs to see: for each
// frame, the square of half-side MarginFactor × the camera-to-marker
// distance around the marker (the camera is always inside it), and the box
// that holds all of them. Longitudes are unwrapped along the plan, as
// Route.BoundingBox does, so a plan crossing the antimeridian gives a small
// area that CrossesAntimeridian, never one spanning the planet; the width is
// capped at one turn. Coordinates are rounded to 1e-7°, so the area — and
// everything derived from it — is the same on every platform.
func (c CameraPlan) AreaOfInterest(tuning SliceTuning) BoundingBox {
	if len(c.Frames) == 0 {
		return BoundingBox{}
	}

	minLat, maxLat := math.Inf(1), math.Inf(-1)
	minLon, maxLon := math.Inf(1), math.Inf(-1)
	unwrapped := c.Frames[0].MarkerLongitude
	previous := unwrapped

	for i, f := range c.Frames {
		if i > 0 {
			delta := f.MarkerLongitude - previous
			switch {
			case delta > 180:
				delta -= 360
			case delta < -180:
				delta += 360
			}
			unwrapped += delta
			previous = f.MarkerLongitude
		}

		meters := tuning.MarginFactor * f.CameraToMarkerDistance
		halfLat := meters / MetersPerDegree
		cosLat := math.Max(math.Cos(f.MarkerLatitude*math.Pi/180), minimumCosLatitude)
		halfLon := meters / (MetersPerDegree * cosLat)

		minLat = math.Min(minLat, f.MarkerLatitude-halfLat)
		maxLat = math.Max(maxLat, f.MarkerLatitude+halfLat)
		minLon = math.Min(minLon, unwrapped-halfLon)
		maxLon = math.Max(maxLon, unwrapped+halfLon)
	}

	area := BoundingBox{
		MinLatitude: math.Max(minLat, -90),
		MaxLatitude: math.Min(maxLat, 90),
	}

	switch {
	case maxLon-minLon >= 360:
		area.MinLongitude, area.MaxLongitude = -180, 180
	case minLon < -180 || maxLon > 180:
		area.MinLongitude, area.MaxLongitude = normalizeLongitude(minLon), normalizeLongitude(maxLon)
		area.CrossesAntimeridian = true
	default:
		area.MinLongitude, area.MaxLongitude = minLon, maxLon
	}

	return area.rounded()
}

// ID identifies the plan by its content: the SHA-256, in lowercase
// hexadecimal, of a canonical encoding of its effective parameters and of every
// frame, with each value rounded to the step the plan itself uses (1e-7° for
// coordinates, 1e-3 for lengths and angles) and written as a 64-bit integer in
// a fixed byte order. It is the same for the same content whether the plan
// comes from memory or was read back from a file, so a slice can say which
// plan it was made from (a plan file's formatting never matters). The summary
// is derived from the frames and does not take part.
func (c CameraPlan) ID() string {
	hash := sha256.New()
	write := func(v int64) {
		var buffer [8]byte
		binary.BigEndian.PutUint64(buffer[:], uint64(v))
		hash.Write(buffer[:])
	}
	writeText := func(text string) {
		write(int64(len(text)))
		hash.Write([]byte(text))
	}
	quantized := func(v, step float64) int64 { return int64(math.Round(v / step)) }

	writeText("sobrevoo-plan-v1")

	var duration int64
	if c.Parameters.Duration != nil {
		duration = int64(*c.Parameters.Duration)
	}
	write(duration)
	write(int64(math.Float64bits(c.Parameters.FrameRate)))
	write(int64(c.Parameters.Distance))
	write(int64(c.Parameters.Tilt))

	write(int64(len(c.Frames)))
	for _, f := range c.Frames {
		write(int64(f.Index))
		writeText(string(f.Phase))
		write(quantized(f.CameraLatitude, coordinateStep))
		write(quantized(f.CameraLongitude, coordinateStep))
		write(quantized(f.CameraAltitude, lengthStep))
		write(quantized(f.Heading, angleStep))
		write(quantized(f.Tilt, angleStep))
		write(quantized(f.MarkerLatitude, coordinateStep))
		write(quantized(f.MarkerLongitude, coordinateStep))
		write(quantized(f.MarkerDistance, lengthStep))
		write(quantized(f.CameraToMarkerDistance, lengthStep))
	}

	return hex.EncodeToString(hash.Sum(nil))
}
