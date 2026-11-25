package pairing

import (
	"context"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

func (f *fakeRepo) InsertShellJob(_ context.Context, job ShellJob, evt eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs[job.ID] = job
	f.outbox = append(f.outbox, evt)
	return nil
}

func (f *fakeRepo) StartShellJob(_ context.Context, id string, at time.Time, evt eventbus.OutboxEvent) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, ok := f.jobs[id]
	if !ok || job.Status != ShellQueued {
		return false, nil
	}
	job.Status, job.StartedAt = ShellRunning, &at
	f.jobs[id] = job
	f.outbox = append(f.outbox, evt)
	return true, nil
}

func (f *fakeRepo) FinishShellJob(_ context.Context, job ShellJob, evt eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, ok := f.jobs[job.ID]
	if !ok || (stored.Status != ShellQueued && stored.Status != ShellRunning) {
		return nil
	}
	job.StartedAt = stored.StartedAt
	f.jobs[job.ID] = job
	f.outbox = append(f.outbox, evt)
	return nil
}

func (f *fakeRepo) GetShellJob(_ context.Context, id string) (*ShellJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, ok := f.jobs[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &job, nil
}

func (f *fakeRepo) ListShellJobs(_ context.Context, computerID, requesterID string, w paging.Window) ([]ShellJob, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []ShellJob
	for _, j := range f.jobs {
		if j.ComputerID == computerID && (requesterID == "" || j.RequesterID == requesterID) {
			j.Output = ""
			out = append(out, j)
		}
	}
	slices.SortFunc(out, func(a, b ShellJob) int { return strings.Compare(b.ID, a.ID) })
	total := len(out)
	out = out[min(w.Offset, total):min(w.Offset+w.Limit, total)]
	return out, total, nil
}

func (f *fakeRepo) PurgeShellOutput(_ context.Context, before time.Time, limit int) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for id, j := range f.jobs {
		if n < int64(limit) && j.Output != "" && j.EndedAt != nil && j.EndedAt.Before(before) {
			j.Output, j.OutputPurged = "", true
			f.jobs[id] = j
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) FailUnfinishedShellJobs(_ context.Context, at time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for id, j := range f.jobs {
		if j.Status == ShellQueued || j.Status == ShellRunning {
			j.Status, j.EndedAt = ShellFailed, &at
			f.jobs[id] = j
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) SetComputerShell(_ context.Context, userID, computerID string, enabled bool, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.computers[computerID]
	if !ok || c.UserID != userID {
		return apperrs.ErrNotFound
	}
	c.ShellEnabled = enabled
	f.computers[computerID] = c
	return nil
}

func (f *fakeRepo) shellJob(id string) ShellJob {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.jobs[id]
}
