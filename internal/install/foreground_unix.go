//go:build !windows

package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const supervisorLock = "supervisor.lock"

// lockSupervisor makes this process the one supervisor of the computer's runner in dir: a second one would start a
// second runner on the same credential. The lock dies with its holder, and its file names the holder's pid.
func lockSupervisor(dir string) (release func(), err error) {
	f, err := os.OpenFile(filepath.Join(dir, supervisorLock), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open the supervisor's lock: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		holder, _ := supervisorPID(dir)
		return nil, errors.Join(fmt.Errorf("this computer's runner is already supervised, by pid %d: stop it first, or leave it running", holder), f.Close())
	}
	if err := f.Truncate(0); err != nil {
		return nil, errors.Join(fmt.Errorf("write the supervisor's pid: %w", err), f.Close())
	}
	if _, err := f.WriteString(strconv.Itoa(os.Getpid())); err != nil {
		return nil, errors.Join(fmt.Errorf("write the supervisor's pid: %w", err), f.Close())
	}
	return func() { _ = f.Close() }, nil // closing the file drops the lock; nothing else to flush
}

// supervisorPID is the pid of the supervisor holding dir's lock, and false when none does.
func supervisorPID(dir string) (int, bool) {
	f, err := os.OpenFile(filepath.Join(dir, supervisorLock), os.O_RDONLY, 0)
	if err != nil {
		return 0, false
	}
	defer func() { _ = f.Close() }() // read-only
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err == nil {
		return 0, false
	}
	data, err := os.ReadFile(filepath.Join(dir, supervisorLock))
	if err != nil {
		return 0, true
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data))) // a holder that has not written its pid yet reads as 0
	return pid, true
}

// stopSupervisor asks the supervisor with this pid to stop.
func stopSupervisor(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}
