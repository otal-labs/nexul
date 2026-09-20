package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reexecRecorder collects reexecFn invocations under a mutex: the stubbed call runs on the client's background
// goroutine while a test's require.Eventually polls it from another, so a bare slice would race.
type reexecRecorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *reexecRecorder) record(exe string) {
	r.mu.Lock()
	r.calls = append(r.calls, exe)
	r.mu.Unlock()
}

func (r *reexecRecorder) Calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

// stubReexec replaces reexecFn for the duration of the test so applyUpdate's success path never actually
// execs over the test binary.
func stubReexec(t *testing.T) *reexecRecorder {
	t.Helper()
	rec := &reexecRecorder{}
	orig := reexecFn
	reexecFn = func(exe string, _, _ []string) error {
		rec.record(exe)
		return nil
	}
	t.Cleanup(func() { reexecFn = orig })
	return rec
}

func writeExe(t *testing.T, content string) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "runner")
	require.NoError(t, os.WriteFile(exe, []byte(content), 0o755))
	return exe
}

func TestApplyUpdate(t *testing.T) {
	t.Run("downloads, verifies, swaps the binary, and re-execs", func(t *testing.T) {
		body := "new-binary-bytes"
		sum := sha256.Sum256([]byte(body))
		sha := hex.EncodeToString(sum[:])
		var gotAuth string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		exe := writeExe(t, "old-binary")
		calls := stubReexec(t)

		frame := Frame{Type: FrameUpdate, Version: "v0.2.0", URL: srv.URL, Sha256: sha}
		require.NoError(t, applyUpdate(context.Background(), frame, "s3cr3t", exe))

		assert.Equal(t, "Bearer s3cr3t", gotAuth)
		assert.Equal(t, []string{exe}, calls.Calls())

		got, err := os.ReadFile(exe)
		require.NoError(t, err)
		assert.Equal(t, body, string(got))

		old, err := os.ReadFile(exe + ".old")
		require.NoError(t, err)
		assert.Equal(t, "old-binary", string(old))

		_, err = os.Stat(exe + ".new")
		assert.True(t, os.IsNotExist(err), "the .new file must be renamed away, not left behind")
	})

	t.Run("empty sha256 skips verification", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("new-binary-bytes"))
		}))
		t.Cleanup(srv.Close)
		exe := writeExe(t, "old-binary")
		stubReexec(t)

		frame := Frame{Type: FrameUpdate, Version: "v0.2.0", URL: srv.URL}
		require.NoError(t, applyUpdate(context.Background(), frame, "s3cr3t", exe))
	})

	t.Run("checksum mismatch deletes the download and leaves the running binary untouched", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("new-binary-bytes"))
		}))
		t.Cleanup(srv.Close)
		exe := writeExe(t, "old-binary")

		frame := Frame{Type: FrameUpdate, Version: "v0.2.0", URL: srv.URL, Sha256: "0000000000000000000000000000000000000000000000000000000000000000"}
		err := applyUpdate(context.Background(), frame, "s3cr3t", exe)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")

		_, err = os.Stat(exe + ".new")
		assert.True(t, os.IsNotExist(err), "a checksum mismatch must delete the partial download")
		got, err := os.ReadFile(exe)
		require.NoError(t, err)
		assert.Equal(t, "old-binary", string(got), "the running binary must be untouched on a failed verification")
		_, err = os.Stat(exe + ".old")
		assert.True(t, os.IsNotExist(err), "a failed verification must never swap the binary")
	})

	t.Run("non-200 status is an error, nothing touched", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		t.Cleanup(srv.Close)
		exe := writeExe(t, "old-binary")

		frame := Frame{Type: FrameUpdate, Version: "v0.2.0", URL: srv.URL}
		err := applyUpdate(context.Background(), frame, "s3cr3t", exe)
		require.Error(t, err)
		_, err = os.Stat(exe + ".new")
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("reexec failure surfaces as an error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("new-binary-bytes"))
		}))
		t.Cleanup(srv.Close)
		exe := writeExe(t, "old-binary")
		orig := reexecFn
		reexecFn = func(string, []string, []string) error { return assert.AnError }
		t.Cleanup(func() { reexecFn = orig })

		frame := Frame{Type: FrameUpdate, Version: "v0.2.0", URL: srv.URL}
		err := applyUpdate(context.Background(), frame, "s3cr3t", exe)
		require.Error(t, err)
	})
}

func TestIsContainerRunner(t *testing.T) {
	t.Run("NEXUL_RUNNER_ID=instance", func(t *testing.T) {
		t.Setenv("NEXUL_RUNNER_ID", "instance")
		assert.True(t, isContainerRunner())
	})

	t.Run("neither signal present", func(t *testing.T) {
		t.Setenv("NEXUL_RUNNER_ID", "host-1")
		assert.False(t, isContainerRunner())
	})
}

