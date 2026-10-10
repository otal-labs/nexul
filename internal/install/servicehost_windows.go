//go:build windows

package install

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

// serviceHost runs a unit under the SCM: it starts the unit's binary with its env and stops it when asked. It
// kills only that process, never its children, so a removal the unit started detached outlives it.
func (h *Host) serviceHost(name string) error {
	if name == computerUnit {
		return h.logonTaskHost()
	}
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

var freeConsole = windows.NewLazySystemDLL("kernel32.dll").NewProc("FreeConsole")

// restartDelay is how long a computer's runner that failed waits to start again, as systemd's RestartSec does.
const restartDelay = 5 * time.Second

// logonTaskHost keeps a computer's runner running inside the person's logon task. Task Scheduler restarts only a task
// that fails to start, so this starts the runner again after it fails, and ends with its clean exit, which is how a
// removed runner leaves. The runner lives in a job that closes with this process, so ending the task ends it too.
func (h *Host) logonTaskHost() error {
	_, _, _ = freeConsole.Call() // the task opens a console window on the person's screen, with nothing to show in it
	p, err := h.computerPerson()
	if err != nil {
		return err
	}
	h.Paths = h.computerPaths(p)
	u, err := h.loadUnit(computerUnit)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("this account has no computer runner installed")
	}
	log, err := os.OpenFile(filepath.Join(u.Dir, "service.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = log.Close() }() // a log that fails to close has nowhere to report it
	job, err := runnerJob()
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(job) }() // closing it ends a runner still in it
	for {
		err := runInJob(job, *u, log)
		if err == nil {
			return nil
		}
		_, _ = fmt.Fprintf(log, "%s exited: %v; starting it again in %s\n", u.Name, err, restartDelay) // best-effort diagnostic
		time.Sleep(restartDelay)
	}
}

// runnerJob is a job whose processes end when its last handle, this process's, closes; a removal the runner starts
// may break away from it, to outlive the task it ends.
func runnerJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, fmt.Errorf("create the runner's job: %w", err)
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
		LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_BREAKAWAY_OK,
	}}
	_, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	if err != nil {
		return 0, errors.Join(fmt.Errorf("set up the runner's job: %w", err), windows.CloseHandle(job))
	}
	return job, nil
}

// runInJob runs the unit's binary in job, with no window, until it exits. NEXUL_SUPERVISED tells the runner to apply
// an update by exiting for this to start the new binary, rather than starting it itself outside the job.
func runInJob(job windows.Handle, u Unit, log *os.File) error {
	cmd := exec.Command(u.Exec)
	cmd.Dir = u.WorkDir
	cmd.Stdout, cmd.Stderr = log, log
	cmd.Env = append(os.Environ(), "NEXUL_SUPERVISED=1")
	for k, v := range u.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	if err := cmd.Start(); err != nil {
		return err
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err == nil {
		err = errors.Join(windows.AssignProcessToJobObject(job, process), windows.CloseHandle(process))
	}
	if err != nil {
		return errors.Join(fmt.Errorf("put %s in its job: %w", u.Name, err), cmd.Process.Kill(), cmd.Wait())
	}
	return cmd.Wait()
}
