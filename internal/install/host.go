package install

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Commander runs a system command and returns its combined output.
type Commander interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// Paths are the host locations the installer writes outside the install directory.
type Paths struct {
	Config       string
	BinDir       string
	Unit         string
	ComposePlugs string
	SystemdProbe string
	DockerEnv    string
}

// Host is everything the installer touches on the machine, swappable in tests.
type Host struct {
	Exec        Commander
	HTTP        *http.Client
	Out         io.Writer
	In          *bufio.Reader
	Interactive bool
	// ReleaseURL is the base the release's files are downloaded from; NEXUL_RELEASE_URL overrides it for testing.
	ReleaseURL string
	// ReleaseAPI is the GitHub API root the newest release is looked up on; empty means api.github.com.
	ReleaseAPI string
	// DockerScriptURL is Docker's install script; ComposeURL is where the Compose plugin's builds are published.
	DockerScriptURL string
	ComposeURL      string
	Paths           Paths
	GOOS            string
	GOARCH          string
	Getuid          func() int
	PortFree        func(port int) bool
	LookPath        func(file string) (string, error)
	Executable      func() (string, error)
	Reexec          func(path string, args []string) error
	// HealthTimeout bounds how long the installer waits for the server to answer after starting the stack.
	HealthTimeout time.Duration
	PollInterval  time.Duration

	aptUpdated bool
}

// NewHost returns a Host wired to this machine.
func NewHost() *Host {
	releaseURL := os.Getenv("NEXUL_RELEASE_URL")
	if releaseURL == "" {
		releaseURL = "https://github.com/otal-labs/nexul/releases/download"
	}
	return &Host{
		Exec:            execCommander{},
		HTTP:            &http.Client{Timeout: 10 * time.Minute},
		Out:             os.Stdout,
		In:              bufio.NewReader(os.Stdin),
		Interactive:     isTerminal(os.Stdin),
		ReleaseURL:      strings.TrimSuffix(releaseURL, "/"),
		DockerScriptURL: "https://get.docker.com",
		ComposeURL:      "https://github.com/docker/compose/releases/latest/download",
		Paths: Paths{
			Config:       "/etc/nexul/nexul.conf",
			BinDir:       "/usr/local/bin",
			Unit:         "/etc/systemd/system/nexul-runner.service",
			ComposePlugs: "/usr/local/lib/docker/cli-plugins",
			SystemdProbe: "/run/systemd/system",
			DockerEnv:    "/.dockerenv",
		},
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		Getuid:        os.Getuid,
		PortFree:      portFree,
		LookPath:      exec.LookPath,
		Executable:    os.Executable,
		Reexec:        func(path string, args []string) error { return syscall.Exec(path, args, os.Environ()) },
		HealthTimeout: 3 * time.Minute,
		PollInterval:  2 * time.Second,
	}
}

// printf writes to the terminal; a failed terminal write has nowhere to be reported.
func (h *Host) printf(format string, a ...any) {
	_, _ = fmt.Fprintf(h.Out, format, a...)
}

func (h *Host) requireRoot() error {
	if h.GOOS != "linux" {
		return fmt.Errorf("nexul install supports Linux servers; on %s, run the single binary with `nexul serve`", h.GOOS)
	}
	if h.Getuid() != 0 {
		return errors.New("run this as root, for example with sudo")
	}
	return nil
}

// PublicIP is the address other machines most likely reach this host on: the source address of the default route.
func (h *Host) PublicIP() string {
	conn, err := net.Dial("udp", "1.1.1.1:53")
	if err != nil {
		return "localhost"
	}
	defer func() { _ = conn.Close() }() // a UDP socket that sent nothing has nothing to flush
	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "localhost"
	}
	return addr.IP.String()
}

func portFree(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = l.Close() // the probe listener accepted nothing
	return true
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

type execCommander struct{}

// Run executes name with args, returning the combined output; a failure carries the output's tail.
func (execCommander) Run(ctx context.Context, name string, args ...string) (string, error) {
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmd.Env = append(childEnv(os.Environ()), "DEBIAN_FRONTEND=noninteractive")
	err := cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err != nil {
		return out, fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, tail(out, 20))
	}
	return out, nil
}

// childEnv drops NEXUL_* variables, which docker compose would otherwise let override the install's .env.
func childEnv(environ []string) []string {
	out := make([]string, 0, len(environ))
	for _, kv := range environ {
		if !strings.HasPrefix(kv, "NEXUL_") {
			out = append(out, kv)
		}
	}
	return out
}

// tail keeps the last n lines of s, where a failing command prints its reason.
func tail(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
