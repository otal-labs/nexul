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

// teardown sends f to a connected runner on machine. With none connected, or a failed send, it is kept and sent when
// one connects.
// ponytail: the wait is in memory like the deploy queue, so a server restart forgets it; persist it if that bites.
func (h *Handler) teardown(ctx context.Context, machine string, f Frame) {
	h.mu.Lock()
	c := h.machineConnLocked(machine)
	if c == nil {
		h.teardowns[machine] = append(h.teardowns[machine], f)
		h.mu.Unlock()
		h.log.Info("stack teardown waits for a runner", "machine", machine, "slug", f.StackSlug)
		return
	}
	h.mu.Unlock()
	if err := h.sendFrame(ctx, c, f); err != nil {
		h.log.Warn("stack teardown send failed, kept for the next connect", "machine", machine, "slug", f.StackSlug, "error", err)
		h.mu.Lock()
		h.teardowns[machine] = append(h.teardowns[machine], f)
		h.mu.Unlock()
	}
}

// flushTeardowns sends the teardowns kept for c's machine; it runs before queued deploys so a slug reused since is
// cleared before it is deployed again.
func (h *Handler) flushTeardowns(ctx context.Context, c *runnerConn) {
	if c.personal || c.machine == "" {
		return
	}
	h.mu.Lock()
	kept := h.teardowns[c.machine]
	delete(h.teardowns, c.machine)
	h.mu.Unlock()
	for i, f := range kept {
		if err := h.sendFrame(ctx, c, f); err != nil {
			h.log.Warn("kept stack teardown send failed", "machine", c.machine, "slug", f.StackSlug, "error", err)
			h.mu.Lock()
			h.teardowns[c.machine] = append(kept[i:], h.teardowns[c.machine]...)
			h.mu.Unlock()
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
