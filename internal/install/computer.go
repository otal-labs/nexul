package install

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/platform/hostcred"
)

// A person's own computer runs one personal runner, as that person, under their own service manager (ADR 0146).
const (
	kindComputer = "computer"
	computerUnit = "nexul-computer"
)

// ComputerOptions are the choices `nexul install computer` takes.
type ComputerOptions struct {
	// Token is the signed token Add a computer's command carries: the instance, the one-time code and the computer.
	Token   string
	Version string
}

// computerClaims is the part of a computer's token the installer reads: where to enroll. The token's signature is
// the instance's to check; a token whose server was swapped fails there, and its single-use code with it.
type computerClaims struct {
	Server string `json:"server"`
}

func (h *Host) runInstallComputer(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("install computer", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	var o ComputerOptions
	fs.StringVar(&o.Token, "token", "", "the token Add a computer's command carries")
	fs.StringVar(&o.Version, "version", "", "release to install (default: this binary's version)")
	fs.Bool("yes", false, "accepted for scripts; this install never asks")
	fs.Bool("y", false, "shorthand for --yes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	return h.InstallComputer(ctx, o)
}

// InstallComputer installs this computer's personal runner as the person's own user service: the runner binary and
// its credential under their home, the nexul command in ~/.local/bin, and nothing as root.
func (h *Host) InstallComputer(ctx context.Context, o ComputerOptions) error {
	if err := h.checkComputerUser(); err != nil {
		return err
	}
	var claims computerClaims
	if err := hostcred.PeekToken(strings.TrimSpace(o.Token), &claims); err != nil {
		return errors.New("--token is not a computer token: copy the whole command from Add a computer")
	}
	ho := HostOptions{Kind: kindComputer, Server: strings.TrimSuffix(claims.Server, "/"), Token: strings.TrimSpace(o.Token)}
	if !strings.HasPrefix(ho.Server, "http://") && !strings.HasPrefix(ho.Server, "https://") {
		return errors.New("--token names no instance address: copy the whole command from Add a computer")
	}
	tag, err := installTag(o.Version)
	if err != nil {
		return err
	}
	h.Paths = h.computerPaths()
	h.printf("Nexul %s computer installer\n\n", tag)
	if err := h.step("Command", h.installSelf); err != nil {
		return err
	}
	if err := h.step("Lingering", func() (string, error) { return h.enableLinger(ctx), nil }); err != nil {
		return err
	}
	if err := h.step("Runner", func() (string, error) { return h.installComputerRunner(ctx, ho, tag) }); err != nil {
		return err
	}
	h.printf("\nThis computer's runner is running as your %s service; Nexul shows the computer connected once it reaches %s.\n", computerUnit, ho.Server)
	h.printf("  Remove it      nexul uninstall computer\n  Logs           journalctl --user -u %s\n", computerUnit)
	return nil
}

// checkComputerUser refuses root: the runner reaches the person's own T3 Code as them, and root would own its files.
func (h *Host) checkComputerUser() error {
	if h.GOOS != "linux" {
		return fmt.Errorf("nexul install computer runs on Linux for now, not %s", h.GOOS)
	}
	if h.Getuid() == 0 {
		return errors.New("run this as yourself, not as root or with sudo: the computer's runner runs as you")
	}
	return nil
}

// computerPaths keeps everything in the person's home: the command in ~/.local/bin, the runner under
// ~/.local/share/nexul, and its unit with their other systemd user units.
func (h *Host) computerPaths() Paths {
	p := h.Paths
	p.BinDir = filepath.Join(h.Home, ".local", "bin")
	p.UnitRoot = filepath.Join(h.Home, ".local", "share", "nexul")
	p.Services = filepath.Join(h.Home, ".config", "systemd", "user")
	return p
}

// installComputerRunner downloads the runner, trades the code for its credential and starts it as a user service. A
// failure removes its directory again, so the next attempt starts clean.
func (h *Host) installComputerRunner(ctx context.Context, o HostOptions, tag string) (string, error) {
	existing, err := h.loadUnit(computerUnit)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return "", errors.New("this computer already has its runner; remove it first with: nexul uninstall computer")
	}
	dir := h.unitDir(computerUnit)
	u := Unit{Name: computerUnit, Kind: kindComputer, Version: tag, Dir: dir, WorkDir: dir, Exec: filepath.Join(dir, h.exeName(binaryFor(kindRunner)))}
	if err := h.setUpComputer(ctx, &u, o, tag); err != nil {
		return "", errors.Join(err, os.RemoveAll(u.Dir))
	}
	return u.Name + ", " + u.Host, nil
}

func (h *Host) setUpComputer(ctx context.Context, u *Unit, o HostOptions, tag string) error {
	if err := os.MkdirAll(u.Dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", u.Dir, err)
	}
	if err := h.releaseAsset(ctx, tag, h.assetName(binaryFor(kindRunner)), u.Exec); err != nil {
		return err
	}
	enrolled, err := h.enroll(ctx, o.Server, o, tag)
	if err != nil {
		return err
	}
	credFile, err := writeCredential(u.Dir, enrolled.Credential)
	if err != nil {
		return err
	}
	u.Host = enrolled.Name
	u.Env = map[string]string{
		"NEXUL_SERVER_URL":      o.Server,
		"NEXUL_CREDENTIAL_FILE": credFile,
		"NEXUL_CTL":             h.ctlPath(),
		"NEXUL_RUNNER_NAME":     enrolled.Name,
		"NEXUL_RUNNER_MODE":     "personal",
	}
	if err := h.saveUnit(*u); err != nil {
		return err
	}
	return userSystemd{h}.install(ctx, *u)
}

// enableLinger keeps the person's service manager running after they log out and from boot: as them first, then
// through sudo without a password prompt. Without it the runner runs while they are logged in, and it says so.
func (h *Host) enableLinger(ctx context.Context) string {
	uid := strconv.Itoa(h.Getuid())
	if out, _ := h.Exec.Run(ctx, "loginctl", "show-user", uid, "--property=Linger", "--value"); strings.TrimSpace(out) == "yes" { // fails for a user with no session yet
		return "on"
	}
	if _, err := h.Exec.Run(ctx, "loginctl", "enable-linger"); err == nil {
		return "turned on"
	}
	if _, err := h.Exec.Run(ctx, "sudo", "-n", "loginctl", "enable-linger", uid); err == nil {
		return "turned on with sudo"
	}
	return "off: the runner stops when you log out; sudo loginctl enable-linger $USER keeps it running"
}

func (h *Host) runUninstallComputer(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("uninstall computer", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	detach := fs.Bool("detach", false, "run the removal outside the runner's own service and return at once")
	yes := fs.Bool("yes", false, "do not ask for confirmation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if *detach {
		return h.detach(ctx, append([]string{"uninstall", kindComputer}, args...))
	}
	return h.UninstallComputer(ctx, *yes)
}

// UninstallComputer removes this computer's runner: its user service and its directory, and tells the instance.
func (h *Host) UninstallComputer(ctx context.Context, yes bool) error {
	if err := h.checkComputerUser(); err != nil {
		return err
	}
	h.Paths = h.computerPaths()
	u, err := h.loadUnit(computerUnit)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("this computer has no Nexul runner installed for your user")
	}
	h.printf("This removes this computer's runner and tells the instance it is gone.\n")
	if err := h.confirm(yes); err != nil {
		return err
	}
	h.printf("\n")
	err = h.step(u.Name, func() (string, error) {
		detail := "removed" + h.tellInstance(ctx, *u)
		if err := (userSystemd{h}).remove(ctx, *u); err != nil {
			return "", err
		}
		if err := os.RemoveAll(u.Dir); err != nil {
			return "", fmt.Errorf("remove %s: %w", u.Dir, err)
		}
		return detail, nil
	})
	if err != nil {
		return err
	}
	h.printf("\nThis computer's runner is uninstalled.\n")
	return nil
}

// userSystemd runs a unit under the person's own systemd user manager, at boot too once lingering is on.
type userSystemd struct{ h *Host }

func (s userSystemd) path(u Unit) string { return filepath.Join(s.h.Paths.Services, u.Name+".service") }

func (s userSystemd) systemctl(ctx context.Context, args ...string) error {
	_, err := s.h.Exec.Run(ctx, "systemctl", append([]string{"--user"}, args...)...)
	return err
}

func (s userSystemd) install(ctx context.Context, u Unit) error {
	if err := s.h.userBus(ctx); err != nil {
		return err
	}
	if err := os.MkdirAll(s.h.Paths.Services, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", s.h.Paths.Services, err)
	}
	if err := os.WriteFile(s.path(u), []byte(userUnit(u)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", s.path(u), err)
	}
	for _, args := range [][]string{{"daemon-reload"}, {"enable", u.Name + ".service"}, {"restart", u.Name + ".service"}} {
		if err := s.systemctl(ctx, args...); err != nil {
			return err
		}
	}
	return nil
}

func (s userSystemd) remove(ctx context.Context, u Unit) error {
	if err := s.h.userBus(ctx); err != nil {
		return err
	}
	_ = s.systemctl(ctx, "disable", "--now", u.Name+".service") // fails for a unit already gone; the removals below are what matter
	if err := os.Remove(s.path(u)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", s.path(u), err)
	}
	return s.systemctl(ctx, "daemon-reload")
}

// userUnit restarts the runner whenever it exits, so a crash the runner cannot recover from costs a few seconds.
func userUnit(u Unit) string {
	return strings.Join([]string{
		"[Unit]",
		"Description=" + u.describe(),
		"",
		"[Service]",
		"EnvironmentFile=" + u.envFile(),
		"WorkingDirectory=" + u.WorkDir,
		"ExecStart=" + u.Exec,
		"Restart=always",
		"RestartSec=5",
		"",
		"[Install]",
		"WantedBy=default.target",
		"",
	}, "\n")
}

// userBus points systemctl --user at the person's manager when the installer's shell has no login session of its
// own (su, sudo -u, a remote command), waiting for the manager lingering has just started.
func (h *Host) userBus(ctx context.Context) error {
	if os.Getenv("XDG_RUNTIME_DIR") != "" {
		return nil
	}
	dir := filepath.Join(h.Paths.UserRuntime, strconv.Itoa(h.Getuid()))
	ctx, cancel := context.WithTimeout(ctx, h.HealthTimeout)
	defer cancel()
	for {
		if _, err := os.Stat(filepath.Join(dir, "bus")); err == nil {
			return os.Setenv("XDG_RUNTIME_DIR", dir)
		}
		select {
		case <-ctx.Done():
			return errors.New("your systemd user manager is not running: log in to this computer, or run sudo loginctl enable-linger $USER, then run this again")
		case <-time.After(h.PollInterval):
		}
	}
}
