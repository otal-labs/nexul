//go:build windows

package runner

import (
	"os"
	"os/exec"
)

// reexec starts exe as a new process with the same args, env and stdio, then exits this one: Windows has no
// syscall.Exec equivalent that replaces the current process image.
func reexec(exe string, args, env []string) error {
	cmd := exec.Command(exe, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
