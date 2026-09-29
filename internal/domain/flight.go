package domain

import "time"

// FlightRequest is everything a single-command run (the seventh stage) needs
// beyond the track reader: the same parameters "plan"/"render all"/"video"
// already accept, the video destination, and where — if anywhere — to keep
// the intermediates.
type FlightRequest struct {
	// Parameters plans the camera: duration, frame rate, distance, tilt and
	// aspect ratio, exactly as PlanParameters already means.
	Parameters PlanParameters

	// Resolution is the size of the frames drawn.
	Resolution Resolution

	// Appearance is the trail, the marker and the background the frames are
	// drawn with (008-frame-appearance).
	Appearance Appearance

	// Overlay is the screen overlay configuration the frames are drawn with
	// (009-frame-overlays).
	Overlay OverlayConfig

	// Quality is the quality of the video.
	Quality VideoQuality

	// Output is the video file to write.
	Output string

	// Keep is the directory the plan, the slice and the frames are kept in
	// and reused from; empty means they live only for this run, in a
	// temporary directory of the run's own, removed at the end.
	Keep string

	// Overwrite allows replacing an existing video at Output, and a
	// plan/slice/frames under Keep that belong to another set.
	Overwrite bool
}

// FlightStage is one of the five stages a single-command run goes through, in
// order.
type FlightStage int

const (
	StageTrackProcessing FlightStage = iota
	StageCameraPlanning
	StageGeoDataSlicing
	StageFrameRendering
	StageVideoEncoding
)

// String is the stage's label, as FlightProgress reports it.
func (s FlightStage) String() string {
	switch s {
	case StageTrackProcessing:
		return "treating the track"
	case StageCameraPlanning:
		return "planning the camera"
	case StageGeoDataSlicing:
		return "slicing the geo data"
	case StageFrameRendering:
		return "drawing the frames"
	case StageVideoEncoding:
		return "encoding the video"
	default:
		return "unknown stage"
	}
}

// FlightProgress is reported as a run moves through its five stages: Stage is
// always set; Render and Video carry the same progress the frame-rendering and
// video-encoding stages already report on their own, forwarded as-is — nil
// outside of their stage.
type FlightProgress struct {
	Stage  FlightStage
	Render *RenderProgress
	Video  *VideoProgress

	// Reused is true when the stage announcement (Render and Video both nil)
	// for StageCameraPlanning or StageGeoDataSlicing means the plan/slice
	// under Keep was reused, not (re)computed; meaningless for the other
	// stages, which are always (re)done.
	Reused bool
}

// FlightSummary is what a run is left with — what completed, what was reused,
// and the summaries of the two stages that already have one of their own.
type FlightSummary struct {
	// Completed are the stages that finished, in order; a run that stopped
	// early ends here on failure or interruption.
	Completed []FlightStage

	// PlanReused/SliceReused say whether the plan/slice under Keep already
	// matched this run's track and values, so Generate was not called for it.
	PlanReused  bool
	SliceReused bool

	// FramesDirectory is the directory the frames were drawn into: the
	// temporary directory of the run's own, or Keep's "frames" subdirectory.
	// Empty until it is resolved (before the frame-rendering stage starts).
	FramesDirectory string

	// Render is FrameService.DrawFrames's own summary; the zero value if
	// frame rendering never started.
	Render RenderSummary

	// Video is VideoService.Assemble's own summary; the zero value if video
	// encoding never started.
	Video VideoSummary

	// Elapsed is how long the whole run took — not just Video.Elapsed.
	Elapsed time.Duration

	// Interrupted is true when the user interrupted the run.
	Interrupted bool
}