func TestClient_handleUpdate(t *testing.T) {
	t.Run("ignored for a container runner", func(t *testing.T) {
		t.Setenv("NEXUL_RUNNER_ID", "instance")
		c := newTestClient("ws://server/ws/runner", &fakeExecutor{})
		c.cfg.Version = "v0.1.6"

		c.handleUpdate(context.Background(), Frame{Type: FrameUpdate, Version: "v0.2.0", URL: "http://unused"})

		c.mu.Lock()
		defer c.mu.Unlock()
		assert.Nil(t, c.pendingUpdate)
	})

	t.Run("ignored for a dev build", func(t *testing.T) {
		t.Setenv("NEXUL_RUNNER_ID", "host-1")
		c := newTestClient("ws://server/ws/runner", &fakeExecutor{})
		c.cfg.Version = "dev"

		c.handleUpdate(context.Background(), Frame{Type: FrameUpdate, Version: "v0.2.0", URL: "http://unused"})

		c.mu.Lock()
		defer c.mu.Unlock()
		assert.Nil(t, c.pendingUpdate)
	})

	t.Run("deferred while a job is running, never touching the executable", func(t *testing.T) {
		t.Setenv("NEXUL_RUNNER_ID", "host-1")
		c := newTestClient("ws://server/ws/runner", &fakeExecutor{})
		c.cfg.Version = "v0.1.6"
		origExeFn := runnerExecutableFn
		runnerExecutableFn = func() (string, error) {
			t.Fatal("applyUpdateFrame must not run while a job is busy")
			return "", nil
		}
		t.Cleanup(func() { runnerExecutableFn = origExeFn })

		_, cancel := context.WithCancel(context.Background())
		c.mu.Lock()
		c.jobID = "d1"
		c.jobCancel = cancel
		c.mu.Unlock()
		t.Cleanup(cancel)

		frame := Frame{Type: FrameUpdate, Version: "v0.2.0", URL: "http://unused"}
		c.handleUpdate(context.Background(), frame)

		c.mu.Lock()
		defer c.mu.Unlock()
		require.NotNil(t, c.pendingUpdate)
		assert.Equal(t, frame, *c.pendingUpdate)
		assert.NotNil(t, c.jobCancel, "handleUpdate must not touch the running job's state")
	})

	t.Run("applied immediately when idle", func(t *testing.T) {
		t.Setenv("NEXUL_RUNNER_ID", "host-1")
		body := "new-binary-bytes"
		sum := sha256.Sum256([]byte(body))
		sha := hex.EncodeToString(sum[:])
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		exe := writeExe(t, "old-binary")
		calls := stubReexec(t)
		origExeFn := runnerExecutableFn
		runnerExecutableFn = func() (string, error) { return exe, nil }
		t.Cleanup(func() { runnerExecutableFn = origExeFn })

		c := newTestClient("ws://server/ws/runner", &fakeExecutor{})
		c.cfg.Version = "v0.1.6"
		c.handleUpdate(context.Background(), Frame{Type: FrameUpdate, Version: "v0.2.0", URL: srv.URL, Sha256: sha})

		assert.Equal(t, []string{exe}, calls.Calls())
	})
}

func TestClient_jobFinished(t *testing.T) {
	t.Run("applies a deferred update and clears the job", func(t *testing.T) {
		body := "new-binary-bytes"
		sum := sha256.Sum256([]byte(body))
		sha := hex.EncodeToString(sum[:])
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		exe := writeExe(t, "old-binary")
		calls := stubReexec(t)
		origExeFn := runnerExecutableFn
		runnerExecutableFn = func() (string, error) { return exe, nil }
		t.Cleanup(func() { runnerExecutableFn = origExeFn })

		c := newTestClient("ws://server/ws/runner", &fakeExecutor{})
		_, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		frame := Frame{Type: FrameUpdate, Version: "v0.2.0", URL: srv.URL, Sha256: sha}
		c.mu.Lock()
		c.jobID = "d1"
		c.jobCancel = cancel
		c.pendingUpdate = &frame
		c.mu.Unlock()

		c.jobFinished(context.Background(), "d1")

		c.mu.Lock()
		assert.Nil(t, c.jobCancel)
		assert.Empty(t, c.jobID)
		assert.Nil(t, c.pendingUpdate)
		c.mu.Unlock()
		assert.Equal(t, []string{exe}, calls.Calls())
	})

	t.Run("ignores a stale job id", func(t *testing.T) {
		c := newTestClient("ws://server/ws/runner", &fakeExecutor{})
		_, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		c.mu.Lock()
		c.jobID = "current"
		c.jobCancel = cancel
		c.mu.Unlock()

		c.jobFinished(context.Background(), "stale")

		c.mu.Lock()
		assert.Equal(t, "current", c.jobID)
		assert.NotNil(t, c.jobCancel)
		c.mu.Unlock()
	})
}
