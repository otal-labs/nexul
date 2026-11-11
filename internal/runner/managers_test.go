package runner

import (
	"context"
	"encoding/json"
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
	c.cfg.Managers = secretref.NewManagers(testLogger(), secretref.Setup{Scheme: "op", Open: func(context.Context) (secretref.Resolver, error) {
		if attempts.Add(1) == 1 {
			return nil, secretref.ErrUnreachable
		}
		return &secretreftest.Fake{}, nil
	}})
	c.cfg.Managers.SetRetry(20*time.Millisecond, 20*time.Millisecond)
	cancel, done := runClient(t, c)

	assert.JSONEq(t, `[{"manager":"op","state":"unreachable"}]`, <-reports)
	assert.JSONEq(t, `[{"manager":"op","state":"ready"}]`, <-reports)
	cancel()
	assert.NoError(t, <-done)
}

func TestClient_APersonalRunnerNeverReportsManagers(t *testing.T) {
	c := newTestClient("ws://server/ws/runner", &fakeExecutor{})
	c.cfg.Managers = secretref.NewManagers(testLogger(), secretref.Setup{Scheme: "op", Open: opens(&secretreftest.Fake{}, nil)})
	c.cfg.Managers.Start(context.Background(), &sync.WaitGroup{})

	u, err := url.Parse(c.wsURL())
	require.NoError(t, err)
	assert.True(t, u.Query().Has("managers"))

	c.cfg.Personal = true
	u, err = url.Parse(c.wsURL())
	require.NoError(t, err)
	assert.False(t, u.Query().Has("managers"))
}
