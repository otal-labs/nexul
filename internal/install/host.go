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
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Commander runs system commands.
type Commander interface {
	// Run returns the command's combined output.
	Run(ctx context.Context, name string, args ...string) (string, error)
	// RunAttached gives the command this terminal, for installers that ask the user something (a password).
	RunAttached(ctx context.Context, name string, args ...string) error
}

// Paths are the host locations the installer writes outside the install directory.
type Paths struct {
	Config       string
	BinDir       string
	Unit         string
	ComposePlugs string
	SystemdProbe string
	// UserPlugins is the per-user Docker CLI plugins directory, where macOS links Homebrew's compose plugin.
	UserPlugins string
	// DockerApp is Docker Desktop's app bundle on macOS, started when it is installed but not running.
	DockerApp string
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
	// BrewInstallURL is Homebrew's install script, run on macOS when Docker and Homebrew are both missing.
	BrewInstallURL string
	// Home is the user's home directory; macOS and Windows installs live under it.
	Home string
	// PrependPath puts a directory in front of this process's PATH, for tools installed while it runs.
	PrependPath func(dir string)
	GOOS        string
	GOARCH      string
	Getuid      func() int
	PortFree    func(port int) bool
	LookPath    func(file string) (string, error)
	Executable  func() (string, error)
	Reexec      func(path string, args []string) error
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
	home, _ := os.UserHomeDir() // empty only on a host without a home, where the Linux defaults below apply anyway
	return &Host{
		Exec:            execCommander{},
		HTTP:            &http.Client{Timeout: 10 * time.Minute},
		Out:             os.Stdout,
		In:              bufio.NewReader(os.Stdin),
		Interactive:     isTerminal(os.Stdin),
		ReleaseURL:      strings.TrimSuffix(releaseURL, "/"),
		DockerScriptURL: "https://get.docker.com",
		ComposeURL:      "https://github.com/docker/compose/releases/latest/download",
		BrewInstallURL:  "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh",
		Paths:           pathsFor(runtime.GOOS, home),
		Home:            home,
		PrependPath: func(dir string) {
			_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH")) // Setenv fails only on an invalid key
		},
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		Getuid:        os.Getuid,
		PortFree:      portFree,
		LookPath:      exec.LookPath,
		Executable:    os.Executable,
		Reexec:        reexec,
		HealthTimeout: 3 * time.Minute,
		PollInterval:  2 * time.Second,
	}
}

// pathsFor places the config and the nexul command where each OS expects them without extra privileges.
func pathsFor(goos, home string) Paths {
	p := Paths{
		Config:       "/etc/nexul/nexul.conf",
		BinDir:       "/usr/local/bin",
		Unit:         "/etc/systemd/system/nexul-runner.service",
		ComposePlugs: "/usr/local/lib/docker/cli-plugins",
		SystemdProbe: "/run/systemd/system",
		UserPlugins:  filepath.Join(home, ".docker", "cli-plugins"),
		DockerApp:    "/Applications/Docker.app",
	}
	if goos == "darwin" {
		p.Config = filepath.Join(home, "Library", "Application Support", "nexul", "nexul.conf")
	}
	if goos == "windows" {
		p.Config = filepath.Join(home, "AppData", "Roaming", "nexul", "nexul.conf")
		p.BinDir = filepath.Join(home, "AppData", "Local", "Programs", "Nexul")
	}
	return p
}

// desktop reports a macOS or Windows host, where Docker runs in a VM and there is no systemd.
func (h *Host) desktop() bool {
	return h.GOOS == "darwin" || h.GOOS == "windows"
}

// defaultDir is /data/nexul on a Linux server and a nexul folder in the home directory elsewhere, because Docker
// on macOS only shares the home directory into its VM.
func (h *Host) defaultDir() string {
	if h.desktop() {
		return filepath.Join(h.Home, "nexul")
	}
	return DefaultDir
}

// printf writes to the terminal; a failed terminal write has nowhere to be reported.
func (h *Host) printf(format string, a ...any) {
	_, _ = fmt.Fprintf(h.Out, format, a...)
}

// checkUser refuses a user the platform's tools cannot work as: Linux needs root for Docker and systemd, and macOS
// must not be root because Homebrew refuses to run as root.
func (h *Host) checkUser() error {
	if h.GOOS == "darwin" && h.Getuid() == 0 {
		return errors.New("run this as your own user, not with sudo; it asks for your password when it needs it")
	}
	if h.desktop() {
		return nil
	}
	if h.GOOS != "linux" {
		return fmt.Errorf("nexul install supports Linux, macOS and Windows, not %s; run the binary with `nexul serve`", h.GOOS)
	}
	if h.Getuid() != 0 {
		return errors.New("run this as root, for example with sudo")
	}
	return nil
}

// PublicIP is the address other machines most likely reach this host on: the source address of the default route.
// A desktop install is for the person at the keyboard, so it is localhost there.
func (h *Host) PublicIP() string {
	if h.desktop() {
		return "localhost"
	}
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

// RunAttached runs name with this process's terminal, so the command can prompt.
func (execCommander) RunAttached(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = childEnv(os.Environ())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// tail keeps the last n lines of s, where a failing command prints its reason.
func tail(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
