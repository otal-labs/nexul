package t3clientv2

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/t3rpc"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

func newFake(t *testing.T, protocol int) (*t3rpctest.Server, *Harness) {
	f := t3rpctest.New(t)
	f.Protocol = protocol
	return f, NewHarness(t3rpc.Options{HTTPClient: f.Client(), RPCTimeout: 5 * time.Second})
}

func laptop(f *t3rpctest.Server) harness.Session {
	s := f.Session()
	s.Name = "Onik's laptop"
	return s
}

// sessionCalls is every harness.Client method that dials a paired computer.
var sessionCalls = []struct {
	name string
	call func(ctx context.Context, h *Harness, s harness.Session) error
}{
	{"ListProjects", func(ctx context.Context, h *Harness, s harness.Session) error {
		_, err := h.ListProjects(ctx, s)
		return err
	}},
	{"ListProviders", func(ctx context.Context, h *Harness, s harness.Session) error {
		_, err := h.ListProviders(ctx, s)
		return err
	}},
	{"Hold", func(ctx context.Context, h *Harness, s harness.Session) error {
		c, err := h.Hold(ctx, s)
		if err == nil {
			_ = c.Close()
		}
		return err
	}},
	{"Settle", func(ctx context.Context, h *Harness, s harness.Session) error {
		return h.Settle(ctx, harness.Target{Session: s, SessionID: "th-1"})
	}},
	{"StartTurn", func(ctx context.Context, h *Harness, s harness.Session) error {
		_, err := h.StartTurn(ctx, harness.Target{Session: s, SessionID: "th-1"}, "title", testPrompts)
		return err
	}},
	{"Interrupt", func(ctx context.Context, h *Harness, s harness.Session) error {
		return h.Interrupt(ctx, harness.Target{Session: s, SessionID: "th-1"})
	}},
	{"Answer", func(ctx context.Context, h *Harness, s harness.Session) error {
		return h.Answer(ctx, harness.Target{Session: s, SessionID: "th-1"}, "rq-1", harness.QuestionAnswer{})
	}},
}

func TestSessionCalls_ServerItCannotFollow_RefusedWithTheMessageToShow(t *testing.T) {
	t.Parallel()
	const wentBack = "T3 Code on Onik's laptop went back to its old orchestrator; Nexul only moves forward. Update T3 Code there."
	tests := []struct {
		name     string
		protocol int
		want     string
	}{
		{"handshake names no protocol", 0, wentBack},
		{"handshake names protocol 1 on a protocol-2 dial", 1, wentBack},
		{"426 naming protocol 3", 3, "T3 Code on Onik's laptop needs a newer Nexul."},
	}
	for _, tt := range tests {
		for _, sc := range sessionCalls {
			t.Run(tt.name+"/"+sc.name, func(t *testing.T) {
				t.Parallel()
				f, h := newFake(t, tt.protocol)
				err := sc.call(t.Context(), h, laptop(f))
				require.ErrorIs(t, err, harness.ErrProtocol)
				assert.EqualError(t, err, tt.want)
				assert.NotErrorIs(t, err, apperrs.ErrRetryable, "redialing cannot change the protocol T3 speaks")
			})
		}
	}
}

func TestSessionCalls_RejectedBearer_IsUnauthorized(t *testing.T) {
	t.Parallel()
	for _, sc := range sessionCalls {
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, 2)
			s := laptop(f)
			s.BearerToken = "revoked"
			assert.ErrorIs(t, sc.call(t.Context(), h, s), apperrs.ErrUnauthorized)
		})
	}
}

func TestPairAndVersion_ReadTheDescriptorBeforeSpendingTheToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		protocol int
		want     string // the refusal, empty when the pairing goes through
	}{
		{"descriptor names no protocol", 0, "went back to its old orchestrator; Nexul only moves forward. Update T3 Code there."},
		{"descriptor names protocol 1", 1, "went back to its old orchestrator; Nexul only moves forward. Update T3 Code there."},
		{"descriptor names protocol 3", 3, "needs a newer Nexul."},
		{"protocol 2", 2, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, h := newFake(t, tt.protocol)
			host, err := url.Parse(f.URL)
			require.NoError(t, err)

			version, versionErr := h.Version(t.Context(), f.URL)
			result, pairErr := h.Pair(t.Context(), f.URL, f.PairToken)
			if tt.want != "" {
				for _, err := range []error{versionErr, pairErr} {
					require.ErrorIs(t, err, harness.ErrProtocol)
					assert.EqualError(t, err, "T3 Code on "+host.Host+" "+tt.want)
				}
				assert.Zero(t, f.Exchanges.Load(), "the one-time token stays unspent for the client that can use it")
				return
			}
			require.NoError(t, versionErr)
			require.NoError(t, pairErr)
			assert.Equal(t, "0.0.34", version)
			assert.Equal(t, harness.PairResult{BearerToken: "bearer-token", ExpiresIn: 30 * 24 * time.Hour, Version: "0.0.34", Kind: harness.KindT3CodeV2}, result)
			assert.Equal(t, int32(1), f.Exchanges.Load())
		})
	}
}

func TestSessionCalls_Protocol2_ReturnWhatProtocol1Returns(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)
	ctx := t.Context()

	providers, err := h.ListProviders(ctx, laptop(f))
	require.NoError(t, err)
	assert.Equal(t, []harness.Provider{{ID: "claudeAgent", Driver: "claudeAgent", Name: "Claude", Version: "2.1.288",
		Models: []harness.ProviderModel{{Slug: "claude-opus-5-5", Name: "Claude Opus 5.5", IsDefault: true}}}}, providers)
	dial := t3rpctest.WaitFor(t, f.Dialed, "providers dial")
	assert.Equal(t, url.Values{"orchestrationProtocol": {"2"}, "clientAppVersion": {"nexul/dev"}}, dial)

	projects, err := h.ListProjects(ctx, laptop(f))
	require.NoError(t, err)
	assert.Equal(t, []harness.Project{
		{ID: "proj-live", Title: "My App", Path: "/home/me/app"},
		{ID: "proj-two", Title: "Second", Path: "/home/me/second"},
	}, projects, "the first snapshot is the registry; the identity-roots one that follows changes no row")

	conn, err := h.Hold(ctx, laptop(f))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	select {
	case <-conn.Done():
		t.Fatal("Hold handed back a connection that is already closed")
	default:
	}
	f.Drop()
	t3rpctest.WaitFor(t, conn.Done(), "presence noticing the dropped socket")
}

func TestSettle_DispatchesThreadSettleWithoutOptionalKeys(t *testing.T) {
	t.Parallel()
	f, h := newFake(t, 2)

	require.NoError(t, h.Settle(t.Context(), harness.Target{Session: laptop(f), SessionID: "th-1"}))
	cmd := t3rpctest.WaitFor(t, f.Dispatched, "thread.settle dispatch")
	assert.NotEmpty(t, cmd["commandId"])
	assert.Equal(t, map[string]any{"type": "thread.settle", "commandId": cmd["commandId"], "threadId": "th-1"}, cmd)

	err := h.Settle(t.Context(), harness.Target{Session: laptop(f)})
	assert.ErrorIs(t, err, apperrs.ErrInvalid, "nothing to settle without a thread")
}
