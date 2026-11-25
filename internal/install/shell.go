package install

import (
	"context"
	"errors"
)

// shellEnvKey turns a computer's shell jobs off when it holds shellOff; the runner reads it at start.
const (
	shellEnvKey = "NEXUL_SHELL"
	shellOff    = "off"
)

const shellUsage = "usage: nexul shell on|off"

// runShellSwitch is `nexul shell on|off`: whether this computer's runner runs the commands Nexul sends it. Nexul can
// never turn back on what this turned off.
func (h *Host) runShellSwitch(ctx context.Context, args []string) error {
	if len(args) != 1 || (args[0] != "on" && args[0] != "off") {
		return errors.New(shellUsage)
	}
	if h.GOOS != "windows" && h.Getuid() != 0 {
		return errors.New("it restarts this computer's runner service, so run it with sudo: sudo nexul shell " + args[0])
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
	if args[0] == "off" {
		u.Env[shellEnvKey] = shellOff
	}
	if err := h.saveUnit(*u); err != nil {
		return err
	}
	if err := h.giveTo(p, u.Dir); err != nil {
		return err
	}
	if u.Foreground {
		h.printf("Commands from Nexul are %s on this computer once nexul run computer starts again.\n", args[0])
		return nil
	}
	if err := h.computerHost().install(ctx, *u); err != nil {
		return err
	}
	h.printf("Commands from Nexul are %s on this computer; its runner restarted.\n", args[0])
	return nil
}
