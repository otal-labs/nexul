//go:build windows

package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows/svc"
)

// serviceHost runs a unit under the SCM: it starts the unit's binary with its env and stops it when asked. It
// kills only that process, never its children, so a removal the unit started detached outlives it.
func (h *Host) serviceHost(name string) error {
	u, err := h.loadUnit(name)
	if err != nil {
		return err
	}
	if u == nil {
		return fmt.Errorf("no unit named %s is installed", name)
	}
	return svc.Run(u.Name, &supervisor{u: *u})
}

type supervisor struct{ u Unit }

// Execute follows the SCM protocol; a unit that exits on its own ends the service with exit code 1, which the
// service's failure actions restart.
func (s *supervisor) Execute(_ []string, req <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	log, err := os.OpenFile(filepath.Join(s.u.Dir, "service.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return false, 1
	}
	defer func() { _ = log.Close() }() // a log that fails to close has nowhere to report it
	cmd := exec.Command(s.u.Exec)
	cmd.Dir = s.u.WorkDir
	cmd.Stdout, cmd.Stderr = log, log
	cmd.Env = os.Environ()
	for k, v := range s.u.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if err := cmd.Start(); err != nil {
		_, _ = fmt.Fprintf(log, "start %s: %v\n", s.u.Exec, err) // best-effort diagnostic
		return false, 1
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case err := <-exited:
			_, _ = fmt.Fprintf(log, "%s exited: %v\n", s.u.Name, err) // best-effort diagnostic
			return false, 1
		case c := <-req:
			if c.Cmd == svc.Interrogate {
				status <- c.CurrentStatus
				continue
			}
			if c.Cmd != svc.Stop && c.Cmd != svc.Shutdown {
				continue
			}
			status <- svc.Status{State: svc.StopPending}
			_ = cmd.Process.Kill() // fails only when it already exited, which the wait below sees
			<-exited
			return false, 0
		}
	}
}
