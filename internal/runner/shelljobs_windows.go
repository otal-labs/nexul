package runner

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"

	"golang.org/x/sys/windows"
)

// shellCommand runs script in pwsh, else Windows PowerShell, or in the shell asked for.
func shellCommand(shell, script string) (string, []string, error) {
	names := []string{"pwsh", "powershell"}
	if shell != "" {
		names = []string{shell}
	}
	if shell == "bash" || shell == "sh" {
		return "", nil, fmt.Errorf("%s does not run on Windows; this computer runs pwsh or powershell", shell)
	}
	path, err := firstOnPath(names...)
	if err != nil {
		return "", nil, err
	}
	return path, []string{"-NoProfile", "-NonInteractive", "-Command", script}, nil
}

// killTree runs the job with no console window of its own, and kills everything it started on a timeout or a cancel;
// a window an app it starts opens still shows on the person's desktop.
func killTree(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	cmd.Cancel = func() error {
		kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
		kill.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
		return kill.Run()
	}
}
