package install

import (
	"os"

	"golang.org/x/sys/windows"
)

// hideEcho turns echo off on f's console and returns the call that restores it.
func hideEcho(f *os.File) (func(), error) {
	h := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return nil, err
	}
	if err := windows.SetConsoleMode(h, mode&^windows.ENABLE_ECHO_INPUT); err != nil {
		return nil, err
	}
	return func() { _ = windows.SetConsoleMode(h, mode) }, nil // a console that went away has nothing to restore
}
