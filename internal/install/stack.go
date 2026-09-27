package install

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// installed is an existing install: the directory recorded in the config file and the .env inside it.
type installed struct {
	Dir string
	Env map[string]string
}

func (p *installed) intEnv(key string, fallback int) int {
	if p == nil {
		return fallback
	}
	n, err := strconv.Atoi(p.Env[key])
	if err != nil {
		return fallback
	}
	return n
}

// loadInstall reads the config file the last install wrote; nil, nil when this host has none.
func (h *Host) loadInstall() (*installed, error) {
	conf, err := readEnvFile(h.Paths.Config)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	dir := conf["NEXUL_DIR"]
	if dir == "" {
		return nil, fmt.Errorf("%s has no NEXUL_DIR", h.Paths.Config)
	}
	env, err := readEnvFile(filepath.Join(dir, ".env"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return &installed{Dir: dir, Env: env}, nil
}

// installAt is the install in dir: prev when it lives there, else whatever .env the directory already holds.
func installAt(dir string, prev *installed) (*installed, error) {
	if prev != nil && prev.Dir == dir {
		return prev, nil
	}
	env, err := readEnvFile(filepath.Join(dir, ".env"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &installed{Dir: dir, Env: env}, nil
}

// writeSettings creates the install directory's folders and writes its .env, keeping every secret and port already
// in it, and records the directory in the config file. On Linux the data and logs folders go to the nexul user the
// server and OpenObserve run as.
func (h *Host) writeSettings(ctx context.Context, o Options, prev *installed, tag string) (map[string]string, error) {
	if err := os.MkdirAll(o.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("create %s: %w", o.Dir, err)
	}
	for _, sub := range []string{"data", "logs", "stacks"} {
		if err := os.MkdirAll(filepath.Join(o.Dir, sub), 0o750); err != nil {
			return nil, fmt.Errorf("create %s: %w", sub, err)
		}
	}
	env := map[string]string{}
	if prev != nil && prev.Dir == o.Dir {
		env = prev.Env
	}
	env["NEXUL_VERSION"] = strings.TrimPrefix(tag, "v")
	env["NEXUL_PORT"] = strconv.Itoa(o.Port)
	env["NEXUL_LOGS_EMAIL"] = logsEmail
	if err := h.pickLogsPorts(env, o.Port); err != nil {
		return nil, err
	}
	// OpenObserve refuses to boot on a password that breaks its rule, so one that does was never in use.
	if !validLogsPassword(env["NEXUL_LOGS_PASSWORD"]) {
		env["NEXUL_LOGS_PASSWORD"] = randomPassword()
	}
	if env["NEXUL_LOGS_TOKEN"] == "" {
		env["NEXUL_LOGS_TOKEN"] = randomSecret()
	}
	if err := writeEnvFile(filepath.Join(o.Dir, ".env"), env); err != nil {
		return nil, err
	}
	if h.GOOS == "linux" {
		if err := h.giveToServiceUser(ctx, filepath.Join(o.Dir, "data"), filepath.Join(o.Dir, "logs")); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(h.Paths.Config), 0o755); err != nil {
		return nil, fmt.Errorf("create %s: %w", filepath.Dir(h.Paths.Config), err)
	}
	return env, writeEnvFile(h.Paths.Config, map[string]string{"NEXUL_DIR": o.Dir})
}

// pickLogsPorts gives OpenObserve's HTTP and gRPC listeners free localhost ports, once: an install that has them
// keeps them, since its own OpenObserve is what holds them.
func (h *Host) pickLogsPorts(env map[string]string, webPort int) error {
	taken := map[string]bool{strconv.Itoa(webPort): true}
	for _, key := range []string{"NEXUL_LOGS_PORT", "NEXUL_LOGS_GRPC_PORT"} {
		if env[key] != "" && !taken[env[key]] {
			taken[env[key]] = true
			continue
		}
		for {
			port, err := h.LocalPort()
			if err != nil {
				return err
			}
			if !taken[strconv.Itoa(port)] {
				env[key] = strconv.Itoa(port)
				taken[env[key]] = true
				break
			}
		}
	}
	return nil
}

// waitHealthy polls the web UI until the server answers, so the summary never points at a server still booting.
func (h *Host) waitHealthy(ctx context.Context, port int, u Unit) error {
	ctx, cancel := context.WithTimeout(ctx, h.HealthTimeout)
	defer cancel()
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		if resp, err := h.HTTP.Do(req); err == nil {
			_ = resp.Body.Close() // only the status matters
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("the server did not answer on port %d; %s", port, h.logHint(u))
		case <-time.After(h.PollInterval):
		}
	}
}

// logHint says where a unit's output goes on this OS.
func (h *Host) logHint(u Unit) string {
	if h.GOOS == "darwin" {
		return "see " + filepath.Join(h.Paths.Logs, u.Name+".log")
	}
	if h.GOOS == "windows" {
		return "see " + filepath.Join(u.Dir, "service.log")
	}
	return "see `journalctl -u " + u.Name + "`"
}

// checkPort refuses a port another program holds; the port this install already serves on is expected busy.
func (h *Host) checkPort(port int, prev *installed) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d is out of range", port)
	}
	if prev.intEnv("NEXUL_PORT", 0) == port || h.PortFree(port) {
		return nil
	}
	return fmt.Errorf("port %d is already in use; pick another with --port", port)
}

// askPort re-asks until the answer is a free port or the port this install already uses.
func (h *Host) askPort(label string, def int, prev *installed) (int, error) {
	for {
		answer, err := h.prompt(label, strconv.Itoa(def))
		if err != nil {
			return 0, err
		}
		port, err := strconv.Atoi(answer)
		if err != nil || port < 1 || port > 65535 {
			h.printf("  %q is not a port number.\n", answer)
			continue
		}
		if prev.intEnv("NEXUL_PORT", 0) != port && !h.PortFree(port) {
			h.printf("  Port %d is already in use.\n", port)
			continue
		}
		return port, nil
	}
}

func (h *Host) prompt(label, def string) (string, error) {
	h.printf("  %-18s [%s]: ", label, def)
	line, err := h.In.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("read answer: %w", err)
	}
	if answer := strings.TrimSpace(line); answer != "" {
		return answer, nil
	}
	return def, nil
}

