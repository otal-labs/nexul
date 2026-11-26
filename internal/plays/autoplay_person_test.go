package plays

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func personFixture(t *testing.T) (*Service, *fakeRepo, map[Type]*Play) {
	t.Helper()
	s, repo, ps := autoPlayFixture(t)
	stack, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Investigate", Type: TypeStack})
	require.NoError(t, err)
	ps[TypeStack] = stack
	return s, repo, ps
}

func TestAutoPlayPerson_ANewOneRunsAsItsCreator(t *testing.T) {
	s, _, ps := personFixture(t)
	a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, AutoPlayInput{Moment: MomentStackServiceCrashed})
	require.NoError(t, err)
	assert.Equal(t, []any{false, RunOnPerson, "editor", []string{}, ""}, []any{a.Enabled, a.RunOn, a.RunAs, a.ComputerIDs, a.ComputerID},
		"a stack auto play runs as whoever saved it, and names no computer")

	ticket, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, AutoPlayInput{Moment: MomentTicketCreated, RunOn: RunOnPerson})
	require.NoError(t, err)
	assert.Equal(t, []any{RunOnPerson, "editor"}, []any{ticket.RunOn, ticket.RunAs}, "a ticket play may run as a person too")
}

func TestAutoPlayPerson_TheOlderComputerSpellingReadsAsPerson(t *testing.T) {
	s, _, ps := personFixture(t)
	a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, AutoPlayInput{Moment: MomentStackServiceCrashed, RunOn: RunOnComputer})
	require.NoError(t, err)
	assert.Equal(t, []any{RunOnPerson, "editor"}, []any{a.RunOn, a.RunAs})
}

func TestAutoPlayPerson_AnUpdateKeepsThePersonUnlessRunAsMe(t *testing.T) {
	s, repo, ps := personFixture(t)
	stack := ps[TypeStack].ID
	a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, stack, AutoPlayInput{Moment: MomentStackServiceCrashed})
	require.NoError(t, err)
	repo.autoPlays[0].RunAs = "someone-else"

	kept, err := s.UpdateAutoPlay(ctxAs("editor"), workspaceID, stack, a.ID, true, AutoPlayInput{Moment: MomentStackServiceRecovered})
	require.NoError(t, err)
	assert.Equal(t, []any{"someone-else", true}, []any{kept.RunAs, kept.Enabled}, "saving another field does not take the run over")

	mine, err := s.UpdateAutoPlay(ctxAs("editor"), workspaceID, stack, a.ID, true, AutoPlayInput{Moment: MomentStackServiceRecovered, RunAsMe: true})
	require.NoError(t, err)
	assert.Equal(t, "editor", mine.RunAs, "run_as_me makes the caller the person it runs as")
}

func TestAutoPlayPerson_SwitchingOnNeedsAPerson(t *testing.T) {
	s, _, ps := personFixture(t)
	ticket := ps[TypeTicket].ID
	a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ticket, AutoPlayInput{Moment: MomentTicketCreated})
	require.NoError(t, err)
	assert.Empty(t, a.RunAs)

	_, err = s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ticket, a.ID, true, AutoPlayInput{Moment: MomentTicketCreated, RunOn: RunOnPerson})
	require.ErrorIs(t, err, apperrs.ErrInvalid, "no one has picked who it runs as")
	assert.ErrorContains(t, err, "pick the person it runs as")

	on, err := s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ticket, a.ID, true, AutoPlayInput{Moment: MomentTicketCreated, RunOn: RunOnPerson, RunAsMe: true})
	require.NoError(t, err)
	assert.Equal(t, []any{true, "editor"}, []any{on.Enabled, on.RunAs})
}

func TestAutoPlayPerson_AnyOtherRunOnForgetsThePerson(t *testing.T) {
	s, _, ps := personFixture(t)
	a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, AutoPlayInput{Moment: MomentTicketCreated, RunOn: RunOnPerson})
	require.NoError(t, err)

	back, err := s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, a.ID, false, AutoPlayInput{Moment: MomentTicketCreated, RunOn: RunOnDeveloper, RunAsMe: true})
	require.NoError(t, err)
	assert.Equal(t, []any{RunOnDeveloper, ""}, []any{back.RunOn, back.RunAs})
}

func TestAutoPlayPerson_OnlyAStackPlayTakesStackMoments(t *testing.T) {
	s, _, ps := personFixture(t)
	_, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, AutoPlayInput{Moment: MomentStackServiceCrashed})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, AutoPlayInput{Moment: MomentTicketCreated})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeStack].ID, AutoPlayInput{Moment: MomentStackServiceCrashed, RunOn: RunOnCauser})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}
