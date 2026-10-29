package storage_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/plays"
)

func TestDocRestore_RecordsTheVersionItCameFromAndAPlainSaveLeavesItNull(t *testing.T) {
	f := newHistoryFixture(t)
	ctx := actor("alice")
	_, err := f.svc.Update(ctx, f.doc.ID, "Spec", "second")
	require.NoError(t, err)
	_, err = f.svc.RestoreVersion(ctx, f.doc.ID, 1)
	require.NoError(t, err)

	hist, err := f.svc.History(ctx, f.doc.ID)
	require.NoError(t, err)
	require.Len(t, hist.Versions, 3)
	require.NotNil(t, hist.Versions[0].RestoredFrom)
	assert.Equal(t, 1, *hist.Versions[0].RestoredFrom, "the restore names the version it copied")
	assert.Nil(t, hist.Versions[1].RestoredFrom, "an ordinary save is not a restore")
	assert.Nil(t, hist.Versions[2].RestoredFrom)

	_, err = f.svc.RestoreVersion(ctx, f.doc.ID, 3)
	require.NoError(t, err)
	hist, err = f.svc.History(ctx, f.doc.ID)
	require.NoError(t, err)
	assert.Len(t, hist.Versions, 3, "restoring what the newest version already holds writes no row")
}

func TestTicketRestore_RecordsTheVersionItCameFromAndAPlainSaveLeavesItNull(t *testing.T) {
	f := newTicketHistoryFixture(t)
	ctx := actor("alice")
	_, err := f.svc.UpdateTicket(ctx, f.ticket.ID, f.ticket.Title, "second")
	require.NoError(t, err)
	_, err = f.svc.RestoreVersion(ctx, f.ticket.ID, 1)
	require.NoError(t, err)

	hist := f.history(t)
	require.Len(t, hist.Versions, 3)
	require.NotNil(t, hist.Versions[0].RestoredFrom)
	assert.Equal(t, 1, *hist.Versions[0].RestoredFrom)
	assert.Nil(t, hist.Versions[1].RestoredFrom)
	assert.Nil(t, hist.Versions[2].RestoredFrom)

	_, err = f.svc.RestoreVersion(ctx, f.ticket.ID, 3)
	require.NoError(t, err)
	assert.Len(t, f.history(t).Versions, 3, "restoring what the newest version already holds writes no row")
}

// playGate lets every caller do everything, so the test reaches the history path rather than the permission one.
type playGate struct{}

func (playGate) HasPermission(context.Context, string, string, permissions.Action, string, string) bool {
	return true
}

func (playGate) PlaysWith(_ context.Context, _, _ string, _ permissions.Action, ids []string) map[string]bool {
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out
}

func (playGate) CallerProjects(context.Context, permissions.Action) ([]string, bool, error) {
	return nil, true, nil
}

func (playGate) DocsWith(context.Context, string, string, permissions.Action) (permissions.DocReach, error) {
	return permissions.DocReach{}, nil
}

func TestPlayRestore_RecordsTheVersionItCameFromAndAPlainSaveLeavesItNull(t *testing.T) {
	s := storage.New(newDB(t), []byte("0123456789abcdef0123456789abcdef"))
	svc := plays.NewService(s.Plays, playGate{})
	ctx := actor("alice")
	p, err := svc.Create(ctx, "workspace-default", plays.CreateInput{Label: "Fix", Type: plays.TypeDoc, Instructions: "First.", Enabled: true})
	require.NoError(t, err)
	_, err = svc.Update(ctx, "workspace-default", p.ID, plays.UpdateInput{Label: "Fix", Instructions: "Second.", Enabled: true})
	require.NoError(t, err)
	_, err = svc.Restore(ctx, p.ID, 1)
	require.NoError(t, err)

	h, err := svc.History(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, h.Versions, 3)
	require.NotNil(t, h.Versions[0].RestoredFrom)
	assert.Equal(t, 1, *h.Versions[0].RestoredFrom)
	assert.Nil(t, h.Versions[1].RestoredFrom)
	assert.Nil(t, h.Versions[2].RestoredFrom)

	_, err = svc.Restore(ctx, p.ID, 3)
	require.NoError(t, err)
	h, err = svc.History(ctx, p.ID)
	require.NoError(t, err)
	assert.Len(t, h.Versions, 3, "restoring what the newest version already holds writes no row")
}

