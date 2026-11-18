package imports

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// retryAfter is how long a source that failed for a passing reason, such as a rate limit, waits to be read again.
const retryAfter = time.Minute

// errStopped ends a run whose import was cancelled meanwhile.
var errStopped = errors.New("import stopped")

// Run works through every open import until ctx ends, woken by a commit to imports and by the next ready import's
// expiry. A restart resumes each import where it stopped.
// shortcut: one import at a time across the instance, so a long one delays another workspace's; a worker per
// workspace once two run at once.
func (s *Service) Run(ctx context.Context, wake func() <-chan struct{}) {
	for {
		committed := wake()
		wait := s.Pass(ctx)
		if !sleep(ctx, wait, committed) {
			return
		}
	}
}

// Pass runs every open import once, as Run does on each wake, and returns how long until one needs looking at again;
// -1 is not until a commit.
func (s *Service) Pass(ctx context.Context) time.Duration {
	open, err := s.repo.Open(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "list open imports", "error", err)
		return retryAfter
	}
	wait := time.Duration(-1)
	soonest := func(d time.Duration) {
		if wait < 0 || d < wait {
			wait = d
		}
	}
	for _, imp := range open {
		if ctx.Err() != nil {
			return wait
		}
		if d, ok := s.step(ctx, imp); ok {
			soonest(d)
		}
	}
	return wait
}

// step moves one import on, returning when it next needs a look if it is still waiting.
func (s *Service) step(ctx context.Context, imp *Import) (time.Duration, bool) {
	ctx = identity.WithActor(ctx, identity.Actor{ID: imp.CreatedBy})
	log := s.log.With("import_id", imp.ID, "source", imp.Source)
	src, ok := s.sources[imp.Source]
	if !ok {
		s.fail(ctx, imp, fmt.Sprintf("source %s is no longer available", imp.Source))
		return 0, false
	}
	switch imp.State {
	case StateReady:
		left := ReadyFor - s.now().Sub(imp.UpdatedAt)
		if left > 0 {
			return left, true
		}
		if _, err := s.move(ctx, imp, StateReady, StateCancelled, "it waited 24 hours without being confirmed"); err != nil {
			log.WarnContext(ctx, "expire import", "error", err)
		}
		return 0, false
	case StateScanning:
		err := s.scan(ctx, src, imp)
		return s.settle(ctx, log, imp, err)
	case StateImporting:
		err := s.write(ctx, src, imp)
		return s.settle(ctx, log, imp, err)
	}
	return 0, false
}

// settle files a run's outcome: a passing failure retries later, any other fails the import.
func (s *Service) settle(ctx context.Context, log interface {
	WarnContext(ctx context.Context, msg string, args ...any)
}, imp *Import, err error) (time.Duration, bool) {
	if err == nil || errors.Is(err, errStopped) || ctx.Err() != nil {
		return 0, false
	}
	if errors.Is(err, apperrs.ErrRetryable) {
		log.WarnContext(ctx, "import paused; reading again shortly", "error", err)
		return retryAfter, true
	}
	log.WarnContext(ctx, "import failed", "error", err)
	s.fail(ctx, imp, failureReason(err))
	return 0, false
}

// failureReason is what a failed import shows: a sentinel's own words say what to fix, without the layers that
// wrapped them; anything else stays in the log.
func failureReason(err error) string {
	for _, known := range []error{apperrs.ErrInvalid, apperrs.ErrNotFound, apperrs.ErrForbidden, apperrs.ErrConflict, apperrs.ErrUnauthorized} {
		if !errors.Is(err, known) {
			continue
		}
		msg, prefix := err.Error(), known.Error()+": "
		if i := strings.LastIndex(msg, prefix); i >= 0 {
			return msg[i+len(prefix):]
		}
		break
	}
	return "the source could not be read; the server log has the details"
}

func (s *Service) fail(ctx context.Context, imp *Import, reason string) {
	current, err := s.repo.Get(ctx, imp.ID)
	if err != nil || current.State.Final() {
		return
	}
	if _, err := s.move(ctx, current, current.State, StateFailed, reason); err != nil {
		s.log.ErrorContext(ctx, "fail import", "import_id", imp.ID, "error", err)
	}
}

func (s *Service) job(imp *Import) Job {
	return Job{
		ImportID: imp.ID, WorkspaceID: imp.WorkspaceID, ProjectID: imp.ProjectID, From: imp.From,
		Credential: imp.Credential, Cursor: imp.Report.Cursor,
	}
}

// scan counts what the source holds and plans its people and statuses, then leaves the import ready.
func (s *Service) scan(ctx context.Context, src Source, imp *Import) error {
	found, err := src.Scan(ctx, s.job(imp))
	if err != nil {
		return err
	}
	if found.Tickets > MaxTickets {
		return fmt.Errorf("%w: it holds %d tickets, more than the %d one import takes", apperrs.ErrInvalid, found.Tickets, MaxTickets)
	}
	if found.Docs > MaxDocs {
		return fmt.Errorf("%w: it holds %d docs, more than the %d one import takes", apperrs.ErrInvalid, found.Docs, MaxDocs)
	}
	members, err := s.people.Members(ctx, imp.ProjectID)
	if err != nil {
		return fmt.Errorf("list members: %w", err)
	}
	cols, err := s.board.Columns(ctx, imp.ProjectID)
	if err != nil {
		return fmt.Errorf("list columns: %w", err)
	}
	next := *imp
	next.From = cmp.Or(found.From, imp.From)
	next.Total = found.Tickets + found.Docs
	next.Report.Tickets, next.Report.Comments, next.Report.Docs = found.Tickets, found.Comments, found.Docs
	next.Report.People = matchPeople(members, found.People)
	next.Report.Statuses = planStatuses(cols, found.Statuses)
	next.State, next.UpdatedAt = StateReady, s.now().UTC()
	if _, err := s.repo.Update(ctx, &next, StateScanning, progressEvent(&next)); err != nil {
		return fmt.Errorf("save scan: %w", err)
	}
	return nil
}

// write hands the source a sink and completes the import once the source is exhausted.
func (s *Service) write(ctx context.Context, src Source, imp *Import) error {
	members, err := s.people.Members(ctx, imp.ProjectID)
	if err != nil {
		return fmt.Errorf("list members: %w", err)
	}
	cols, err := s.board.Columns(ctx, imp.ProjectID)
	if err != nil {
		return fmt.Errorf("list columns: %w", err)
	}
	k := &sink{s: s, imp: imp, info: src.Info(), members: members, cols: cols}
	if err := src.Import(ctx, s.job(imp), k); err != nil {
		return err
	}
	if _, err := s.move(ctx, k.imp, StateImporting, StateCompleted, ""); err != nil {
		if errors.Is(err, apperrs.ErrConflict) {
			return errStopped
		}
		return err
	}
	return nil
}

// sleep waits out wait (forever when negative) or a commit, and reports false once ctx ends.
func sleep(ctx context.Context, wait time.Duration, committed <-chan struct{}) bool {
	var timeout <-chan time.Time
	if wait >= 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		timeout = timer.C
	}
	select {
	case <-ctx.Done():
		return false
	case <-committed:
		return true
	case <-timeout:
		return true
	}
}