// readEnvFile parses KEY=VALUE lines, skipping blanks and comments.
func readEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	env := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.HasPrefix(line, "#") {
			continue
		}
		env[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return env, nil
}

// writeEnvFile writes env sorted by key, readable by root only since it holds the logs credentials.
func writeEnvFile(path string, env map[string]string) error {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, env[k])
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// randomPassword meets OpenObserve's root password rule: upper and lower case, a digit and a special character. The
// special character is a dash, which no env file parser treats specially.
func randomPassword() string {
	s := randomSecret()
	return s[:12] + "-" + s[12:]
}

// validLogsPassword is OpenObserve's root password rule: 8 to 128 characters with a lower and an upper case letter, a
// digit and a special character.
func validLogsPassword(p string) bool {
	return len(p) >= 8 && len(p) <= 128 &&
		strings.ContainsFunc(p, unicode.IsLower) && strings.ContainsFunc(p, unicode.IsUpper) &&
		strings.ContainsFunc(p, unicode.IsDigit) &&
		strings.ContainsFunc(p, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// randomSecret is 24 characters mixing upper case, lower case and digits.
func randomSecret() string {
	for {
		s := rand.Text()[:12] + strings.ToLower(rand.Text()[:12])
		if strings.ContainsAny(s, "234567") && strings.ContainsAny(s, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") && strings.ContainsAny(s, "abcdefghijklmnopqrstuvwxyz") {
			return s
		}
	}
}
