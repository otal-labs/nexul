package pairing

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

var testNow = time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

func newSetupService(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	repo.computers["c1"] = Computer{ID: "c1", UserID: "u1", Name: "home"}
	return newTestService(repo, &fakeExchanger{}), repo
}

func TestSetup_ErrorPaths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		call    func(context.Context, *Service) error
		wantErr error
	}{
		{"get without user", func(ctx context.Context, s *Service) error { _, err := s.GetSetup(ctx, "", "c1"); return err }, apperrs.ErrUnauthorized},
		{"get without computer", func(ctx context.Context, s *Service) error { _, err := s.GetSetup(ctx, "u1", " "); return err }, apperrs.ErrInvalid},
		{"get another user's computer", func(ctx context.Context, s *Service) error { _, err := s.GetSetup(ctx, "u2", "c1"); return err }, apperrs.ErrNotFound},
		{"confirm another user's computer", func(ctx context.Context, s *Service) error { _, err := s.ConfirmSetup(ctx, "u2", "c1"); return err }, apperrs.ErrNotFound},
		{"unconfirm another user's computer", func(ctx context.Context, s *Service) error { _, err := s.UnconfirmSetup(ctx, "u2", "c1"); return err }, apperrs.ErrNotFound},
		{"confirm provider on another user's computer", func(ctx context.Context, s *Service) error {
			_, err := s.ConfirmProviderSetup(ctx, "u2", "c1", "claude", []string{"tdd"})
			return err
		}, apperrs.ErrNotFound},
		{"unconfirm provider on another user's computer", func(ctx context.Context, s *Service) error {
			_, err := s.UnconfirmProviderSetup(ctx, "u2", "c1", "claude")
			return err
		}, apperrs.ErrNotFound},
		{"confirm provider without provider", func(ctx context.Context, s *Service) error {
			_, err := s.ConfirmProviderSetup(ctx, "u1", "c1", " ", []string{"tdd"})
			return err
		}, apperrs.ErrInvalid},
		{"confirm provider without skills", func(ctx context.Context, s *Service) error {
			_, err := s.ConfirmProviderSetup(ctx, "u1", "c1", "claude", []string{" ", ""})
			return err
		}, apperrs.ErrInvalid},
		{"unconfirm provider without provider", func(ctx context.Context, s *Service) error {
			_, err := s.UnconfirmProviderSetup(ctx, "u1", "c1", "")
			return err
		}, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := newSetupService(t)
			require.ErrorIs(t, tt.call(t.Context(), svc), tt.wantErr)
			assert.Empty(t, repo.outbox, "a rejected write publishes nothing")
		})
	}
}

func TestSetup_RepoFailures_AreWrapped(t *testing.T) {
	t.Parallel()
	t.Run("list provider setups", func(t *testing.T) {
		t.Parallel()
		svc, repo := newSetupService(t)
		repo.listSetupErr = errBoom
		_, err := svc.GetSetup(t.Context(), "u1", "c1")
		require.ErrorIs(t, err, errBoom)
	})
	t.Run("save overall", func(t *testing.T) {
		t.Parallel()
		svc, repo := newSetupService(t)
		repo.saveErr = errBoom
		_, err := svc.ConfirmSetup(t.Context(), "u1", "c1")
		require.ErrorIs(t, err, errBoom)
	})
	t.Run("save provider", func(t *testing.T) {
		t.Parallel()
		svc, repo := newSetupService(t)
		repo.saveErr = errBoom
		_, err := svc.ConfirmProviderSetup(t.Context(), "u1", "c1", "claude", []string{"tdd"})
		require.ErrorIs(t, err, errBoom)
	})
}

