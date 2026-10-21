package install

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const (
	// foregroundRestartDelay is how long a child that exited waits to start again, as the systemd unit's RestartSec does.
	foregroundRestartDelay = 5 * time.Second
	// foregroundGrace is how long the runner and T3 Code get to stop on SIGTERM before they are killed.
	foregroundGrace = 15 * time.Second
	// t3SupervisedEnv tells the runner this supervisor restarts T3 Code, so it never runs `t3 service restart`.
	t3SupervisedEnv = "NEXUL_T3_SUPERVISED"
)

// runsForeground reports whether the computer's runner runs under `nexul run computer` instead of a service
// manager: when asked to, and on a Linux host that is not running systemd (ADR 0163).
func (h *Host) runsForeground(o ComputerOptions) (bool, error) {
	if o.Foreground && h.GOOS != "linux" {
		return false, fmt.Errorf("--foreground is for Linux hosts with no systemd; %s runs the computer's runner as a service of its own", h.GOOS)
	}
	if o.Foreground {
		return true, nil
	}
	return h.GOOS == "linux" && !fileExists(h.Paths.SystemdProbe), nil
}

// foregroundPerson is the account that runs the install: the supervisor installs nothing that needs root, so the
// runner belongs to whoever starts it, and under sudo that would be the wrong person.
func (h *Host) foregroundPerson() (person, error) {
	if h.Getuid() == 0 && os.Getenv("SUDO_USER") != "" && os.Getenv("SUDO_USER") != "root" {
		return person{}, errors.New("run this without sudo: on a host with no systemd it installs no service, and the runner runs as whoever runs the command")
	}
	return h.selfPerson()
}

func (h *Host) selfPerson() (person, error) {
	account, err := h.CurrentUser()
	if err != nil {
		return person{}, fmt.Errorf("look up this account: %w", err)
	}
	p, err := personOf(account.Username, account)
	p.self = true
	return p, err
}

// foregroundInstall is this account's foreground computer install, nil when it has none; h.Paths is left alone then.
func (h *Host) foregroundInstall() (person, *Unit, error) {
	if h.GOOS != "linux" {
		return person{}, nil, nil
	}
	p, err := h.selfPerson()
	if err != nil {
		return person{}, nil, err
	}
	saved := h.Paths
	h.Paths = h.computerPaths(p)
	u, err := h.loadUnit(computerUnit)
	if err != nil || u == nil || !u.Foreground {
		h.Paths = saved
		return p, nil, err
	}
	return p, u, nil
}

func (h *Host) runRunComputer(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("run computer", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	return h.RunComputer(ctx)
}

// RunComputer runs this account's computer runner, and T3 Code when the install set it up, in the foreground until it
// is told to stop or the runner is removed.
func (h *Host) RunComputer(ctx context.Context) error {
	if h.GOOS != "linux" {
		return fmt.Errorf("nexul run computer is for Linux hosts with no systemd; %s runs the computer's runner as a service of its own", h.GOOS)
	}
	p, u, err := h.foregroundInstall()
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("this account has no computer runner to run here: add the computer with nexul install computer --foreground")
	}
	return h.supervise(ctx, p, *u)
}

// supervise keeps the runner and T3 Code running, and takes the install away when the runner was removed.
func (h *Host) supervise(ctx context.Context, p person, u Unit) error {
	release, err := lockSupervisor(u.Dir)
	if err != nil {
		return err
	}
	defer release()
	s := &childSupervisor{
		log:      slog.New(slog.NewJSONHandler(os.Stderr, nil)),
		children: h.foregroundChildren(p, u),
		delay:    foregroundRestartDelay,
		grace:    foregroundGrace,
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)
	if !s.run(ctx, stop) {
		return nil
	}
	s.log.Info("the runner was removed from the instance; removing this install", "dir", u.Dir)
	return h.removeForeground(u)
}

// foregroundChildren are the processes the units would run: the runner with the unit's env and, when the install set
// T3 Code up for it, `t3 serve`.
func (h *Host) foregroundChildren(p person, u Unit) []childSpec {
	request := filepath.Join(u.Dir, removeRequest)
	runner := childSpec{
		name: "runner",
		command: func() *exec.Cmd {
			env := u.Env
			if len(u.T3) > 0 {
				env = withEnv(env, t3SupervisedEnv, "1")
			}
			return foregroundCommand(p, u.WorkDir, env, u.Exec)
		},
		ends: func() bool { return fileExists(request) },
	}
	if len(u.T3) == 0 {
		return []childSpec{runner}
	}
	t3 := childSpec{name: "t3", command: func() *exec.Cmd { return foregroundCommand(p, p.home, nil, u.T3...) }}
	return []childSpec{runner, t3}
}

