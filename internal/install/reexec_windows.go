//go:build windows

package install

import (
	"errors"
	"os"
	"os/exec"
)

// reexec runs the binary at path with this terminal and reports its exit code: Windows cannot replace a running
// process in place, so this process finishes once the new one has.
func reexec(path string, args []string) error {
	cmd := exec.Command(path, args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return &ExitCodeError{Code: exit.ExitCode()}
	}
	return err
}
