package install

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// serviceManager puts a unit under the OS service manager: systemd on Linux, launchd on macOS, the SCM on Windows.
type serviceManager interface {
	// install writes the unit's service definition, enables it at boot and (re)starts it.
	install(ctx context.Context, u Unit) error
	// remove stops and disables the unit and deletes its definition; a unit already gone is not an error.
	remove(ctx context.Context, u Unit) error
	// state is the service manager's word for the unit's state, e.g. active or running.
	state(ctx context.Context, u Unit) string
}

func (h *Host) services() serviceManager {
	if h.GOOS == "darwin" {
		return launchd{h}
	}
	if h.GOOS == "windows" {
		return scm{h}
	}
	return systemd{h}
}

type systemd struct{ h *Host }

func (s systemd) path(u Unit) string { return filepath.Join(s.h.Paths.Services, u.Name+".service") }

func (s systemd) install(ctx context.Context, u Unit) error {
	if _, err := os.Stat(s.h.Paths.SystemdProbe); err != nil {
		return errors.New("this host is not running systemd, which the Nexul services run under")
	}
	if err := os.WriteFile(s.path(u), []byte(systemdUnit(u)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", s.path(u), err)
	}
	for _, args := range [][]string{{"daemon-reload"}, {"enable", u.Name + ".service"}, {"restart", u.Name + ".service"}} {
		if _, err := s.h.Exec.Run(ctx, "systemctl", args...); err != nil {
			return err
		}
	}
	return nil
}

func (s systemd) remove(ctx context.Context, u Unit) error {
	// disable --now fails for a unit already gone; the removals below are what matter.
	_, _ = s.h.Exec.Run(ctx, "systemctl", "disable", "--now", u.Name+".service")
	if err := os.Remove(s.path(u)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", s.path(u), err)
	}
	_, err := s.h.Exec.Run(ctx, "systemctl", "daemon-reload")
	return err
}

func (s systemd) state(ctx context.Context, u Unit) string {
	out, _ := s.h.Exec.Run(ctx, "systemctl", "is-active", u.Name+".service") // exits non-zero for every state but active; the output is the answer
	return firstLine(out, "unknown")
}

// systemdUnit is the unit file: the env file carries the settings and secrets, so the world-readable unit holds none.
func systemdUnit(u Unit) string {
	deps := "network-online.target"
	if u.Kind == kindRunner {
		deps += " docker.service"
	}
	lines := []string{
		"[Unit]",
		"Description=" + u.describe(),
		"Wants=" + deps,
		"After=" + deps,
		"",
		"[Service]",
	}
	if u.User != "" {
		lines = append(lines, "User="+u.User, "Group="+u.User)
	}
	lines = append(lines, "EnvironmentFile="+u.envFile(), "WorkingDirectory="+u.WorkDir, "ExecStart="+u.Exec)
	if u.Kind == kindServer {
		lines = append(lines, "AmbientCapabilities=CAP_NET_BIND_SERVICE")
	}
	if u.Kind == kindLogs {
		lines = append(lines, "MemoryMax=1G")
	}
	lines = append(lines, "Restart=always", "RestartSec=5", "", "[Install]", "WantedBy=multi-user.target", "")
	return strings.Join(lines, "\n")
}

type launchd struct{ h *Host }

func (l launchd) label(u Unit) string   { return "io.nexul." + u.Name }
func (l launchd) path(u Unit) string    { return filepath.Join(l.h.Paths.Services, l.label(u)+".plist") }
func (l launchd) domain() string        { return fmt.Sprintf("gui/%d", l.h.Getuid()) }
func (l launchd) target(u Unit) string  { return l.domain() + "/" + l.label(u) }
func (l launchd) logPath(u Unit) string { return filepath.Join(l.h.Paths.Logs, u.Name+".log") }

func (l launchd) install(ctx context.Context, u Unit) error {
	for _, dir := range []string{l.h.Paths.Services, l.h.Paths.Logs} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(l.path(u), []byte(launchdPlist(l.label(u), u, l.logPath(u))), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", l.path(u), err)
	}
	// bootout fails when the agent is not loaded yet, which is the first install.
	_, _ = l.h.Exec.Run(ctx, "launchctl", "bootout", l.target(u))
	if _, err := l.h.Exec.Run(ctx, "launchctl", "enable", l.target(u)); err != nil {
		return err
	}
	_, err := l.h.Exec.Run(ctx, "launchctl", "bootstrap", l.domain(), l.path(u))
	return err
}

func (l launchd) remove(ctx context.Context, u Unit) error {
	_, _ = l.h.Exec.Run(ctx, "launchctl", "bootout", l.target(u)) // fails for an agent already unloaded
	if err := os.Remove(l.path(u)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", l.path(u), err)
	}
	return nil
}

func (l launchd) state(ctx context.Context, u Unit) string {
	out, err := l.h.Exec.Run(ctx, "launchctl", "print", l.target(u))
	if err != nil {
		return "not loaded"
	}
	for _, line := range strings.Split(out, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), " = "); ok && k == "state" {
			return v
		}
	}
	return "unknown"
}