func withEnv(env map[string]string, k, v string) map[string]string {
	out := map[string]string{k: v}
	for key, value := range env {
		out[key] = value
	}
	return out
}

// foregroundCommand is argv in dir with this process's environment plus env, which the child's own settings win over;
// a host with no HOME gives the child the account's.
func foregroundCommand(p person, dir string, env map[string]string, argv ...string) *exec.Cmd {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.Env = os.Environ()
	if os.Getenv("HOME") == "" {
		cmd.Env = append(cmd.Env, "HOME="+p.home)
	}
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	return cmd
}

// removeForeground deletes what a foreground install made: the runner's folder and the nexul command beside it. The
// credential goes with the folder, so a supervisor started again finds nothing to run.
func (h *Host) removeForeground(u Unit) error {
	if err := os.RemoveAll(u.Dir); err != nil {
		return fmt.Errorf("remove %s: %w", u.Dir, err)
	}
	return removeIfPresent(filepath.Join(h.Paths.BinDir, h.commandName()))
}

// requestForegroundRemoval is what a removed runner calls (`nexul uninstall computer --detach`) under the supervisor:
// it leaves the request for the supervisor, which removes the install once the runner has exited.
func (h *Host) requestForegroundRemoval(u Unit) error {
	if err := os.WriteFile(filepath.Join(u.Dir, removeRequest), nil, 0o600); err != nil {
		return fmt.Errorf("ask for this computer's runner to be removed: %w", err)
	}
	h.printf("Asked for this computer's runner to be removed; its supervisor removes it once the runner exits.\n")
	return nil
}

// uninstallForeground stops this computer's supervisor, if one runs, removes its install and tells the instance.
func (h *Host) uninstallForeground(ctx context.Context, u Unit, yes bool) error {
	h.printf("This removes this computer's runner and tells the instance it is gone.\n")
	if err := h.confirm(yes); err != nil {
		return err
	}
	h.printf("\n")
	err := h.step(u.Name, func() (string, error) {
		told := h.tellInstance(ctx, u)
		if err := h.stopForeground(ctx, u.Dir); err != nil {
			return "", err
		}
		return "removed" + told, h.removeForeground(u)
	})
	if err != nil {
		return err
	}
	h.printf("\nThis computer's runner is uninstalled.\n")
	return nil
}

// stopForeground stops the supervisor running dir's runner and waits until it has let go of its lock.
func (h *Host) stopForeground(ctx context.Context, dir string) error {
	pid, running := supervisorPID(dir)
	if !running {
		return nil
	}
	if pid == 0 {
		return errors.New("this computer's supervisor is starting; run the command again in a moment")
	}
	if err := stopSupervisor(pid); err != nil {
		return fmt.Errorf("stop the supervisor, pid %d: %w", pid, err)
	}
	ctx, cancel := context.WithTimeout(ctx, h.HealthTimeout)
	defer cancel()
	for {
		if _, running := supervisorPID(dir); !running {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("the supervisor, pid %d, did not stop", pid)
		case <-time.After(h.PollInterval):
		}
	}
}

// foregroundStarted prints, once, how to start the supervisor again: nothing starts it at boot.
func (h *Host) foregroundStarted(p person, server string) {
	h.printf("\nThis computer's runner now runs under Nexul's own supervisor, in this terminal, as %s; Nexul shows the computer connected once it reaches %s.\n", p.name, server)
	h.printf("  Stop it        Ctrl-C, or SIGTERM to the supervisor\n  Remove it      nexul uninstall computer\n")
	h.printf("  After a reboot nothing starts it for you; run this from a container entrypoint or an @reboot cron entry:\n    %s run computer\n\n", h.ctlPath())
}

// t3ServeArgs is T3 Code's command line under the supervisor: loopback only, on its port, with no browser.
func t3ServeArgs(command string, port int) []string {
	return []string{command, "serve", "--host", "127.0.0.1", "--port", strconv.Itoa(port)}
}
