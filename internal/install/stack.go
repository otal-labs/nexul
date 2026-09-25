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

	"github.com/otal-labs/nexul"
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

// requireInstall is loadInstall for the commands that only make sense on an installed host.
func (h *Host) requireInstall() (*installed, error) {
	prev, err := h.loadInstall()
	if err != nil {
		return nil, err
	}
	if prev == nil {
		return nil, errors.New("nexul is not installed on this host; run nexul install")
	}
	return prev, nil
}

// writeStack writes the compose file and .env into the install directory, keeping any secret already in .env, and
// records the directory in the config file.
func (h *Host) writeStack(o Options, prev *installed, tag string) (map[string]string, error) {
	for _, sub := range []string{"data", "logs"} {
		if err := os.MkdirAll(filepath.Join(o.Dir, sub), 0o750); err != nil {
			return nil, fmt.Errorf("create %s: %w", sub, err)
		}
	}
	if err := os.WriteFile(filepath.Join(o.Dir, "docker-compose.yml"), nexul.Compose, 0o644); err != nil {
		return nil, fmt.Errorf("write compose file: %w", err)
	}
	env := map[string]string{}
	if prev != nil && prev.Dir == o.Dir {
		env = prev.Env
	}
	env["NEXUL_VERSION"] = strings.TrimPrefix(tag, "v")
	env["NEXUL_PORT"] = strconv.Itoa(o.Port)
	env["NEXUL_LOGS_PORT"] = strconv.Itoa(o.LogsPort)
	env["NEXUL_LOGS_EMAIL"] = logsEmail
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
	if err := os.MkdirAll(filepath.Dir(h.Paths.Config), 0o755); err != nil {
		return nil, fmt.Errorf("create %s: %w", filepath.Dir(h.Paths.Config), err)
	}
	return env, writeEnvFile(h.Paths.Config, map[string]string{"NEXUL_DIR": o.Dir})
}

// compose runs a docker compose subcommand against the install directory's project.
func (h *Host) compose(ctx context.Context, dir string, args ...string) error {
	_, err := h.Exec.Run(ctx, "docker", slices.Concat([]string{"compose", "--project-directory", dir}, args)...)
	return err
}

// waitHealthy polls the web UI until the server answers, so the summary never points at a server still booting.
func (h *Host) waitHealthy(ctx context.Context, port int) error {
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
			return fmt.Errorf("the server did not answer on port %d; see `docker compose --project-directory <dir> logs server`", port)
		case <-time.After(h.PollInterval):
		}
	}
}

// checkPorts refuses ports another program holds; the ports this install already serves on are expected busy.
func (h *Host) checkPorts(o Options, prev *installed) error {
	if o.Port == o.LogsPort {
		return fmt.Errorf("the web port and the logs port are both %d", o.Port)
	}
	for _, p := range []struct {
		port int
		key  string
		flag string
	}{{o.Port, "NEXUL_PORT", "--port"}, {o.LogsPort, "NEXUL_LOGS_PORT", "--logs-port"}} {
		if p.port < 1 || p.port > 65535 {
			return fmt.Errorf("port %d is out of range", p.port)
		}
		if prev.intEnv(p.key, 0) == p.port {
			continue
		}
		if !h.PortFree(p.port) {
			return fmt.Errorf("port %d is already in use; pick another with %s", p.port, p.flag)
		}
	}
	return nil
}

// askPorts prompts for both ports with their defaults filled in; Enter keeps the default.
func (h *Host) askPorts(o Options, prev *installed) (Options, error) {
	var err error
	if o.Port, err = h.askPort("Web port", o.Port, "NEXUL_PORT", prev); err != nil {
		return o, err
	}
	if o.LogsPort, err = h.askPort("Logs UI port", o.LogsPort, "NEXUL_LOGS_PORT", prev); err != nil {
		return o, err
	}
	return o, nil
}

// askPort re-asks until the answer is a free port or the port this install already uses.
func (h *Host) askPort(label string, def int, key string, prev *installed) (int, error) {
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
		if prev.intEnv(key, 0) != port && !h.PortFree(port) {
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
// special character is a dash because compose reads $ in .env as a variable.
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
