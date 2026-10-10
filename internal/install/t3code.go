package install

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// t3DesktopWinget is T3 Code's desktop app, what Windows gets since T3 Code has no Windows service.
const t3DesktopWinget = "T3Tools.T3Code"

// t3Found is what the install finds of T3 Code in the person's home.
type t3Found struct {
	port      int
	answering bool
	// serviceManaged is set when T3 Code's own background service runs the server that answers.
	serviceManaged bool
	desktop        bool
	// command is T3 Code's t3 command line, when it is installed.
	command string
}

// installT3 reuses a T3 Code it finds and otherwise runs T3 Code's own installer and service as the person (ADR 0146).
func (h *Host) installT3(ctx context.Context, p person, o ComputerOptions) error {
	if o.NoT3 {
		return h.step("T3 Code", func() (string, error) { return "skipped (--no-t3)", nil })
	}
	port := o.T3Port
	if port == 0 {
		port = h.T3Port
	}
	found := h.findT3(ctx, p, port)
	if found.answering {
		return h.step("T3 Code", func() (string, error) { return h.keepT3(ctx, p, found) })
	}
	if found.desktop {
		return h.step("T3 Code", func() (string, error) {
			return "the desktop app is installed but closed: Open T3 Code, and Nexul pairs it once it runs", nil
		})
	}
	if found.command != "" {
		return h.step("T3 Code", func() (string, error) { return h.startT3(ctx, p, found.command, port) })
	}
	if h.GOOS == "windows" {
		h.printf("  Installing T3 Code's desktop app, which Nexul pairs with (winget install %s). Pass --no-t3 to skip this.\n", t3DesktopWinget)
		return h.step("T3 Code", func() (string, error) { return h.installT3Desktop(ctx) })
	}
	if h.GOOS == "darwin" && port != h.T3Port {
		return fmt.Errorf("T3 Code's macOS service always starts on port %d; install T3 Code yourself and run t3 serve --port %d, or leave out --t3-port", h.T3Port, port)
	}
	h.printf("  Installing T3 Code, which Nexul pairs with: its command line from %s into ~/.local/bin, run as %s's background service on port %d. Pass --no-t3 to skip this.\n",
		h.T3InstallerURL, p.name, port)
	return h.step("T3 Code", func() (string, error) { return h.installT3CommandLine(ctx, p, port) })
}

