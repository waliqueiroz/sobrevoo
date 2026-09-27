package videoencoder

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// readProgress reads what ffmpeg writes for -progress — blocks of key=value lines
// — until it ends, and calls progress with the number of frames encoded each time
// it grows. Whatever it cannot make sense of is left aside, and whatever is left
// to read when it stops is read and thrown away, so the program that writes it
// never blocks. progress may be nil.
func readProgress(r io.Reader, progress func(encoded int)) {
	scanner := bufio.NewScanner(r)

	last := 0
	for scanner.Scan() {
		value, ok := strings.CutPrefix(scanner.Text(), "frame=")
		if !ok {
			continue
		}

		encoded, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || encoded <= last {
			continue
		}
		last = encoded
		if progress != nil {
			progress(encoded)
		}
	}

	// The scanner stops at a line it cannot hold as well as at the end.
	_, _ = io.Copy(io.Discard, r)
}
