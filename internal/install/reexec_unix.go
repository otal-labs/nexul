//go:build !windows

package install

import (
	"os"
	"syscall"
)

// reexec replaces this process with the binary at path, so an upgrade continues under the new version.
func reexec(path string, args []string) error {
	return syscall.Exec(path, args, os.Environ())
}
