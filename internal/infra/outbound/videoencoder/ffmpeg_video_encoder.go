// Package videoencoder implements the outbound adapters of the domain.VideoEncoder
// port: the programs, outside this tool, that join the frames of a flight into a
// video. The package is named after the port and each strategy after the program
// it runs.
package videoencoder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

const (
	// codec is the video encoder of ffmpeg that makes the H.264 of the video.
	codec = "libx264"

	// x264Threads is how many threads libx264 encodes with. x264 gives the same
	// bytes only for the same number of threads, so it is fixed, and not as many as
	// the machine has: the same video on any machine. Changing it changes the bytes
	// of every video: a new version of the tool.
	x264Threads = 4

	// errorTailBytes is how much of what ffmpeg says on its standard error goes
	// into the error of an encoding that failed: the end, where the reason is.
	errorTailBytes = 2048
)

// FFmpeg implements domain.VideoEncoder with the ffmpeg program: it reads the
// frames as a sequence of images and writes an MP4 of H.264 video, with no audio
// (specs/006-video-assembly/research.md, items 1 to 6, and contracts/video-file.md).
type FFmpeg struct {
	binary string
}

// NewFFmpeg creates an FFmpeg that runs the program called binary, which is
// looked for on the PATH unless it is a path.
func NewFFmpeg(binary string) FFmpeg {
	return FFmpeg{binary: binary}
}

// Probe finds the program, asks its version and checks that it has the libx264
// encoder.
func (e FFmpeg) Probe(ctx context.Context) (domain.EncoderInfo, error) {
	path, err := exec.LookPath(e.binary)
	if err != nil {
		return domain.EncoderInfo{}, e.unavailable(err)
	}

	version, err := exec.CommandContext(ctx, path, "-hide_banner", "-version").Output()
	if err != nil {
		return domain.EncoderInfo{}, e.failedToRun(ctx, err)
	}
	encoders, err := exec.CommandContext(ctx, path, "-hide_banner", "-encoders").Output()
	if err != nil {
		return domain.EncoderInfo{}, e.failedToRun(ctx, err)
	}
	info := domain.EncoderInfo{Name: "ffmpeg", Version: versionOf(version), Codec: codec}
	if !hasEncoder(encoders, codec) {
		found := filepath.Base(e.binary)
		if info.Version != "" {
			found += " " + info.Version
		}
		return domain.EncoderInfo{}, fmt.Errorf("%w: %s has no %s encoder; install a build that includes it (%s)",
			domain.ErrEncoderUnavailable, found, codec, installHintUnix)
	}

	return info, nil
}

// How to get ffmpeg, as the messages of a program that is missing say it.
const (
	installHintUnix = "macOS: brew install ffmpeg; Debian/Ubuntu: sudo apt install ffmpeg"
	installHintAll  = installHintUnix + "; Windows: winget install Gyan.FFmpeg"
)

// failedToRun is the error for a program that could not be run to the end: the
// error of the context when it is what stopped it, since that says nothing about
// the program.
func (e FFmpeg) failedToRun(ctx context.Context, cause error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("stopped: %w", ctx.Err())
	}
	return e.unavailable(cause)
}

// unavailable is the error for a program that cannot be run: one that is not
// there says what to install and how to check it; any other says why it could
// not be run.
func (e FFmpeg) unavailable(cause error) error {
	if errors.Is(cause, exec.ErrNotFound) || errors.Is(cause, fs.ErrNotExist) {
		where := ""
		if !strings.ContainsRune(e.binary, filepath.Separator) && !strings.ContainsRune(e.binary, '/') {
			where = " on the PATH"
		}
		return fmt.Errorf("%w: %q was not found%s; install it (%s), then check it with: %s -version",
			domain.ErrEncoderUnavailable, e.binary, where, installHintAll, e.binary)
	}
	return fmt.Errorf("%w: %q could not be run: %w", domain.ErrEncoderUnavailable, e.binary, cause)
}

// versionOf reads the version off the first line of what ffmpeg prints for
// -version: "ffmpeg version 7.1 Copyright ...".
func versionOf(output []byte) string {
	first, _, _ := strings.Cut(string(output), "\n")
	fields := strings.Fields(first)
	if len(fields) >= 3 && fields[1] == "version" {
		return fields[2]
	}
	return ""
}

// hasEncoder says whether the list ffmpeg prints for -encoders has the encoder:
// a line of flags, the name of the encoder and what it is.
func hasEncoder(output []byte, name string) bool {
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == name {
			return true
		}
	}
	return false
}

