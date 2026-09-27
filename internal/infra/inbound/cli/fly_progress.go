package cli

import (
	"fmt"
	"io"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// flyProgressCount is how many stages a single-command run always goes
// through (FlightStage's five values).
const flyProgressCount = 5

// flyProgressPrinter announces each of the five stages a single-command run
// goes through, and forwards the frame-rendering and video-encoding progress
// to the same printers "render all" and "video" already use, so the two most
// demanding stages read exactly like their own commands.
type flyProgressPrinter struct {
	out      io.Writer
	terminal bool

	stage     domain.FlightStage
	announced bool
	render    progressPrinter
	video     videoProgressPrinter
}

func newFlyProgressPrinter(out io.Writer, terminal bool) *flyProgressPrinter {
	return &flyProgressPrinter{
		out:      out,
		terminal: terminal,
		render:   progressPrinter{out: out, terminal: terminal},
		video:    videoProgressPrinter{out: out, terminal: terminal},
	}
}

func (p *flyProgressPrinter) report(progress domain.FlightProgress) {
	if !p.announced || progress.Stage != p.stage {
		p.render.finish()
		p.video.finish()
		fmt.Fprintf(p.out, "Stage %d/%d: %s%s\n", int(progress.Stage)+1, flyProgressCount, progress.Stage, reuseAnnouncement(progress))
		p.stage = progress.Stage
		p.announced = true
	}

	switch {
	case progress.Render != nil:
		p.render.report(*progress.Render)
	case progress.Video != nil:
		p.video.report(*progress.Video)
	}
}

// reuseAnnouncement says, for the camera-planning and geo-data-slicing stage
// announcements, that the plan/slice under --keep was reused instead of
// (re)computed (contracts/intermediates-directory.md); empty otherwise.
func reuseAnnouncement(progress domain.FlightProgress) string {
	if !progress.Reused {
		return ""
	}
	switch progress.Stage {
	case domain.StageCameraPlanning:
		return " (unchanged since the last run under --keep, reusing plan.json)"
	case domain.StageGeoDataSlicing:
		return " (unchanged, reusing slice.zip)"
	default:
		return ""
	}
}

// finish ends the line a terminal was rewriting.
func (p *flyProgressPrinter) finish() {
	p.render.finish()
	p.video.finish()
}
