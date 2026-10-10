package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// The kinds of unit a machine runs.
const (
	kindServer      = "server"
	kindLogs        = "logs"
	kindRunner      = "runner"
	kindAutomations = "automations"
)

// serviceUser is the Linux system user the server, OpenObserve and automations hosts run as.
const serviceUser = "nexul"

var hostNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

// Unit is one Nexul service on this machine. Its record, unit.json in the unit's own directory, is how status,
// upgrade, uninstall and the Windows service host find it again.
type Unit struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	// Host is a runner's or automations host's name; empty for the server and OpenObserve.
	Host    string            `json:"host,omitempty"`
	Version string            `json:"version"`
	Dir     string            `json:"dir"`
	Exec    string            `json:"exec"`
	WorkDir string            `json:"work_dir,omitempty"`
	Env     map[string]string `json:"env"`
	// User is the Linux account the unit runs as; empty is root.
	User string `json:"user,omitempty"`
}

// unitName is the service name for a kind: nexul-server, nexul-openobserve, nexul-runner-<host>.
func unitName(kind, host string) string {
	if kind == kindServer {
		return "nexul-server"
	}
	if kind == kindLogs {
		return "nexul-openobserve"
	}
	return "nexul-" + kind + "-" + host
}

// unitDir is the unit's own directory: <root>/server, <root>/openobserve, <root>/runner-<host>.
func (h *Host) unitDir(name string) string {
	return filepath.Join(h.Paths.UnitRoot, strings.TrimPrefix(name, "nexul-"))
}

// binaryFor is the release binary a kind runs, which is also its file name in the unit directory.
func binaryFor(kind string) string {
	if kind == kindLogs {
		return "openobserve"
	}
	return "nexul-" + kind
}

func (h *Host) exeName(binary string) string {
	if h.GOOS == "windows" {
		return binary + ".exe"
	}
	return binary
}

// newUnit lays out a unit in its own directory, running its kind's binary from there.
func (h *Host) newUnit(kind, host, version string) Unit {
	name := unitName(kind, host)
	dir := h.unitDir(name)
	return Unit{Name: name, Kind: kind, Host: host, Version: version, Dir: dir, WorkDir: dir,
		Exec: filepath.Join(dir, h.exeName(binaryFor(kind))), Env: map[string]string{}}
}

func (u Unit) envFile() string { return filepath.Join(u.Dir, "env") }

// describe is the unit's human name, used as the service's description.
func (u Unit) describe() string {
	if u.Kind == kindServer {
		return "Nexul server"
	}
	if u.Kind == kindLogs {
		return "Nexul logs (OpenObserve)"
	}
	if u.Kind == kindRunner {
		return "Nexul runner " + u.Host
	}
	if u.Kind == kindComputer {
		return "Nexul computer runner"
	}
	return "Nexul automations host " + u.Host
}

// saveUnit writes the unit's env file and record, both 0600 because the env holds its secrets.
func (h *Host) saveUnit(u Unit) error {
	if err := os.MkdirAll(u.Dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", u.Dir, err)
	}
	if err := writeEnvFile(u.envFile(), u.Env); err != nil {
		return err
	}
	data, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", u.Name, err)
	}
	if err := os.WriteFile(filepath.Join(u.Dir, "unit.json"), data, 0o600); err != nil {
		return fmt.Errorf("write %s record: %w", u.Name, err)
	}
	return nil
}

// loadUnit reads one unit's record; nil, nil when no unit by that name is installed.
func (h *Host) loadUnit(name string) (*Unit, error) {
	data, err := os.ReadFile(filepath.Join(h.unitDir(name), "unit.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s record: %w", name, err)
	}
	var u Unit
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, fmt.Errorf("read %s record: %w", name, err)
	}
	return &u, nil
}

// loadUnits is every unit on this machine: the server, OpenObserve, then runners and automations hosts by name.
func (h *Host) loadUnits() ([]Unit, error) {
	records, err := filepath.Glob(filepath.Join(h.Paths.UnitRoot, "*", "unit.json"))
	if err != nil {
		return nil, fmt.Errorf("list units: %w", err)
	}
	units := make([]Unit, 0, len(records))
	for _, rec := range records {
		u, err := h.loadUnit("nexul-" + filepath.Base(filepath.Dir(rec)))
		if err != nil {
			return nil, err
		}
		units = append(units, *u)
	}
	order := map[string]int{kindServer: 0, kindLogs: 1, kindRunner: 2, kindAutomations: 3}
	slices.SortFunc(units, func(a, b Unit) int {
		if d := order[a.Kind] - order[b.Kind]; d != 0 {
			return d
		}
		return strings.Compare(a.Name, b.Name)
	})
	return units, nil
}
