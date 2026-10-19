package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
)

func TestIntegration_AccessNames_NamesAPersonWithoutTheirEmail(t *testing.T) {
	t.Parallel()
	s := storage.New(migrated.DB(t), []byte("0123456789abcdef0123456789abcdef"))
	_, _, err := s.Users.UpsertUser(t.Context(), &auth.Identity{UserID: "u-lena", Provider: auth.ProviderDiscord, ProviderUserID: "d-lena", Login: "lena@example.com", Name: "Lena", Username: "lena_d"})
	require.NoError(t, err)

	name, err := accessNames{store: s}.Name(t.Context(), access.TargetPerson, "u-lena")

	require.NoError(t, err)
	assert.Equal(t, "Lena", name, "a sentence about a Discord or Google account never shows the email it signed in with")
}
