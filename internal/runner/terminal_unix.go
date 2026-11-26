//go:build !windows

package runner

import (
	"path/filepath"
	"syscall"

	"github.com/aymanbagabas/go-pty"
)

// terminalShell is the shell a terminal runs: bash as a login shell, else sh.
func terminalShell() (string, []string, error) {
	path, err := firstOnPath("bash", "sh")
	if err != nil {
		return "", nil, err
	}
	if filepath.Base(path) == "bash" {
		return path, []string{"-l"}, nil
	}
	return path, nil, nil
}

// releaseTty closes this process's copy of the terminal's tty, so reading the terminal ends once the shell exits.
func releaseTty(p pty.Pty) {
	if u, ok := p.(pty.UnixPty); ok {
		_ = u.Slave().Close()
	}
}

// killTerminal kills the shell's session: the shell leads it, so everything it started goes too.
func killTerminal(pid int) {
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