// sizeOf reads a version's stored size as "+added -removed", or "none" when it has no size.
func sizeOf(added, removed *int) string {
	if added == nil || removed == nil {
		return "none"
	}
	return fmt.Sprintf("+%d -%d", *added, *removed)
}

func TestDocSave_StoresTheSizeAgainstThePreviousVersionIncludingTheTitle(t *testing.T) {
	f := newHistoryFixture(t)
	ctx := actor("alice")
	_, err := f.svc.Update(ctx, f.doc.ID, "Spec", "first second")
	require.NoError(t, err)
	_, err = f.svc.Update(ctx, f.doc.ID, "Spec renamed", "first second")
	require.NoError(t, err)
	_, err = f.svc.Update(ctx, f.doc.ID, "Spec renamed", "first")
	require.NoError(t, err)

	hist, err := f.svc.History(ctx, f.doc.ID)
	require.NoError(t, err)
	got := []string{}
	for _, v := range hist.Versions {
		got = append(got, sizeOf(v.WordsAdded, v.WordsRemoved))
	}
	assert.Equal(t, []string{"+0 -1", "+1 -0", "+1 -0", "none"}, got, "newest first: a dropped word, a title word, a body word, and no size on the first version")
}

// TestDocUpdate_APruneThatDropsANamedVersionsPredecessor_ResizesItAndClearsTheNewOldest keeps two named versions
// ahead of the prune: B sits right after the unnamed version the prune drops, so its predecessor becomes A.
func TestDocUpdate_APruneThatDropsANamedVersionsPredecessor_ResizesItAndClearsTheNewOldest(t *testing.T) {
	f := newHistoryFixture(t)
	ctx := actor("alice")
	_, err := f.svc.CreateNamedVersion(ctx, f.doc.ID, "A")
	require.NoError(t, err)
	_, err = f.svc.Update(ctx, f.doc.ID, "Spec", "second")
	require.NoError(t, err)
	_, err = f.svc.CreateNamedVersion(ctx, f.doc.ID, "B")
	require.NoError(t, err)

	sizes := func() map[string]string {
		hist, err := f.svc.History(ctx, f.doc.ID)
		require.NoError(t, err)
		out := map[string]string{}
		for _, v := range hist.Versions {
			out[v.Name] = sizeOf(v.WordsAdded, v.WordsRemoved)
		}
		return out
	}
	require.Equal(t, "+0 -0", sizes()["B"], "before the prune B repeats the unnamed version just before it")
	require.Equal(t, "+0 -0", sizes()["A"], "and A repeats the first version")

	for i := range 10 {
		_, err = f.svc.Update(ctx, f.doc.ID, "Spec", fmt.Sprintf("edit %d", i))
		require.NoError(t, err)
	}

	got := sizes()
	assert.Equal(t, "none", got["A"], "A is the oldest kept once the first version is pruned")
	assert.Equal(t, "+1 -1", got["B"], "B now follows A, whose text differs by one word")
}

func TestTicketUpdate_StoresTheSizeOverPlainTextAndPruneClearsTheNewOldest(t *testing.T) {
	f := newTicketHistoryFixture(t)
	ctx := actor("alice")
	richBody := `{"type":"doc","content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"first"}]},{"type":"paragraph","content":[{"type":"text","text":"two","marks":[{"type":"bold"}]},{"type":"text","text":" words"}]}]}`
	_, err := f.svc.UpdateTicket(ctx, f.ticket.ID, f.ticket.Title, richBody)
	require.NoError(t, err)
	hist := f.history(t)
	assert.Equal(t, "+2 -0", sizeOf(hist.Versions[0].WordsAdded, hist.Versions[0].WordsRemoved), "markup and block boundaries add no words")
	assert.Equal(t, "none", sizeOf(hist.Versions[1].WordsAdded, hist.Versions[1].WordsRemoved), "the first version has no size")

	for i := range 10 {
		_, err = f.svc.UpdateTicket(ctx, f.ticket.ID, f.ticket.Title, fmt.Sprintf("body %d", i))
		require.NoError(t, err)
	}

	hist = f.history(t)
	require.Len(t, hist.Versions, 10)
	assert.Equal(t, "+1 -1", sizeOf(hist.Versions[0].WordsAdded, hist.Versions[0].WordsRemoved))
	last := hist.Versions[9]
	assert.Equal(t, "none", sizeOf(last.WordsAdded, last.WordsRemoved), "version %d lost its predecessor to the prune", last.Version)
}
