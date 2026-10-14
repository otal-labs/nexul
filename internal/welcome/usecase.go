package welcome

import (
	"context"
	"fmt"
	"slices"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/welcomestep"
)

// Service is the welcome use-case layer.
type Service struct {
	repo  Repo
	seams Seams
	now   func() time.Time
}

// NewService wires the welcome use-cases over their records and the other domains' reads.
func NewService(repo Repo, seams Seams) *Service {
	return &Service{repo: repo, seams: seams, now: time.Now}
}

// Progress is userID's own progress, for the person themselves.
func (s *Service) Progress(ctx context.Context, userID string) (Progress, error) {
	if userID == "" {
		return Progress{}, apperrs.ErrUnauthorized
	}
	all, err := s.ProgressFor(ctx, []string{userID})
	if err != nil {
		return Progress{}, err
	}
	return all[userID], nil
}

// ProgressFor works out the progress of every person in userIDs with one statement per seam, whatever their number.
// It checks nothing: a caller shows it only beside an account it already may read.
func (s *Service) ProgressFor(ctx context.Context, userIDs []string) (map[string]Progress, error) {
	ticked, err := s.seams.TickedSteps(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("read ticked steps: %w", err)
	}
	sets := []PeopleWith{s.seams.GitHub, s.seams.Computer, s.seams.InAProject, s.seams.DeployServer, s.seams.FirstRunDone}
	answers := make([]map[string]bool, len(sets))
	for i, set := range sets {
		if answers[i], err = set(ctx, userIDs); err != nil {
			return nil, fmt.Errorf("read welcome facts: %w", err)
		}
	}
	records, err := s.repo.Records(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("read welcome records: %w", err)
	}
	byUser := make(map[string]map[string]Record, len(userIDs))
	for _, r := range records {
		if byUser[r.UserID] == nil {
			byUser[r.UserID] = map[string]Record{}
		}
		byUser[r.UserID][r.Step] = r
	}
	out := make(map[string]Progress, len(userIDs))
	for _, id := range userIDs {
		f := Facts{Ticked: ticked[id], GitHub: answers[0][id], Computer: answers[1][id], InAProject: answers[2][id],
			DeployServer: answers[3][id], FirstRunDone: answers[4][id]}
		out[id] = progressOf(f, byUser[id])
	}
	return out, nil
}

// Skip silences one of userID's own steps until it is given to them again (ADR 0149).
func (s *Service) Skip(ctx context.Context, userID, step string) (Progress, error) {
	if err := s.ownStep(ctx, userID, step); err != nil {
		return Progress{}, err
	}
	at := s.now().UTC()
	if err := s.repo.SetSkipped(ctx, userID, step, &at, changed(userID)...); err != nil {
		return Progress{}, fmt.Errorf("skip %s for %s: %w", step, userID, err)
	}
	return s.Progress(ctx, userID)
}

// Unskip takes back one of userID's own skips.
func (s *Service) Unskip(ctx context.Context, userID, step string) (Progress, error) {
	if err := s.ownStep(ctx, userID, step); err != nil {
		return Progress{}, err
	}
	if err := s.repo.SetSkipped(ctx, userID, step, nil, changed(userID)...); err != nil {
		return Progress{}, fmt.Errorf("unskip %s for %s: %w", step, userID, err)
	}
	return s.Progress(ctx, userID)
}

// FinishFirstRun records that userID's Welcome wizard had its first run, so it never opens by itself again.
func (s *Service) FinishFirstRun(ctx context.Context, userID string) (Progress, error) {
	if userID == "" {
		return Progress{}, apperrs.ErrUnauthorized
	}
	if err := s.seams.MarkFirstRun(ctx, userID); err != nil {
		return Progress{}, fmt.Errorf("finish the first run of %s: %w", userID, err)
	}
	if err := s.repo.Announce(ctx, changed(userID)...); err != nil {
		return Progress{}, fmt.Errorf("announce the first run of %s: %w", userID, err)
	}
	return s.Progress(ctx, userID)
}

// ownStep refuses a step key that isn't one of userID's steps.
func (s *Service) ownStep(ctx context.Context, userID, step string) error {
	if userID == "" {
		return apperrs.ErrUnauthorized
	}
	ticked, err := s.seams.TickedSteps(ctx, []string{userID})
	if err != nil {
		return fmt.Errorf("read ticked steps of %s: %w", userID, err)
	}
	if !slices.Contains(ticked[userID], step) {
		return fmt.Errorf("%w: %q is not one of your welcome steps; GET /api/me/welcome lists them", apperrs.ErrNotFound, step)
	}
	return nil
}

// give hands steps to people again: each skip goes, and after a person's first run the step is marked given.
func (s *Service) give(ctx context.Context, userIDs, steps []string) error {
	if len(userIDs) == 0 {
		return nil
	}
	if len(steps) == 0 {
		return s.repo.Announce(ctx, changed(userIDs...)...)
	}
	firstRun, err := s.seams.FirstRunDone(ctx, userIDs)
	if err != nil {
		return err
	}
	at := s.now().UTC()
	var gifts []Gift
	for _, id := range userIDs {
		for _, step := range steps {
			g := Gift{UserID: id, Step: step}
			if firstRun[id] {
				g.GivenAt = &at
			}
			gifts = append(gifts, g)
		}
	}
	return s.repo.Give(ctx, gifts, changed(userIDs...)...)
}

// LightsDot is the dot rule: a step left lights it, except projects, which a person can't do alone, until given.
func LightsDot(s Step) bool {
	return s.State == Left && (s.Key != welcomestep.Projects || s.Given)
}

// progressOf works out a person's progress from what other domains hold and the records kept for their steps.
func progressOf(f Facts, records map[string]Record) Progress {
	p := Progress{FirstRunDone: f.FirstRunDone, Steps: []Step{}, InAProject: f.InAProject}
	done := map[string]bool{
		welcomestep.GitHub: f.GitHub, welcomestep.Computer: f.Computer, welcomestep.Projects: f.InAProject,
		welcomestep.DeployServer: f.DeployServer, welcomestep.Phone: records[welcomestep.Phone].DoneAt != nil,
	}
	for _, key := range welcomestep.All {
		if !slices.Contains(f.Ticked, key) {
			continue
		}
		p.Steps = append(p.Steps, stepOf(key, done[key], records[key]))
		p.Dot = p.Dot || LightsDot(p.Steps[len(p.Steps)-1])
	}
	return p
}

func stepOf(key string, done bool, r Record) Step {
	s := Step{Key: key, State: Left, Given: r.GivenAt != nil}
	if done {
		s.State = Done
		if key == welcomestep.Phone {
			s.At = r.DoneAt
		}
		return s
	}
	if r.SkippedAt != nil {
		s.State, s.At = Skipped, r.SkippedAt
	}
	return s
}
