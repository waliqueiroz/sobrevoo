//go:build !windows

package videoencoder_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/videoencoder"
)

// job is a job to encode: the frames are where the test says, the output next to
// them.
func job(t *testing.T) domain.EncodeJob {
	t.Helper()

	dir := t.TempDir()
	return domain.EncodeJob{
		Directory: dir,
		Frames:    1139,
		FrameRate: 29.97,
		Quality:   domain.VideoQualityMedium,
		Output:    filepath.Join(dir, "out.tmp"),
	}
}

// indexOf is the position of the argument, and fails the test if it is not there.
func indexOf(t *testing.T, args []string, argument string) int {
	t.Helper()
	for i, given := range args {
		if given == argument {
			return i
		}
	}
	require.Failf(t, "argument not found", "%q is not among %v", argument, args)
	return -1
}

// assertInOrder asserts that the arguments appear, one after the other and next
// to each other, in the list.
func assertInOrder(t *testing.T, args []string, sequence ...string) {
	t.Helper()
	start := indexOf(t, args, sequence[0])
	require.LessOrEqual(t, start+len(sequence), len(args), "%v is cut short", sequence)
	assert.Equal(t, sequence, args[start:start+len(sequence)])
}

func Test_FFmpeg_Probe(t *testing.T) {
	t.Run("should say the program, its version and that it has libx264", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)

		// when
		info, err := videoencoder.NewFFmpeg(fake.binary).Probe(context.Background())

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.EncoderInfo{Name: "ffmpeg", Version: "7.1", Codec: "libx264"}, info)
		assert.Equal(t, "ffmpeg 7.1 (libx264)", info.String())
	})

	t.Run("should not encode anything, and leave no trace of a call to encode", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)

		// when
		_, err := videoencoder.NewFFmpeg(fake.binary).Probe(context.Background())

		// then
		require.NoError(t, err)
		assert.Nil(t, fake.args(t))
	})

	t.Run("should refuse an encoder that is not there, as unavailable", func(t *testing.T) {
		// given
		binary := filepath.Join(t.TempDir(), "no-such-program")

		// when
		_, err := videoencoder.NewFFmpeg(binary).Probe(context.Background())

		// then
		assert.ErrorIs(t, err, domain.ErrEncoderUnavailable)
	})
}

