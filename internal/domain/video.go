package domain

//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/video_encoder.go -package mockdomain . VideoEncoder
//go:generate go run go.uber.org/mock/mockgen -destination mockdomain/video_exporter.go -package mockdomain . VideoExporter

import (
	"context"
	"math"
	"strings"
	"time"
)

// VideoEncoder joins the frames of a flight into a video with an encoder that is
// not part of this tool. Concrete implementations live in
// internal/infra/outbound.
type VideoEncoder interface {
	// Probe says whether the encoder is there and can make the video. When it
	// cannot, the error is ErrEncoderUnavailable, saying what is missing and what
	// to do about it.
	Probe(ctx context.Context) (EncoderInfo, error)

	// Encode writes the video of job to job.Output, calling progress with how
	// many frames are encoded as it goes (progress may be nil). It stops early,
	// with an error that wraps the context's, when ctx is done; any other
	// failure is ErrVideoEncodingFailed. What it wrote is not the encoder's to
	// keep: whoever asked deletes it.
	Encode(ctx context.Context, job EncodeJob, progress func(encoded int)) error
}

// VideoExporter keeps a video in a file the user chose. Concrete implementations
// live in internal/infra/outbound.
type VideoExporter interface {
	// Check says, before anything is encoded, whether path can receive the video:
	// ErrVideoDestinationExists when it exists and overwrite is not true;
	// ErrVideoDestinationInvalid when it is a directory, or its folder is not
	// there or cannot be written to. It leaves nothing behind.
	Check(path string, overwrite bool) error

	// Export gives produce the path of a temporary file next to path — a real
	// path, that can be written to and moved around in — and publishes what
	// produce wrote as path, whole or not at all: when produce fails, or the file
	// cannot be published, nothing is left behind and a previous file at path
	// stays as it was. It returns the size of the file. Failing to publish is one
	// of the errors of Check; an error of produce is returned as it is.
	Export(path string, overwrite bool, produce func(temporary string) error) (size int64, err error)
}

// VideoRequest asks for the video of a plan from the frames in a directory.
type VideoRequest struct {
	Directory string
	Output    string
	Quality   VideoQuality
	Overwrite bool
}

// VideoProgress says how far the encoding got: Done frames of Total are encoded,
// after Elapsed.
type VideoProgress struct {
	Done, Total int
	Elapsed     time.Duration
}

// EncodeJob is what the encoder is asked: join the Frames frames of Directory —
// the files named FrameFileName(0) to FrameFileName(Frames-1) — into a video of
// the given frame rate and quality, and write it to Output.
type EncodeJob struct {
	Directory string
	Frames    int
	FrameRate float64
	Quality   VideoQuality
	Output    string
}

// EncoderInfo says which encoder made, or would make, the video.
type EncoderInfo struct {
	Name, Version, Codec string
}

// String is the encoder as the summary shows it: ffmpeg 7.1 (libx264).
func (i EncoderInfo) String() string {
	var b strings.Builder
	b.WriteString(i.Name)
	if i.Version != "" {
		b.WriteString(" " + i.Version)
	}
	b.WriteString(" (" + i.Codec + ")")
	return b.String()
}

// VideoSummary says what an assembly did (FR-017).
type VideoSummary struct {
	Frames     int
	FrameRate  float64
	Resolution Resolution
	Quality    VideoQuality
	Encoder    EncoderInfo

	// SizeBytes is the size of the video that was written.
	SizeBytes int64

	// Elapsed is how long the run took. It is about the run, and never goes into
	// the video.
	Elapsed time.Duration

	// Encoded is how many frames were encoded when the run ended; Interrupted is
	// true when the user interrupted it.
	Encoded     int
	Interrupted bool
}

// Duration is the duration of the video: its frames over its frame rate, to the
// millisecond. It is zero when there is no frame rate to divide by.
func (s VideoSummary) Duration() time.Duration {
	if s.FrameRate <= 0 || s.Frames <= 0 {
		return 0
	}
	return time.Duration(math.Round(float64(s.Frames)*1000/s.FrameRate)) * time.Millisecond
}
