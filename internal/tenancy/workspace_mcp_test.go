package tenancy

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func TestWorkspaceList(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	mine, err := s.Create(context.Background(), "u-1", "Acme")
	require.NoError(t, err)
	_, err = s.Create(context.Background(), "u-2", "Other")
	require.NoError(t, err)
	call := WorkspaceMCPTools(s)[0].Call

	t.Run("no caller is unauthorized", func(t *testing.T) {
		_, err := call(context.Background(), json.RawMessage(`{}`))
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})
	t.Run("an unknown argument is invalid", func(t *testing.T) {
		ctx := identity.WithActor(context.Background(), identity.Actor{ID: "u-1"})
		_, err := call(ctx, json.RawMessage(`{"user_id":"u-2"}`))
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("lists only the caller's workspaces, with their role", func(t *testing.T) {
		ctx := identity.WithActor(context.Background(), identity.Actor{ID: "u-1"})
		out, err := call(ctx, json.RawMessage(`{}`))
		require.NoError(t, err)
		page, ok := out.(mcptool.Page[workspaceResult])
		require.True(t, ok)
		require.Len(t, page.Items, 1)
		assert.Equal(t, mine.ID, page.Items[0].ID)
		assert.Equal(t, "Acme", page.Items[0].Name)
		assert.NotEmpty(t, page.Items[0].Role)
	})
}
