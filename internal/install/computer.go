package install

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/hostcred"
)

// A person's own computer runs one personal runner, as a system service that runs as that person (ADR 0146).
const (
	kindComputer = "computer"
	computerUnit = "nexul-computer"
	// cleanupUnit is the root path unit and oneshot service that remove the runner once it asks, through a file it
	// may write, with no sudo rule and no input from the person.
	cleanupUnit = "nexul-computer-cleanup"
	// removeRequest is the file whose existence asks for the removal; its contents are never read.
	removeRequest = "remove-requested"
)

// ComputerOptions are the choices `nexul install computer` takes.
type ComputerOptions struct {
	// Token is the signed token Add a computer's command carries: the instance, the one-time code and the computer.
	Token   string
	Version string
	// NoT3 skips the T3 Code step; T3Port is the port T3 Code is installed on, when not its own 3773.
	NoT3   bool
	T3Port int
}

// computerClaims is the part of a computer's token the installer reads: where to enroll. The token's signature is
// the instance's to check; a token whose server was swapped fails there, and its single-use code with it.
type computerClaims struct {
	Server string `json:"server"`
}

// person is the account a computer's runner is installed for and runs as: whoever typed sudo.
type person struct {
	name     string
	uid, gid int
	home     string
}

func (h *Host) runInstallComputer(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("install computer", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	var o ComputerOptions
	fs.StringVar(&o.Token, "token", "", "the token Add a computer's command carries")
	fs.StringVar(&o.Version, "version", "", "release to install (default: this binary's version)")
	fs.BoolVar(&o.NoT3, "no-t3", false, "leave T3 Code alone: neither look for it nor install it")
	fs.IntVar(&o.T3Port, "t3-port", 0, "port T3 Code's service listens on when this installs it (default 3773)")
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

// InstallComputer installs this computer's personal runner for the person who typed sudo: the nexul command, the
// runner and its credential under their home and owned by them, and a system service that runs it as them.
func (h *Host) InstallComputer(ctx context.Context, o ComputerOptions) error {
	p, err := h.computerPerson()
	if err != nil {
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
	h.Paths = h.computerPaths(p)
	h.printf("Nexul %s computer installer, for %s\n\n", tag, p.name)
	err = h.step("Command", func() (string, error) {
		ctl, err := h.installSelf()
		if err != nil {
			return "", err
		}
		return ctl, h.giveTo(p, ctl)
	})
	if err != nil {
		return err
	}
	if err := h.installT3(ctx, p, o); err != nil {
		return err
	}
	if err := h.step("Runner", func() (string, error) { return h.installComputerRunner(ctx, ho, tag, p) }); err != nil {
		return err
	}
	h.printf("\nThis computer's runner runs as %s under the %s system service, from boot and after you log out; Nexul shows the computer connected once it reaches %s.\n", p.name, computerUnit, ho.Server)
	h.printf("  Remove it      nexul uninstall computer\n  Logs           journalctl -u %s\n", computerUnit)
	return nil
}

// computerPerson is the person a computer's runner is for. The install runs as root under sudo only to place the
// service; a direct root login names nobody, and the runner must never run as root.
func (h *Host) computerPerson() (person, error) {
	if h.GOOS != "linux" {
		return person{}, fmt.Errorf("nexul install computer runs on Linux for now, not %s", h.GOOS)
	}
	if h.Getuid() != 0 {
		return person{}, errors.New("run this with sudo: it installs a system service that runs the computer's runner as you")
	}
	name := os.Getenv("SUDO_USER")
	if name == "" || name == "root" {
		return person{}, errors.New("run this from your own account with sudo, not as root: the computer's runner runs as the person who typed sudo")
	}
	return h.lookupPerson(name)
}

func (h *Host) lookupPerson(name string) (person, error) {
	account, err := h.LookupUser(name)
	if err != nil {
		return person{}, fmt.Errorf("look up %s: %w", name, err)
	}
	uid, uidErr := strconv.Atoi(account.Uid)
	gid, gidErr := strconv.Atoi(account.Gid)
	if err := errors.Join(uidErr, gidErr); err != nil {
		return person{}, fmt.Errorf("look up %s: %w", name, err)
	}
	return person{name: name, uid: uid, gid: gid, home: account.HomeDir}, nil
}

// computerPaths keeps everything but the service in the person's home: the command in ~/.local/bin and the runner
// under ~/.local/share/nexul. The service itself is a system unit, root's.
func (h *Host) computerPaths(p person) Paths {
	paths := h.Paths
	paths.BinDir = filepath.Join(p.home, ".local", "bin")
	paths.UnitRoot = filepath.Join(p.home, ".local", "share", "nexul")
	return paths
}

// installComputerRunner downloads the runner, trades the code for its credential and starts it as a system service
// running as the person. A failure removes its directory again, so the next attempt starts clean.
func (h *Host) installComputerRunner(ctx context.Context, o HostOptions, tag string, p person) (string, error) {
	existing, err := h.loadUnit(computerUnit)
	if err != nil {
		return "", err
	}
	if existing != nil && fileExists(h.userUnitPath(p)) {
		return h.moveToSystemService(ctx, *existing, p)
	}
	if existing != nil {
		return "", errors.New("this computer already has its runner; remove it first with: nexul uninstall computer")
	}
	if fileExists(systemd{h}.path(Unit{Name: computerUnit})) {
		return "", errors.New("this computer's runner is installed for another account; that account removes it with: nexul uninstall computer")
	}
	dir := h.unitDir(computerUnit)
	u := Unit{Name: computerUnit, Kind: kindComputer, Version: tag, Dir: dir, WorkDir: dir, Exec: filepath.Join(dir, h.exeName(binaryFor(kindRunner))), User: p.name}
	if err := h.setUpComputer(ctx, &u, o, tag, p); err != nil {
		return "", errors.Join(err, h.removeCleanup(ctx), os.RemoveAll(u.Dir), h.giveTo(p, filepath.Dir(u.Dir)))
	}
	return u.Name + ", " + u.Host, nil
}

func (h *Host) setUpComputer(ctx context.Context, u *Unit, o HostOptions, tag string, p person) error {
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
	if err := h.giveTo(p, u.Dir); err != nil {
		return err
	}
	if err := h.installCleanup(ctx, p); err != nil {
		return err
	}
	return systemd{h}.install(ctx, *u)
}

// moveToSystemService takes over a runner an earlier release installed as the person's own user service: its
// credential stays, so the computer keeps its runner and its row, and the token goes unused.
func (h *Host) moveToSystemService(ctx context.Context, u Unit, p person) (string, error) {
	if err := h.dropUserUnit(ctx, p); err != nil {
		return "", err
	}
	u.User = p.name
	if err := h.saveUnit(u); err != nil {
		return "", err
	}
	if err := h.giveTo(p, u.Dir); err != nil {
		return "", err
	}
	if err := h.installCleanup(ctx, p); err != nil {
		return "", err
	}
	if err := (systemd{h}).install(ctx, u); err != nil {
		return "", err
	}
	return u.Name + ", " + u.Host + ", moved from your user service", nil
}

// userUnitPath is where an earlier release put the runner's systemd user unit.
func (h *Host) userUnitPath(p person) string {
	return filepath.Join(p.home, ".config", "systemd", "user", computerUnit+".service")
}

// dropUserUnit stops and deletes an earlier release's user unit as the person, so it never runs beside the system
// service and a link in their home leads root nowhere.
func (h *Host) dropUserUnit(ctx context.Context, p person) error {
	path := h.userUnitPath(p)
	if !fileExists(path) {
		return nil
	}
	runtime := "XDG_RUNTIME_DIR=/run/user/" + strconv.Itoa(p.uid)
	_, _ = h.Exec.Run(ctx, "runuser", "-u", p.name, "--", "env", runtime, "systemctl", "--user", "disable", "--now", computerUnit+".service") // a user manager that is not running has nothing running to stop
	_, err := h.Exec.Run(ctx, "runuser", "-u", p.name, "--", "rm", "-f", path)
	return err
}

// giveTo hands what root made under the person's home to them: each path, everything in it, and the folders above
// it up to their home, which the install may have created.
func (h *Host) giveTo(p person, paths ...string) error {
	for _, path := range paths {
		if !strings.HasPrefix(path, p.home+string(filepath.Separator)) {
			continue
		}
		for dir := filepath.Dir(path); dir != p.home; dir = filepath.Dir(dir) {
			if err := h.Chown(dir, p.uid, p.gid); err != nil {
				return fmt.Errorf("give %s to %s: %w", dir, p.name, err)
			}
		}
		err := filepath.WalkDir(path, func(f string, _ fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			return h.Chown(f, p.uid, p.gid)
		})
		if err != nil {
			return fmt.Errorf("give %s to %s: %w", path, p.name, err)
		}
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// installCleanup lets the runner remove itself without sudo: root's own copy of nexul, a request folder only the
// person may write in, and a path unit that runs the copy's fixed removal once a request appears there.
func (h *Host) installCleanup(ctx context.Context, p person) error {
	exe, err := h.executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(h.Paths.Libexec, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", h.Paths.Libexec, err)
	}
	if err := copyFile(exe, h.cleanupExe()); err != nil {
		return err
	}
	if err := os.RemoveAll(h.Paths.Requests); err != nil { // a request left from before would fire at once
		return fmt.Errorf("clear %s: %w", h.Paths.Requests, err)
	}
	if err := os.Mkdir(h.Paths.Requests, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", h.Paths.Requests, err)
	}
	if err := h.Chown(h.Paths.Requests, p.uid, p.gid); err != nil {
		return fmt.Errorf("give %s to %s: %w", h.Paths.Requests, p.name, err)
	}
	files := map[string]string{
		h.cleanupPath(".path"):    cleanupPathUnit(filepath.Join(h.Paths.Requests, removeRequest)),
		h.cleanupPath(".service"): cleanupServiceUnit(h.cleanupExe(), p.name),
	}
	for path, text := range files {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	for _, args := range [][]string{{"daemon-reload"}, {"enable", "--now", cleanupUnit + ".path"}} {
		if _, err := h.Exec.Run(ctx, "systemctl", args...); err != nil {
			return err
		}
	}
	return nil
}

// removeCleanup takes the cleanup away again; any piece already gone is not an error.
func (h *Host) removeCleanup(ctx context.Context) error {
	_, _ = h.Exec.Run(ctx, "systemctl", "disable", "--now", cleanupUnit+".path") // fails once the unit is gone
	for _, path := range []string{h.cleanupPath(".path"), h.cleanupPath(".service"), h.cleanupExe()} {
		if err := removeIfPresent(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}
	if err := os.RemoveAll(h.Paths.Requests); err != nil {
		return fmt.Errorf("remove %s: %w", h.Paths.Requests, err)
	}
	_, err := h.Exec.Run(ctx, "systemctl", "daemon-reload")
	return err
}

func (h *Host) cleanupPath(ext string) string {
	return filepath.Join(h.Paths.Services, cleanupUnit+ext)
}

func (h *Host) cleanupExe() string { return filepath.Join(h.Paths.Libexec, "nexul-computer-uninstall") }

// cleanupPathUnit starts the cleanup once the request file exists; the path unit reads nothing from it.
func cleanupPathUnit(request string) string {
	return strings.Join([]string{
		"[Unit]",
		"Description=Remove the Nexul computer runner when it asks",
		"",
		"[Path]",
		"PathExists=" + request,
		"Unit=" + cleanupUnit + ".service",
		"",
		"[Install]",
		"WantedBy=multi-user.target",
		"",
	}, "\n")
}

// cleanupServiceUnit runs root's copy for the person named at install, so nothing the person can change steers it.
func cleanupServiceUnit(exe, name string) string {
	return strings.Join([]string{
		"[Unit]",
		"Description=Remove the Nexul computer runner",
		"",
		"[Service]",
		"Type=oneshot",
		"ExecStart=" + exe + " uninstall computer --cleanup-for " + name,
		"",
	}, "\n")
}

func (h *Host) runUninstallComputer(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("uninstall computer", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	detach := fs.Bool("detach", false, "ask for the removal and return at once, as a removed runner does")
	cleanupFor := fs.String("cleanup-for", "", "the removal the cleanup service runs as root for this person")
	yes := fs.Bool("yes", false, "do not ask for confirmation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if *cleanupFor != "" {
		return h.cleanUpComputer(ctx, *cleanupFor)
	}
	if *detach {
		if err := h.requestRemoval(); err != nil {
			return err
		}
		h.printf("Asked for this computer's runner to be removed; journalctl -u %s shows it.\n", cleanupUnit)
		return nil
	}
	return h.UninstallComputer(ctx, *yes)
}

// requestRemoval asks root's cleanup to remove this computer's runner, as the person: it is what a removed runner
// calls (`nexul uninstall computer --detach`), and it returns at once.
func (h *Host) requestRemoval() error {
	path := filepath.Join(h.Paths.Requests, removeRequest)
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		return fmt.Errorf("ask for this computer's runner to be removed: %w; remove it with: sudo ~/.local/bin/nexul uninstall computer", err)
	}
	return nil
}

// UninstallComputer removes this computer's runner and tells the instance. Under sudo it removes it at once; as the
// person, it asks root's cleanup to, as a removed runner does.
func (h *Host) UninstallComputer(ctx context.Context, yes bool) error {
	if h.GOOS != "linux" {
		return fmt.Errorf("nexul uninstall computer runs on Linux for now, not %s", h.GOOS)
	}
	p := person{name: "your account", home: h.Home}
	if h.Getuid() == 0 {
		var err error
		if p, err = h.computerPerson(); err != nil {
			return err
		}
	}
	h.Paths = h.computerPaths(p)
	u, err := h.loadUnit(computerUnit)
	if err != nil {
		return err
	}
	if u == nil {
		return fmt.Errorf("this computer has no Nexul runner installed for %s", p.name)
	}
	h.printf("This removes this computer's runner and tells the instance it is gone.\n")
	if err := h.confirm(yes); err != nil {
		return err
	}
	h.printf("\n")
	err = h.step(u.Name, func() (string, error) {
		told := h.tellInstance(ctx, *u)
		if h.Getuid() != 0 {
			return "removal asked for" + told, h.requestRemoval()
		}
		return "removed" + told, h.removeComputer(ctx, p)
	})
	if err != nil {
		return err
	}
	if h.Getuid() != 0 {
		h.printf("\nThis computer's cleanup service removes the runner in a moment; journalctl -u %s shows it.\n", cleanupUnit)
		return nil
	}
	h.printf("\nThis computer's runner is uninstalled.\n")
	return nil
}

// cleanUpComputer is the cleanup service's removal, run as root because the runner asked. It reads nothing the
// person can write, neither the request nor the runner's own record, so asking can only ever remove the runner.
func (h *Host) cleanUpComputer(ctx context.Context, name string) error {
	if h.Getuid() != 0 {
		return errors.New("--cleanup-for is the cleanup service's, which runs as root")
	}
	p, err := h.lookupPerson(name)
	if err != nil {
		return err
	}
	h.Paths = h.computerPaths(p)
	return h.removeComputer(ctx, p)
}

// removeComputer removes everything the install made: the service, the cleanup, a user service an earlier release
// left, and, as the person, so a link in their home leads nowhere else, the runner's folder and the nexul command.
func (h *Host) removeComputer(ctx context.Context, p person) error {
	if err := (systemd{h}).remove(ctx, Unit{Name: computerUnit}); err != nil {
		return err
	}
	if err := h.removeCleanup(ctx); err != nil {
		return err
	}
	if err := h.dropUserUnit(ctx, p); err != nil {
		return err
	}
	_, err := h.Exec.Run(ctx, "runuser", "-u", p.name, "--", "rm", "-rf", h.unitDir(computerUnit), filepath.Join(h.Paths.BinDir, "nexul"))
	return err
}
