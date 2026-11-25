package composite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/shelljob"
)

// fakeCommands is both the computers' and the machines' side: a command ends at once with the output it is given.
type fakeCommands struct {
	ran    []string
	output string
	err    error
}

func (f *fakeCommands) run(where string, in shelljob.Input) (*shelljob.Job, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.ran = append(f.ran, where+": "+in.Command+" via "+in.Adapter)
	return &shelljob.Job{ID: "j-1", TimeoutSeconds: 60}, nil
}

func (f *fakeCommands) ended() (*shelljob.Job, error) {
	code := 2
	return &shelljob.Job{ID: "j-1", Status: shelljob.Failed, ExitCode: &code, Output: f.output}, nil
}

type fakeComputerCommands struct{ *fakeCommands }

func (f fakeComputerCommands) RunCommand(_ context.Context, userID, computerID string, in shelljob.Input) (*shelljob.Job, error) {
	return f.run(userID+" on computer "+computerID, in)
}

func (f fakeComputerCommands) WaitShellJob(context.Context, string, string) (*shelljob.Job, error) {
	return f.ended()
}

type fakeMachineCommands struct{ *fakeCommands }

func (f fakeMachineCommands) RunCommand(_ context.Context, runnerID string, in shelljob.Input) (*shelljob.Job, error) {
	return f.run("runner "+runnerID, in)
}

func (f fakeMachineCommands) WaitShellJob(context.Context, string) (*shelljob.Job, error) {
	return f.ended()
}

func callCommandRun(t *testing.T, f *fakeCommands, args string) (commandResult, error) {
	t.Helper()
	ctx := identity.WithActor(t.Context(), identity.Actor{ID: "alice"})
	got, err := CommandTools(fakeComputerCommands{f}, fakeMachineCommands{f})[0].Call(ctx, json.RawMessage(args))
	if err != nil {
		return commandResult{}, err
	}
	return got.(commandResult), nil
}

func TestCommandRun_RefusesWhatItCannotRunOrWaitFor(t *testing.T) {
	tests := []struct {
		name, args string
	}{
		{"neither target", `{"command":"id"}`},
		{"both targets", `{"computer_id":"c-1","runner_id":"r-1","command":"id"}`},
		{"over five minutes", `{"computer_id":"c-1","command":"sleep 600","timeout_seconds":301}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeCommands{}
			_, err := callCommandRun(t, f, tt.args)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Empty(t, f.ran)
		})
	}
}

func TestCommandRun_PassesOnTheUseCasesRefusal(t *testing.T) {
	f := &fakeCommands{err: fmt.Errorf("%w: runner r-9", apperrs.ErrNotFound)}
	_, err := callCommandRun(t, f, `{"runner_id":"r-9","command":"id"}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestCommandRun_RunsWhereItIsToldAndReturnsTheEndOfTheOutput(t *testing.T) {
	f := &fakeCommands{output: strings.Repeat("a", shelljob.AgentTail) + "boom\n"}

	got, err := callCommandRun(t, f, `{"runner_id":"r-1","command":"make test","timeout_seconds":120}`)
	require.NoError(t, err)
	assert.Equal(t, shelljob.Failed, got.Status)
	require.NotNil(t, got.ExitCode)
	assert.Equal(t, 2, *got.ExitCode)
	assert.True(t, got.OutputTruncated)
	assert.Len(t, got.Output, shelljob.AgentTail)
	assert.True(t, strings.HasSuffix(got.Output, "boom\n"))

	_, err = callCommandRun(t, f, `{"computer_id":"c-1","command":"ls"}`)
	require.NoError(t, err)
	assert.Equal(t, []string{"runner r-1: make test via mcp", "alice on computer c-1: ls via mcp"}, f.ran)
}