// launchdPath is the PATH a LaunchAgent gets, which otherwise lacks Homebrew's and Docker Desktop's docker.
const launchdPath = "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

// launchdPlist inlines the env because launchd has no env file; the plist is 0600 for that reason.
func launchdPlist(label string, u Unit, logPath string) string {
	env := map[string]string{"PATH": launchdPath}
	for k, v := range u.Env {
		env[k] = v
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)
	fmt.Fprintf(&b, "  <key>Label</key>\n  <string>%s</string>\n", xmlText(label))
	fmt.Fprintf(&b, "  <key>ProgramArguments</key>\n  <array>\n    <string>%s</string>\n  </array>\n", xmlText(u.Exec))
	b.WriteString("  <key>EnvironmentVariables</key>\n  <dict>\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "    <key>%s</key>\n    <string>%s</string>\n", xmlText(k), xmlText(env[k]))
	}
	b.WriteString("  </dict>\n")
	fmt.Fprintf(&b, "  <key>WorkingDirectory</key>\n  <string>%s</string>\n", xmlText(u.WorkDir))
	b.WriteString("  <key>RunAtLoad</key>\n  <true/>\n  <key>KeepAlive</key>\n  <true/>\n")
	fmt.Fprintf(&b, "  <key>StandardOutPath</key>\n  <string>%s</string>\n", xmlText(logPath))
	fmt.Fprintf(&b, "  <key>StandardErrorPath</key>\n  <string>%s</string>\n", xmlText(logPath))
	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

func xmlText(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s)) // a strings.Builder never fails a write
	return b.String()
}

// scm registers units with the Windows Service Control Manager; each runs `nexul service-host <unit>`, which
// supervises the unit's binary with its env.
type scm struct{ h *Host }

func (s scm) install(ctx context.Context, u Unit) error {
	bin := fmt.Sprintf(`"%s" service-host %s`, s.h.ctlPath(), u.Name)
	verb := "create"
	if _, err := s.h.Exec.Run(ctx, "sc.exe", "query", u.Name); err == nil {
		verb = "config"
		s.stop(ctx, u)
	}
	calls := [][]string{
		{verb, u.Name, "binPath=", bin, "start=", "auto", "DisplayName=", u.describe()},
		{"failure", u.Name, "reset=", "86400", "actions=", "restart/5000/restart/5000/restart/5000"},
		// The service host exits non-zero when its unit dies, and this makes that count as a failure to restart on.
		{"failureflag", u.Name, "1"},
		{"start", u.Name},
	}
	for _, args := range calls {
		if _, err := s.h.Exec.Run(ctx, "sc.exe", args...); err != nil {
			return err
		}
	}
	return nil
}

func (s scm) remove(ctx context.Context, u Unit) error {
	if _, err := s.h.Exec.Run(ctx, "sc.exe", "query", u.Name); err != nil {
		return nil
	}
	s.stop(ctx, u)
	_, err := s.h.Exec.Run(ctx, "sc.exe", "delete", u.Name)
	return err
}

// stop asks the service to stop and waits until it has, because sc.exe returns while it is still stopping.
func (s scm) stop(ctx context.Context, u Unit) {
	_, _ = s.h.Exec.Run(ctx, "sc.exe", "stop", u.Name) // fails when it is already stopped
	ctx, cancel := context.WithTimeout(ctx, s.h.HealthTimeout)
	defer cancel()
	for s.state(ctx, u) != "stopped" {
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.h.PollInterval):
		}
	}
}

func (s scm) state(ctx context.Context, u Unit) string {
	out, err := s.h.Exec.Run(ctx, "sc.exe", "query", u.Name)
	if err != nil {
		return "not installed"
	}
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(line, ":")
		fields := strings.Fields(v)
		if ok && strings.TrimSpace(k) == "STATE" && len(fields) >= 2 {
			return strings.ToLower(fields[1])
		}
	}
	return "unknown"
}

func firstLine(s, fallback string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	if line == "" {
		return fallback
	}
	return line
}
