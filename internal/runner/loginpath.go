package runner

import (
	"cmp"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
)

const (
	loginPathTimeout = 10 * time.Second
	loginPathMarker  = "__nexul_path__"
)

// UseLoginPath puts the PATH of the person's interactive login shell ahead of the one the service manager started the
// runner with, so a tool installed by Homebrew, Linuxbrew or a version manager is found as it is in their terminal.
func UseLoginPath(ctx context.Context, log *slog.Logger) {
	if runtime.GOOS == "windows" {
		return
	}
	shell := cmp.Or(os.Getenv("SHELL"), "/bin/sh")
	ctx, cancel := context.WithTimeout(ctx, loginPathTimeout)
	defer cancel()
	// Interactive too: a stock .bashrc returns early for other shells, and that is where brew's shellenv usually sits.
	cmd := exec.CommandContext(ctx, shell, "-ilc", `printf '%s%s' "`+loginPathMarker+`" "$PATH"`)
	out, err := cmd.Output()
	_, login, found := strings.Cut(string(out), loginPathMarker)
	if err != nil || !found {
		log.Warn("read the login shell's PATH", "shell", shell, "error", err)
		return
	}
	merged := mergePaths(strings.TrimSpace(login), os.Getenv("PATH"))
	if err := os.Setenv("PATH", merged); err != nil {
		log.Warn("set PATH from the login shell", "error", err)
	}
}

// mergePaths is first's directories, then those of second it lacks, each once.
func mergePaths(first, second string) string {
	var dirs []string
	for _, dir := range filepath.SplitList(first + string(filepath.ListSeparator) + second) {
		if dir != "" && !slices.Contains(dirs, dir) {
			dirs = append(dirs, dir)
		}
	}
	return strings.Join(dirs, string(filepath.ListSeparator))
}
