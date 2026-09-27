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
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/inbound/cli"
)

func Test_RenderCommands_Interruption(t *testing.T) {
	plan := planOfFrames(60)
	slice := builddomain.NewGeoSliceBuilder().Build()

	// interrupt sends the signal to this process, as the user's Ctrl+C does, and
	// waits for the context the command gave the service to be done.
	interrupt := func(t *testing.T, ctx context.Context) bool {
		t.Helper()
		if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
			t.Fatal(err)
		}
		select {
		case <-ctx.Done():
			return true
		case <-time.After(3 * time.Second):
			return false
		}
	}

	t.Run("should cancel the context of the drawing of all the frames when the process is interrupted", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		var cancelled bool
		m.frameService.EXPECT().DrawFrames(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, _ domain.CameraPlan, _ domain.GeoSlice, _ domain.FrameSetRequest, _ func(domain.RenderProgress)) (domain.RenderSummary, error) {
				cancelled = interrupt(t, ctx)
				return domain.RenderSummary{Requested: 60, Drawn: 3, Resolution: defaultResolution, Interrupted: true}, domain.ErrRenderInterrupted
			})

		// when
		_, _, err := executeRenderAllWith(t, m, nil, "plan.json", "slice.zip", "--output", "/tmp/q")

		// then
		assert.True(t, cancelled, "the signal cancelled the context")
		assert.Equal(t, 38, cli.ExitCode(err))
	})

	t.Run("should cancel the context of the drawing of a single frame when the process is interrupted", func(t *testing.T) {
		// given
		m := newRenderCommandMocks(t)
		m.planService.EXPECT().Load(gomock.Any()).Return(plan, nil)
		m.sliceService.EXPECT().Load(gomock.Any()).Return(slice, nil)
		var cancelled bool
		m.frameService.EXPECT().DrawFrame(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, _ domain.CameraPlan, _ domain.GeoSlice, _ domain.SingleFrameRequest) (domain.RenderSummary, error) {
				cancelled = interrupt(t, ctx)
				return domain.RenderSummary{Requested: 1, Interrupted: true}, domain.ErrRenderInterrupted
			})

		// when
		_, _, err := executeRenderFrameCommand(t, m, "plan.json", "slice.zip", "--number", "1", "--output", "f.png")

		// then
		assert.True(t, cancelled)
		assert.Equal(t, 38, cli.ExitCode(err))
	})
}
