//go:build windows

package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

// startDetached starts name outside this console, process group and job, so stopping the service that asked for
// it does not stop it too.
func startDetached(name string, args []string, logPath string) error {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(logPath), err)
	}
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open %s: %w", logPath, err)
	}
	defer func() { _ = log.Close() }() // the child holds its own handle
	flags := uint32(windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP)
	err = start(name, args, log, flags|windows.CREATE_BREAKAWAY_FROM_JOB)
	if err == nil {
		return nil
	}
	// A job that forbids breakaway refuses the flag; outside such a job the process is already free of it.
	return start(name, args, log, flags)
}

func start(name string, args []string, log *os.File, flags uint32) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = log, log
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flags}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
