package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

func newTestDeploy(id string) *deploy.Deploy {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	return &deploy.Deploy{
		ID: id, Service: "api", Target: "10.0.0.1:22", Image: "ghcr.io/onik/api:v1",
		Status: deploy.StatusPending, Strategy: deploy.StrategyCompose,
		CreatedAt: now, UpdatedAt: now,
	}
}

func TestDeploysRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Deploys.GetByID(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDeploysRepo_Create_DuplicateID_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	err := s.Deploys.Create(context.Background(), newTestDeploy("dep-1"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestDeploysRepo_Create_GetByID_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestDeploy("dep-1")
	require.NoError(t, s.Deploys.Create(context.Background(), want))

	got, err := s.Deploys.GetByID(context.Background(), "dep-1")
	require.NoError(t, err)
	assert.Equal(t, "api", got.Service)
	assert.Equal(t, "10.0.0.1:22", got.Target)
	assert.Equal(t, deploy.StatusPending, got.Status)
	assert.Equal(t, deploy.StrategyCompose, got.Strategy)
}

func TestDeploysRepo_BuildKind_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	build := newTestDeploy("dep-b")
	build.Kind = deploy.KindBuild
	build.Image = ""
	require.NoError(t, s.Deploys.Create(context.Background(), build))

	got, err := s.Deploys.GetByID(context.Background(), "dep-b")
	require.NoError(t, err)
	assert.Equal(t, deploy.KindBuild, got.Kind, "a repo-driven build round-trips as kind build")
	assert.Equal(t, "", got.Image)

	// Legacy rows without a kind (migration default) read back as deploys.
	legacy := newTestDeploy("dep-legacy")
	require.NoError(t, s.Deploys.Create(context.Background(), legacy))
	gotLegacy, err := s.Deploys.GetByID(context.Background(), "dep-legacy")
	require.NoError(t, err)
	assert.Equal(t, deploy.KindDeploy, gotLegacy.Kind)
}

func TestDeploysRepo_List_ReturnsAll(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-2")))

	got, err := s.Deploys.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestDeploysRepo_ListByService_ScopesToService(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-2")))
	other := newTestDeploy("dep-3")
	other.Service = "worker"
	require.NoError(t, s.Deploys.Create(context.Background(), other))

	got, err := s.Deploys.ListByService(context.Background(), "api")
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestDeploysRepo_ListByStatus_ScopesToStatus(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	running := newTestDeploy("dep-2")
	running.Status = deploy.StatusRunning
	require.NoError(t, s.Deploys.Create(context.Background(), running))

	got, err := s.Deploys.ListByStatus(context.Background(), deploy.StatusPending)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "dep-1", got[0].ID)
}

func TestDeploysRepo_UpdateStatus_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Deploys.UpdateStatus(context.Background(), "missing", deploy.StatusHealthy)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDeploysRepo_UpdateStatus_Persists(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	require.NoError(t, s.Deploys.UpdateStatus(context.Background(), "dep-1", deploy.StatusHealthy))

	got, err := s.Deploys.GetByID(context.Background(), "dep-1")
	require.NoError(t, err)
	assert.Equal(t, deploy.StatusHealthy, got.Status)
}

func TestDeploysRepo_UpdateStatus_WritesOutboxInSameTx(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	evt := eventbus.OutboxEvent{ID: "evt-status", Topic: deploy.TopicDeployUpdated, Payload: deploy.DeployUpdatedEvent{ID: "dep-1", Status: "healthy"}}
	require.NoError(t, s.Deploys.UpdateStatus(context.Background(), "dep-1", deploy.StatusHealthy, evt))

	var raw []byte
	require.NoError(t, s.db.QueryRowContext(context.Background(), `SELECT payload FROM outbox WHERE id = 'evt-status'`).Scan(&raw))
	var e deploy.DeployUpdatedEvent
	require.NoError(t, json.Unmarshal(raw, &e))
	assert.Equal(t, deploy.DeployUpdatedEvent{ID: "dep-1", Status: "healthy"}, e)
}

func TestDeploysRepo_UpdateStatus_NotFound_WritesNoOutbox(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	evt := eventbus.OutboxEvent{ID: "evt-missing", Topic: deploy.TopicDeployUpdated, Payload: deploy.DeployUpdatedEvent{ID: "missing"}}
	require.ErrorIs(t, s.Deploys.UpdateStatus(context.Background(), "missing", deploy.StatusHealthy, evt), apperrs.ErrNotFound)

	var n int
	require.NoError(t, s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM outbox WHERE id = 'evt-missing'`).Scan(&n))
	assert.Equal(t, 0, n)
}

func TestDeploysRepo_SetAddress_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Deploys.SetAddress(context.Background(), "missing", "172.18.0.4")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDeploysRepo_SetAddress_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	require.NoError(t, s.Deploys.SetAddress(context.Background(), "dep-1", "172.18.0.4"))

	got, err := s.Deploys.GetByID(context.Background(), "dep-1")
	require.NoError(t, err)
	assert.Equal(t, "172.18.0.4", got.Address)
}

func TestDeploysRepo_AppendLogLines_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Deploys.AppendLogLines(context.Background(), "missing", []deploy.LogLine{{TS: 1, Text: "line"}})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDeploysRepo_AppendLogLines_WritesOutboxInSameTx(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	evt := eventbus.OutboxEvent{ID: "evt-log", Topic: deploy.TopicDeployUpdated, Payload: deploy.DeployUpdatedEvent{ID: "dep-1", Status: "pending"}}
	require.NoError(t, s.Deploys.AppendLogLines(context.Background(), "dep-1", []deploy.LogLine{{TS: 1, Phase: "build", Text: "Step 1/3"}}, evt))

	var topic string
	require.NoError(t, s.db.QueryRowContext(context.Background(), `SELECT topic FROM outbox WHERE id = 'evt-log'`).Scan(&topic))
	assert.Equal(t, deploy.TopicDeployUpdated, topic)
}

func TestDeploysRepo_AppendLogLines_NotFound_WritesNoOutbox(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	evt := eventbus.OutboxEvent{ID: "evt-orphan", Topic: deploy.TopicDeployUpdated, Payload: deploy.DeployUpdatedEvent{ID: "missing"}}
	require.ErrorIs(t, s.Deploys.AppendLogLines(context.Background(), "missing", []deploy.LogLine{{TS: 1, Text: "line"}}, evt), apperrs.ErrNotFound)

	var n int
	require.NoError(t, s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM outbox WHERE id = 'evt-orphan'`).Scan(&n))
	assert.Equal(t, 0, n)
}

func TestDeploysRepo_AppendLogLines_NoLines_IsNoop(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.AppendLogLines(context.Background(), "missing", nil))
}

