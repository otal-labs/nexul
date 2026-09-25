package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const runnerService = "nexul-runner.service"

// installRunner puts the instance runner on the host as a systemd service: it drives the host's Docker daemon and
// checks stacks out onto the host filesystem, which is why it is not a container (ADR 0032).
func (h *Host) installRunner(ctx context.Context, o Options, tag string) (string, error) {
	if _, err := os.Stat(h.Paths.SystemdProbe); err != nil {
		return "", errors.New("the instance runner runs as a systemd service, and this host is not running systemd")
	}
	bin := filepath.Join(h.Paths.BinDir, "nexul-runner")
	if err := h.releaseAsset(ctx, tag, h.assetName("nexul-runner"), bin); err != nil {
		return "", err
	}
	if err := os.WriteFile(h.Paths.Unit, []byte(runnerUnit(bin, o)), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", h.Paths.Unit, err)
	}
	for _, args := range [][]string{{"daemon-reload"}, {"enable", runnerService}, {"restart", runnerService}} {
		if _, err := h.Exec.Run(ctx, "systemctl", args...); err != nil {
			return "", err
		}
	}
	return runnerService, nil
}

func runnerUnit(bin string, o Options) string {
	return fmt.Sprintf(`[Unit]
Description=Nexul instance runner
Wants=network-online.target docker.service
After=network-online.target docker.service

[Service]
Environment=NEXUL_SERVER_WS=ws://127.0.0.1:%d/ws/runner
Environment=NEXUL_RUNNER_SECRET_FILE=%s
Environment=NEXUL_RUNNER_ID=instance
Environment=NEXUL_RUNNER_NAME=instance
Environment=NEXUL_MACHINE=instance
ExecStart=%s
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, o.Port, filepath.Join(o.Dir, "data", "runner-secret"), bin)
}

// removeRunner stops and deletes the runner service and binary; a host without them is already in that state.
func (h *Host) removeRunner(ctx context.Context) error {
	// disable --now fails on a host that never had the unit; the removals below are what uninstall promises.
	_, _ = h.Exec.Run(ctx, "systemctl", "disable", "--now", runnerService)
	for _, path := range []string{h.Paths.Unit, filepath.Join(h.Paths.BinDir, "nexul-runner")} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}
	_, err := h.Exec.Run(ctx, "systemctl", "daemon-reload")
	return err
}

// installSelf copies the running binary to the bin directory, so `nexul upgrade` and `nexul status` are on the PATH.
func (h *Host) installSelf() (string, error) {
	exe, err := h.executable()
	if err != nil {
		return "", err
	}
	dest := filepath.Join(h.Paths.BinDir, "nexul")
	if exe == dest {
		return dest, nil
	}
	if err := copyFile(exe, dest); err != nil {
		return "", err
	}
	return dest, nil
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
	if err := os.Rename(tmp, dest); err != nil {
		return fmt.Errorf("install %s: %w", dest, err)
	}
	return nil
}
