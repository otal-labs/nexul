package deploy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeReaders answers ManagersOn from a fixed list, whatever the machine.
type fakeReaders struct{ managers []string }

func (f fakeReaders) ManagersOn(context.Context, string) ([]string, error) { return f.managers, nil }

const refusal = "no runner on 10.0.0.1:22 can read 1Password references"

// newReferenceService is a service whose stack svc-1 holds an op:// reference (when withRef) and whose machine's
// runners can read the managers given.
func newReferenceService(t *testing.T, withRef bool, readable ...string) (*Service, *fakeRepo) {
	t.Helper()
	stack := validStack()
	if withRef {
		stack.Env = map[string]string{"DB_PASSWORD": "op://vault/item/password", "PORT": "8080"}
	}
	stacks := newFakeStackRepo()
	requireStack(t, stacks, stack)
	repo := newFakeRepo()
	s := newTestServiceWith(repo, stacks, newFakeContainerRepo(), newFakeProjects(), newFakeBus())
	s.SetSecretReaders(fakeReaders{managers: readable})
	seedDeploy(t, repo, &Deploy{ID: "failed", StackID: "svc-1", Status: StatusFailed, Image: "img:1", CreatedAt: at(1)})
	seedDeploy(t, repo, &Deploy{ID: "good", StackID: "svc-1", Status: StatusHealthy, Image: "img:0", CreatedAt: at(0)})
	return s, repo
}

// A deploy of a stack with references is refused at once, over HTTP and MCP and for a retry and a rollback, when no
// runner on its machine can read the manager; a stack without references, or a machine that can read, goes through.
func TestDeploy_ReferencesWithNoReadingRunner_AreRefusedEverywhere(t *testing.T) {
	t.Run("HTTP answers 409 with the machine and the manager", func(t *testing.T) {
		s, repo := newReferenceService(t, true)
		rec := httptest.NewRecorder()
		NewHandler(s).Routes().ServeHTTP(rec, withActor(httptest.NewRequest(http.MethodPost, "/api/deploys",
			strings.NewReader(`{"stack_id":"svc-1","image":"img:2"}`))))
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), refusal)
		assert.Empty(t, repo.of(TopicDeployRequested).Topic, "a refused deploy leaves no deploy.requested behind")
	})

	t.Run("MCP stack_deploy, its retry and its rollback are refused", func(t *testing.T) {
		s, _ := newReferenceService(t, true)
		for _, args := range []string{
			`{"id":"svc-1","image":"img:2"}`,
			`{"id":"svc-1","retry":"failed"}`,
			`{"id":"svc-1","rollback":true}`,
		} {
			_, err := call(t, s, "stack_deploy", args)
			require.ErrorIs(t, err, apperrs.ErrConflict, args)
			assert.Contains(t, err.Error(), refusal, args)
		}
	})

	t.Run("a stack without references is not refused", func(t *testing.T) {
		s, _ := newReferenceService(t, false)
		_, err := s.Deploy(t.Context(), validRequest())
		require.NoError(t, err)
	})

	t.Run("a machine with a runner that reads the manager is not refused, and the event names the manager", func(t *testing.T) {
		s, repo := newReferenceService(t, true, "op")
		_, err := s.Deploy(t.Context(), validRequest())
		require.NoError(t, err)
		ev, ok := repo.of(TopicDeployRequested).Payload.(DeployRequestedEvent)
		require.True(t, ok)
		assert.Equal(t, []string{"op"}, ev.SecretManagers)
		assert.Equal(t, "", ev.Env["DB_PASSWORD"], "the event still carries no value")
	})
}
