//go:build !windows

package runner

import "syscall"

// reexec replaces the running process image with exe; on success it never returns.
func reexec(exe string, args, env []string) error {
	return syscall.Exec(exe, append([]string{exe}, args...), env)
}