func Test_FFmpeg_Encode(t *testing.T) {
	t.Run("should run in the directory of the frames and read them by their names alone", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		encoding := job(t)

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), encoding, nil)

		// then
		require.NoError(t, err)
		want, _ := filepath.EvalSymlinks(encoding.Directory)
		got, _ := filepath.EvalSymlinks(fake.workingDirectory(t))
		assert.Equal(t, want, got)
		assertInOrder(t, fake.args(t), "-framerate", "29.97", "-start_number", "0", "-i", "frame_%06d.png", "-frames:v", "1139")
	})

	t.Run("should write the frame rate as the shortest decimal that stands for it", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)

		// when
		for _, rate := range []float64{30, 29.97, 59.94, 12.5, 120, 1} {
			encoding := job(t)
			encoding.FrameRate = rate
			require.NoError(t, videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), encoding, nil))
		}

		// then
		var rates []string
		args := fake.args(t)
		for i, argument := range args {
			if argument == "-framerate" {
				rates = append(rates, args[i+1])
			}
		}
		assert.Equal(t, []string{"30", "29.97", "59.94", "12.5", "120", "1"}, rates)
	})

	t.Run("should ask for a video of the frames of the job, of the medium quality", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), nil)

		// then
		require.NoError(t, err)
		args := fake.args(t)
		assertInOrder(t, args, "-nostdin", "-hide_banner", "-loglevel", "error", "-y")
		assertInOrder(t, args, "-c:v", "libx264", "-preset", "medium", "-crf", "23", "-profile:v", "high")
		assertInOrder(t, args, "-colorspace", "bt709", "-color_primaries", "bt709", "-color_trc", "bt709", "-color_range", "tv")
		assertInOrder(t, args, "-an", "-movflags", "+faststart", "-f", "mp4")
		assertInOrder(t, args, "-vf", "scale=out_color_matrix=bt709:out_range=tv:flags=accurate_rnd+full_chroma_int+bitexact,format=yuv420p")
	})

	t.Run("should write the video to the output of the job, given as a file and not as anything else the encoder reads", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		encoding := job(t)

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), encoding, nil)

		// then
		require.NoError(t, err)
		args := fake.args(t)
		assert.Equal(t, "file:"+encoding.Output, args[len(args)-1])
		content, readErr := os.ReadFile(encoding.Output)
		require.NoError(t, readErr)
		assert.Equal(t, "fake-mp4", string(content))
	})

	t.Run("should not put the directory of the frames among the arguments, only in the working directory", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		encoding := job(t)

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), encoding, nil)

		// then
		require.NoError(t, err)
		for _, argument := range fake.args(t) {
			if argument != "file:"+encoding.Output {
				assert.NotContains(t, argument, encoding.Directory)
			}
		}
	})

	t.Run("should fail with the reason the encoder gave when it exits with an error", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		fake.mode(t, "fail")

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), nil)

		// then
		require.ErrorIs(t, err, domain.ErrVideoEncodingFailed)
		assert.EqualError(t, err, "video encoding failed: ffmpeg exited with status 1: fake: No space left on device")
	})

	t.Run("should not wait for an encoder that fills its outputs, and keep only the end of what it said", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		fake.mode(t, "floodfail")
		done := make(chan error, 1)

		// when
		go func() { done <- videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), nil) }()

		// then
		select {
		case err := <-done:
			require.ErrorIs(t, err, domain.ErrVideoEncodingFailed)
			assert.LessOrEqual(t, len(err.Error()), 2048+128, "the end of the error output, and no more")
			assert.True(t, strings.HasSuffix(err.Error(), "y"), "the end of what was said, not the start")
		case <-time.After(10 * time.Second):
			require.Fail(t, "the encoder blocked on its outputs")
		}
	})

	t.Run("should not wait for an encoder that fills its outputs and succeeds", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		fake.mode(t, "flood")
		done := make(chan error, 1)

		// when
		go func() { done <- videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), nil) }()

		// then
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(10 * time.Second):
			require.Fail(t, "the encoder blocked on its outputs")
		}
	})

	t.Run("should fail as unavailable, not as a failed encoding, when the program cannot be run at all", func(t *testing.T) {
		// given
		binary := filepath.Join(t.TempDir(), "no-such-program")

		// when
		err := videoencoder.NewFFmpeg(binary).Encode(context.Background(), job(t), nil)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEncoderUnavailable)
	})
}

