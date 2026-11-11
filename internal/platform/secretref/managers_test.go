package secretref_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/secretref"
	"github.com/otal-labs/nexul/internal/platform/secretref/secretreftest"
)

func opens(res secretref.Resolver, err error) func(context.Context) (secretref.Resolver, error) {
	return func(context.Context) (secretref.Resolver, error) { return res, err }
}

func startManagers(t *testing.T, setups ...secretref.Setup) *secretref.Managers {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	m := secretref.NewManagers(slog.Default(), setups...)
	m.SetRetry(time.Millisecond, 5*time.Millisecond)
	m.Start(ctx, &wg)
	t.Cleanup(func() { cancel(); wg.Wait() })
	return m
}

func TestManagers_StateAfterStart(t *testing.T) {
	tests := []struct {
		name  string
		setup []secretref.Setup
		want  []secretref.ManagerState
	}{
		{"a token the manager accepts is ready", []secretref.Setup{{Scheme: "op", Open: opens(&secretreftest.Fake{}, nil)}},
			[]secretref.ManagerState{{Manager: "op", State: secretref.ManagerReady}}},
		{"any failure that is not unreachable is a rejected token", []secretref.Setup{{Scheme: "op", Open: opens(nil, errors.New("invalid service account token"))}},
			[]secretref.ManagerState{{Manager: "op", State: secretref.ManagerTokenRejected}}},
		{"no token reports no manager", nil, []secretref.ManagerState{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, startManagers(t, tt.setup...).States())
		})
	}
}

func TestManagers_UnreachableAtStartIsRetriedUntilItOpens_ARejectedTokenIsNot(t *testing.T) {
	var attempts atomic.Int32
	m := startManagers(t, secretref.Setup{Scheme: "op", Open: func(context.Context) (secretref.Resolver, error) {
		if attempts.Add(1) < 3 {
			return nil, secretref.ErrUnreachable
		}
		return &secretreftest.Fake{}, nil
	}})
	assert.Equal(t, []secretref.ManagerState{{Manager: "op", State: secretref.ManagerUnreachable}}, m.States())

	require.Eventually(t, func() bool { return m.States()[0].State == secretref.ManagerReady }, 2*time.Second, time.Millisecond)
	res, err := m.Registry().For("op://v/i/f")
	require.NoError(t, err)
	assert.Equal(t, "op", res.Scheme())

	var rejects atomic.Int32
	rejected := startManagers(t, secretref.Setup{Scheme: "op", Open: func(context.Context) (secretref.Resolver, error) {
		rejects.Add(1)
		return nil, errors.New("rejected")
	}})
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, []secretref.ManagerState{{Manager: "op", State: secretref.ManagerTokenRejected}}, rejected.States())
	assert.EqualValues(t, 1, rejects.Load(), "a rejected token is not retried")
}

func TestManagers_OnRejectedFiresOnceWhenAStateTurnsTokenRejected(t *testing.T) {
	var attempts atomic.Int32
	tests := []struct {
		name  string
		open  func(context.Context) (secretref.Resolver, error)
		fired int32
	}{
		{"a token the manager accepts", opens(&secretreftest.Fake{}, nil), 0},
		{"a token refused at once", opens(nil, errors.New("rejected")), 1},
		{"unreachable, then refused when it answers", func(context.Context) (secretref.Resolver, error) {
			if attempts.Add(1) < 3 {
				return nil, secretref.ErrUnreachable
			}
			return nil, errors.New("rejected")
		}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			var fired atomic.Int32
			m := secretref.NewManagers(slog.Default(), secretref.Setup{Scheme: "op", Open: tt.open})
			m.SetRetry(time.Millisecond, 5*time.Millisecond)
			m.SetOnRejected(func(_ context.Context, scheme string) {
				assert.Equal(t, "op", scheme)
				fired.Add(1)
			})
			m.Start(ctx, &wg)
			t.Cleanup(func() { cancel(); wg.Wait() })

			require.Eventually(t, func() bool { return m.States()[0].State != secretref.ManagerUnreachable }, 2*time.Second, time.Millisecond)
			wg.Wait()
			assert.Equal(t, tt.fired, fired.Load())
		})
	}
}
