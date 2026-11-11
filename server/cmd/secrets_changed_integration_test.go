package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
	"github.com/otal-labs/nexul/internal/runner"
)

// TestSecretsChanged_ARunnersFindingReachesTheStackAndItsViewersUntilADeployClearsIt: a machine runner's secret check on
// the bus puts the variable names on the stack's read for whoever can read the stack, the event reaches the same people
// and nobody else, and the stack's next healthy deploy takes the notice away.
func TestSecretsChanged_ARunnersFindingReachesTheStackAndItsViewersUntilADeployClearsIt(t *testing.T) {
	t.Parallel()
	db := migrated.DB(t)
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: store.ProcessedEvents, DeadLetterStore: store.DeadLetters})
	t.Cleanup(func() { _ = bus.Close() })
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "nexul.db"), HTTPAddr: "127.0.0.1:0", AuthSecret: "0123456789abcdef0123456789abcdef"}
	svc := wireCoreServices(cfg, store, []byte("0123456789abcdef0123456789abcdef"), bus, logger)
	wireDomainEventSubscriptions(context.Background(), bus, svc, logger)
	f := seedPermFixture(t, db, svc, store)
	ctx := context.Background()
	h := deploy.NewHandler(svc.deploySvc).Routes()
	read := func(user string) (int, deploy.Stack) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks/"+f.stack, nil).WithContext(as(user)))
		var stack deploy.Stack
		_ = json.Unmarshal(rec.Body.Bytes(), &stack)
		return rec.Code, stack
	}

	require.NoError(t, bus.Publish(ctx, runner.TopicSecretsChanged, runner.SecretsChangedReport{Machine: "m1", Changes: []runner.SecretChange{
		{Slug: "web", Variables: []string{"DB_PASSWORD"}},
	}}))
	require.Eventually(t, func() bool {
		_, stack := read(uReader)
		return len(stack.SecretsChanged) > 0
	}, 5*time.Second, 10*time.Millisecond)
	_, stack := read(uReader)
	assert.Equal(t, []string{"DB_PASSWORD"}, stack.SecretsChanged)
	assert.NotEqual(t, http.StatusOK, func() int { code, _ := read(uPlain); return code }(), "someone who cannot read the stack is told nothing")

	a := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	frame := deploy.SecretsChangedEvent{ProjectID: "project-general", StackID: f.stack, StackName: "web", Variables: []string{"DB_PASSWORD"}}
	assert.True(t, a.allows(as(uReader), deploy.TopicSecretsChanged, frame), "whoever reads the stack hears it")
	assert.False(t, a.allows(as(uPlain), deploy.TopicSecretsChanged, frame), "and nobody else does")

	now := time.Now().UTC()
	require.NoError(t, store.Deploys.Create(ctx, &deploy.Deploy{ID: "deploy-2", StackID: f.stack, Service: "web", Status: deploy.StatusRunning, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, bus.Publish(ctx, runner.TopicDeployStatusChanged, runner.DeployStatusChangedEvent{ID: "deploy-2", Status: runner.DeployStatusHealthy}))
	require.Eventually(t, func() bool {
		_, stack := read(uReader)
		return len(stack.SecretsChanged) == 0
	}, 5*time.Second, 10*time.Millisecond, "a healthy deploy ends the notice")
}
