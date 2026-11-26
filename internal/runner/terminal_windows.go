package runner

import (
	"os/exec"
	"strconv"
	"syscall"

	"github.com/aymanbagabas/go-pty"
	"golang.org/x/sys/windows"
)

// terminalShell is the shell a terminal runs: pwsh, else Windows PowerShell.
func terminalShell() (string, []string, error) {
	path, err := firstOnPath("pwsh", "powershell")
	if err != nil {
		return "", nil, err
	}
	return path, []string{"-NoLogo"}, nil
}

// releaseTty has nothing to release on Windows, where the pseudo console holds its own pipes.
func releaseTty(pty.Pty) {}

// killTerminal kills the shell and everything it started.
func killTerminal(pid int) {
	kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
	kill.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	_ = kill.Run()
}