func Test_FFmpeg_Probe_Unavailable(t *testing.T) {
	t.Run("should say the program was not found on the PATH, what to install and how to check it, with the name it was given", func(t *testing.T) {
		// given
		encoder := videoencoder.NewFFmpeg("sobrevoo-no-such-encoder")

		// when
		_, err := encoder.Probe(context.Background())

		// then
		require.ErrorIs(t, err, domain.ErrEncoderUnavailable)
		assert.EqualError(t, err, `video encoder not available: "sobrevoo-no-such-encoder" was not found on the PATH; `+
			`install it (macOS: brew install ffmpeg; Debian/Ubuntu: sudo apt install ffmpeg; Windows: winget install Gyan.FFmpeg), `+
			`then check it with: sobrevoo-no-such-encoder -version`)
	})

	t.Run("should say a path to a program that is not there was not found, without saying it is on the PATH", func(t *testing.T) {
		// given
		binary := filepath.Join(t.TempDir(), "no-such-program")

		// when
		_, err := videoencoder.NewFFmpeg(binary).Probe(context.Background())

		// then
		require.ErrorIs(t, err, domain.ErrEncoderUnavailable)
		assert.ErrorContains(t, err, `"`+binary+`" was not found`)
		assert.NotContains(t, err.Error(), "on the PATH")
		assert.ErrorContains(t, err, "brew install ffmpeg")
	})

	t.Run("should say what the program found does not have, its version, and what to install", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		fake.mode(t, "nolibx264")

		// when
		_, err := videoencoder.NewFFmpeg(fake.binary).Probe(context.Background())

		// then
		require.ErrorIs(t, err, domain.ErrEncoderUnavailable)
		assert.EqualError(t, err, "video encoder not available: ffmpeg 7.1 has no libx264 encoder; "+
			"install a build that includes it (macOS: brew install ffmpeg; Debian/Ubuntu: sudo apt install ffmpeg)")
	})

	t.Run("should say a file that cannot be run could not be, with the reason", func(t *testing.T) {
		// given
		binary := filepath.Join(t.TempDir(), "ffmpeg")
		require.NoError(t, os.WriteFile(binary, []byte("#!/bin/sh\n"), 0o644))

		// when
		_, err := videoencoder.NewFFmpeg(binary).Probe(context.Background())

		// then
		require.ErrorIs(t, err, domain.ErrEncoderUnavailable)
		assert.ErrorContains(t, err, `"`+binary+`" could not be run`)
	})

	t.Run("should say a program that fails when asked its version could not be run", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		fake.mode(t, "noversion")

		// when
		_, err := videoencoder.NewFFmpeg(fake.binary).Probe(context.Background())

		// then
		require.ErrorIs(t, err, domain.ErrEncoderUnavailable)
		assert.ErrorContains(t, err, "could not be run")
	})

	t.Run("should say the same when the program cannot be started to encode", func(t *testing.T) {
		// given
		encoder := videoencoder.NewFFmpeg(filepath.Join(t.TempDir(), "no-such-program"))

		// when
		err := encoder.Encode(context.Background(), job(t), nil)

		// then
		require.ErrorIs(t, err, domain.ErrEncoderUnavailable)
		assert.ErrorContains(t, err, "was not found")
	})
}

func Test_FFmpeg_Encode_Quality(t *testing.T) {
	// encodeWith encodes a job of the quality and gives the arguments the fake was
	// called with.
	encodeWith := func(t *testing.T, quality domain.VideoQuality) []string {
		t.Helper()
		fake := fakeFFmpeg(t)
		encoding := job(t)
		encoding.Quality = quality
		require.NoError(t, videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), encoding, nil))
		return fake.args(t)
	}

	t.Run("should encode the low quality with the veryfast preset and a constant rate factor of 28", func(t *testing.T) {
		// given / when
		args := encodeWith(t, domain.VideoQualityLow)

		// then
		assertInOrder(t, args, "-preset", "veryfast", "-crf", "28")
	})

	t.Run("should encode the medium quality with the medium preset and a constant rate factor of 23", func(t *testing.T) {
		// given / when
		args := encodeWith(t, domain.VideoQualityMedium)

		// then
		assertInOrder(t, args, "-preset", "medium", "-crf", "23")
	})

	t.Run("should encode the high quality with the slow preset and a constant rate factor of 18", func(t *testing.T) {
		// given / when
		args := encodeWith(t, domain.VideoQualityHigh)

		// then
		assertInOrder(t, args, "-preset", "slow", "-crf", "18")
	})

	t.Run("should ask for the same in every quality but the preset and the rate factor", func(t *testing.T) {
		// given
		without := func(args []string) []string {
			var kept []string
			for i := 0; i < len(args); i++ {
				if args[i] == "-preset" || args[i] == "-crf" {
					i++
					continue
				}
				kept = append(kept, args[i])
			}
			return kept
		}

		// when
		low := without(encodeWith(t, domain.VideoQualityLow))
		medium := without(encodeWith(t, domain.VideoQualityMedium))
		high := without(encodeWith(t, domain.VideoQualityHigh))

		// then: the output is a different temporary file each time, and the rest is the same
		assert.Equal(t, low[:len(low)-1], medium[:len(medium)-1])
		assert.Equal(t, medium[:len(medium)-1], high[:len(high)-1])
	})
}

