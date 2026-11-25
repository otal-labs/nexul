//go:build !windows

package runner

import (
	"fmt"
	"os/exec"
	"syscall"
)

// shellCommand runs script in bash, else sh, or in the shell asked for.
func shellCommand(shell, script string) (string, []string, error) {
	names := []string{"bash", "sh"}
	if shell != "" {
		names = []string{shell}
	}
	if shell == "pwsh" || shell == "powershell" {
		return "", nil, fmt.Errorf("%s runs only on Windows; this computer runs bash or sh", shell)
	}
	path, err := firstOnPath(names...)
	if err != nil {
		return "", nil, err
	}
	return path, []string{"-c", script}, nil
}

// killTree starts the job in its own process group, so a timeout or a cancel kills everything it started.
func killTree(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
