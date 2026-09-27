//go:build !windows

package cli_test

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

func Test_VideoCommand_Interruption(t *testing.T) {
	plan := planOfFrames(60)

	// interrupt sends the signal to this process, as the user's Ctrl+C does, and
	// waits for the context the command gave the service to be done.
	interrupt := func(t *testing.T, signal syscall.Signal, ctx context.Context) bool {
		t.Helper()
		if err := syscall.Kill(os.Getpid(), signal); err != nil {
			t.Fatal(err)
		}
		select {
		case <-ctx.Done():
			return true
		case <-time.After(3 * time.Second):
			return false
		}
	}

	// assembling makes the video service stop when the signal arrives.
	assembling := func(t *testing.T, signal syscall.Signal) (cancelled *bool, m videoCommandMocks) {
		t.Helper()
		m = newVideoCommandMocks(t)
		cancelled = new(bool)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.videoService.EXPECT().Assemble(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, _ domain.CameraPlan, _ domain.VideoRequest, _ func(domain.VideoProgress)) (domain.VideoSummary, error) {
				*cancelled = interrupt(t, signal, ctx)
				return domain.VideoSummary{Frames: 60, Encoded: 3, Interrupted: true}, domain.ErrVideoInterrupted
			})
		return cancelled, m
	}

	t.Run("should cancel the context of the assembly when the process is interrupted", func(t *testing.T) {
		// given
		cancelled, m := assembling(t, syscall.SIGINT)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4")

		// then
		assert.True(t, *cancelled, "the signal cancelled the context")
		assert.Equal(t, 49, cli.ExitCode(err))
	})

	t.Run("should cancel the context of the assembly when the process is asked to stop", func(t *testing.T) {
		// given
		cancelled, m := assembling(t, syscall.SIGTERM)

		// when
		_, _, err := executeVideoCommand(t, m, domain.VideoQualityMedium, "plan.json", "frames", "--output", "flight.mp4")

		// then
		assert.True(t, *cancelled, "the signal cancelled the context")
		assert.Equal(t, 49, cli.ExitCode(err))
	})
}
