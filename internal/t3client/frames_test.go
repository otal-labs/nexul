package t3client

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/t3rpc"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

func connectFake(t *testing.T, ctx context.Context, f *t3rpctest.Server) *conn {
	c, err := t3rpc.Connect(ctx, f.Session(), Options{HTTPClient: f.Client(), RPCTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return &conn{Conn: c, log: slog.Default()}
}

// Protocol-1 stream items.

func eventItem(eventType string, payload any) map[string]any {
	return map[string]any{"kind": "event", "event": map[string]any{"type": eventType, "payload": payload}}
}

func messageSent(threadID, messageID, role, text string, streaming bool) map[string]any {
	return eventItem("thread.message-sent", map[string]any{
		"threadId": threadID, "messageId": messageID, "role": role,
		"text": text, "streaming": streaming, "turnId": "turn-1",
	})
}

func toolActivity(threadID, kind, summary string) map[string]any {
	return eventItem("thread.activity-appended", map[string]any{
		"threadId": threadID,
		"activity": map[string]any{"id": "act-1", "tone": "tool", "kind": kind, "summary": summary, "payload": map[string]any{}},
	})
}

func sessionSet(threadID, status string, activeTurnID, lastError any) map[string]any {
	return eventItem("thread.session-set", map[string]any{
		"threadId": threadID,
		"session": map[string]any{
			"threadId": threadID, "status": status, "providerName": "claude",
			"activeTurnId": activeTurnID, "lastError": lastError,
		},
	})
}

// collect drains a subscription to completion, splitting updates by kind.
func collect(t *testing.T, sub *Subscription) ([]MessageSnapshot, []harness.Activity, []ApprovalRequest, *TurnResult) {
	t.Helper()
	var snaps []MessageSnapshot
	var activities []harness.Activity
	var approvals []ApprovalRequest
	var terminal *TurnResult
	for u := range sub.Updates() {
		if u.Snapshot != nil {
			snaps = append(snaps, *u.Snapshot)
		}
		if u.Activity != nil {
			activities = append(activities, *u.Activity)
		}
		if u.Approval != nil {
			approvals = append(approvals, *u.Approval)
		}
		if u.Terminal != nil {
			terminal = u.Terminal
		}
	}
	return snaps, activities, approvals, terminal
}
