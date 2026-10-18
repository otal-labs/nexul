package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/tickets"
)

// ign drops the value of a read, which is not what these tests look at.
func ign[T any](_ T, err error) error { return err }

// TestRepos_AnIdNobodyStoredIsNotFound holds every read, update and delete of a missing row to ErrNotFound, never an
// empty value or a nil error.
func TestRepos_AnIdNobodyStoredIsNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	tests := []struct {
		name string
		call func(ctx context.Context) error
	}{
		{"AttachmentsRepo.GetByID", func(ctx context.Context) error { return ign(s.Attachments.GetByID(ctx, "missing")) }},
		{"AttachmentsRepo.Delete", func(ctx context.Context) error { return s.Attachments.Delete(ctx, "missing") }},
		{"UsersRepo.GetUserByID", func(ctx context.Context) error { return ign(s.Users.GetUserByID(ctx, "missing")) }},
		{"UsersRepo.MarkFirstLoginDone", func(ctx context.Context) error { return s.Users.MarkFirstLoginDone(ctx, "ghost") }},
		{"AutomationRunsRepo.Get", func(ctx context.Context) error { return ign(s.AutomationRuns.Get(ctx, "missing")) }},
		{"AutomationsRepo.Get", func(ctx context.Context) error { return ign(s.Automations.Get(ctx, "nope")) }},
		{"AutomationsRepo.GetByTokenHash", func(ctx context.Context) error { return ign(s.Automations.GetByTokenHash(ctx, "no-such-hash")) }},
		{"ChatRepo.GetConversation", func(ctx context.Context) error { return ign(s.Chat.GetConversation(ctx, "missing")) }},
		{"ChatRepo.GetTicketThread", func(ctx context.Context) error { return ign(s.Chat.GetTicketThread(ctx, "t-1")) }},
		{"ChatRepo.GetDocThread", func(ctx context.Context) error { return ign(s.Chat.GetDocThread(ctx, "doc-1")) }},
		{"ChatRepo.GetMessage", func(ctx context.Context) error { return ign(s.Chat.GetMessage(ctx, "missing")) }},
		{"ChatRepo.UpdateMessage", func(ctx context.Context) error { return s.Chat.UpdateMessage(ctx, "missing", "hi", nil, chatFixedNow) }},
		{"ChatRepo.DeleteMessage", func(ctx context.Context) error { return s.Chat.DeleteMessage(ctx, "missing", chatFixedNow) }},
		{"CodeReviewsRepo.Get", func(ctx context.Context) error { return ign(s.CodeReviews.Get(ctx, "nope")) }},
		{"ConnectorsRepo.GetCredentials", func(ctx context.Context) error { return ign(s.Connectors.GetCredentials(ctx, "github")) }},
		{"DeadLettersRepo.Get", func(ctx context.Context) error { return ign(s.DeadLetters.Get(ctx, "missing")) }},
		{"DeadLettersRepo.Delete", func(ctx context.Context) error { return s.DeadLetters.Delete(ctx, "missing") }},
		{"DeploysRepo.GetByID", func(ctx context.Context) error { return ign(s.Deploys.GetByID(ctx, "missing")) }},
		{"DeploysRepo.SetAddress", func(ctx context.Context) error { return s.Deploys.SetAddress(ctx, "missing", "172.18.0.4") }},
		{"DocsRepo.GetByID", func(ctx context.Context) error { return ign(s.Docs.GetByID(ctx, "missing")) }},
		{"DocsRepo.Update", func(ctx context.Context) error { return s.Docs.Update(ctx, newTestDoc("missing"), "") }},
		{"DocsRepo.Delete", func(ctx context.Context) error { return s.Docs.Delete(ctx, "missing") }},
		{"DocsRepo.DefaultFolder", func(ctx context.Context) error { return ign(s.Docs.DefaultFolder(ctx, "no-such-project")) }},
		{"DocsRepo.CommitBody", func(ctx context.Context) error { return s.Docs.CommitBody(ctx, newTestDoc("nope"), "") }},
		{"ErrorRateRepo.GetOpen", func(ctx context.Context) error { return ign(s.ErrorRate.GetOpen(ctx, "stack-1")) }},
		{"MachinesRepo.Get", func(ctx context.Context) error { return ign(s.Machines.Get(ctx, "missing")) }},
		{"MachinesRepo.Rename", func(ctx context.Context) error { return s.Machines.Rename(ctx, "missing", "x") }},
		{"MachinesRepo.Touch", func(ctx context.Context) error { return s.Machines.Touch(ctx, "missing", "x", time.Now()) }},
		{"MemoriesRepo.GetByID", func(ctx context.Context) error { return ign(s.Memories.GetByID(ctx, "missing")) }},
		{"MemoriesRepo.Update", func(ctx context.Context) error {
			return s.Memories.Update(ctx, newTestMemory("missing", "project-general"), "")
		}},
		{"MemoriesRepo.Delete", func(ctx context.Context) error { return s.Memories.Delete(ctx, "missing") }},
		{"OutagesRepo.GetOpen", func(ctx context.Context) error { return ign(s.Outages.GetOpen(ctx, "stack-1", "web")) }},
		{"OutboxRepo.MarkPublished", func(ctx context.Context) error { return s.Outbox.MarkPublished(ctx, "missing") }},
		{"PATsRepo.GetByHash", func(ctx context.Context) error { return ign(s.PATs.GetByHash(ctx, "missing")) }},
		{"PlaysRepo.Get", func(ctx context.Context) error { return ign(s.Plays.Get(ctx, "missing")) }},
		{"PlaysRepo.Update", func(ctx context.Context) error { return s.Plays.Update(ctx, newTestPlay("missing", "ws-1")) }},
		{"PlaysRepo.Delete", func(ctx context.Context) error { return s.Plays.Delete(ctx, "missing") }},
		{"RolesRepo.Get", func(ctx context.Context) error { return ign(s.Roles.Get(ctx, "missing")) }},
		{"RolesRepo.Update", func(ctx context.Context) error { return s.Roles.Update(ctx, newTestRole("missing", "ws-1")) }},
		{"RolesRepo.Delete", func(ctx context.Context) error { return s.Roles.Delete(ctx, "missing") }},
		{"RunnersRepo.GetByID", func(ctx context.Context) error { return ign(s.Runners.GetByID(ctx, "missing")) }},
		{"RunnersRepo.SetVersion", func(ctx context.Context) error { return s.Runners.SetVersion(ctx, "missing", "v0.1.6") }},
		{"RunnersRepo.Heartbeat", func(ctx context.Context) error { return s.Runners.Heartbeat(ctx, "missing", time.Now()) }},
		{"RunnersRepo.GetCredential", func(ctx context.Context) error { return ign(s.Runners.GetCredential(ctx, "nope")) }},
		{"ServicesRepo.Get", func(ctx context.Context) error { return ign(s.Services.Get(ctx, "missing")) }},
		{"StacksRepo.GetByID", func(ctx context.Context) error { return ign(s.Stacks.GetByID(ctx, "missing")) }},
		{"StacksRepo.Update", func(ctx context.Context) error { return s.Stacks.Update(ctx, newTestStack("svc-1")) }},
		{"WorkspacesRepo.Get", func(ctx context.Context) error { return ign(s.Workspaces.Get(ctx, "missing")) }},
		{"TicketsRepo.GetByID", func(ctx context.Context) error { return ign(s.Tickets.GetByID(ctx, "missing")) }},
		{"TicketsRepo.UpdateTicket", func(ctx context.Context) error {
			return s.Tickets.UpdateTicket(ctx, "missing", "New title", "New body")
		}},
		{"TicketsRepo.UpdateStatus", func(ctx context.Context) error {
			return s.Tickets.UpdateStatus(ctx, "missing", tickets.StatusDone, nil)
		}},
		{"TicketsRepo.Delete", func(ctx context.Context) error { return s.Tickets.Delete(ctx, "missing", nil) }},
		{"TopologyRepo.Get", func(ctx context.Context) error { return ign(s.Topology.Get(ctx, "prod")) }},
		{"InstanceUpgradesRepo.GetByID", func(ctx context.Context) error { return ign(s.InstanceUpgrades.GetByID(ctx, "missing")) }},
		{"InstanceUpgradesRepo.Latest", func(ctx context.Context) error { return ign(s.InstanceUpgrades.Latest(ctx)) }},
		{"InstanceUpgradesRepo.SetStatus", func(ctx context.Context) error { return s.InstanceUpgrades.SetStatus(ctx, "missing", "failed", "boom") }},
		{"ProjectsRepo.Get", func(ctx context.Context) error { return ign(s.Projects.Get(ctx, "missing")) }},
		{"ProjectsRepo.Update", func(ctx context.Context) error { return s.Projects.Update(ctx, newTestProject("missing", "X", 0)) }},
		{"ProjectsRepo.Delete", func(ctx context.Context) error { return s.Projects.Delete(ctx, "missing") }},
		{"ProjectsRepo.RemoveRepo", func(ctx context.Context) error { return s.Projects.RemoveRepo(ctx, "acme", "app") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, tt.call(t.Context()), apperrs.ErrNotFound)
		})
	}
}
