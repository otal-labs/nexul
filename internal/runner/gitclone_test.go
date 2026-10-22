package runner

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestCloneRepository_AsksOnlyARunnerNewEnoughToClone(t *testing.T) {
	for _, v := range []string{"", "v0.3.40-beta", lastWithoutGitClone} {
		t.Run("version "+v, func(t *testing.T) {
			f := newT3Fixture(t)
			conn, _, err := dialRunner(t.Context(), f.srv, f.alice, "/ws/runner?version="+v)
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.CloseNow() })
			eventually(t, 2*time.Second, func() bool { return len(f.h.Runners()) == 1 })

			_, err = f.h.CloneRepository(t.Context(), "c-laptop", "acme/api", "~/Code/api", "tok")
			require.ErrorIs(t, err, ErrCannotClone)
		})
	}
}

func TestCloneRepository_ReturnsWhatTheRunnerAnswered(t *testing.T) {
	f := newT3Fixture(t)
	conn, _, err := dialRunner(t.Context(), f.srv, f.alice, "/ws/runner?version=v0.3.56-beta")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.CloseNow() })
	eventually(t, 2*time.Second, func() bool { return len(f.h.Runners()) == 1 })

	answer := func(reply Frame) (string, error) {
		type result struct {
			dir string
			err error
		}
		got := make(chan result, 1)
		go func() {
			dir, err := f.h.CloneRepository(t.Context(), "c-laptop", "acme/api", "~/Code/api", "tok")
			got <- result{dir, err}
		}()
		var req Frame
		for req.Type != FrameGitClone {
			require.NoError(t, wsjson.Read(t.Context(), conn, &req))
		}
		assert.Equal(t, Frame{Type: FrameGitClone, ID: req.ID, Repo: "acme/api", Path: "~/Code/api", GitToken: "tok"}, req)
		reply.ID = req.ID
		require.NoError(t, wsjson.Write(t.Context(), conn, reply))
		r := <-got
		return r.dir, r.err
	}

	dir, err := answer(Frame{Type: FrameGitCloneResult, Path: "/home/alice/Code/api"})
	require.NoError(t, err)
	assert.Equal(t, "/home/alice/Code/api", dir)

	_, err = answer(Frame{Type: FrameGitCloneResult, Error: "git clone acme/api failed (exit status 128): destination path already exists"})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "destination path already exists")
}

// gitCloneAnswer asks a personal runner to clone acme/api into ~/Code/api with a token and returns its answer and logs.
func gitCloneAnswer(t *testing.T) (Frame, string) {
	t.Helper()
	answer := make(chan Frame, 1)
	var logs bytes.Buffer
	stop := personalClient(t, t.TempDir(), "", &logs, func(ctx context.Context, conn *websocket.Conn) {
		_ = wsjson.Write(ctx, conn, Frame{Type: FrameGitClone, ID: "g1", Repo: "acme/api", Path: "~/Code/api", GitToken: "secret-clone-tok"})
		for {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type == FrameGitCloneResult {
				answer <- f
				return
			}
		}
	})
	select {
	case f := <-answer:
		stop()
		return f, logs.String()
	case <-time.After(30 * time.Second):
		t.Fatal("no git_clone_result within 30s")
		return Frame{}, ""
	}
}

// githubAt points the test's git at base for every https://github.com/ URL, through a home folder of its own.
func githubAt(t *testing.T, base string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	git(t, "", "config", "--global", "url."+base+".insteadOf", "https://github.com/")
	return home
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func TestPersonalRunner_ClonesIntoItsHomeWithTheTokenKeptOutOfTheCheckout(t *testing.T) {
	repos := t.TempDir()
	src := filepath.Join(repos, "acme", "api.git")
	git(t, "", "init", "--quiet", "--bare", src)
	work := t.TempDir()
	git(t, "", "clone", "--quiet", src, work)
	require.NoError(t, os.WriteFile(filepath.Join(work, "README.md"), []byte("api\n"), 0o600))
	git(t, work, "add", ".")
	git(t, work, "-c", "user.name=Alice", "-c", "user.email=alice@example.com", "commit", "--quiet", "-m", "first")
	git(t, work, "push", "--quiet", "origin", "HEAD")
	home := githubAt(t, "file://"+repos+"/")

	got, logs := gitCloneAnswer(t)

	want := filepath.Join(home, "Code", "api")
	assert.Equal(t, Frame{Type: FrameGitCloneResult, ID: "g1", Path: want}, got)
	assert.FileExists(t, filepath.Join(want, "README.md"))
	config, err := os.ReadFile(filepath.Join(want, ".git", "config"))
	require.NoError(t, err)
	assert.NotContains(t, string(config), "secret-clone-tok")
	assert.NotContains(t, string(config), base64.StdEncoding.EncodeToString([]byte("x-access-token:secret-clone-tok")))
	assert.NotContains(t, logs, "secret-clone-tok")
}

func TestPersonalRunner_SendsTheTokenToGitHubAndAnswersWhyGitFailed(t *testing.T) {
	headers := make(chan string, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers <- r.Header.Get("Authorization")
		http.Error(w, "repository not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	githubAt(t, srv.URL+"/")

	got, logs := gitCloneAnswer(t)

	assert.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("x-access-token:secret-clone-tok")), <-headers)
	assert.Contains(t, got.Error, "git clone acme/api failed")
	assert.Empty(t, got.Path)
	assert.NotContains(t, got.Error+logs, "secret-clone-tok")
}
