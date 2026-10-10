package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/t3rpc"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

// countingDialer counts the streams the harness client opens.
type countingDialer struct {
	next  computerDialer
	dials atomic.Int32
}

func (d *countingDialer) DialComputer(ctx context.Context, computerID string) (net.Conn, error) {
	d.dials.Add(1)
	return d.next.DialComputer(ctx, computerID)
}

func TestIntegration_PersonalRunnerReachesT3Code(t *testing.T) {
	t3 := t3rpctest.New(t)
	db := mentionsTestDB(t)
	router, store, svc := routerOver(t, db)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	ctx := t.Context()

	now := time.Now().UTC()
	_, codeHash, err := hostcred.MintCode()
	require.NoError(t, err)
	require.NoError(t, store.Runners.CreateEnrollment(ctx, &runner.EnrollmentCode{CodeHash: codeHash, Name: "computer-ab12cd34", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}))
	credential, credentialHash, err := hostcred.MintCredential("nxr_")
	require.NoError(t, err)
	require.NoError(t, store.Runners.Enroll(ctx, codeHash, &runner.Runner{ID: "r-laptop", Name: "computer-ab12cd34", LastSeen: now, CreatedAt: now}, credentialHash, now))
	_, err = db.Exec(`UPDATE runners SET owner_user_id = 'u-alice', computer_id = 'c-laptop' WHERE id = 'r-laptop'`)
	require.NoError(t, err)

	t3URL, err := url.Parse(t3.URL)
	require.NoError(t, err)
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "userdata"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(home, "userdata", "server-runtime.json"),
		[]byte(`{"version":1,"pid":4242,"host":"127.0.0.1","port":`+t3URL.Port()+`,"origin":"`+t3.URL+`"}`), 0o600))
	base := "ws" + strings.TrimPrefix(srv.URL, "http")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	personal := runner.NewClient(runner.ClientConfig{
		URL: base + "/ws/runner", StreamURL: base + "/api/runners/streams", Credential: credential, Name: "computer-ab12cd34",
		Logger: logger, Executor: runner.NewShellExecutor(nil, runner.ExecutorConfig{}, logger), Personal: true, T3Home: home,
	})
	runCtx, stop := context.WithCancel(ctx)
	stopped := make(chan error, 1)
	go func() { stopped <- personal.Run(runCtx) }()
	t.Cleanup(func() {
		stop()
		<-stopped
	})
	require.Eventually(t, func() bool {
		rn, err := store.Runners.GetByID(ctx, "r-laptop")
		return err == nil && rn.Connected
	}, 5*time.Second, 10*time.Millisecond)

	dials := &countingDialer{next: svc.computers}
	client := harnessHTTPClient(dials, func(context.Context, string) (string, string, bool, error) { return "", "", false, nil })
	address := "http://c-laptop" + computerHostSuffix

	desc, err := t3rpc.Describe(ctx, client, address)
	require.NoError(t, err)
	assert.Equal(t, t3rpc.Descriptor{ServerVersion: "0.0.34", Protocol: 1}, desc, "the fake's descriptor, read through the runner")
	_, err = t3rpc.Describe(ctx, client, address)
	require.NoError(t, err)
	assert.Equal(t, int32(1), dials.dials.Load(), "the second request reused the pooled stream")

	session, err := t3rpc.Connect(ctx, harness.Session{ServerURL: address, BearerToken: t3.Session().BearerToken}, t3rpc.Options{HTTPClient: client, Logger: logger, RPCTimeout: 5 * time.Second})
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	assert.Equal(t, int32(1), t3.ConfigCalls.Load(), "server.getConfig completed through the runner")
	assert.Equal(t, int32(2), dials.dials.Load(), "the ticket reused the pool; the upgrade request pools apart, so the session socket opened the second stream")
}
