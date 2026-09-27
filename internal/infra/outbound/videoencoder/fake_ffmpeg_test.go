//go:build !windows

package videoencoder_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeScript is a stand-in for ffmpeg: a shell script that answers what the
// adapter asks of the real one (its version, its encoders), records how it was
// called, and behaves as FAKE_FFMPEG_MODE says when it is asked to encode:
//
//	(unset)      writes the output file and succeeds
//	nolibx264    has no libx264 among its encoders
//	noversion    fails when asked for its version
//	fail         says "No space left on device" and exits with 1
//	progress     prints blocks of -progress to its standard output, then succeeds
//	sleep        writes its pid to FAKE_FFMPEG_PID and waits for 30 seconds
//	flood        writes 1 MiB to its standard output and to its standard error, then succeeds
//	floodfail    the same, and exits with 1
const fakeScript = `#!/bin/sh
for a in "$@"; do
  if [ "$a" = "-version" ]; then
    if [ "$FAKE_FFMPEG_MODE" = "noversion" ]; then exit 1; fi
    echo "ffmpeg version 7.1 Copyright (c) 2000-2024 the FFmpeg developers"
    echo "built with a fake compiler"
    exit 0
  fi
  if [ "$a" = "-encoders" ]; then
    echo "Encoders:"
    echo " V....D mpeg4                mpeg4 codec"
    if [ "$FAKE_FFMPEG_MODE" != "nolibx264" ]; then
      echo " V....D libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (codec h264)"
    fi
    exit 0
  fi
done
if [ -n "$FAKE_FFMPEG_LOG" ]; then
  for a in "$@"; do printf '%s\n' "$a" >> "$FAKE_FFMPEG_LOG"; done
fi
if [ -n "$FAKE_FFMPEG_CWD" ]; then pwd > "$FAKE_FFMPEG_CWD"; fi
for last; do :; done
out="${last#file:}"
case "$FAKE_FFMPEG_MODE" in
  fail)
    echo "fake: No space left on device" >&2
    exit 1;;
  progress)
    printf 'frame=10\nfps=25.0\nprogress=continue\n'
    printf 'frame=60\nfps=25.0\nprogress=continue\n'
    printf 'frame=120\nfps=25.0\nprogress=end\n';;
  sleep)
    echo $$ > "$FAKE_FFMPEG_PID"
    exec sleep 30;;
  flood|floodfail)
    head -c 1048576 /dev/zero | tr '\000' 'x'
    head -c 1048576 /dev/zero | tr '\000' 'y' >&2
    if [ "$FAKE_FFMPEG_MODE" = "floodfail" ]; then exit 1; fi;;
esac
printf 'fake-mp4' > "$out"
exit 0
`

// fakeProgram is the fake ffmpeg of a test and where it records what it saw.
type fakeProgram struct {
	binary, log, cwd, pid string
}

// fakeFFmpeg writes the fake ffmpeg to a directory of the test and points its
// records at files there. A test picks a mode with mode.
func fakeFFmpeg(t *testing.T) fakeProgram {
	t.Helper()

	dir := t.TempDir()
	fake := fakeProgram{
		binary: filepath.Join(dir, "ffmpeg"),
		log:    filepath.Join(dir, "args.txt"),
		cwd:    filepath.Join(dir, "cwd.txt"),
		pid:    filepath.Join(dir, "pid.txt"),
	}
	require.NoError(t, os.WriteFile(fake.binary, []byte(fakeScript), 0o755))

	t.Setenv("FAKE_FFMPEG_LOG", fake.log)
	t.Setenv("FAKE_FFMPEG_CWD", fake.cwd)
	t.Setenv("FAKE_FFMPEG_PID", fake.pid)
	t.Setenv("FAKE_FFMPEG_MODE", "")
	return fake
}

// mode makes the fake behave as the mode says.
func (f fakeProgram) mode(t *testing.T, mode string) {
	t.Helper()
	t.Setenv("FAKE_FFMPEG_MODE", mode)
}

// args are the arguments the fake was called with to encode, one per line
// recorded, or nil if it was not.
func (f fakeProgram) args(t *testing.T) []string {
	t.Helper()

	data, err := os.ReadFile(f.log)
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

// workingDirectory is where the fake ran when it was asked to encode.
func (f fakeProgram) workingDirectory(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile(f.cwd)
	require.NoError(t, err)
	return strings.TrimSpace(string(data))
}