func TestSetup_NewComputer_StartsUnconfirmed(t *testing.T) {
	t.Parallel()
	svc, _ := newSetupService(t)
	setup, err := svc.GetSetup(t.Context(), "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, "c1", setup.ComputerID)
	assert.Nil(t, setup.ConfirmedAt)
	assert.Empty(t, setup.Providers)
}

func TestSetup_ConfirmAndUnconfirmOverall_PublishesEachChange(t *testing.T) {
	t.Parallel()
	svc, repo := newSetupService(t)

	setup, err := svc.ConfirmSetup(t.Context(), "u1", "c1")
	require.NoError(t, err)
	require.NotNil(t, setup.ConfirmedAt)
	assert.Equal(t, testNow, *setup.ConfirmedAt)

	setup, err = svc.UnconfirmSetup(t.Context(), "u1", "c1")
	require.NoError(t, err)
	assert.Nil(t, setup.ConfirmedAt)

	require.Len(t, repo.outbox, 2)
	assert.Equal(t, TopicSetupConfirmed, repo.outbox[0].Topic)
	assert.Equal(t, SetupChangedEvent{ComputerID: "c1", UserID: "u1", ConfirmedAt: &testNow}, repo.outbox[0].Payload)
	assert.Equal(t, TopicSetupUnconfirmed, repo.outbox[1].Topic)
	assert.Equal(t, SetupChangedEvent{ComputerID: "c1", UserID: "u1"}, repo.outbox[1].Payload)
	assert.NotEqual(t, repo.outbox[0].ID, repo.outbox[1].ID)
}

func TestSetup_ConfirmAndUnconfirmProvider(t *testing.T) {
	t.Parallel()
	svc, repo := newSetupService(t)

	setup, err := svc.ConfirmProviderSetup(t.Context(), "u1", "c1", " Claude ", []string{"tdd", " tdd ", "", "diagnose"})
	require.NoError(t, err)
	assert.Nil(t, setup.ConfirmedAt, "a provider's confirmation leaves the overall one alone")
	require.Len(t, setup.Providers, 1)
	assert.Equal(t, ProviderSetup{Provider: "claude", ConfirmedAt: &testNow, Skills: []string{"tdd", "diagnose"}}, setup.Providers[0])

	setup, err = svc.UnconfirmProviderSetup(t.Context(), "u1", "c1", "claude")
	require.NoError(t, err)
	require.Len(t, setup.Providers, 1)
	assert.Equal(t, ProviderSetup{Provider: "claude", Skills: []string{}}, setup.Providers[0])

	require.Len(t, repo.outbox, 2)
	assert.Equal(t, TopicSetupConfirmed, repo.outbox[0].Topic)
	assert.Equal(t, SetupChangedEvent{ComputerID: "c1", UserID: "u1", Provider: "claude", ConfirmedAt: &testNow, Skills: []string{"tdd", "diagnose"}}, repo.outbox[0].Payload)
	assert.Equal(t, TopicSetupUnconfirmed, repo.outbox[1].Topic)
}

func TestSetup_RePair_KeepsTheConfirmation(t *testing.T) {
	t.Parallel()
	svc, repo := newSetupService(t)
	_, err := svc.ConfirmSetup(t.Context(), "u1", "c1")
	require.NoError(t, err)

	stored := repo.computers["c1"]
	stored.Name = "renamed"
	require.NoError(t, repo.SaveComputer(t.Context(), stored))

	setup, err := svc.GetSetup(t.Context(), "u1", "c1")
	require.NoError(t, err)
	assert.NotNil(t, setup.ConfirmedAt)
}

func TestTopics_ListsBothSetupTopics(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"computer.setup_confirmed", "computer.setup_unconfirmed"}, Topics())
}

func toolNamed(t *testing.T, tools []mcptool.Tool, name string) mcptool.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found", name)
	return mcptool.Tool{}
}

func actorCtx(t *testing.T, userID string) context.Context {
	return identity.WithActor(t.Context(), identity.Actor{ID: userID})
}

func TestMCPTools_Shape(t *testing.T) {
	t.Parallel()
	svc, _ := newSetupService(t)
	var names []string
	for _, tool := range MCPTools(svc) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.Call)
		assert.Contains(t, tool.InputSchema["required"], "computer_id")
	}
	assert.ElementsMatch(t, []string{
		"computer_setup_get", "computer_setup_confirm_provider", "computer_setup_unconfirm_provider",
		"computer_setup_confirm", "computer_setup_unconfirm",
	}, names)
}

