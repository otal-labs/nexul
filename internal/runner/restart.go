package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/coder/websocket"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/version"
)

const (
	// restartAnswerTimeout bounds the wait for a runner to agree to restart; it answers before it goes.
	restartAnswerTimeout = 15 * time.Second
	// lastWithoutRestart is the newest release whose personal runner does not answer restart.
	lastWithoutRestart = "v0.3.77-beta"
)

// ErrCannotRestart means the computer's runner is too old to restart when Nexul asks.
var ErrCannotRestart = errors.New("the runner cannot restart on request")

// ---- server side ----

// RestartComputerRunner asks computerID's runner to restart itself, and returns once it agreed, before it is back.
func (h *Handler) RestartComputerRunner(ctx context.Context, computerID string) error {
	c, err := h.computerConn(ctx, computerID)
	if err != nil {
		return err
	}
	if c.version == "" || !version.Newer(c.version, lastWithoutRestart) {
		return ErrCannotRestart
	}
	f, err := h.ask(ctx, c, Frame{Type: FrameRestart}, restartAnswerTimeout)
	if err != nil {
		return fmt.Errorf("ask the runner of computer %s to restart: %w", computerID, err)
	}
	if f.Error != "" {
		return fmt.Errorf("%w: %s", apperrs.ErrConflict, f.Error)
	}
	return nil
}

// ---- runner side ----

// answerRestart agrees, then starts the runner again in place, so it comes back with the PATH its computer has saved
// now; a runner that is not personal, or cannot find its own binary, says why and keeps running.
func (c *Client) answerRestart(ctx context.Context, conn *websocket.Conn, f Frame) {
	exe, err := c.restartTarget()
	if err != nil {
		c.sendFrame(ctx, conn, Frame{Type: FrameRestartResult, ID: f.ID, Error: err.Error()})
		return
	}
	c.sendFrame(ctx, conn, Frame{Type: FrameRestartResult, ID: f.ID})
	c.log.Info("restarting on request", "runner", c.cfg.Name)
	if err := reexecFn(exe, os.Args[1:], os.Environ()); err != nil {
		c.log.Error("restart failed; still running", "runner", c.cfg.Name, "error", err)
	}
}

func (c *Client) restartTarget() (string, error) {
	if !c.cfg.Personal {
		return "", errors.New("this runner is not a personal runner")
	}
	exe, err := runnerExecutableFn()
	if err != nil {
		return "", fmt.Errorf("find the Nexul app's own binary: %w", err)
	}
	return exe, nil
}
