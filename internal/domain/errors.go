package domain

import "errors"

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
)
