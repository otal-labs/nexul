package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// isContainerRunner reports whether this process is the compose-managed instance runner: it never swaps its own
// binary, since the next image pull replaces the whole container instead.
func isContainerRunner() bool {
	if os.Getenv("NEXUL_RUNNER_ID") == "instance" {
		return true
	}
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

// handleUpdate applies an update frame immediately when the runner is idle, or defers it until the running
// job's result frame has been sent (a mid-job binary swap would kill the job).
func (c *Client) handleUpdate(ctx context.Context, frame Frame) {
	if isContainerRunner() {
		c.log.Info("update frame ignored: container runner updates via image pull", "runner_id", c.cfg.RunnerID)
		return
	}
	if c.cfg.Version == "dev" {
		c.log.Info("update frame ignored: dev build", "runner_id", c.cfg.RunnerID)
		return
	}
	c.mu.Lock()
	busy := c.jobCancel != nil
	if busy {
		fr := frame
		c.pendingUpdate = &fr
	}
	c.mu.Unlock()
	if busy {
		c.log.Info("update deferred until the running job finishes", "runner_id", c.cfg.RunnerID, "version", frame.Version)
		return
	}
	c.applyUpdateFrame(ctx, frame)
}

// jobFinished marks the runner idle once id's job sent its result frame, and applies any update that arrived
// while it was running. Guarded on id still matching the current job so a stale send from a job startJob already
// superseded can't clear the new job's state (mirrors cancelJob's own id guard).
func (c *Client) jobFinished(ctx context.Context, id string) {
	c.mu.Lock()
	if c.jobID != id {
		c.mu.Unlock()
		return
	}
	c.jobCancel = nil
	c.jobID = ""
	pending := c.pendingUpdate
	c.pendingUpdate = nil
	c.mu.Unlock()
	if pending != nil {
		c.applyUpdateFrame(ctx, *pending)
	}
}

// applyUpdateFrame resolves the running executable and applies frame, logging the outcome; a failed update
// leaves the runner exactly as it was, connected and running the old binary.
func (c *Client) applyUpdateFrame(ctx context.Context, frame Frame) {
	if frame.Sha256 == "" {
		c.log.Warn("update has no checksum; skipping verification", "runner_id", c.cfg.RunnerID, "version", frame.Version)
	}
	exe, err := runnerExecutableFn()
	if err != nil {
		c.log.Error("update failed: resolve executable", "runner_id", c.cfg.RunnerID, "error", err)
		return
	}
	if err := applyUpdate(ctx, frame, c.cfg.Token, exe); err != nil {
		c.log.Error("update failed", "runner_id", c.cfg.RunnerID, "version", frame.Version, "error", err)
		return
	}
	// applyUpdate re-execs the process in place on success; reaching this line means it didn't.
}

// runnerExecutableFn resolves the running binary's path; a package var so tests can point it at a temp file
// instead of the test binary itself.
var runnerExecutableFn = runnerExecutable

// reexecFn replaces the running process with the updated binary; a package var so tests can stub out the actual
// process replacement (reexec never returns on success on unix, which would kill the test binary).
var reexecFn = reexec

// runnerExecutable resolves the running binary's real path, following symlinks so the update swaps the actual
// file rather than a link to it.
func runnerExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolve executable symlinks: %w", err)
	}
	return resolved, nil
}

// applyUpdate downloads frame's binary to "<exe>.new", verifies its checksum when frame.Sha256 is set, swaps it
// in for exe (via "<exe>.old"), and re-execs the process. A checksum mismatch deletes the partial download and
// returns an error without touching exe. Split out from the Client so it's testable with an httptest server and
// a temp-dir exe.
func applyUpdate(ctx context.Context, frame Frame, token, exe string) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, frame.URL, nil)
	if err != nil {
		return fmt.Errorf("build update request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download update: status %d", resp.StatusCode)
	}

	newExe := exe + ".new"
	if err := downloadTo(newExe, resp.Body, frame.Sha256); err != nil {
		return err
	}

	oldExe := exe + ".old"
	if err := os.Rename(exe, oldExe); err != nil {
		_ = os.Remove(newExe)
		return fmt.Errorf("rename current binary: %w", err)
	}
	if err := os.Rename(newExe, exe); err != nil {
		return fmt.Errorf("rename new binary into place: %w", err)
	}

	return reexecFn(exe, os.Args[1:], os.Environ())
}

// downloadTo streams body to path as an executable (0755), verifying its sha256 against want when want is
// non-empty. A write failure or checksum mismatch deletes the partial file.
func downloadTo(path string, body io.Reader, want string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	h := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(f, h), body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(path)
		return fmt.Errorf("write %s: %w", path, copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close %s: %w", path, closeErr)
	}
	if want == "" {
		return nil
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		_ = os.Remove(path)
		return fmt.Errorf("checksum mismatch: got %s want %s", got, want)
	}
	return nil
}
