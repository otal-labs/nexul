package install

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/otal-labs/nexul/internal/platform/release"
	"github.com/otal-labs/nexul/internal/platform/version"
)

func (h *Host) runUpgrade(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	target := fs.String("version", "", "release to upgrade or roll back to (default: the newest on this install's channel)")
	detach := fs.Bool("detach", false, "run the upgrade outside the calling service and return at once")
	fs.Bool("yes", false, "accepted for scripts; upgrade never asks")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *detach {
		return h.detach(ctx, append([]string{"upgrade"}, args...))
	}
	return h.Upgrade(ctx, *target)
}

// Upgrade moves every unit on this machine to target, or to the newest release on this binary's channel. A
// different release swaps this binary for that release's first and re-runs the upgrade with it, so the new
// version's installer does the rest. Rolling back is the same call with an older target.
func (h *Host) Upgrade(ctx context.Context, target string) error {
	if err := h.checkUser(); err != nil {
		return err
	}
	units, err := h.loadUnits()
	if err != nil {
		return err
	}
	if len(units) == 0 {
		return errors.New("nothing of Nexul is installed on this machine; run nexul install")
	}
	tag, err := h.resolveTarget(ctx, target)
	if err != nil {
		return err
	}
	if tag != version.Version {
		return h.switchBinary(ctx, tag)
	}

	h.printf("Upgrading Nexul to %s\n\n", tag)
	fetched := map[string]string{}
	for _, u := range units {
		if err := h.step(u.Name, func() (string, error) { return h.upgradeUnit(ctx, u, tag, fetched) }); err != nil {
			return err
		}
	}
	if err := h.recordVersion(tag); err != nil {
		return err
	}
	h.printf("\nNexul %s is running.\n", tag)
	return nil
}

// upgradeUnit replaces a unit's binary with the target's and restarts it; a unit already there is left running.
// Each release binary is downloaded once and copied to every further unit of its kind.
func (h *Host) upgradeUnit(ctx context.Context, u Unit, tag string, fetched map[string]string) (string, error) {
	want := tag
	if u.Kind == kindLogs {
		want = h.OpenObserve.Version
	}
	if u.Version == want {
		return want + ", already", nil
	}
	if err := h.fetchBinary(ctx, u, tag, fetched); err != nil {
		return "", err
	}
	u.Version = want
	if err := h.saveUnit(u); err != nil {
		return "", err
	}
	if err := h.services().install(ctx, u); err != nil {
		return "", err
	}
	return u.Version, nil
}

func (h *Host) fetchBinary(ctx context.Context, u Unit, tag string, fetched map[string]string) error {
	if u.Kind == kindLogs {
		return h.installOpenObserve(ctx, u.Exec)
	}
	asset := h.assetName(binaryFor(u.Kind))
	if src, ok := fetched[asset]; ok {
		return copyFile(src, u.Exec)
	}
	if err := h.releaseAsset(ctx, tag, asset, u.Exec); err != nil {
		return err
	}
	fetched[asset] = u.Exec
	return nil
}

