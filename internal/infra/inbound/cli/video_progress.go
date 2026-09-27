package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// videoLogInterval is how often the progress of an encoding is written when
// stderr is not a terminal: the encoder reports every half second, which would
// fill a log.
const videoLogInterval = 5 * time.Second

// videoProgressPrinter shows how far an encoding got: on a terminal, one line that
// is rewritten with each report; elsewhere (a log), a new line at most every five
// seconds and at the last report.
type videoProgressPrinter struct {
	out         io.Writer
	terminal    bool
	written     bool
	lastPrinted time.Duration
}

func (p *videoProgressPrinter) report(progress domain.VideoProgress) {
	text := fmt.Sprintf("Encoding frame %d/%d (%.1f%%), elapsed %s",
		progress.Done, progress.Total, 100*float64(progress.Done)/float64(progress.Total), formatElapsed(progress.Elapsed))

	if p.terminal {
		fmt.Fprintf(p.out, "\r%s", text)
		p.written = true
		return
	}
	if progress.Done == progress.Total || progress.Elapsed-p.lastPrinted >= videoLogInterval {
		fmt.Fprintln(p.out, text)
		p.lastPrinted = progress.Elapsed
	}
}

// finish ends the line a terminal was rewriting.
func (p *videoProgressPrinter) finish() {
	if p.terminal && p.written {
		fmt.Fprintln(p.out)
	}
}
