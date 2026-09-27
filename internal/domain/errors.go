package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for the domain's business failures (Constitution
// Principle VII). Each inbound/outbound adapter is responsible for
// translating these into its own representation (e.g. process exit codes
// in the CLI, HTTP status codes in a future REST adapter).
var (
	// ErrEmptyFile is returned when the input file has no content at all (FR-005).
	ErrEmptyFile = errors.New("track file is empty")

	// ErrUnsupportedFormat is returned when the input content does not match
	// any supported track format (FR-004).
	ErrUnsupportedFormat = errors.New("unsupported track file format")

	// ErrInsufficientPoints is returned when the track has fewer points than
	// the minimum required to compute the summary, before any cleaning is
	// applied (FR-006).
	ErrInsufficientPoints = errors.New("track has insufficient points")

	// ErrInsufficientPointsAfterCleaning is returned when the track has
	// enough points originally, but falls below the minimum required after
	// discarding invalid points (FR-006).
	ErrInsufficientPointsAfterCleaning = errors.New("track has insufficient points after cleaning")

	// ErrDataFileNotFound is returned when a geo data file path does not
	// exist (FR-004).
	ErrDataFileNotFound = errors.New("geo data file not found")

	// ErrDataFileUnreadable is returned when a geo data file exists but
	// cannot be opened/read (FR-005).
	ErrDataFileUnreadable = errors.New("geo data file cannot be read")

	// ErrUnsupportedDataFormat is returned when a geo data file's content
	// does not match any recognized base map or elevation format (FR-006).
	ErrUnsupportedDataFormat = errors.New("unsupported geo data file format")

	// ErrDataSourceNameAlreadyUsed is returned when registering a name that
	// already identifies another registered source (FR-007).
	ErrDataSourceNameAlreadyUsed = errors.New("geo data source name already in use")

	// ErrDataSourceNotRegistered is returned when a name does not match any
	// registered geo data source (FR-012).
	ErrDataSourceNotRegistered = errors.New("geo data source not registered")

	// ErrInvalidDuration reports a requested video duration that is not
	// greater than zero or exceeds the maximum accepted duration.
	ErrInvalidDuration = errors.New("invalid duration")

	// ErrInvalidFrameRate reports a frame rate that is not a finite number
	// within the accepted range.
	ErrInvalidFrameRate = errors.New("invalid frame rate")

	// ErrInvalidAspectRatio reports a video aspect ratio that is not a
	// "WIDTH:HEIGHT" pair of whole numbers within the accepted range.
	ErrInvalidAspectRatio = errors.New("invalid aspect ratio")

	// ErrDurationTooShort reports a requested duration shorter than the
	// minimum needed to plan a smooth flight over the track.
	ErrDurationTooShort = errors.New("duration too short for this track")

	// ErrTrackTooShort reports a track whose length is below the minimum
	// that can be followed by a camera.
	ErrTrackTooShort = errors.New("track too short to plan a camera path")

	// ErrTrackTooLarge reports a track whose span exceeds the maximum the
	// planner supports.
	ErrTrackTooLarge = errors.New("track too large to plan a camera path")

	// ErrPlanDestinationExists reports an export destination that already
	// holds a file, when overwriting was not requested.
	ErrPlanDestinationExists = errors.New("plan destination already exists")

	// ErrPlanDestinationInvalid reports an export destination that cannot be
	// written (missing directory, no permission, ...).
	ErrPlanDestinationInvalid = errors.New("plan destination is not writable")

	// ErrPlanFileInvalid reports a camera plan file that cannot be used: not
	// a plan at all, missing required fields, truncated, or incoherent with
	// itself.
	ErrPlanFileInvalid = errors.New("camera plan file is invalid")

	// ErrPlanFormatVersionUnsupported reports a camera plan file whose
	// format version this tool does not recognize.
	ErrPlanFormatVersionUnsupported = errors.New("camera plan file format version is not supported")

	// ErrAreaNotCovered reports that the area of a plan is not fully covered
	// by the registered base maps and elevation data. It is carried by
	// AreaNotCoveredError, which also holds the coverage report.
	ErrAreaNotCovered = errors.New("area is not fully covered by the registered geo data")

	// ErrSliceTooLarge reports a slice bigger than the documented maximum.
	ErrSliceTooLarge = errors.New("geo data slice is too large")

	// ErrGeoDataContentUnreadable reports a registered file whose content
	// cannot be read: corrupted, truncated, or using an encoding this tool
	// does not support.
	ErrGeoDataContentUnreadable = errors.New("geo data content cannot be read")

	// ErrElevationUnitUnsupported reports an elevation file whose vertical
	// unit cannot be converted to meters.
	ErrElevationUnitUnsupported = errors.New("elevation unit is not supported")

	// ErrSliceDestinationExists reports an export destination that already
	// exists, when overwriting was not requested.
	ErrSliceDestinationExists = errors.New("slice destination already exists")

	// ErrSliceDestinationInvalid reports an export destination that cannot
	// be written (missing directory, no permission, ...).
	ErrSliceDestinationInvalid = errors.New("slice destination is not writable")

	// ErrElevationNotCovered reports a coordinate that no registered
	// elevation source covers. It differs from an elevation reading without
	// a value: there, a source covers the point but the file has no data.
	ErrElevationNotCovered = errors.New("no registered elevation data covers this point")

	// ErrInvalidCoordinate reports a latitude or longitude that is not a
	// finite number within the valid range.
	ErrInvalidCoordinate = errors.New("invalid coordinate")

	// ErrSliceFileInvalid reports a geo data slice file that cannot be used:
	// not a slice at all, missing required fields, truncated, corrupted,
	// incoherent with itself, without the identification of its plan, or
	// holding a tile that is not an image.
	ErrSliceFileInvalid = errors.New("geo data slice file is invalid")

	// ErrSliceFormatVersionUnsupported reports a slice file whose format
	// version this tool does not recognize.
	ErrSliceFormatVersionUnsupported = errors.New("geo data slice file format version is not supported")

	// ErrSliceDoesNotMatchPlan reports a slice that was made from another
	// plan than the one it is used with.
	ErrSliceDoesNotMatchPlan = errors.New("geo data slice does not match the camera plan")

	// ErrSliceDoesNotCoverPlan reports a slice whose area does not contain the
	// area the plan needs.
	ErrSliceDoesNotCoverPlan = errors.New("geo data slice does not cover the camera plan")

	// ErrTileFormatUnsupported reports base map tiles this tool cannot draw:
	// vector tiles, or any format that is not an image.
	ErrTileFormatUnsupported = errors.New("tile format is not supported for drawing")

	// ErrNoElevationData reports a slice in which no elevation sample has a
	// value: there is no terrain to draw and no height to put the camera at.
	ErrNoElevationData = errors.New("geo data slice has no elevation data")

	// ErrFrameOutOfRange reports a frame number that is not a whole number of
	// the plan's frames.
	ErrFrameOutOfRange = errors.New("frame number is out of range")

	// ErrInvalidResolution reports an image resolution outside the documented
	// limits, or that is not written as WIDTHxHEIGHT.
	ErrInvalidResolution = errors.New("invalid resolution")

	// ErrFrameDestinationInvalid reports a frame destination that cannot be
	// used (not a directory, missing directory, no permission, ...).
	ErrFrameDestinationInvalid = errors.New("frame destination cannot be used")

	// ErrFrameDestinationExists reports a single frame file that already
	// exists, when overwriting was not requested.
	ErrFrameDestinationExists = errors.New("frame destination already exists")

	// ErrFrameSetConflict reports a directory that holds frames of another set
	// (or files named like frames that are not this tool's), when overwriting
	// was not requested.
	ErrFrameSetConflict = errors.New("frame destination holds frames of another set")

	// ErrRenderInterrupted reports a drawing the user interrupted; the frames
	// that were finished stay valid.
	ErrRenderInterrupted = errors.New("drawing was interrupted")

	// ErrFrameDirectoryInvalid reports a frame directory that cannot be joined
	// into a video: one that does not exist, is not a directory, cannot be read,
	// or holds no frame this tool drew.
	ErrFrameDirectoryInvalid = errors.New("frame directory cannot be used")

	// ErrFrameSequenceInvalid reports frames that do not number the frames of
	// the plan: some missing, some the plan has no number for, or a number that
	// repeats.
	ErrFrameSequenceInvalid = errors.New("frames do not form the flight of the plan")

	// ErrFrameResolutionInvalid reports frames of different resolutions, or of
	// an odd width or height, which the video cannot have.
	ErrFrameResolutionInvalid = errors.New("frame resolution cannot be used for the video")

	// ErrFramesDoNotMatchPlan reports frames drawn from another plan, or of more
	// than one set mixed in a directory.
	ErrFramesDoNotMatchPlan = errors.New("frames were not drawn from the plan")

	// ErrFramesWithoutPlanID reports frames of this tool that do not say which
	// plan they were drawn from, as the ones drawn before frames did.
	ErrFramesWithoutPlanID = errors.New("frames do not say which plan they were drawn from")

	// ErrFrameFileInvalid reports a frame file that is not a whole PNG image.
	ErrFrameFileInvalid = errors.New("frame file cannot be used")

	// ErrEncoderUnavailable reports a video encoder that is not there, or that
	// cannot make the video.
	ErrEncoderUnavailable = errors.New("video encoder not available")

	// ErrVideoDestinationExists reports a video destination that already exists,
	// when overwriting was not requested.
	ErrVideoDestinationExists = errors.New("video destination already exists")

	// ErrVideoDestinationInvalid reports a video destination that cannot be
	// written: a directory, or a file whose folder is missing or not writable.
	ErrVideoDestinationInvalid = errors.New("video destination cannot be used")

	// ErrVideoInterrupted reports an assembly the user interrupted; no video was
	// written.
	ErrVideoInterrupted = errors.New("video assembly was interrupted")

	// ErrVideoEncodingFailed reports a video encoder that failed while encoding.
	ErrVideoEncodingFailed = errors.New("video encoding failed")
)

// AreaNotCoveredError is the error for an area that the registered geo data
// does not fully cover (ErrAreaNotCovered). It carries the coverage report so
// an adapter can tell the user exactly what is missing.
type AreaNotCoveredError struct {
	Report CoverageReport
}

// Error lists every uncovered stretch, one per line, the way "geodata
// check" shows them.
func (e *AreaNotCoveredError) Error() string {
	var b strings.Builder
	b.WriteString(ErrAreaNotCovered.Error())
	for _, segment := range e.Report.UncoveredSegments {
		fmt.Fprintf(&b, "\n  missing %s from (%.6f, %.6f) to (%.6f, %.6f)",
			segment.Missing,
			segment.StartLatitude, segment.StartLongitude,
			segment.EndLatitude, segment.EndLongitude)
	}
	return b.String()
}

// Is makes errors.Is(err, ErrAreaNotCovered) true for this error.
func (e *AreaNotCoveredError) Is(target error) bool {
	return target == ErrAreaNotCovered
}
