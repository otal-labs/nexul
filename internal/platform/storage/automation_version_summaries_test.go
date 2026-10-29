package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestAutomationVersionsRepo_ListSummaries_OmitsCodeAndGetBySequenceFindsOne(t *testing.T) {
	s := newTestStore(t)
	seedTestAutomation(t, s, "a1")
	ctx := context.Background()
	v1 := &automations.Version{ID: "v1", AutomationID: "a1", Code: "code v1", Source: "src v1", PusherID: "owner", Message: "first", Status: automations.VersionActive, CreatedAt: time.Now().UTC()}
	require.NoError(t, s.AutomationVersions.InsertActive(ctx, v1))
	for _, id := range []string{"v2", "v3"} {
		require.NoError(t, s.AutomationVersions.ReplacePending(ctx, &automations.Version{ID: id, AutomationID: "a1", Code: "code " + id, PusherID: "owner", Status: automations.VersionPending, CreatedAt: time.Now().UTC()}))
	}

	list, err := s.AutomationVersions.ListSummaries(ctx, "a1")
	require.NoError(t, err)
	require.Len(t, list, 2, "the pending version v3 replaced v2")
	assert.Equal(t, []int{3, 1}, []int{list[0].Sequence, list[1].Sequence}, "newest first")
	assert.Equal(t, "first", list[1].Message)
	for _, v := range list {
		assert.Empty(t, v.Code)
		assert.Empty(t, v.Source)
	}

	got, err := s.AutomationVersions.GetBySequence(ctx, "a1", 1)
	require.NoError(t, err)
	assert.Equal(t, "code v1", got.Code)

	_, err = s.AutomationVersions.GetBySequence(ctx, "a1", 2)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a replaced pending version's number is never read back")
}
