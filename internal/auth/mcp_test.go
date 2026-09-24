package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func whoamiTool(t *testing.T, s *Service) mcptool.Tool {
	t.Helper()
	for _, tool := range MCPTools(s) {
		if tool.Name == "account_whoami" {
			return tool
		}
	}
	t.Fatal("account_whoami not registered")
	return mcptool.Tool{}
}

func TestAccountWhoami(t *testing.T) {
	t.Parallel()
	s, users, _, _ := newTestHarness(&fakeGitHub{})
	u, _, err := users.UpsertUser(t.Context(), &User{ID: "u1", Provider: ProviderGitHub, ProviderUserID: "1", Login: "onik97"})
	require.NoError(t, err)
	call := whoamiTool(t, s).Call

	tests := []struct {
		name    string
		actorID string
		wantErr error
	}{
		{"no token user", "", apperrs.ErrUnauthorized},
		{"unknown user", "ghost", apperrs.ErrNotFound},
		{"token user", u.ID, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := identity.WithActor(t.Context(), identity.Actor{ID: tt.actorID})
			got, err := call(ctx, nil)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "onik97", got.(*User).Login)
		})
	}
}
