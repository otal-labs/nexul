package pairing

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// fakeRunners is the runner domain's side of a computer: the codes it minted and the runners enrolled per computer.
type fakeRunners struct {
	mu      sync.Mutex
	minted  []string
	runners map[string]ComputerRunner
	err     error
	// token is what PairingToken hands out, or tokenErr its failure; tokens counts the asks.
	token    string
	tokenErr error
	tokens   int
}

func (f *fakeRunners) PairingToken(_ context.Context, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokens++
	return f.token, f.tokenErr
}

func (f *fakeRunners) EnrollComputer(_ context.Context, userID, computerID string) (Enrollment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return Enrollment{}, f.err
	}
	if _, ok := f.runners[computerID]; ok {
		return Enrollment{}, apperrs.ErrConflict
	}
	f.minted = append(f.minted, userID+"/"+computerID)
	token := "eyJ." + computerID + ".sig"
	return Enrollment{Token: token, ExpiresAt: testNow.Add(time.Hour), Commands: InstallCommands{Unix: "curl computer.sh | sh -s -- " + token}}, nil
}

func (f *fakeRunners) ComputerRunner(_ context.Context, computerID string) (ComputerRunner, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.runners[computerID]; ok {
		return r, nil
	}
	return ComputerRunner{}, apperrs.ErrNotFound
}

func newComputerService() (*Service, *fakeRepo, *fakeRunners) {
	repo := newFakeRepo()
	runners := &fakeRunners{runners: map[string]ComputerRunner{}}
	svc := NewService(Config{Repo: repo, Harnesses: registry(&fakeExchanger{}), EncryptionKey: testEncKey, Tokens: newFakeTokens(), Runners: runners,
		Now: func() time.Time { return testNow }})
	return svc, repo, runners
}

func TestAddComputer_MakesAComputerWaitingForItsRunner(t *testing.T) {
	t.Parallel()
	svc, repo, runners := newComputerService()

	got, err := svc.AddComputer(t.Context(), "u-alice", " ")

	require.NoError(t, err)
	assert.Equal(t, "eyJ."+got.Computer.ID+".sig", got.Token)
	assert.Equal(t, "curl computer.sh | sh -s -- "+got.Token, got.Commands.Unix)
	assert.Equal(t, []string{"u-alice/" + got.Computer.ID}, runners.minted, "the code is bound to the caller and this computer")
	stored, err := repo.GetComputer(t.Context(), "u-alice", got.Computer.ID)
	require.NoError(t, err)
	assert.Empty(t, stored.Name, "named after its hostname once the runner enrolls")
	assert.False(t, stored.Paired())
}

func TestAddComputer_Refusals(t *testing.T) {
	t.Parallel()
	t.Run("signed out", func(t *testing.T) {
		svc, _, _ := newComputerService()
		_, err := svc.AddComputer(t.Context(), "", "Laptop")
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})
	t.Run("a code that cannot be minted leaves no computer behind", func(t *testing.T) {
		svc, repo, runners := newComputerService()
		runners.err = apperrs.ErrConflict
		_, err := svc.AddComputer(t.Context(), "u-alice", "Laptop")
		require.ErrorIs(t, err, apperrs.ErrConflict)
		assert.Empty(t, repo.computers)
	})
	t.Run("personal runners not wired", func(t *testing.T) {
		repo := newFakeRepo()
		_, err := NewService(Config{Repo: repo}).AddComputer(t.Context(), "u-alice", "Laptop")
		require.ErrorIs(t, err, apperrs.ErrFatal)
		assert.Empty(t, repo.computers)
	})
}

