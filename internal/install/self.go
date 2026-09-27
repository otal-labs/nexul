package install

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// installSelf copies the running binary to the bin directory, so `nexul upgrade` and `nexul status` are on the PATH
// and every unit's NEXUL_CTL points at one stable path. A copy that fails leaves the command where it is, and that
// path is what the units get.
func (h *Host) installSelf() (string, error) {
	exe, err := h.executable()
	if err != nil {
		return "", err
	}
	h.ctl = filepath.Join(h.Paths.BinDir, h.commandName())
	if exe == h.ctl {
		return h.ctl, nil
	}
	if err := os.MkdirAll(h.Paths.BinDir, 0o755); err != nil {
		h.ctl = exe
		return exe, nil
	}
	if err := copyFile(exe, h.ctl); err != nil {
		h.ctl = exe
		return exe, nil
	}
	return h.ctl, nil
}

// ctlPath is the nexul command the units call: where installSelf put it, else its place in the bin directory.
func (h *Host) ctlPath() string {
	if h.ctl != "" {
		return h.ctl
	}
	return filepath.Join(h.Paths.BinDir, h.commandName())
}

// commandName is the nexul binary's file name on this OS.
func (h *Host) commandName() string {
	if h.GOOS == "windows" {
		return "nexul.exe"
	}
	return "nexul"
}

// executable resolves the running binary's real path, following symlinks.
func (h *Host) executable() (string, error) {
	exe, err := h.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	return resolved, nil
}

// copyFile copies src to dest (0755) through a temporary file, so a running dest is replaced, never truncated.
func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer func() { _ = in.Close() }() // read-only
	tmp := dest + ".download"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create %s: %w", tmp, err)
	}
	_, copyErr := io.Copy(out, in)
	if err := errors.Join(copyErr, out.Close()); err != nil {
		return errors.Join(fmt.Errorf("copy to %s: %w", tmp, err), os.Remove(tmp))
	}
	return replaceFile(tmp, dest)
}