func TestDeploysRepo_ListLogLines_OrdersByTimeThenSeq(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-2")))
	require.NoError(t, s.Deploys.AppendLogLines(context.Background(), "dep-1", []deploy.LogLine{
		{TS: 20, Phase: "build", Text: "Step 1/3"},
		{TS: 20, Phase: "build", Text: "Step 2/3"},
	}))
	require.NoError(t, s.Deploys.AppendLogLines(context.Background(), "dep-1", []deploy.LogLine{{TS: 10, Phase: "checkout", Text: "late arrival"}}))
	require.NoError(t, s.Deploys.AppendLogLines(context.Background(), "dep-2", []deploy.LogLine{{TS: 1, Text: "other deploy"}}))

	got, err := s.Deploys.ListLogLines(context.Background(), "dep-1")
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, deploy.LogLine{Seq: 3, TS: 10, Phase: "checkout", Text: "late arrival"}, got[0])
	assert.Equal(t, deploy.LogLine{Seq: 1, TS: 20, Phase: "build", Text: "Step 1/3"}, got[1])
	assert.Equal(t, deploy.LogLine{Seq: 2, TS: 20, Phase: "build", Text: "Step 2/3"}, got[2])
}

func TestDeploysRepo_ListLogLines_Empty(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	got, err := s.Deploys.ListLogLines(context.Background(), "dep-1")
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

// TestDeploysRepo_Page_FiltersAndScopesInSQL: each filter and scope keeps exactly the deploys the per-stack rule keeps,
// newest first, paged once each, with deploys created in the same second.
func TestDeploysRepo_Page_FiltersAndScopesInSQL(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	s := newTestStore(t)
	seedProject(t, s, "p-hidden", "workspace-default", "HID")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	projectOf := map[string]string{"s-gen": "project-general", "s-hid": "p-hidden", "s-infra": "", "s-gone": ""}
	for _, id := range []string{"s-gen", "s-hid", "s-infra", "s-gone"} {
		require.NoError(t, s.Stacks.Create(ctx, &deploy.Stack{ID: id, ProjectID: projectOf[id], Name: id, Slug: id, Machine: "m1", Strategy: deploy.StrategyRun, CreatedAt: now, UpdatedAt: now}))
	}
	stacks := []string{"s-gen", "s-hid", "s-infra", "s-gone"}
	var all []*deploy.Deploy
	for i := range 60 {
		d := newTestDeploy(fmt.Sprintf("dep-%02d", i))
		d.StackID = stacks[i%4]
		d.Status = []deploy.Status{deploy.StatusHealthy, deploy.StatusFailed}[i%2]
		d.CreatedAt = now.Add(time.Duration(i/4) * time.Second)
		require.NoError(t, s.Deploys.Create(ctx, d))
		all = append(all, d)
	}
	require.NoError(t, s.Stacks.Delete(ctx, "s-gone"))

	scopes := map[string]deploy.DeployScope{
		"everything":                   {All: true},
		"a project and the instance's": {ProjectIDs: []string{"project-general"}, Anywhere: true},
		"a project only":               {ProjectIDs: []string{"project-general"}},
		"nothing":                      {ProjectIDs: []string{}},
	}
	filters := []deploy.DeployFilter{{}, {Status: deploy.StatusFailed}, {StackID: "s-gen"}, {StackID: "s-gen", Status: deploy.StatusFailed}}
	for name, scope := range scopes {
		for _, f := range filters {
			var want []string
			for i := len(all) - 1; i >= 0; i-- {
				d := all[i]
				project := projectOf[d.StackID]
				shown := scope.All || slices.Contains(scope.ProjectIDs, project) || (project == "" && scope.Anywhere)
				if shown && (f.StackID == "" || d.StackID == f.StackID) && (f.Status == "" || d.Status == f.Status) {
					want = append(want, d.ID)
				}
			}

			got := pageAll(t, 7, func(offset, limit int) ([]string, int) {
				ds, total, err := s.Deploys.Page(ctx, f, scope, paging.Window{Offset: offset, Limit: limit})
				require.NoError(t, err)
				ids := make([]string, len(ds))
				for i, d := range ds {
					ids[i] = d.ID
				}
				return ids, total
			})

			assert.Equal(t, want, got, "%s %+v", name, f)
		}
	}
}
