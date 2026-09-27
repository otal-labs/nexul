//go:build !windows

package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// startDetached starts name in a session of its own, which launchd does not kill with the agent that started it.
func startDetached(name string, args []string, logPath string) error {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(logPath), err)
	}
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open %s: %w", logPath, err)
	}
	defer func() { _ = log.Close() }() // the child holds its own copy of the descriptor
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = log, log
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
