package storage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil"
)

// Each background runner picks its work through one repo read, so that read is where an archive must hide it.
func TestArchivedWorkspace_BackgroundReadsFindNothingUntilRestored(t *testing.T) {
	const workspaceID = "workspace-default"
	ctx := actorCtx()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	svc := docs.NewService(s.Docs, allowAll{}, nil)
	_, _, err := s.Users.UpsertUser(ctx, &auth.Identity{UserID: "u-alice", Provider: auth.ProviderGitHub, ProviderUserID: "u-alice", Login: "alice"})
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO conversations (id, workspace_id, kind, name, created_by, created_at, updated_at, agent_thread_id)
		VALUES ('conv-1', ?, 'channel', 'general', 'u-alice', 1, 1, 'thread-1')`, workspaceID)
	require.NoError(t, err)
	stack := &deploy.Stack{
		ID: "stack-api", ProjectID: testutil.GeneralProjectID, Name: "api", Slug: "api", Machine: "m1", Strategy: deploy.StrategyRun,
		BuildSource: &deploy.BuildSource{RepoOwner: "acme", RepoName: "api", Dockerfile: "Dockerfile"},
		CreatedAt:   time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.Stacks.Create(ctx, stack))

	runners := []struct {
		name string
		// found runs the runner's read once and counts what it would act on.
		found func(t *testing.T) int
	}{
		{"branch deploys on a push to the repo", func(t *testing.T) int {
			got, err := s.Stacks.ListByBuildRepo(ctx, "acme", "api")
			require.NoError(t, err)
			return len(got)
		}},
		{"following a harness thread", func(t *testing.T) int {
			_, err := s.Chat.GetConversationByAgentThread(ctx, "thread-1")
			if err != nil {
				return 0
			}
			return 1
		}},
		{"a doc edit settling into doc.settled", func(t *testing.T) int {
			d, err := svc.Create(ctx, testutil.GeneralProjectID, "Spec", "draft")
			require.NoError(t, err)
			var settled []docs.SettledEvent
			require.NoError(t, s.Docs.SettleDue(ctx, time.Now().Add(docs.SettleWindow+time.Minute), func(e []docs.SettledEvent) []eventbus.OutboxEvent {
				settled = e
				out := make([]eventbus.OutboxEvent, 0, len(e))
				for _, ev := range e {
					out = append(out, eventbus.OutboxEvent{ID: ids.New(), Topic: docs.TopicSettled, Payload: ev})
				}
				return out
			}))
			for _, e := range settled {
				assert.Equal(t, d.ID, e.Doc.ID)
			}
			return len(settled)
		}},
	}
	setArchived := func(archived bool) {
		var at *time.Time
		if archived {
			now := time.Now()
			at = &now
		}
		require.NoError(t, s.Workspaces.SetArchived(ctx, workspaceID, at, time.Now()))
	}
	for _, r := range runners {
		t.Run(r.name, func(t *testing.T) {
			require.Equal(t, 1, r.found(t), "a live workspace is read")
			setArchived(true)
			assert.Zero(t, r.found(t), "an archived workspace is not")
			setArchived(false)
			assert.Equal(t, 1, r.found(t), "a restored one is read again")
		})
	}
}
