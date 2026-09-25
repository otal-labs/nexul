package install

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/release"
	"github.com/otal-labs/nexul/internal/platform/version"
)

func (h *Host) runUpgrade(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	target := fs.String("version", "", "release to upgrade or roll back to (default: the newest on this install's channel)")
	fs.Bool("yes", false, "accepted for scripts; upgrade never asks")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return h.Upgrade(ctx, *target)
}

// Upgrade moves the install to target, or the newest release on this binary's channel. A different release swaps
// this binary for that release's first and re-runs the upgrade with it, so the new version's compose file is what
// gets written. Rolling back is the same call with an older target.
func (h *Host) Upgrade(ctx context.Context, target string) error {
	if err := h.requireRoot(); err != nil {
		return err
	}
	prev, err := h.requireInstall()
	if err != nil {
		return err
	}
	tag, err := h.resolveTarget(ctx, target)
	if err != nil {
		return err
	}
	if tag != version.Version {
		return h.switchBinary(ctx, tag)
	}

	h.printf("Upgrading Nexul to %s\n\n", tag)
	o := Options{Dir: prev.Dir, Port: prev.intEnv("NEXUL_PORT", defaultPort), LogsPort: prev.intEnv("NEXUL_LOGS_PORT", defaultLogsPort)}
	if err := h.step("Files", func() (string, error) {
		_, err := h.writeStack(o, prev, tag)
		return o.Dir, err
	}); err != nil {
		return err
	}
	if err := h.startStack(ctx, o); err != nil {
		return err
	}
	h.printf("\nNexul %s is running. Runners update themselves when they reconnect.\n", tag)
	return nil
}

func (h *Host) resolveTarget(ctx context.Context, target string) (string, error) {
	if target != "" {
		return "v" + strings.TrimPrefix(target, "v"), nil
	}
	channel := version.Channel()
	if channel == "dev" {
		channel = "stable"
	}
	rel, err := release.New(release.Config{HTTP: h.HTTP, APIBase: h.ReleaseAPI}).Latest(ctx, channel)
	if err != nil {
		return "", fmt.Errorf("find the newest %s release: %w", channel, err)
	}
	return rel.Tag, nil
}

// switchBinary replaces this binary with tag's and re-executes the upgrade under it.
func (h *Host) switchBinary(ctx context.Context, tag string) error {
	exe, err := h.executable()
	if err != nil {
		return err
	}
	h.printf("Downloading nexul %s\n", tag)
	if err := h.releaseAsset(ctx, tag, h.assetName("nexul"), exe); err != nil {
		return err
	}
	return h.Reexec(exe, []string{exe, "upgrade", "--version", tag})
}

func (h *Host) runUninstall(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	purge := fs.Bool("purge", false, "also delete the install directory: the database, logs and stack checkouts")
	yes := fs.Bool("yes", false, "do not ask for confirmation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return h.Uninstall(ctx, *purge, *yes)
}

// Uninstall stops the stack and removes what install put on the host. The install directory stays unless purge is
// set, so `nexul install --dir <dir>` brings the same instance back.
func (h *Host) Uninstall(ctx context.Context, purge, yes bool) error {
	if err := h.requireRoot(); err != nil {
		return err
	}
	prev, err := h.requireInstall()
	if err != nil {
		return err
	}
	h.printf("This stops Nexul and removes the instance runner service. Stacks you deployed keep running.\n")
	if purge {
		h.printf("It also DELETES %s: the database, the logs and every stack checkout.\n", prev.Dir)
	}
	if !purge {
		h.printf("%s is kept; `nexul install --dir %s` brings this instance back.\n", prev.Dir, prev.Dir)
	}
	if err := h.confirm(yes); err != nil {
		return err
	}
	h.printf("\n")

	if err := h.step("Stack", func() (string, error) {
		return "stopped", h.compose(ctx, prev.Dir, "down", "--remove-orphans")
	}); err != nil {
		return err
	}
	if err := h.step("Runner", func() (string, error) { return "removed", h.removeRunner(ctx) }); err != nil {
		return err
	}
	if err := h.step("Files", func() (string, error) { return h.removeFiles(prev.Dir, purge) }); err != nil {
		return err
	}
	h.printf("\nNexul is uninstalled.\n")
	return nil
}

func (h *Host) confirm(yes bool) error {
	if yes {
		return nil
	}
	if !h.Interactive {
		return errors.New("pass --yes to uninstall without a terminal to confirm on")
	}
	answer, err := h.prompt("Continue? (y/N)", "N")
	if err != nil {
		return err
	}
	if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
		return errors.New("cancelled")
	}
	return nil
}

func (h *Host) removeFiles(dir string, purge bool) (string, error) {
	paths := []string{h.Paths.Config, filepath.Join(h.Paths.BinDir, "nexul")}
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("remove %s: %w", path, err)
		}
	}
	_ = os.Remove(filepath.Dir(h.Paths.Config)) // removes the config directory only when nothing else lives in it
	if !purge {
		return "kept " + dir, nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("remove %s: %w", dir, err)
	}
	return "deleted " + dir, nil
}

// Status prints the installed version, where it lives, and whether each part is running.
func (h *Host) Status(ctx context.Context) error {
	prev, err := h.requireInstall()
	if err != nil {
		return err
	}
	runner, _ := h.Exec.Run(ctx, "systemctl", "is-active", runnerService) // is-active exits non-zero for any state but active; its output is the answer
	services, err := h.Exec.Run(ctx, "docker", "compose", "--project-directory", prev.Dir, "ps", "--all", "--format", "{{.Service}}\t{{.State}}\t{{.Status}}")
	if err != nil {
		services = "unavailable: " + err.Error()
	}
	h.printf("Nexul %s\n", "v"+prev.Env["NEXUL_VERSION"])
	h.printf("  Directory  %s\n", prev.Dir)
	h.printf("  Web UI     %s\n", siteURL(h.PublicIP(), prev.intEnv("NEXUL_PORT", defaultPort)))
	h.printf("  Runner     %s (%s)\n", runner, runnerService)
	h.printf("  Command    %s\n", version.Version)
	h.printf("  Services\n")
	for _, line := range strings.Split(services, "\n") {
		h.printf("    %s\n", line)
	}
	return nil
}
