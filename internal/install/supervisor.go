package install

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// childSpec is one process the supervisor keeps running.
type childSpec struct {
	name string
	// command makes a fresh command for each start.
	command func() *exec.Cmd
	// ends is asked each time the child exits on its own; true ends the supervisor instead of restarting it.
	ends func() bool
}

// childSupervisor keeps its children running in the foreground, for a host with no service manager (ADR 0163). A child
// that re-execs itself in place keeps its pid, so to the supervisor it never exited.
type childSupervisor struct {
	log      *slog.Logger
	children []childSpec
	// delay is how long an exited child waits to start again, as systemd's RestartSec does.
	delay time.Duration
	// grace is how long a child gets to stop after it is signalled before it is killed.
	grace time.Duration

	mu   sync.Mutex
	live map[string]*os.Process
}

// run starts every child and keeps it running until a signal arrives on stop, or a child's ends reports true. It
// then signals the rest and waits for them, and reports whether a child ended it.
func (s *childSupervisor) run(ctx context.Context, stop <-chan os.Signal) bool {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var ended atomic.Bool
	var wg sync.WaitGroup
	s.live = map[string]*os.Process{}
	for _, c := range s.children {
		wg.Go(func() {
			if s.keep(ctx, c) {
				ended.Store(true)
				cancel()
			}
		})
	}
	select {
	case sig := <-stop:
		s.log.Info("stopping its children", "signal", sig.String())
		cancel()
		s.signalAll(sig)
	case <-ctx.Done():
		s.signalAll(syscall.SIGTERM)
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(s.grace):
		s.log.Warn("a child did not stop in time; killing it")
		s.signalAll(syscall.SIGKILL)
		<-done
	}
	return ended.Load()
}

// keep runs c until ctx ends, starting it again after each exit; it reports whether c's ends asked to stop.
func (s *childSupervisor) keep(ctx context.Context, c childSpec) bool {
	for {
		err := s.runOnce(ctx, c)
		if ctx.Err() != nil {
			return false
		}
		if c.ends != nil && c.ends() {
			s.log.Info("child exited and asked the supervisor to end", "child", c.name)
			return true
		}
		s.log.Warn("child exited; starting it again", "child", c.name, "error", err, "in", s.delay.String())
		timer := time.NewTimer(s.delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false
		case <-timer.C:
		}
	}
}

// runOnce starts c and waits for it to exit.
func (s *childSupervisor) runOnce(ctx context.Context, c childSpec) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := c.command()
	s.mu.Lock()
	if err := cmd.Start(); err != nil {
		s.mu.Unlock()
		return err
	}
	s.live[c.name] = cmd.Process
	if ctx.Err() != nil { // a stop that came after the check above found nothing live to signal
		_ = cmd.Process.Signal(syscall.SIGTERM) // fails only when it already exited, which Wait reports
	}
	s.mu.Unlock()
	s.log.Info("child started", "child", c.name, "pid", cmd.Process.Pid)
	err := cmd.Wait()
	s.mu.Lock()
	delete(s.live, c.name)
	s.mu.Unlock()
	return err
}

func (s *childSupervisor) signalAll(sig os.Signal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.live {
		_ = p.Signal(sig) // fails only when it already exited, which its Wait reports
	}
}
