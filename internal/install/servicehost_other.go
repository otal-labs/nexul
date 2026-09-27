//go:build !windows

package install

import "errors"

// serviceHost exists only on Windows, where services run through it; systemd and launchd run units directly.
func (h *Host) serviceHost(string) error {
	return errors.New("service-host runs units under the Windows Service Control Manager; this is not Windows")
}
