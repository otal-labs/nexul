package plays

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// sharedComputers answers ComputerGate from a table keyed by user and computer: absent is a computer they cannot see.
type sharedComputers map[string]permissions.Set

func (g sharedComputers) ComputerPermissions(_ context.Context, userID, computerID string) (string, permissions.Set, error) {
	held, ok := g[userID+"/"+computerID]
	if !ok {
		return "", nil, apperrs.ErrNotFound
	}
	return "u-owner", held, nil
}

func computerFixture(t *testing.T) (*Service, *fakeRepo, map[Type]*Play) {
	t.Helper()
	s, repo, ps := autoPlayFixture(t)
	s.SetComputers(sharedComputers{
		"editor/c-own":   permissions.SetOf(permissions.ComputerActions...),
		"editor/c-run":   permissions.SetOf(permissions.ComputersSee, permissions.ComputersAgents),
		"editor/c-see":   permissions.SetOf(permissions.ComputersSee),
		"writer/c-own":   permissions.SetOf(permissions.ComputerActions...),
		"writer/c-other": permissions.SetOf(permissions.ComputerActions...),
	})
	stack, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Investigate", Type: TypeStack})
	require.NoError(t, err)
	ps[TypeStack] = stack
	return s, repo, ps
}

func TestAutoPlayComputer_IsPickedAsWhoeverSavesIt(t *testing.T) {
	s, _, ps := computerFixture(t)
	for _, computer := range []string{"c-own", "c-run"} {
		a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, AutoPlayInput{Moment: MomentStackServiceCrashed, ComputerID: computer})
		require.NoError(t, err, computer)
		assert.Equal(t, []any{RunOnComputer, computer, "editor"}, []any{a.RunOn, a.ComputerID, a.RunAs}, "a computer they own or may run agents on")
	}
}

func TestAutoPlayComputer_RefusesOneThePickerMayNotRunAgentsOn(t *testing.T) {
	s, repo, ps := computerFixture(t)
	repo.published = nil
	for _, computer := range []string{"c-see", "c-gone"} {
		_, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, AutoPlayInput{Moment: MomentStackServiceCrashed, ComputerID: computer})
		require.ErrorIs(t, err, apperrs.ErrInvalid, computer)
		assert.ErrorContains(t, err, "own or may run agents on")
	}
	assert.Empty(t, repo.published)

}

func TestAutoPlayComputer_NeedsAShareCheckToPickOne(t *testing.T) {
	s, _, _ := autoPlayFixture(t)
	stack, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Investigate", Type: TypeStack})
	require.NoError(t, err)

	_, err = s.CreateAutoPlay(ctxAs("editor"), workspaceID, stack.ID, AutoPlayInput{Moment: MomentStackServiceCrashed, ComputerID: "c-own"})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestAutoPlayComputer_SwitchingOnNeedsAComputer(t *testing.T) {
	s, _, ps := computerFixture(t)
	a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, AutoPlayInput{Moment: MomentStackServiceCrashed})
	require.NoError(t, err)
	assert.Equal(t, []any{false, RunOnComputer, "", ""}, []any{a.Enabled, a.RunOn, a.ComputerID, a.RunAs}, "a stack auto play waits for its computer")

	_, err = s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, a.ID, true, AutoPlayInput{Moment: MomentStackServiceCrashed})
	require.ErrorIs(t, err, apperrs.ErrInvalid)

	on, err := s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, a.ID, true, AutoPlayInput{Moment: MomentStackServiceCrashed, ComputerID: "c-own"})
	require.NoError(t, err)
	assert.Equal(t, []any{true, "c-own", "editor"}, []any{on.Enabled, on.ComputerID, on.RunAs})
}

func TestAutoPlayComputer_KeepingTheComputerKeepsWhoPickedIt(t *testing.T) {
	s, _, ps := computerFixture(t)
	a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, AutoPlayInput{Moment: MomentStackServiceCrashed, ComputerID: "c-own"})
	require.NoError(t, err)

	// Another editor changes only the moment; the run still takes the identity of whoever picked the computer.
	s.SetComputers(sharedComputers{"writer/c-own": permissions.SetOf(permissions.ComputerActions...), "writer/c-other": permissions.SetOf(permissions.ComputerActions...)})
	kept, err := s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, a.ID, false, AutoPlayInput{Moment: MomentStackServiceRecovered, ComputerID: "c-own"})
	require.NoError(t, err)
	assert.Equal(t, "editor", kept.RunAs)

	moved, err := s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, a.ID, false, AutoPlayInput{Moment: MomentStackServiceRecovered, ComputerID: "c-other"})
	require.ErrorIs(t, err, apperrs.ErrInvalid, "the editor holds nothing on the new computer")
	assert.Nil(t, moved)
}

func TestAutoPlayComputer_AnyOtherRunOnForgetsIt(t *testing.T) {
	s, _, ps := computerFixture(t)
	a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, AutoPlayInput{Moment: MomentTicketCreated, RunOn: RunOnComputer, ComputerID: "c-run"})
	require.NoError(t, err)
	assert.Equal(t, []any{RunOnComputer, "c-run", "editor"}, []any{a.RunOn, a.ComputerID, a.RunAs}, "a ticket play may run on a chosen computer too")

	back, err := s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, a.ID, false, AutoPlayInput{Moment: MomentTicketCreated, RunOn: RunOnDeveloper, ComputerID: "c-run"})
	require.NoError(t, err)
	assert.Equal(t, []any{RunOnDeveloper, "", ""}, []any{back.RunOn, back.ComputerID, back.RunAs})
}

func TestAutoPlayComputer_OnlyAStackPlayTakesStackMoments(t *testing.T) {
	s, _, ps := computerFixture(t)
	_, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, AutoPlayInput{Moment: MomentStackServiceCrashed})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, AutoPlayInput{Moment: MomentTicketCreated})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, AutoPlayInput{Moment: MomentStackServiceCrashed, RunOn: RunOnCauser})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}
