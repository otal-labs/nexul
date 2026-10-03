package harness_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
)

// scripted answers every call with err and records which methods ran.
func scripted(kind harness.Kind, err error) (*harnesstest.Client, *[]string) {
	calls := &[]string{}
	rec := func(name string) error {
		*calls = append(*calls, name)
		return err
	}
	return &harnesstest.Client{
		KindValue: kind,
		PairFn: func(context.Context, string, string) (harness.PairResult, error) {
			return harness.PairResult{Kind: kind}, rec("Pair")
		},
		VersionFn:      func(context.Context, string) (string, error) { return string(kind), rec("Version") },
		ListProjectsFn: func(context.Context, harness.Session) ([]harness.Project, error) { return nil, rec("ListProjects") },
		ListProvidersFn: func(context.Context, harness.Session) ([]harness.Provider, error) {
			return nil, rec("ListProviders")
		},
		HoldFn: func(context.Context, harness.Session) (harness.Conn, error) { return nil, rec("Hold") },
		StartTurnFn: func(context.Context, harness.Target, string, harness.TurnPrompts) (harness.StartResult, error) {
			return harness.StartResult{SessionID: string(kind)}, rec("StartTurn")
		},
		InterruptFn: func(context.Context, harness.Target) error { return rec("Interrupt") },
		AnswerFn: func(context.Context, harness.Target, string, harness.QuestionAnswer) error {
			return rec("Answer")
		},
		SettleFn: func(context.Context, harness.Target) error { return rec("Settle") },
	}, calls
}

var laptop = harness.Session{ComputerID: "c1", Name: "laptop", ServerURL: "https://laptop.example.com", BearerToken: "b"}

// calls is every Client method; session marks the ones that reach a paired computer's session.
var calls = []struct {
	name    string
	session bool
	call    func(ctx context.Context, c harness.Client) error
}{
	{"Pair", false, func(ctx context.Context, c harness.Client) error {
		_, err := c.Pair(ctx, laptop.ServerURL, "tok")
		return err
	}},
	{"Version", false, func(ctx context.Context, c harness.Client) error {
		_, err := c.Version(ctx, laptop.ServerURL)
		return err
	}},
	{"ListProjects", true, func(ctx context.Context, c harness.Client) error {
		_, err := c.ListProjects(ctx, laptop)
		return err
	}},
	{"ListProviders", true, func(ctx context.Context, c harness.Client) error {
		_, err := c.ListProviders(ctx, laptop)
		return err
	}},
	{"Hold", true, func(ctx context.Context, c harness.Client) error {
		_, err := c.Hold(ctx, laptop)
		return err
	}},
	{"StartTurn", true, func(ctx context.Context, c harness.Client) error {
		_, err := c.StartTurn(ctx, harness.Target{Session: laptop}, "title", harness.TurnPrompts{})
		return err
	}},
	{"Interrupt", true, func(ctx context.Context, c harness.Client) error {
		return c.Interrupt(ctx, harness.Target{Session: laptop})
	}},
	{"Answer", true, func(ctx context.Context, c harness.Client) error {
		return c.Answer(ctx, harness.Target{Session: laptop}, "req-1", harness.QuestionAnswer{})
	}},
	{"Settle", true, func(ctx context.Context, c harness.Client) error {
		return c.Settle(ctx, harness.Target{Session: laptop})
	}},
}

type movedCall struct {
	session harness.Session
	to      harness.Kind
}

func TestForward_MovedFails_ReturnsItsErrorWithoutCallingTo(t *testing.T) {
	t.Parallel()
	errStore := errors.New("store down")
	for _, c := range calls {
		if !c.session {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			from, _ := scripted(harness.KindT3Code, &harness.MovedError{To: harness.KindT3CodeV2})
			to, toCalls := scripted(harness.KindT3CodeV2, nil)
			fwd := harness.Forward(from, to, func(context.Context, harness.Session, harness.Kind) error { return errStore })

			assert.ErrorIs(t, c.call(t.Context(), fwd), errStore)
			assert.Empty(t, *toCalls, "a move that was not recorded is not followed")
		})
	}
}

func TestForward_FromAnswers_ToIsNeverAsked(t *testing.T) {
	t.Parallel()
	errDown := errors.New("connection refused")
	for _, fromErr := range []error{nil, errDown} {
		for _, c := range calls {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				from, fromCalls := scripted(harness.KindT3Code, fromErr)
				to, toCalls := scripted(harness.KindT3CodeV2, nil)
				var moves []movedCall
				fwd := harness.Forward(from, to, func(_ context.Context, s harness.Session, k harness.Kind) error {
					moves = append(moves, movedCall{s, k})
					return nil
				})

				err := c.call(t.Context(), fwd)
				if fromErr == nil {
					require.NoError(t, err)
				}
				if fromErr != nil {
					require.ErrorIs(t, err, fromErr, "a failure that is not a move stays from's")
				}
				assert.Equal(t, []string{c.name}, *fromCalls)
				assert.Empty(t, *toCalls)
				assert.Empty(t, moves)
			})
		}
	}
}

func TestForward_FromMovedOn_SessionCallsRecordTheMoveOnceThenRunOnTo(t *testing.T) {
	t.Parallel()
	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			from, fromCalls := scripted(harness.KindT3Code, &harness.MovedError{To: harness.KindT3CodeV2})
			to, toCalls := scripted(harness.KindT3CodeV2, nil)
			var moves []movedCall
			fwd := harness.Forward(from, to, func(_ context.Context, s harness.Session, k harness.Kind) error {
				moves = append(moves, movedCall{s, k})
				return nil
			})

			require.NoError(t, c.call(t.Context(), fwd))
			assert.Equal(t, []string{c.name}, *fromCalls)
			assert.Equal(t, []string{c.name}, *toCalls)
			if c.session {
				assert.Equal(t, []movedCall{{laptop, harness.KindT3CodeV2}}, moves)
				return
			}
			assert.Empty(t, moves, "pairing and the version probe have no computer to move")
		})
	}
}

func TestForward_ResultsComeFromTheClientThatAnswered(t *testing.T) {
	t.Parallel()
	from, _ := scripted(harness.KindT3Code, &harness.MovedError{To: harness.KindT3CodeV2})
	to, _ := scripted(harness.KindT3CodeV2, nil)
	fwd := harness.Forward(from, to, func(context.Context, harness.Session, harness.Kind) error { return nil })

	assert.Equal(t, harness.KindT3Code, fwd.Kind(), "the registry entry keeps the kind computers are stored under")
	paired, err := fwd.Pair(t.Context(), laptop.ServerURL, "tok")
	require.NoError(t, err)
	assert.Equal(t, harness.KindT3CodeV2, paired.Kind, "a pairing says where it landed")
	started, err := fwd.StartTurn(t.Context(), harness.Target{Session: laptop}, "title", harness.TurnPrompts{})
	require.NoError(t, err)
	assert.Equal(t, "t3code-v2", started.SessionID)
}
