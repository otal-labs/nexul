package install

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/version"
)

// fakeExec records every command and answers from canned results keyed by the command line's prefix.
type fakeExec struct {
	mu      sync.Mutex
	calls   []string
	answers map[string]fakeAnswer
	// onRun lets a test change the world when a command runs, e.g. make `docker` appear after the install script.
	onRun func(line string)
}

type fakeAnswer struct {
	out string
	err error
}

func (f *fakeExec) Run(_ context.Context, name string, args ...string) (string, error) {
	line := strings.TrimSpace(name + " " + strings.Join(args, " "))
	f.mu.Lock()
	f.calls = append(f.calls, line)
	onRun := f.onRun
	f.mu.Unlock()
	if onRun != nil {
		onRun(line)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	answer, longest := fakeAnswer{}, -1
	for prefix, a := range f.answers {
		if strings.HasPrefix(line, prefix) && len(prefix) > longest {
			answer, longest = a, len(prefix)
		}
	}
	return answer.out, answer.err
}

func (f *fakeExec) ran(prefix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func (f *fakeExec) set(prefix, out string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.answers == nil {
		f.answers = map[string]fakeAnswer{}
	}
	f.answers[prefix] = fakeAnswer{out: out, err: err}
}

// fakeRelease serves a fake GitHub release: /<tag>/<file> from files, checksums.txt computed from them.
type fakeRelease struct {
	files    map[string][]byte
	checksum map[string]string
}

func (r *fakeRelease) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		name := filepath.Base(req.URL.Path)
		if name == "checksums.txt" {
			var b strings.Builder
			for n, data := range r.files {
				sum := sha256.Sum256(data)
				if override, ok := r.checksum[n]; ok {
					fmt.Fprintf(&b, "%s  %s\n", override, n)
					continue
				}
				fmt.Fprintf(&b, "%s  %s\n", hex.EncodeToString(sum[:]), n)
			}
			_, _ = w.Write([]byte(b.String()))
			return
		}
		data, ok := r.files[name]
		if !ok {
			http.NotFound(w, req)
			return
		}
		_, _ = w.Write(data)
	})
}

// testHost is a Host rooted in a temp dir, with a fake exec, a fake release server, and an out buffer.
type testHost struct {
	*Host
	exec    *fakeExec
	out     *bytes.Buffer
	root    string
	release *fakeRelease
	web     *httptest.Server
	reexec  []string
}

func newTestHost(t *testing.T) *testHost {
	t.Helper()
	root := t.TempDir()
	rel := &fakeRelease{files: map[string][]byte{
		"nexul-runner-linux-amd64": []byte("runner-binary"),
		"nexul-linux-amd64":        []byte("nexul-binary"),
	}}
	srv := httptest.NewServer(rel.handler())
	t.Cleanup(srv.Close)
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	t.Cleanup(web.Close)

	systemd := filepath.Join(root, "run-systemd")
	require.NoError(t, os.MkdirAll(systemd, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "bin"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "systemd"), 0o755))
	self := filepath.Join(root, "downloads", "nexul")
	require.NoError(t, os.MkdirAll(filepath.Dir(self), 0o755))
	require.NoError(t, os.WriteFile(self, []byte("this-binary"), 0o755))

	fe := &fakeExec{}
	out := &bytes.Buffer{}
	th := &testHost{exec: fe, out: out, root: root, release: rel, web: web}
	th.Host = &Host{
		Exec:       fe,
		HTTP:       srv.Client(),
		Out:        out,
		In:         bufio.NewReader(strings.NewReader("")),
		ReleaseURL: srv.URL,
		Paths: Paths{
			Config:       filepath.Join(root, "etc", "nexul.conf"),
			BinDir:       filepath.Join(root, "bin"),
			Unit:         filepath.Join(root, "systemd", "nexul-runner.service"),
			ComposePlugs: filepath.Join(root, "cli-plugins"),
			SystemdProbe: systemd,
		},
		GOOS:          "linux",
		GOARCH:        "amd64",
		Getuid:        func() int { return 0 },
		PortFree:      func(int) bool { return true },
		LookPath:      func(file string) (string, error) { return "/usr/bin/" + file, nil },
		Executable:    func() (string, error) { return self, nil },
		HealthTimeout: time.Second,
		PollInterval:  time.Millisecond,
	}
	th.Reexec = func(path string, args []string) error {
		th.reexec = args
		return nil
	}
	th.set("docker version", "29.1.3", nil)
	th.set("docker compose version", "2.40.0", nil)
	return th
}

func (th *testHost) set(prefix, out string, err error) { th.exec.set(prefix, out, err) }

// webPort is the fake web server's port; options using it pass the health check.
func (th *testHost) webPort(t *testing.T) int {
	t.Helper()
	var port int
	_, err := fmt.Sscanf(th.web.URL[strings.LastIndex(th.web.URL, ":")+1:], "%d", &port)
	require.NoError(t, err)
	return port
}

// withVersion stamps version.Version for the rest of the test.
func withVersion(t *testing.T, v string) {
	t.Helper()
	orig := version.Version
	version.Version = v
	t.Cleanup(func() { version.Version = orig })
}

// missing makes LookPath fail for the named commands.
func (th *testHost) missing(names ...string) {
	th.LookPath = func(file string) (string, error) {
		for _, n := range names {
			if n == file {
				return "", errors.New("not found")
			}
		}
		return "/usr/bin/" + file, nil
	}
}
