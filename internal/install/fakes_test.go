package install

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// RunAttached records the command like Run; the attached terminal makes no difference to a fake.
func (f *fakeExec) RunAttached(ctx context.Context, name string, args ...string) error {
	_, err := f.Run(ctx, name, args...)
	return err
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

// fakeInstance is the instance's enroll and self-remove endpoints plus the health check on /.
type fakeInstance struct {
	mu       sync.Mutex
	requests []instanceRequest
	// status overrides the answer for a path; unset paths answer 200.
	status map[string]int
}

type instanceRequest struct {
	path, auth string
	body       map[string]string
}

func (f *fakeInstance) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			return
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body) // a body-less request records an empty body
		f.mu.Lock()
		f.requests = append(f.requests, instanceRequest{path: r.URL.Path, auth: r.Header.Get("Authorization"), body: body})
		status, ok := f.status[r.URL.Path]
		f.mu.Unlock()
		if ok {
			w.WriteHeader(status)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/enroll") {
			_, _ = fmt.Fprintf(w, `{"id":"id-%s","name":%q,"credential":"cred-%s"}`, body["name"], body["name"], body["name"])
		}
	})
}

func (f *fakeInstance) calls(path string) []instanceRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []instanceRequest
	for _, r := range f.requests {
		if r.path == path {
			out = append(out, r)
		}
	}
	return out
}

// testHost is a Host rooted in a temp dir, with a fake exec, fake release, OpenObserve and instance servers, and an
// out buffer.
type testHost struct {
	*Host
	exec     *fakeExec
	out      *bytes.Buffer
	root     string
	release  *fakeRelease
	instance *fakeInstance
	web      *httptest.Server
	reexec   []string
	detached [][]string
}

var testTargets = []string{"linux-amd64", "darwin-arm64", "windows-amd64"}

func newTestHost(t *testing.T) *testHost {
	t.Helper()
	root := t.TempDir()
	rel := &fakeRelease{files: map[string][]byte{}}
	for _, target := range testTargets {
		ext := ""
		if strings.HasPrefix(target, "windows") {
			ext = ".exe"
		}
		for _, bin := range []string{"nexul", "nexul-server", "nexul-runner", "nexul-automations"} {
			rel.files[bin+"-"+target+ext] = []byte(bin + "-binary")
		}
	}
	srv := httptest.NewServer(rel.handler())
	t.Cleanup(srv.Close)
	inst := &fakeInstance{status: map[string]int{}}
	web := httptest.NewServer(inst.handler())
	t.Cleanup(web.Close)
	oo := newFakeOpenObserve(t, "v1.0.4")

	systemd := filepath.Join(root, "run-systemd")
	for _, dir := range []string{systemd, filepath.Join(root, "bin"), filepath.Join(root, "systemd"), filepath.Join(root, "sudoers.d")} {
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	self := filepath.Join(root, "downloads", "nexul")
	require.NoError(t, os.MkdirAll(filepath.Dir(self), 0o755))
	require.NoError(t, os.WriteFile(self, []byte("this-binary"), 0o755))

	fe := &fakeExec{}
	out := &bytes.Buffer{}
	th := &testHost{exec: fe, out: out, root: root, release: rel, instance: inst, web: web}
	nextPort := 15080
	th.Host = &Host{
		Exec:       fe,
		HTTP:       srv.Client(),
		Out:        out,
		In:         bufio.NewReader(strings.NewReader("")),
		ReleaseURL: srv.URL,
		Paths: Paths{
			Config:       filepath.Join(root, "etc", "nexul.conf"),
			BinDir:       filepath.Join(root, "bin"),
			UnitRoot:     filepath.Join(root, "opt"),
			Services:     filepath.Join(root, "systemd"),
			Sudoers:      filepath.Join(root, "sudoers.d"),
			Logs:         filepath.Join(root, "home", "Library", "Logs", "nexul"),
			ComposePlugs: filepath.Join(root, "cli-plugins"),
			SystemdProbe: systemd,
			UserPlugins:  filepath.Join(root, "home", ".docker", "cli-plugins"),
			DockerApp:    filepath.Join(root, "Applications", "Docker.app"),
		},
		Home:        filepath.Join(root, "home"),
		PrependPath: func(string) {},
		GOOS:        "linux",
		GOARCH:      "amd64",
		Getuid:      func() int { return 0 },
		Hostname:    func() (string, error) { return "box-1", nil },
		PortFree:    func(int) bool { return true },
		LookPath:    func(file string) (string, error) { return "/usr/bin/" + file, nil },
		Executable:  func() (string, error) { return self, nil },
		LocalPort: func() (int, error) {
			nextPort++
			return nextPort - 1, nil
		},
		OpenObserve:   oo,
		HealthTimeout: time.Second,
		PollInterval:  time.Millisecond,
	}
	th.Reexec = func(path string, args []string) error {
		th.reexec = args
		return nil
	}
	th.StartDetached = func(name string, args []string, logPath string) error {
		th.detached = append(th.detached, append([]string{name, logPath}, args...))
		return nil
	}
	th.set("docker version", "29.1.3", nil)
	th.set("docker compose version", "2.40.0", nil)
	th.set("sc.exe query", "", errors.New("does not exist"))
	return th
}

// newFakeOpenObserve serves an archive per test target holding a fake openobserve binary, pinned by its real sum.
func newFakeOpenObserve(t *testing.T, v string) OpenObserve {
	t.Helper()
	archives := map[string][]byte{}
	sums := map[string]string{}
	for _, target := range []string{"linux-amd64-musl", "darwin-arm64", "windows-amd64"} {
		name := fmt.Sprintf("openobserve-%s-%s.tar.gz", v, target)
		data := tarGz(t, "openobserve", []byte("openobserve-"+v))
		if strings.HasPrefix(target, "windows") {
			name = fmt.Sprintf("openobserve-%s-%s.zip", v, target)
			data = zipped(t, "openobserve.exe", []byte("openobserve-"+v))
		}
		archives["/"+v+"/"+name] = data
		sum := sha256.Sum256(data)
		sums[target] = hex.EncodeToString(sum[:])
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := archives[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return OpenObserve{URL: srv.URL, Version: v, SHA256: sums}
}

func tarGz(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "LICENSE", Mode: 0o644, Size: 2, Typeflag: tar.TypeReg}))
	_, err := tw.Write([]byte("no"))
	require.NoError(t, err)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data)), Typeflag: tar.TypeReg}))
	_, err = tw.Write(data)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func zipped(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	require.NoError(t, err)
	_, err = w.Write(data)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func (th *testHost) set(prefix, out string, err error) { th.exec.set(prefix, out, err) }

// webPort is the fake instance's port; options using it pass the health check and reach its enroll endpoints.
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

// bootedServer makes the fake server write the bundled hosts' enrollment codes, as a real one does at boot.
func (th *testHost) bootedServer(t *testing.T, dir string) {
	t.Helper()
	enroll := filepath.Join(dir, "data", "enroll")
	require.NoError(t, os.MkdirAll(enroll, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(enroll, "runner-instance"), []byte("nxe_runner\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(enroll, "automations-instance"), []byte("nxe_automations\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(enroll, "setup"), []byte("nxs_setup\n"), 0o600))
}

// readFile is os.ReadFile for tests, as a string.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

// callsTo lists the recorded commands starting with prefix, in order.
func (f *fakeExec) callsTo(prefix string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}