func TestEnrollComputer_GivesAFreshCodeUntilTheRunnerEnrolls(t *testing.T) {
	t.Parallel()
	svc, _, runners := newComputerService()
	added, err := svc.AddComputer(t.Context(), "u-alice", "Laptop")
	require.NoError(t, err)

	again, err := svc.EnrollComputer(t.Context(), "u-alice", added.Computer.ID)
	require.NoError(t, err)
	assert.Equal(t, added.Computer.ID, again.Computer.ID)

	runners.runners[added.Computer.ID] = ComputerRunner{Connected: true}
	_, err = svc.EnrollComputer(t.Context(), "u-alice", added.Computer.ID)
	require.ErrorIs(t, err, apperrs.ErrConflict, "one runner per computer")
}

func TestRenameComputer(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newComputerService()
	added, err := svc.AddComputer(t.Context(), "u-alice", "")
	require.NoError(t, err)

	_, err = svc.RenameComputer(t.Context(), "u-alice", added.Computer.ID, "  ")
	var field *FieldError
	require.ErrorAs(t, err, &field)
	assert.Equal(t, "name", field.Field)

	got, err := svc.RenameComputer(t.Context(), "u-alice", added.Computer.ID, " Work laptop ")
	require.NoError(t, err)
	assert.Equal(t, "Work laptop", got.Name)
	stored, err := repo.GetComputer(t.Context(), "u-alice", added.Computer.ID)
	require.NoError(t, err)
	assert.Equal(t, "Work laptop", stored.Name)
}

func TestListComputers_ShowsEachComputersRunner(t *testing.T) {
	t.Parallel()
	svc, _, runners := newComputerService()
	waiting, err := svc.AddComputer(t.Context(), "u-alice", "Desk")
	require.NoError(t, err)
	connected, err := svc.AddComputer(t.Context(), "u-alice", "Laptop")
	require.NoError(t, err)
	runners.runners[connected.Computer.ID] = ComputerRunner{Connected: true, LastSeen: testNow}

	listed, err := svc.ListComputers(t.Context(), "u-alice")
	require.NoError(t, err)
	byID := map[string]Computer{}
	for _, c := range listed {
		byID[c.ID] = c
	}
	assert.Nil(t, byID[waiting.Computer.ID].Runner)
	assert.Equal(t, &ComputerRunner{Connected: true, LastSeen: testNow}, byID[connected.Computer.ID].Runner)
}

func runnerEvent(t *testing.T, computerID, userID, state, hostname string) eventbus.Event {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"computer_id": computerID, "user_id": userID, "state": state, "hostname": hostname, "members_only": true})
	require.NoError(t, err)
	return eventbus.Event{Topic: "runner.personal_changed", Payload: raw}
}

func TestHandleRunnerChanged_NamesAnUnnamedComputerAfterItsHostname(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newComputerService()
	unnamed, err := svc.AddComputer(t.Context(), "u-alice", "")
	require.NoError(t, err)
	named, err := svc.AddComputer(t.Context(), "u-alice", "Work laptop")
	require.NoError(t, err)

	for _, ev := range []eventbus.Event{
		runnerEvent(t, unnamed.Computer.ID, "u-alice", "connected", "ignored"),
		runnerEvent(t, unnamed.Computer.ID, "u-bob", "enrolled", "not-bobs"),
		runnerEvent(t, "c-gone", "u-alice", "enrolled", "gone"),
		runnerEvent(t, unnamed.Computer.ID, "u-alice", "enrolled", " alice-desktop "),
		runnerEvent(t, named.Computer.ID, "u-alice", "enrolled", "alice-laptop"),
	} {
		require.NoError(t, svc.HandleRunnerChanged(t.Context(), ev))
	}

	got, err := repo.GetComputer(t.Context(), "u-alice", unnamed.Computer.ID)
	require.NoError(t, err)
	assert.Equal(t, "alice-desktop", got.Name)
	got, err = repo.GetComputer(t.Context(), "u-alice", named.Computer.ID)
	require.NoError(t, err)
	assert.Equal(t, "Work laptop", got.Name, "a name the person chose stays")

	err = svc.HandleRunnerChanged(t.Context(), eventbus.Event{Topic: "runner.personal_changed", Payload: []byte("{")})
	require.ErrorIs(t, err, apperrs.ErrFatal)
}
