//go:build linux || darwin

package install

import (
	"os"

	"golang.org/x/sys/unix"
)

// hideEcho turns echo off on f's terminal and returns the call that restores it.
func hideEcho(f *os.File) (func(), error) {
	fd := int(f.Fd())
	old, err := unix.IoctlGetTermios(fd, getTermios)
	if err != nil {
		return nil, err
	}
	quiet := *old
	quiet.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(fd, setTermios, &quiet); err != nil {
		return nil, err
	}
	return func() { _ = unix.IoctlSetTermios(fd, setTermios, old) }, nil // a terminal that went away has nothing to restore
}