func TestMCPTools_ArgumentErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tool string
		args map[string]any
	}{
		{"get without computer", "computer_setup_get", map[string]any{}},
		{"confirm provider without provider", "computer_setup_confirm_provider", map[string]any{"computer_id": "c1", "skills": []any{"tdd"}}},
		{"confirm provider without skills", "computer_setup_confirm_provider", map[string]any{"computer_id": "c1", "provider": "claude"}},
		{"confirm provider with a non-string skill", "computer_setup_confirm_provider", map[string]any{"computer_id": "c1", "provider": "claude", "skills": []any{"tdd", 3.0}}},
		{"unconfirm provider without provider", "computer_setup_unconfirm_provider", map[string]any{"computer_id": "c1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, _ := newSetupService(t)
			_, err := toolNamed(t, MCPTools(svc), tt.tool).Call(actorCtx(t, "u1"), tt.args)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
}

func TestMCPTools_OwnerOnly(t *testing.T) {
	t.Parallel()
	svc, repo := newSetupService(t)
	for _, tool := range MCPTools(svc) {
		_, err := tool.Call(actorCtx(t, "u2"), map[string]any{"computer_id": "c1", "provider": "claude", "skills": []any{"tdd"}})
		assert.ErrorIs(t, err, apperrs.ErrNotFound, tool.Name)
		_, err = tool.Call(t.Context(), map[string]any{"computer_id": "c1", "provider": "claude", "skills": []any{"tdd"}})
		assert.ErrorIs(t, err, apperrs.ErrUnauthorized, tool.Name)
	}
	assert.Empty(t, repo.outbox)
}

func TestMCPTools_ConfirmFlow(t *testing.T) {
	t.Parallel()
	svc, _ := newSetupService(t)
	tools := MCPTools(svc)
	ctx := actorCtx(t, "u1")

	_, err := toolNamed(t, tools, "computer_setup_confirm_provider").Call(ctx, map[string]any{"computer_id": "c1", "provider": "codex", "skills": []any{"tdd"}})
	require.NoError(t, err)
	_, err = toolNamed(t, tools, "computer_setup_confirm").Call(ctx, map[string]any{"computer_id": "c1"})
	require.NoError(t, err)

	got, err := toolNamed(t, tools, "computer_setup_get").Call(ctx, map[string]any{"computer_id": "c1"})
	require.NoError(t, err)
	setup := got.(Setup)
	assert.NotNil(t, setup.ConfirmedAt)
	require.Len(t, setup.Providers, 1)
	assert.Equal(t, "codex", setup.Providers[0].Provider)

	_, err = toolNamed(t, tools, "computer_setup_unconfirm_provider").Call(ctx, map[string]any{"computer_id": "c1", "provider": "codex"})
	require.NoError(t, err)
	got, err = toolNamed(t, tools, "computer_setup_unconfirm").Call(ctx, map[string]any{"computer_id": "c1"})
	require.NoError(t, err)
	setup = got.(Setup)
	assert.Nil(t, setup.ConfirmedAt)
	assert.Nil(t, setup.Providers[0].ConfirmedAt)
}

func TestHandler_GetSetup(t *testing.T) {
	t.Parallel()
	h, repo, _ := newTestHandler()
	repo.computers["c1"] = Computer{ID: "c1", UserID: "u1", Name: "home"}
	routes := h.Routes()

	rec := doRequest(routes, http.MethodGet, "/api/pairing/computers/c1/setup", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"computer_id":"c1","confirmed_at":null,"providers":[]}`, rec.Body.String())

	rec = doRequest(routes, http.MethodGet, "/api/pairing/computers/c1/setup", "u2", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code, "another user's computer is invisible")
}

func TestHandler_Setup_HasNoWriteRoute(t *testing.T) {
	t.Parallel()
	h, repo, _ := newTestHandler()
	repo.computers["c1"] = Computer{ID: "c1", UserID: "u1", Name: "home"}
	routes := h.Routes()
	paths := []string{"/api/pairing/computers/c1/setup", "/api/pairing/computers/c1/setup/claude"}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, path := range paths {
			rec := doRequest(routes, method, path, "u1", map[string]any{"provider": "claude"})
			assert.Contains(t, []int{http.StatusMethodNotAllowed, http.StatusNotFound}, rec.Code, "%s %s", method, path)
		}
	}
	assert.Empty(t, repo.outbox)
}
