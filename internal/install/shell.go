package install

import (
	"context"
	"errors"
)

// shellEnvKey turns a computer's shell jobs off when it holds shellOff, and a machine's runner's on when it holds
// shellOn; the runner reads it at start.
const (
	shellEnvKey = "NEXUL_SHELL"
	shellOff    = "off"
	shellOn     = "on"
)

const shellUsage = "usage: nexul shell on|off [<runner>]"

// runShellSwitch is `nexul shell on|off`: whether this computer's runner, or with a name this machine's runner of that
// name, runs the commands Nexul sends it. Nexul can never turn on what this turned off.
func (h *Host) runShellSwitch(ctx context.Context, args []string) error {
	if len(args) < 1 || len(args) > 2 || (args[0] != "on" && args[0] != "off") {
		return errors.New(shellUsage)
	}
	if len(args) == 2 {
		return h.runnerShellSwitch(ctx, args[1], args[0] == "on")
	}
	return h.computerShellSwitch(ctx, args[0])
}

// computerShellSwitch turns this computer's runner's commands from Nexul on or off and restarts it.
func (h *Host) computerShellSwitch(ctx context.Context, state string) error {
	if h.GOOS != "windows" && h.Getuid() != 0 {
		return errors.New("it restarts this computer's runner service, so run it with sudo: sudo nexul shell " + state)
	}
	p, err := h.computerPerson()
	if err != nil {
		return err
	}
	h.Paths = h.computerPaths(p)
	u, err := h.loadUnit(computerUnit)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("this computer has no Nexul runner installed; Add a computer in Nexul gives the command that installs it")
	}
	if u.Env == nil {
		u.Env = map[string]string{}
	}
	delete(u.Env, shellEnvKey)
	if state == "off" {
		u.Env[shellEnvKey] = shellOff
	}
	if err := h.saveUnit(*u); err != nil {
		return err
	}
	if err := h.giveTo(p, u.Dir); err != nil {
		return err
	}
	if u.Foreground {
		h.printf("Commands from Nexul are %s on this computer once nexul run computer starts again.\n", state)
		return nil
	}
	if err := h.computerHost().install(ctx, *u); err != nil {
		return err
	}
	h.printf("Commands from Nexul are %s on this computer; its runner restarted.\n", state)
	return nil
}

// runnerShellSwitch turns the commands Nexul sends the runner called name on or off and restarts it. A machine's runner
// runs them as the user it runs as, root on Linux.
func (h *Host) runnerShellSwitch(ctx context.Context, name string, on bool) error {
	if err := h.checkUser(); err != nil {
		return err
	}
	u, err := h.hostUnit(kindRunner, name)
	if err != nil {
		return err
	}
	if u.Env == nil {
		u.Env = map[string]string{}
	}
	delete(u.Env, shellEnvKey)
	state := "off"
	if on {
		u.Env[shellEnvKey], state = shellOn, "on"
	}
	if err := h.saveUnit(*u); err != nil {
		return err
	}
	if err := h.services().install(ctx, *u); err != nil {
		return err
	}
	h.printf("Commands from Nexul are %s on the runner %s; it restarted.\n", state, name)
	return nil
}
