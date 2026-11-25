package pairing

import (
	"context"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/shelljob"
)

func (f *fakeRepo) InsertShellJob(_ context.Context, job shelljob.Job, evt eventbus.OutboxEvent) error {
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
	if !ok || job.Status != shelljob.Queued {
		return false, nil
	}
	job.Status, job.StartedAt = shelljob.Running, &at
	f.jobs[id] = job
	f.outbox = append(f.outbox, evt)
	return true, nil
}

func (f *fakeRepo) FinishShellJob(_ context.Context, job shelljob.Job, evt eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, ok := f.jobs[job.ID]
	if !ok || (stored.Status != shelljob.Queued && stored.Status != shelljob.Running) {
		return nil
	}
	job.StartedAt = stored.StartedAt
	f.jobs[job.ID] = job
	f.outbox = append(f.outbox, evt)
	return nil
}

func (f *fakeRepo) GetShellJob(_ context.Context, id string) (*shelljob.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, ok := f.jobs[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &job, nil
}

func (f *fakeRepo) ListShellJobs(_ context.Context, computerID, requesterID string, w paging.Window) ([]shelljob.Job, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []shelljob.Job
	for _, j := range f.jobs {
		if j.ComputerID == computerID && (requesterID == "" || j.RequesterID == requesterID) {
			j.Output = ""
			out = append(out, j)
		}
	}
	slices.SortFunc(out, func(a, b shelljob.Job) int { return strings.Compare(b.ID, a.ID) })
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
		if j.Status == shelljob.Queued || j.Status == shelljob.Running {
			j.Status, j.EndedAt = shelljob.Failed, &at
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

func (f *fakeRepo) shellJob(id string) shelljob.Job {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.jobs[id]
}
