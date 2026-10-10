package install

import (
	"context"
	"crypto/rand"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// detach re-runs `nexul <args>` outside the calling service's process tree and returns at once, so a runner or
// automations host can remove or upgrade itself without being killed halfway by its own service stopping. The
// re-run has no terminal to confirm on, so it gets --yes.
func (h *Host) detach(ctx context.Context, args []string) error {
	exe, err := h.executable()
	if err != nil {
		return err
	}
	child := slices.DeleteFunc(slices.Clone(args), isDetachFlag)
	child = append(child, "--yes")
	if h.GOOS == "linux" && h.Getuid() != 0 {
		// The sudoers drop-in allows the call exactly as it was made, --detach included.
		_, err := h.Exec.Run(ctx, "sudo", append([]string{"-n", exe}, args...)...)
		return err
	}
	if h.GOOS == "linux" {
		unit := "nexul-" + args[0] + "-" + strings.ToLower(rand.Text()[:8])
		if _, err := h.Exec.Run(ctx, "systemd-run", append([]string{"--unit", unit, "--collect", "--quiet", exe}, child...)...); err != nil {
			return err
		}
		h.printf("Started %s; journalctl -u %s shows its progress.\n", unit, unit)
		return nil
	}
	log := filepath.Join(h.Paths.UnitRoot, "nexul-"+args[0]+".log")
	if err := h.StartDetached(exe, child, log); err != nil {
		return fmt.Errorf("start nexul %s: %w", args[0], err)
	}
	h.printf("Started nexul %s in the background; %s shows its progress.\n", args[0], log)
	return nil
}

func isDetachFlag(arg string) bool {
	name, _, _ := strings.Cut(strings.TrimLeft(arg, "-"), "=")
	return strings.HasPrefix(arg, "-") && name == "detach"
}