// Encode runs ffmpeg over the frames of job, which it reads from the directory of
// the frames by their names alone, so nothing in the path is taken for a pattern
// or a protocol.
func (e FFmpeg) Encode(ctx context.Context, job domain.EncodeJob, progress func(encoded int)) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("encoding stopped: %w", err)
	}

	cmd := exec.CommandContext(ctx, e.binary, e.arguments(job)...)
	cmd.Dir = job.Directory

	stderr := &tailBuffer{limit: errorTailBytes}
	cmd.Stderr = stderr

	// What ffmpeg says on its standard output is the progress, read as it comes;
	// its standard error, the reason for a failure, is kept by the command.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return e.unavailable(err)
	}
	if err := cmd.Start(); err != nil {
		return e.failedToRun(ctx, err)
	}
	readProgress(stdout, progress)

	// A context that is done kills ffmpeg (exec.CommandContext); the video is
	// not to be kept, and it did not fail.
	err = cmd.Wait()
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return fmt.Errorf("encoding stopped: %w", ctx.Err())
	}

	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		return e.unavailable(err)
	}
	return e.failed(exit.ExitCode(), stderr.String())
}

// failed is the error of an ffmpeg that exited with an error, with what it said.
func (e FFmpeg) failed(code int, said string) error {
	name := filepath.Base(e.binary)
	if said == "" {
		return fmt.Errorf("%w: %s exited with status %d", domain.ErrVideoEncodingFailed, name, code)
	}
	return fmt.Errorf("%w: %s exited with status %d: %s", domain.ErrVideoEncodingFailed, name, code, said)
}

// x264Settings is what a quality asks of libx264: the preset says how much time
// it takes to look for a smaller video of the same look, the constant rate factor
// how faithful the video is (the lower, the more, and the bigger the file).
type x264Settings struct {
	preset string
	crf    int
}

// settingsFor is what each quality asks of libx264 (specs/006-video-assembly/
// contracts/video-file.md): the two go together, so the size of the file grows
// from low to high. Changing them changes the bytes of the video: a new version
// of the tool.
func settingsFor(quality domain.VideoQuality) x264Settings {
	switch quality {
	case domain.VideoQualityLow:
		return x264Settings{preset: "veryfast", crf: 28}
	case domain.VideoQualityHigh:
		return x264Settings{preset: "slow", crf: 18}
	default:
		return x264Settings{preset: "medium", crf: 23}
	}
}

// arguments are the arguments of ffmpeg for the job.
func (e FFmpeg) arguments(job domain.EncodeJob) []string {
	settings := settingsFor(job.Quality)

	return []string{
		// -bitexact (global) is what actually keeps ffmpeg and libx264 from
		// writing their own version and build into the file — an mp4-muxer
		// "encoder" tag with the version (-fflags/-flags:v +bitexact only drop
		// the version, not the tag itself, which -metadata:s:v:0 encoder=
		// clears below) and a "user data unregistered" SEI inside the H.264
		// stream itself, which no libx264 option turns off — that one is
		// removed with the bitstream filter near the end. -xerror is what makes
		// a frame that cannot be decoded (its PNG signature and chunks are
		// whole, so Verify let it through, but the compressed pixel data inside
		// is broken) a hard failure instead of something ffmpeg tolerates and
		// quietly works around.
		"-nostdin", "-hide_banner", "-loglevel", "error", "-y", "-bitexact", "-xerror",
		"-nostats", "-progress", "pipe:1",
		"-framerate", strconv.FormatFloat(job.FrameRate, 'f', -1, 64),
		"-start_number", "0",
		"-i", domain.FrameFilePattern,
		"-frames:v", strconv.Itoa(job.Frames),
		"-vf", "scale=out_color_matrix=bt709:out_range=tv:flags=accurate_rnd+full_chroma_int+bitexact,format=yuv420p",
		"-c:v", codec, "-preset", settings.preset, "-crf", strconv.Itoa(settings.crf), "-profile:v", "high",
		"-x264-params", "threads=" + strconv.Itoa(x264Threads),
		"-colorspace", "bt709", "-color_primaries", "bt709", "-color_trc", "bt709", "-color_range", "tv",
		// Nothing of the machine or the moment goes into the file: bitexact stops the
		// muxer and the encoder from writing the date and their own version, no
		// metadata of the frames is carried over, the encoder tag itself is cleared,
		// and the SEI the encoder writes into the stream (type 6, which nothing this
		// tool needs) is removed (specs/006-video-assembly/contracts/video-file.md).
		"-fflags", "+bitexact", "-flags:v", "+bitexact", "-map_metadata", "-1", "-metadata:s:v:0", "encoder=",
		"-bsf:v", "filter_units=remove_types=6",
		"-an", "-movflags", "+faststart", "-f", "mp4",
		"file:" + job.Output,
	}
}

// tailBuffer keeps the last limit bytes written to it: what a program said last
// is what says why it stopped.
type tailBuffer struct {
	limit int
	data  []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	if len(b.data) > b.limit {
		b.data = append([]byte(nil), b.data[len(b.data)-b.limit:]...)
	}
	return len(p), nil
}

// String is what was kept, with the blank space at both ends taken off.
func (b *tailBuffer) String() string {
	return string(bytes.TrimSpace(b.data))
}