// findT3 looks in the runner's order: the runtime file's port (else port) answering, the desktop app, the t3 command.
func (h *Host) findT3(ctx context.Context, p person, port int) t3Found {
	found := t3Found{port: port}
	t3Home := filepath.Join(p.home, ".t3")
	var runtime struct {
		Port           int  `json:"port"`
		ServiceManaged bool `json:"serviceManaged"`
	}
	if data, err := os.ReadFile(filepath.Join(t3Home, "userdata", "server-runtime.json")); err == nil && json.Unmarshal(data, &runtime) == nil && runtime.Port > 0 {
		found.port, found.serviceManaged = runtime.Port, runtime.ServiceManaged
	}
	found.answering = h.t3Answers(ctx, found.port)
	launcher := "t3"
	if h.GOOS == "windows" {
		launcher = "t3.cmd"
	}
	found.desktop = isExecutable(filepath.Join(t3Home, "bin", launcher))
	if path, err := h.LookPath("t3"); err == nil {
		found.command = path
	}
	if cli := filepath.Join(p.home, ".local", "bin", "t3"); found.command == "" && isExecutable(cli) {
		found.command = cli
	}
	return found
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

// keepT3 reuses a T3 Code that answers; T3 Code's Linux service is a user service, which needs lingering past logout.
func (h *Host) keepT3(ctx context.Context, p person, found t3Found) (string, error) {
	detail := fmt.Sprintf("running on port %d", found.port)
	if h.GOOS != "linux" || !found.serviceManaged {
		return detail, nil
	}
	if _, err := h.Exec.Run(ctx, "loginctl", "enable-linger", p.name); err != nil {
		return "", err
	}
	return detail + ", its service kept running after you log out", nil
}

// installT3CommandLine runs T3 Code's own installer as the person, then starts T3 Code's background service.
func (h *Host) installT3CommandLine(ctx context.Context, p person, port int) (string, error) {
	if h.GOOS == "linux" {
		if err := h.ensureLibatomic(ctx); err != nil {
			return "", err
		}
	}
	dir, err := os.MkdirTemp("", "nexul-t3-")
	if err != nil {
		return "", fmt.Errorf("make a folder for T3 Code's installer: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()     // a leftover copy of a public script harms nothing
	if err := os.Chmod(dir, 0o755); err != nil { // the person, not root, reads the installer
		return "", fmt.Errorf("open %s to %s: %w", dir, p.name, err)
	}
	script := filepath.Join(dir, "install.sh")
	if err := h.download(ctx, h.T3InstallerURL, script, 0o644); err != nil {
		return "", err
	}
	if _, err := h.asPerson(ctx, p, "sh", script); err != nil {
		return "", fmt.Errorf("T3 Code's installer failed; install T3 Code yourself from https://t3.codes, or run the command again with --no-t3: %w", err)
	}
	detail, err := h.startT3(ctx, p, filepath.Join(p.home, ".local", "bin", "t3"), port)
	if err != nil {
		return "", err
	}
	return "installed, " + detail, nil
}

// startT3 starts T3 Code's own background service for the person and waits until it answers.
func (h *Host) startT3(ctx context.Context, p person, t3 string, port int) (string, error) {
	if h.GOOS == "windows" {
		return fmt.Sprintf("installed at %s but not running: start it with t3 serve --port %d, or open the desktop app", t3, port), nil
	}
	if h.GOOS == "linux" {
		if err := h.lingerFor(ctx, p, port); err != nil {
			return "", err
		}
	}
	if _, err := h.asPerson(ctx, p, t3, "service", "install"); err != nil {
		return "", fmt.Errorf("T3 Code's background service did not start; run %s serve --port %d in a terminal you keep open, or run the command again with --no-t3: %w", t3, port, err)
	}
	if err := h.waitForT3(ctx, port); err != nil {
		return "", fmt.Errorf("%w; run %s service status to see why", err, t3)
	}
	if h.GOOS == "darwin" { // T3 Code's macOS service is a LaunchAgent, which launchd runs only in a login at the screen
		return fmt.Sprintf("running on port %d as %s's background service while %s is logged in", port, p.name, p.name), nil
	}
	return fmt.Sprintf("running on port %d as %s's background service, also after you log out", port, p.name), nil
}

// lingerFor starts the person's user manager, T3 Code's service's home, now, at boot and past logout.
func (h *Host) lingerFor(ctx context.Context, p person, port int) error {
	if _, err := h.Exec.Run(ctx, "loginctl", "enable-linger", p.name); err != nil {
		return err
	}
	if _, err := h.Exec.Run(ctx, "systemctl", "start", "user@"+strconv.Itoa(p.uid)+".service"); err != nil {
		return err
	}
	if port == h.T3Port {
		return nil
	}
	dropin := filepath.Join(p.home, ".config", "systemd", "user", "t3code.service.d")
	_, err := h.asPerson(ctx, p, "sh", "-c", `mkdir -p "$1" && printf '[Service]\nEnvironment=T3CODE_PORT=%s\n' "$2" >"$1/nexul-port.conf"`, "sh", dropin, strconv.Itoa(port))
	return err
}

// ensureLibatomic adds the C library T3 Code's Linux build links, which minimal server images leave out.
func (h *Host) ensureLibatomic(ctx context.Context) error {
	if out, err := h.Exec.Run(ctx, "ldconfig", "-p"); err == nil && strings.Contains(out, "libatomic.so.1") {
		return nil
	}
	pm, ok := h.packageManager()
	if !ok {
		return nil
	}
	pkg, ok := map[string]string{"apt-get": "libatomic1", "zypper": "libatomic1", "dnf": "libatomic", "yum": "libatomic", "apk": "libatomic"}[pm.name]
	if !ok {
		return nil
	}
	return h.installPackage(ctx, pm, pkg)
}

func (h *Host) installT3Desktop(ctx context.Context) (string, error) {
	_, err := h.Exec.Run(ctx, "winget", "install", "--id", t3DesktopWinget, "--exact", "--silent", "--accept-source-agreements", "--accept-package-agreements")
	if err != nil {
		return "", err
	}
	return "installed the desktop app: Open T3 Code once, and Nexul pairs it once it runs", nil
}

// asPerson runs a command as the person, with their home, never as root; on Linux it reaches their user manager.
func (h *Host) asPerson(ctx context.Context, p person, name string, args ...string) (string, error) {
	if h.GOOS == "windows" {
		return h.Exec.Run(ctx, name, args...)
	}
	env := []string{"env", "HOME=" + p.home}
	if h.GOOS == "linux" {
		env = append(env, "XDG_RUNTIME_DIR=/run/user/"+strconv.Itoa(p.uid))
	}
	cmd, cmdArgs := h.asPersonArgs(p, append(append(env, name), args...)...)
	return h.Exec.Run(ctx, cmd, cmdArgs...)
}

// t3Answers reports whether T3 Code's unauthenticated descriptor answers on the loopback port.
func (h *Host) t3Answers(ctx context.Context, port int) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/.well-known/t3/environment", port), nil)
	if err != nil {
		return false
	}
	res, err := h.HTTP.Do(req)
	if err != nil {
		return false
	}
	_ = res.Body.Close() // only the status matters
	return res.StatusCode == http.StatusOK
}

// waitForT3 gives a T3 Code service that just started the health timeout to answer.
func (h *Host) waitForT3(ctx context.Context, port int) error {
	ctx, cancel := context.WithTimeout(ctx, h.HealthTimeout)
	defer cancel()
	for !h.t3Answers(ctx, port) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("T3 Code's service is installed but nothing answers on port %d", port)
		case <-time.After(h.PollInterval):
		}
	}
	return nil
}
