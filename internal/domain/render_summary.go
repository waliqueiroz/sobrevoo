package domain

import "time"

// RenderSummary says what a drawing did (FR-026).
type RenderSummary struct {
	// Requested is how many frames were asked for; Drawn how many this run
	// drew, and Kept how many it left as they were because the destination
	// already had them.
	Requested, Drawn, Kept int

	// MapHoleFrames and ElevationHoleFrames count the frames this run drew that
	// show terrain with no map image, or over cells with no elevation; a frame
	// with both is in both.
	MapHoleFrames, ElevationHoleFrames int

	// Removed is how many frames of a previous set the run removed.
	Removed int

	Resolution Resolution

	// Elapsed is how long the run took. It is about the run, and never goes
	// into an image.
	Elapsed time.Duration

	// Interrupted is true when the user interrupted the run.
	Interrupted bool
}

// Add counts a frame that was just drawn.
func (s *RenderSummary) Add(stats FrameStats) {
	s.Drawn++
	if stats.MapHole {
		s.MapHoleFrames++
	}
	if stats.ElevationHole {
		s.ElevationHoleFrames++
	}
}

// RenderProgress says how far a drawing got: Done frames of Total are ready
// (those the destination already had count), after Elapsed.
type RenderProgress struct {
	Done, Total int
	Elapsed     time.Duration
}
