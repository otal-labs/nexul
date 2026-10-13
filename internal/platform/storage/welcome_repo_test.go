package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/welcome"
)

// TestWelcomeRepo_ARecordKeepsWhatLaterWritesMustNotUndo: the phone's first sign-in time stays through later ones,
// giving clears a skip but never forgets an earlier gift, and a skip can be taken back.
func TestWelcomeRepo_ARecordKeepsWhatLaterWritesMustNotUndo(t *testing.T) {
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, &auth.Identity{UserID: "u-lena", Provider: auth.ProviderGitHub, ProviderUserID: "1", Login: "lena"})
	require.NoError(t, err)
	first := time.Unix(1_000, 0).UTC()
	later := first.Add(time.Hour)

	require.NoError(t, s.Welcome.MarkDone(ctx, "u-lena", "phone", first))
	require.NoError(t, s.Welcome.MarkDone(ctx, "u-lena", "phone", later))
	require.NoError(t, s.Welcome.SetSkipped(ctx, "u-lena", "projects", &first))
	require.NoError(t, s.Welcome.Give(ctx, []welcome.Gift{{UserID: "u-lena", Step: "projects", GivenAt: &first}}))
	require.NoError(t, s.Welcome.SetSkipped(ctx, "u-lena", "projects", &later))
	require.NoError(t, s.Welcome.Give(ctx, []welcome.Gift{{UserID: "u-lena", Step: "projects"}}))
	require.NoError(t, s.Welcome.SetSkipped(ctx, "u-lena", "github", &first))
	require.NoError(t, s.Welcome.SetSkipped(ctx, "u-lena", "github", nil))

	got, err := s.Welcome.Records(ctx, []string{"u-lena", "u-nobody"})
	require.NoError(t, err)
	byStep := map[string]welcome.Record{}
	for _, r := range got {
		byStep[r.Step] = r
	}
	assert.Equal(t, &first, byStep["phone"].DoneAt)
	assert.Nil(t, byStep["projects"].SkippedAt, "a gift clears the skip")
	assert.Equal(t, &first, byStep["projects"].GivenAt, "a gift before the first run keeps the earlier one")
	assert.Nil(t, byStep["github"].SkippedAt)
	require.NoError(t, s.Welcome.Announce(ctx))
}