// recordVersion keeps the server install's .env in step with the release it now runs.
func (h *Host) recordVersion(tag string) error {
	prev, err := h.loadInstall()
	if err != nil || prev == nil || prev.Env == nil {
		return err
	}
	prev.Env["NEXUL_VERSION"] = strings.TrimPrefix(tag, "v")
	return writeEnvFile(filepath.Join(prev.Dir, ".env"), prev.Env)
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
	if len(args) > 0 && (args[0] == kindRunner || args[0] == kindAutomations) {
		return h.runUninstallHost(ctx, args[0], args[1:])
	}
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(h.Out)
	purge := fs.Bool("purge", false, "also delete the install directory: the database, logs and stack checkouts")
	yes := fs.Bool("yes", false, "do not ask for confirmation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return h.Uninstall(ctx, *purge, *yes)
}

// Uninstall removes every unit on this machine, the server's settings and the nexul command. The install directory
// stays unless purge is set, so `nexul install --dir <dir>` brings the same instance back.
func (h *Host) Uninstall(ctx context.Context, purge, yes bool) error {
	if err := h.checkUser(); err != nil {
		return err
	}
	units, err := h.loadUnits()
	if err != nil {
		return err
	}
	prev, err := h.loadInstall()
	if err != nil {
		return err
	}
	if len(units) == 0 && prev == nil {
		return errors.New("nexul is not installed on this host; run nexul install")
	}
	h.printUninstallPlan(prev, purge)
	if err := h.confirm(yes); err != nil {
		return err
	}
	h.printf("\n")

	// Hosts first, while the server they tell about their removal is still up.
	for _, u := range slices.Backward(units) {
		if err := h.step(u.Name, func() (string, error) { return h.removeUnit(ctx, u) }); err != nil {
			return err
		}
	}
	dir := ""
	if prev != nil {
		dir = prev.Dir
	}
	if err := h.step("Files", func() (string, error) { return h.removeFiles(dir, purge) }); err != nil {
		return err
	}
	// Kept data is still owned by the service user, so only a purge removes it.
	if purge && h.GOOS == "linux" {
		if err := h.step("User", func() (string, error) { return h.removeServiceUser(ctx) }); err != nil {
			return err
		}
	}
	h.printf("\nNexul is uninstalled.\n")
	return nil
}

func (h *Host) printUninstallPlan(prev *installed, purge bool) {
	h.printf("This stops and removes every Nexul service on this machine. Stacks you deployed keep running.\n")
	if prev != nil && purge {
		h.printf("It also DELETES %s and the database, the logs and every stack checkout.\n", prev.Dir)
	}
	if prev != nil && !purge {
		h.printf("The data is kept; `nexul install --dir %s` brings this instance back.\n", prev.Dir)
	}
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
	paths := []string{h.Paths.Config}
	// Windows cannot delete the executable that is running this uninstall; the summary names it instead.
	if h.GOOS != "windows" {
		paths = append(paths, filepath.Join(h.Paths.BinDir, h.commandName()))
	}
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("remove %s: %w", path, err)
		}
	}
	_ = os.Remove(filepath.Dir(h.Paths.Config)) // removes the config directory only when nothing else lives in it
	_ = os.Remove(h.Paths.UnitRoot)             // likewise, once every unit directory is gone
	kept := ""
	if h.GOOS == "windows" {
		kept = "; delete " + filepath.Join(h.Paths.BinDir, h.commandName()) + " yourself"
	}
	if dir == "" {
		return "removed" + kept, nil
	}
	if !purge {
		return "kept " + dir + kept, nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("remove %s: %w", dir, err)
	}
	return "deleted " + dir + kept, nil
}

// Status prints every Nexul unit on this machine with its kind, state and version, plus the server install's
// directory and address when there is one.
func (h *Host) Status(ctx context.Context) error {
	// Unit records hold each service's credentials, so on a server only root can read them.
	if err := h.checkUser(); err != nil {
		return err
	}
	units, err := h.loadUnits()
	if err != nil {
		return err
	}
	prev, err := h.loadInstall()
	if err != nil {
		return err
	}
	if len(units) == 0 && prev == nil {
		return errors.New("nexul is not installed on this host; run nexul install")
	}
	h.printf("nexul %s\n", version.Version)
	if prev != nil {
		h.printf("  Directory  %s\n", prev.Dir)
		h.printf("  Web UI     %s\n", siteURL(h.PublicIP(), prev.intEnv("NEXUL_PORT", defaultPort)))
	}
	h.printf("\n")
	tw := tabwriter.NewWriter(h.Out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "  KIND\tNAME\tSTATE\tVERSION") // terminal writes have nowhere to report failure
	for _, u := range units {
		_, _ = fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", u.Kind, u.Name, h.services().state(ctx, u), u.Version)
	}
	_ = tw.Flush() // likewise
	return nil
}
