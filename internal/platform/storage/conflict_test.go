package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/otal-labs/nexul/internal/chat"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/workspace"
)

// twice returns what the second of two writes answered, once the first has landed.
func twice(first, second error) error {
	if first != nil {
		return fmt.Errorf("the first write: %w", first)
	}
	return second
}

// TestRepos_ADuplicateOrADanglingReferenceIsAConflict holds every create that repeats a stored id or name, or points
// at a row nobody stored, to ErrConflict, so the layers above tell a clash from an outage.
func TestRepos_ADuplicateOrADanglingReferenceIsAConflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	now := time.Now().UTC()
	channel := func(id string) *chat.Conversation {
		return newTestConversation(id, chat.KindChannel, "general", "", "u-1")
	}
	tests := []struct {
		name string
		call func(ctx context.Context) error
	}{
		{"an automation run with a stored id", func(ctx context.Context) error {
			return twice(s.AutomationRuns.Create(ctx, newTestRun("r1", "a1", now)), s.AutomationRuns.Create(ctx, newTestRun("r1", "a1", now)))
		}},
		{"a channel with a stored name", func(ctx context.Context) error {
			return twice(s.Chat.CreateConversation(ctx, channel("conv-1"), []string{"u-1"}), s.Chat.CreateConversation(ctx, channel("conv-2"), []string{"u-1"}))
		}},
		{"a message in a conversation nobody stored", func(ctx context.Context) error {
			return s.Chat.CreateMessage(ctx, &chat.Message{ID: "msg-1", ConversationID: "does-not-exist", AuthorID: "u-1", Body: "hi", CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow})
		}},
		{"a dead letter with a stored id", func(ctx context.Context) error {
			return twice(s.DeadLetters.Put(ctx, newTestDeadLetter("dl-1")), s.DeadLetters.Put(ctx, newTestDeadLetter("dl-1")))
		}},
		{"a deploy with a stored id", func(ctx context.Context) error {
			return twice(s.Deploys.Create(ctx, newTestDeploy("dep-1")), s.Deploys.Create(ctx, newTestDeploy("dep-1")))
		}},
		{"a doc with a stored id", func(ctx context.Context) error {
			return twice(s.Docs.Create(ctx, newTestDoc("doc-1")), s.Docs.Create(ctx, newTestDoc("doc-1")))
		}},
		{"a doc in a project nobody stored", func(ctx context.Context) error {
			d := newTestDoc("doc-2")
			d.ProjectID = "does-not-exist"
			return s.Docs.Create(ctx, d)
		}},
		{"an interview answer in a project nobody stored", func(ctx context.Context) error {
			a := newTestAnswer("a-1", 0, "Stack")
			a.ProjectID = "project-missing"
			return ign(s.Memories.UpsertAnswer(ctx, a))
		}},
		{"a machine with a stored name", func(ctx context.Context) error {
			return twice(s.Machines.Create(ctx, newTestMachine("m-1", "prod")), s.Machines.Create(ctx, newTestMachine("m-2", "prod")))
		}},
		{"a memory with a stored id", func(ctx context.Context) error {
			return twice(s.Memories.Create(ctx, newTestMemory("mem-1", "project-general"), ""), s.Memories.Create(ctx, newTestMemory("mem-1", "project-general"), ""))
		}},
		{"a memory in a project nobody stored", func(ctx context.Context) error {
			return s.Memories.Create(ctx, newTestMemory("mem-2", "does-not-exist"), "")
		}},
		{"a play in a workspace nobody stored", func(ctx context.Context) error {
			return s.Plays.Create(ctx, newTestPlay("play-1", "missing-workspace"))
		}},
		{"a role in a workspace nobody stored", func(ctx context.Context) error {
			return s.Roles.Create(ctx, newTestRole("role-1", "missing-workspace"))
		}},
		{"a stack in a project nobody stored", func(ctx context.Context) error {
			stack := newTestStack("svc-1")
			stack.ProjectID = "no-such-project"
			return s.Stacks.Create(ctx, stack)
		}},
		{"a workspace with a stored id", func(ctx context.Context) error {
			return twice(s.Workspaces.Create(ctx, newTestWorkspace("ws-1", "A")), s.Workspaces.Create(ctx, newTestWorkspace("ws-1", "B")))
		}},
		{"a ticket on a doc nobody stored", func(ctx context.Context) error {
			return s.Tickets.Create(ctx, newTestTicket("t-1", "does-not-exist"))
		}},
		{"an instance upgrade with a stored id", func(ctx context.Context) error {
			return twice(s.InstanceUpgrades.Create(ctx, newTestUpgrade("u-1", "pending")), s.InstanceUpgrades.Create(ctx, newTestUpgrade("u-1", "pending")))
		}},
		{"a project with a stored id", func(ctx context.Context) error {
			return twice(s.Projects.Create(ctx, newTestProject("p-1", "A", 0)), s.Projects.Create(ctx, newTestProject("p-1", "B", 1)))
		}},
		{"a project in a workspace nobody stored", func(ctx context.Context) error {
			p := newTestProject("p-2", "A", 0)
			p.WorkspaceID = "missing"
			return s.Projects.Create(ctx, p)
		}},
		{"a repository on a project nobody stored", func(ctx context.Context) error {
			return s.Projects.AddRepo(ctx, "missing", workspace.RepoRef{Owner: "acme", Name: "app"})
		}},
		{"a ticket moved to a project nobody stored", func(ctx context.Context) error {
			return twice(s.Tickets.Create(ctx, newTestTicket("t-2", "")), s.Projects.MoveTicket(ctx, "t-2", "missing"))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, tt.call(t.Context()), apperrs.ErrConflict)
		})
	}
}
