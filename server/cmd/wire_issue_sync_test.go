package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
)

func TestIssueSource_NeverShowsANexulLogin(t *testing.T) {
	t.Parallel()
	store := storage.New(migrated.DB(t), []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()
	_, _, err := store.Users.UpsertUser(ctx, &auth.Identity{UserID: "u-sam", Provider: auth.ProviderGoogle, ProviderUserID: "g-sam", Login: "sam@example.com"})
	require.NoError(t, err)
	_, _, err = store.Users.UpsertUser(ctx, &auth.Identity{UserID: "u-alice", Provider: auth.ProviderGitHub, ProviderUserID: "gh-alice", Login: "alice", Name: "Alice"})
	require.NoError(t, err)
	src := issueSource{users: store.Users, instanceURL: func(context.Context) (string, error) { return "https://nexul.example.com", nil }}

	body, err := src.markdown(ctx, "Ask [@sam@example.com](/people/u-sam) and [@alice](/people/u-alice)", false)
	require.NoError(t, err)
	assert.Equal(t, "Ask [a Nexul member](https://nexul.example.com/people/u-sam) and @alice", body,
		"someone without GitHub, or a name, shows as neither their login nor their email")

	sam, ok := src.person(ctx, "", "sam@example.com")
	require.True(t, ok)
	assert.Equal(t, "", sam.Name+sam.Login, "a message header falls back to a neutral name")
	agent, ok := src.person(ctx, "", "agent")
	assert.True(t, ok)
	assert.Equal(t, "Agent", agent.Name, "@Agent pings nobody on GitHub")
}
