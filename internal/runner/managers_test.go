package runner

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/secretref"
	"github.com/otal-labs/nexul/internal/platform/secretref/secretreftest"
)

func opens(res secretref.Resolver, err error) func(context.Context) (secretref.Resolver, error) {
	return func(context.Context) (secretref.Resolver, error) { return res, err }
}

func startManagers(t *testing.T, setups ...ManagerSetup) *Managers {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	m := NewManagers(testLogger(), setups...)
	m.base, m.max = time.Millisecond, 5*time.Millisecond
	m.Start(ctx, &wg)
	t.Cleanup(func() { cancel(); wg.Wait() })
	return m
}

func TestManagers_StateAfterStart(t *testing.T) {
	tests := []struct {
		name  string
		setup []ManagerSetup
		want  []ManagerState
	}{
		{"a token the manager accepts is ready", []ManagerSetup{{Scheme: "op", Open: opens(&secretreftest.Fake{}, nil)}},
			[]ManagerState{{Manager: "op", State: ManagerReady}}},
		{"any failure that is not unreachable is a rejected token", []ManagerSetup{{Scheme: "op", Open: opens(nil, errors.New("invalid service account token"))}},
			[]ManagerState{{Manager: "op", State: ManagerTokenRejected}}},
		{"no token reports no manager", nil, []ManagerState{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, startManagers(t, tt.setup...).States())
		})
	}
}

func TestManagers_UnreachableAtStartIsRetriedUntilItOpens_ARejectedTokenIsNot(t *testing.T) {
	var attempts atomic.Int32
	m := startManagers(t, ManagerSetup{Scheme: "op", Open: func(context.Context) (secretref.Resolver, error) {
		if attempts.Add(1) < 3 {
			return nil, secretref.ErrUnreachable
		}
		return &secretreftest.Fake{}, nil
	}})
	assert.Equal(t, []ManagerState{{Manager: "op", State: ManagerUnreachable}}, m.States())

	eventually(t, 2*time.Second, func() bool { return m.States()[0].State == ManagerReady })
	res, err := m.Registry().For("op://v/i/f")
	require.NoError(t, err)
	assert.Equal(t, "op", res.Scheme())

	var rejects atomic.Int32
	rejected := startManagers(t, ManagerSetup{Scheme: "op", Open: func(context.Context) (secretref.Resolver, error) {
		rejects.Add(1)
		return nil, errors.New("rejected")
	}})
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, []ManagerState{{Manager: "op", State: ManagerTokenRejected}}, rejected.States())
	assert.EqualValues(t, 1, rejects.Load(), "a rejected token is not retried")
}

func TestParseManagers_DropsWhatIsNotAKnownState(t *testing.T) {
	good := ManagerState{Manager: "op", State: ManagerReady}
	raw, err := json.Marshal([]ManagerState{good, {Manager: "op", State: "great"}, {Manager: "", State: ManagerReady}})
	require.NoError(t, err)

	assert.Equal(t, []ManagerState{good}, parseManagers(string(raw)))
	assert.Nil(t, parseManagers("not json"))
	assert.Nil(t, parseManagers(""))
}

func TestClient_ReportsItsManagersOnEveryConnect_AndReconnectsWhenOneChanges(t *testing.T) {
	reports := make(chan string, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		reports <- r.URL.Query().Get("managers")
		var raw json.RawMessage
		for wsjson.Read(r.Context(), conn, &raw) == nil {
		}
	}))
	t.Cleanup(srv.Close)

	var attempts atomic.Int32
	c := newTestClient(wsURL(srv), &fakeExecutor{})
	c.cfg.Managers = NewManagers(testLogger(), ManagerSetup{Scheme: "op", Open: func(context.Context) (secretref.Resolver, error) {
		if attempts.Add(1) == 1 {
			return nil, secretref.ErrUnreachable
		}
		return &secretreftest.Fake{}, nil
	}})
	c.cfg.Managers.base, c.cfg.Managers.max = 20*time.Millisecond, 20*time.Millisecond
	cancel, done := runClient(t, c)

	assert.JSONEq(t, `[{"manager":"op","state":"unreachable"}]`, <-reports)
	assert.JSONEq(t, `[{"manager":"op","state":"ready"}]`, <-reports)
	cancel()
	assert.NoError(t, <-done)
}

func TestClient_APersonalRunnerNeverReportsManagers(t *testing.T) {
	c := newTestClient("ws://server/ws/runner", &fakeExecutor{})
	c.cfg.Managers = NewManagers(testLogger(), ManagerSetup{Scheme: "op", Open: opens(&secretreftest.Fake{}, nil)})
	c.cfg.Managers.Start(context.Background(), &sync.WaitGroup{})

	u, err := url.Parse(c.wsURL())
	require.NoError(t, err)
	assert.True(t, u.Query().Has("managers"))

	c.cfg.Personal = true
	u, err = url.Parse(c.wsURL())
	require.NoError(t, err)
	assert.False(t, u.Query().Has("managers"))
}
