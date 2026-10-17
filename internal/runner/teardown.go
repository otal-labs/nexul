package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// stackTeardownTimeout bounds one teardown; `docker compose down` waits up to ten seconds per container to stop it.
const stackTeardownTimeout = 2 * time.Minute

// handleStackDeleted is the service.deleted consumer: a deleted managed stack's containers go with it. An unmanaged
// stack's were started by someone else, so they stay.
func (h *Handler) handleStackDeleted(ctx context.Context, ev eventbus.Event) error {
	var e StackDeletedEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", TopicStackDeleted, err))
	}
	if !e.Managed || e.Machine == "" {
		return nil
	}
	f := Frame{Type: FrameStackTeardown, StackSlug: e.Slug, Strategy: e.Strategy}
	if err := f.Validate(); err != nil {
		h.log.Warn("stack teardown skipped", "machine", e.Machine, "slug", e.Slug, "error", err)
		return nil
	}
	h.teardown(ctx, e.Machine, f)
	return nil
}

// teardown sends f to a connected runner on machine. With none connected, or a failed send, it is saved and sent when
// one connects.
func (h *Handler) teardown(ctx context.Context, machine string, f Frame) {
	h.mu.Lock()
	c := h.machineConnLocked(machine)
	h.mu.Unlock()
	if c == nil {
		h.log.Info("stack teardown waits for a runner", "machine", machine, "slug", f.StackSlug)
		h.saveTeardown(ctx, machine, f)
		return
	}
	if err := h.sendFrame(ctx, c, f); err != nil {
		h.log.Warn("stack teardown send failed, kept for the next connect", "machine", machine, "slug", f.StackSlug, "error", err)
		h.saveTeardown(ctx, machine, f)
	}
}

func (h *Handler) saveTeardown(ctx context.Context, machine string, f Frame) {
	if h.cfg.Teardowns == nil {
		h.log.Warn("stack teardown dropped, no store to keep it", "machine", machine, "slug", f.StackSlug)
		return
	}
	if err := h.cfg.Teardowns.Save(ctx, machine, f); err != nil {
		h.log.Error("stack teardown could not be kept", "machine", machine, "slug", f.StackSlug, "error", err)
	}
}

// flushTeardowns sends the teardowns kept for c's machine and drops each once sent; it runs before queued deploys so a
// slug reused since is cleared before it is deployed again. A failed send leaves that teardown and the ones after it
// for the next connect.
func (h *Handler) flushTeardowns(ctx context.Context, c *runnerConn) {
	if c.personal || c.machine == "" || h.cfg.Teardowns == nil {
		return
	}
	kept, err := h.cfg.Teardowns.List(ctx, c.machine)
	if err != nil {
		h.log.Error("kept stack teardowns could not be read", "machine", c.machine, "error", err)
		return
	}
	for _, f := range kept {
		if err := h.sendFrame(ctx, c, f); err != nil {
			h.log.Warn("kept stack teardown send failed", "machine", c.machine, "slug", f.StackSlug, "error", err)
			return
		}
		if err := h.cfg.Teardowns.Delete(ctx, c.machine, f.StackSlug); err != nil {
			h.log.Error("sent stack teardown could not be dropped", "machine", c.machine, "slug", f.StackSlug, "error", err)
			return
		}
	}
}

// Teardown implements Executor.Teardown: a compose stack's project comes down with its orphans, a run stack's container
// is force-removed. Volumes, networks and the checkout stay.
func (e *ShellExecutor) Teardown(ctx context.Context, slug, strategy string, send func(Frame)) {
	result := Frame{Type: FrameStackTeardownResult, StackSlug: slug, Status: BuildStatusSuccess}
	if err := e.teardownStack(ctx, slug, strategy); err != nil {
		e.log.Warn("stack_teardown failed", "slug", slug, "error", err)
		result.Status = BuildStatusFailed
		result.Error = err.Error()
	}
	send(result)
}

func (e *ShellExecutor) teardownStack(ctx context.Context, slug, strategy string) error {
	if strategy == "compose" {
		return e.cmd(ctx, "", discardLog, "docker", "compose", "-p", slug, "down", "--remove-orphans")
	}
	err := e.cmd(ctx, "", discardLog, "docker", "rm", "-f", slug)
	if err != nil && strings.Contains(err.Error(), "No such container") {
		return nil
	}
	return err
}
