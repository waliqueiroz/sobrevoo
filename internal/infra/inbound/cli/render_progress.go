package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// RenderOption changes how a command that draws frames talks to its user.
type RenderOption func(*renderSettings)

type renderSettings struct {
	isTerminal func(io.Writer) bool
}

// WithTerminalCheck replaces how a command tells whether stderr is a terminal,
// where it rewrites one line of progress instead of writing a new line now and
// then. A test uses it; by default the command asks the file.
func WithTerminalCheck(check func(io.Writer) bool) RenderOption {
	return func(s *renderSettings) { s.isTerminal = check }
}

func newRenderSettings(options []RenderOption) renderSettings {
	settings := renderSettings{isTerminal: isTerminal}
	for _, option := range options {
		option(&settings)
	}
	return settings
}

// isTerminal says whether w is a file that is a terminal.
func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// interruptContext is the context of a command, cancelled when the user
// interrupts the process (Ctrl+C) or asks it to stop: the drawing then ends in
// an orderly way, without a partial image, and says what was done.
func interruptContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

// progressPrinter shows how far a drawing got: on a terminal, one line that is
// rewritten after each frame; elsewhere (a log), a new line every ten frames and
// at the last one, so it does not fill the file.
type progressPrinter struct {
	out      io.Writer
	terminal bool
	written  bool
}

func (p *progressPrinter) report(progress domain.RenderProgress) {
	text := fmt.Sprintf("Drawing frame %d/%d (%.1f%%), elapsed %s",
		progress.Done, progress.Total, 100*float64(progress.Done)/float64(progress.Total), formatElapsed(progress.Elapsed))

	if p.terminal {
		fmt.Fprintf(p.out, "\r%s", text)
		p.written = true
		return
	}
	if progress.Done%10 == 0 || progress.Done == progress.Total {
		fmt.Fprintln(p.out, text)
	}
}

// finish ends the line a terminal was rewriting.
func (p *progressPrinter) finish() {
	if p.terminal && p.written {
		fmt.Fprintln(p.out)
	}
}