func Test_FFmpeg_Encode_Progress(t *testing.T) {
	t.Run("should ask ffmpeg to say how far it got on its standard output, and not to print statistics", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), nil)

		// then
		require.NoError(t, err)
		assertInOrder(t, fake.args(t), "-nostats", "-progress", "pipe:1")
	})

	t.Run("should report the frames encoded, in order, as ffmpeg says them", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		fake.mode(t, "progress")
		var reported []int

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), func(encoded int) { reported = append(reported, encoded) })

		// then
		require.NoError(t, err)
		assert.Equal(t, []int{10, 60, 120}, reported)
	})

	t.Run("should accept no function to report to", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		fake.mode(t, "progress")

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), nil)

		// then
		assert.NoError(t, err)
	})

	t.Run("should keep the reason the encoder gave when it fails after having reported progress", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		fake.mode(t, "fail")

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), func(int) {})

		// then
		assert.ErrorContains(t, err, "No space left on device")
	})

	t.Run("should not block on an encoder that fills its standard output with what is not progress", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		fake.mode(t, "flood")
		done := make(chan error, 1)

		// when
		go func() { done <- videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), func(int) {}) }()

		// then
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(10 * time.Second):
			require.Fail(t, "the encoder blocked on its outputs")
		}
	})
}

func Test_FFmpeg_Encode_Cancel(t *testing.T) {
	// waitForPid waits for the fake to say it is running, and gives its pid.
	waitForPid := func(t *testing.T, fake fakeProgram) int {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if data, err := os.ReadFile(fake.pid); err == nil {
				if pid, convErr := strconv.Atoi(strings.TrimSpace(string(data))); convErr == nil {
					return pid
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
		require.Fail(t, "the encoder did not start")
		return 0
	}

	t.Run("should stop, with the error of the context and not as a failed encoding, when the context is cancelled", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		fake.mode(t, "sleep")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- videoencoder.NewFFmpeg(fake.binary).Encode(ctx, job(t), nil) }()
		waitForPid(t, fake)

		// when
		cancel()

		// then
		select {
		case err := <-done:
			require.ErrorIs(t, err, context.Canceled)
			assert.NotErrorIs(t, err, domain.ErrVideoEncodingFailed)
		case <-time.After(3 * time.Second):
			require.Fail(t, "the encoding did not stop")
		}
	})

	t.Run("should not leave the encoder running", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		fake.mode(t, "sleep")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- videoencoder.NewFFmpeg(fake.binary).Encode(ctx, job(t), nil) }()
		pid := waitForPid(t, fake)

		// when
		cancel()
		<-done

		// then
		gone := false
		for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
			if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
				gone = true
				break
			}
		}
		assert.True(t, gone, "the process %d is still there", pid)
	})

	t.Run("should not even start the encoder when the context is already cancelled", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(ctx, job(t), nil)

		// then
		require.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, fake.args(t), "the encoder was not called")
	})

	t.Run("should not take a cancelled probe for an encoder that is not there", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// when
		_, err := videoencoder.NewFFmpeg(fake.binary).Probe(ctx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func Test_FFmpeg_Encode_Reproducible(t *testing.T) {
	// argumentsOf encodes the job with a fresh fake and gives the arguments it got.
	argumentsOf := func(t *testing.T, encoding domain.EncodeJob) []string {
		t.Helper()
		fake := fakeFFmpeg(t)
		require.NoError(t, videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), encoding, nil))
		return fake.args(t)
	}

	t.Run("should keep the date, the version and the metadata of the inputs out of the video", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), nil)

		// then
		require.NoError(t, err)
		args := fake.args(t)
		assertInOrder(t, args, "-fflags", "+bitexact", "-flags:v", "+bitexact", "-map_metadata", "-1", "-metadata:s:v:0", "encoder=")
	})

	t.Run("should ask for the global bitexact mode, which is what actually keeps ffmpeg's and libx264's own version out of the file", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), nil)

		// then
		require.NoError(t, err)
		assert.Contains(t, fake.args(t), "-bitexact")
	})

	t.Run("should ask ffmpeg to stop on any decode error, so a frame that cannot be decoded fails the encoding instead of being silently tolerated", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), nil)

		// then
		require.NoError(t, err)
		assert.Contains(t, fake.args(t), "-xerror")
	})

	t.Run("should strip the SEI libx264 writes into the stream itself, which no encoder option turns off", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), nil)

		// then
		require.NoError(t, err)
		assertInOrder(t, fake.args(t), "-bsf:v", "filter_units=remove_types=6")
	})

	t.Run("should encode with a number of threads that does not depend on the machine, and without the message that says the version", func(t *testing.T) {
		// given
		fake := fakeFFmpeg(t)

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), job(t), nil)

		// then
		require.NoError(t, err)
		assertInOrder(t, fake.args(t), "-x264-params", "threads=4")
	})

	t.Run("should ask for exactly this, in this order, for a job of the medium quality", func(t *testing.T) {
		// given
		encoding := job(t)
		encoding.Frames = 380
		encoding.FrameRate = 10

		// when
		args := argumentsOf(t, encoding)

		// then: what changes the bytes of the video is what is written here; changing it is a new version of the tool
		assert.Equal(t, []string{
			"-nostdin", "-hide_banner", "-loglevel", "error", "-y", "-bitexact", "-xerror",
			"-nostats", "-progress", "pipe:1",
			"-framerate", "10", "-start_number", "0", "-i", "frame_%06d.png", "-frames:v", "380",
			"-vf", "scale=out_color_matrix=bt709:out_range=tv:flags=accurate_rnd+full_chroma_int+bitexact,format=yuv420p",
			"-c:v", "libx264", "-preset", "medium", "-crf", "23", "-profile:v", "high",
			"-x264-params", "threads=4",
			"-colorspace", "bt709", "-color_primaries", "bt709", "-color_trc", "bt709", "-color_range", "tv",
			"-fflags", "+bitexact", "-flags:v", "+bitexact", "-map_metadata", "-1", "-metadata:s:v:0", "encoder=",
			"-bsf:v", "filter_units=remove_types=6",
			"-an", "-movflags", "+faststart", "-f", "mp4",
			"file:" + encoding.Output,
		}, args)
	})

	t.Run("should ask for the same whatever the frames, the frame rate and the folders of the job are, but for their own values", func(t *testing.T) {
		// given
		first := job(t)
		second := job(t)
		second.Frames, second.FrameRate = 1500, 25

		// when
		one := argumentsOf(t, first)
		other := argumentsOf(t, second)

		// then: the frame rate and the number of frames are the only values of the job
		differing := 0
		for i := range one[:len(one)-1] {
			if one[i] != other[i] {
				differing++
			}
		}
		assert.Equal(t, 2, differing)
		assert.Len(t, other, len(one))
	})

	t.Run("should ask for the same at another time and from another working directory", func(t *testing.T) {
		// given
		first := job(t)
		second := job(t)
		second.Frames, second.FrameRate = first.Frames, first.FrameRate

		// when
		one := argumentsOf(t, first)
		time.Sleep(1100 * time.Millisecond)
		t.Chdir(t.TempDir())
		other := argumentsOf(t, second)

		// then
		assert.Equal(t, one[:len(one)-1], other[:len(other)-1])
	})

	t.Run("should not put anything of the environment among the arguments", func(t *testing.T) {
		// given
		encoding := job(t)
		fake := fakeFFmpeg(t)
		host, _ := os.Hostname()
		home, _ := os.UserHomeDir()
		user := os.Getenv("USER")

		// when
		err := videoencoder.NewFFmpeg(fake.binary).Encode(context.Background(), encoding, nil)

		// then
		require.NoError(t, err)
		for _, argument := range fake.args(t) {
			if argument == "file:"+encoding.Output {
				continue
			}
			for _, secret := range []string{host, home, user, encoding.Directory, time.Now().Format("2006")} {
				// Short names would match inside any argument.
				if len(secret) >= 5 {
					assert.NotContains(t, argument, secret)
				}
			}
		}
	})
}
