package pairing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/jsonx"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// Runners is the runner domain as pairing reaches it: a computer's personal runner (ADR 0146). Pairing never writes a
// runner, and the runner domain never writes a computer.
type Runners interface {
	// EnrollComputer mints the one-time code that enrolls userID's personal runner for computerID.
	EnrollComputer(ctx context.Context, userID, computerID string) (Enrollment, error)
	// ComputerRunner returns the runner that reaches computerID, or ErrNotFound while none is enrolled.
	ComputerRunner(ctx context.Context, computerID string) (ComputerRunner, error)
	// PairingToken asks computerID's connected runner for a one-time T3 Code pairing token minted on the computer.
	PairingToken(ctx context.Context, computerID string) (string, error)
	// RetireComputerRunner revokes the runner that reaches computerID, which then uninstalls itself, and voids any
	// unused code for it; a computer with no runner has nothing to retire.
	RetireComputerRunner(ctx context.Context, computerID string) error
}

const (
	// repairWindow is how close to its end a computer's session is replaced through its runner.
	repairWindow = 7 * 24 * time.Hour
	// pairTimeout bounds one pairing through a runner: the token's mint on the computer, then its exchange.
	pairTimeout = time.Minute
	diskGranule = 1 << 30
	// snapshotTimeout bounds reading a computer's providers and projects from its T3 Code for its facts.
	snapshotTimeout = 30 * time.Second
)

// Enrollment is the signed token carrying a personal runner's one-time code, and the command that installs it.
type Enrollment struct {
	Token     string          `json:"token"`
	ExpiresAt time.Time       `json:"expires_at"`
	Commands  InstallCommands `json:"commands"`
}

// InstallCommands are the installer one-liners per shell; Windows stays empty until computers install there.
type InstallCommands struct {
	Unix    string `json:"unix"`
	Windows string `json:"windows"`
}

// ComputerEnrollment is a computer waiting for its runner, with the command that adds it.
type ComputerEnrollment struct {
	Computer Computer `json:"computer"`
	Enrollment
}

// ComputerRunner is the state of the personal runner that reaches a computer.
type ComputerRunner struct {
	Connected bool      `json:"connected"`
	LastSeen  time.Time `json:"last_seen"`
}

// AddComputer makes a computer waiting for its runner and the command that installs the runner there. It takes no
// permission: anyone signed in adds their own computers, a Restricted member included. An empty name becomes the
// hostname the computer reports when its runner enrolls.
func (s *Service) AddComputer(ctx context.Context, userID, name string) (*ComputerEnrollment, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	runners, err := s.runnerSeam()
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	now := s.now().UTC()
	computer := Computer{ID: ids.New(), UserID: userID, Kind: harness.KindT3Code, Name: name, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.SaveComputer(ctx, computer); err != nil {
		return nil, fmt.Errorf("save computer: %w", err)
	}
	enrollment, err := runners.EnrollComputer(ctx, userID, computer.ID)
	if err != nil {
		// A computer no code can ever reach would wait forever, so it goes with the failure.
		return nil, errors.Join(err, s.repo.DeleteComputer(ctx, userID, computer.ID))
	}
	s.notifyComputersChanged(userID)
	return &ComputerEnrollment{Computer: computer, Enrollment: enrollment}, nil
}

// EnrollComputer mints a fresh command for one of the caller's computers that has no runner yet, for one whose
// last code expired before the install ran.
func (s *Service) EnrollComputer(ctx context.Context, userID, computerID string) (*ComputerEnrollment, error) {
	computer, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	runners, err := s.runnerSeam()
	if err != nil {
		return nil, err
	}
	enrollment, err := runners.EnrollComputer(ctx, userID, computer.ID)
	if err != nil {
		return nil, err
	}
	if err := s.forgetRunnerSession(ctx, computer); err != nil {
		return nil, err
	}
	computer.BearerToken = ""
	return &ComputerEnrollment{Computer: *computer, Enrollment: enrollment}, nil
}

// forgetRunnerSession drops the session of a computer reached through a runner that is gone: its runner ended every
// Nexul session in T3 Code on the way out, so the new runner pairs it afresh. A tunnel computer keeps its session.
func (s *Service) forgetRunnerSession(ctx context.Context, c *Computer) error {
	if !c.Paired() || c.ServerURL != runnerAddress(c.ID) {
		return nil
	}
	c.BearerToken, c.TokenExpiresAt, c.UpdatedAt = "", time.Time{}, s.now().UTC()
	if err := s.repo.SaveComputer(ctx, *c); err != nil {
		return fmt.Errorf("forget the session of computer %s: %w", c.ID, err)
	}
	s.notifyComputersChanged(c.UserID)
	return nil
}

// RenameComputer renames one of the caller's computers.
func (s *Service) RenameComputer(ctx context.Context, userID, computerID, name string) (*Computer, error) {
	computer, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if computer.Name, err = validateName(name); err != nil {
		return nil, &FieldError{Field: "name", Err: err}
	}
	computer.UpdatedAt = s.now().UTC()
	if err := s.repo.SaveComputer(ctx, *computer); err != nil {
		return nil, fmt.Errorf("save computer %s: %w", computer.ID, err)
	}
	s.notifyComputersChanged(userID)
	computer.BearerToken = ""
	return computer, nil
}

// HandleRunnerChanged is the runner.personal_changed consumer: a computer still unnamed when its runner enrolls takes
// the hostname the computer reported.
func (s *Service) HandleRunnerChanged(ctx context.Context, ev eventbus.Event) error {
	var p struct {
		ComputerID string `json:"computer_id"`
		UserID     string `json:"user_id"`
		State      string `json:"state"`
		Hostname   string `json:"hostname"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	if p.State != "enrolled" || strings.TrimSpace(p.Hostname) == "" {
		return nil
	}
	computer, err := s.ownComputer(ctx, p.UserID, p.ComputerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if computer.Name != "" {
		return nil
	}
	computer.Name = strings.TrimSpace(p.Hostname)
	computer.UpdatedAt = s.now().UTC()
	if err := s.repo.SaveComputer(ctx, *computer); err != nil {
		return fmt.Errorf("name computer %s: %w", computer.ID, err)
	}
	s.notifyComputersChanged(p.UserID)
	return nil
}

// HandleFactsReported stores what a computer's runner reported, and pairs the computer through its runner first when its
// T3 Code answers and its session ends within repairWindow.
func (s *Service) HandleFactsReported(ctx context.Context, ev eventbus.Event) error {
	var p struct {
		ComputerID string `json:"computer_id"`
		UserID     string `json:"user_id"`
		Facts      Facts  `json:"facts"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	computer, err := s.ownComputer(ctx, p.UserID, p.ComputerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	// Whole GiB only: free space moves by a few bytes between reports, and a report that changes nothing writes nothing.
	p.Facts.FreeDiskBytes -= p.Facts.FreeDiskBytes % diskGranule
	if p.Facts.T3.State == t3Answering && (!computer.Paired() || !computer.TokenExpiresAt.After(s.now().Add(repairWindow))) {
		s.pairOnReport(ctx, *computer, p.Facts.Hostname)
	}
	return s.recordFacts(ctx, p.UserID, p.ComputerID, &p.Facts)
}

// pairOnReport pairs c through its runner; a failure is the row's pair_error, never the report's.
func (s *Service) pairOnReport(ctx context.Context, c Computer, hostname string) {
	if c.Name == "" {
		c.Name = strings.TrimSpace(hostname)
	}
	ctx, cancel := context.WithTimeout(ctx, pairTimeout)
	defer cancel()
	paired, err := s.pairThroughRunner(ctx, c)
	if err != nil {
		logging.FromCtx(ctx).Warn("pairing through the runner failed", "computer_id", c.ID, "error", err)
		return
	}
	logging.FromCtx(ctx).Info("computer paired through its runner", "computer_id", paired.ID, "kind", paired.Kind, "version", paired.HarnessVersion)
}

// recordFacts stores a computer's facts when they changed, and only then publishes computer.facts_changed. reported
// replaces the runner's part (nil keeps it); what T3 Code lists is read again unless the report says it is not answering.
func (s *Service) recordFacts(ctx context.Context, userID, computerID string, reported *Facts) error {
	computer, err := s.ownComputer(ctx, userID, computerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var next Facts
	if computer.Facts != nil {
		next = *computer.Facts
	}
	if reported != nil {
		providers, projects := next.Providers, next.Projects
		next = *reported
		next.Providers, next.Projects = providers, projects
	}
	if reported == nil || reported.T3.State == t3Answering {
		s.readT3Snapshot(ctx, userID, computerID, &next)
	}
	same, err := sameFacts(computer.Facts, next)
	if err != nil || same {
		return err
	}
	at := s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicFactsChanged, Payload: FactsChangedEvent{
		ComputerID: computerID, UserID: userID, FactsAt: at, MembersOnly: true,
	}}
	if err := s.repo.SetFacts(ctx, userID, computerID, next, at, evt); err != nil {
		return fmt.Errorf("save facts of computer %s: %w", computerID, err)
	}
	return nil
}

// sameFacts compares facts as they are stored, so an empty list and a missing one read alike.
func sameFacts(stored *Facts, next Facts) (bool, error) {
	if stored == nil {
		return false, nil
	}
	a, err := jsonx.Marshal(stored)
	if err != nil {
		return false, fmt.Errorf("encode stored facts: %w", err)
	}
	b, err := jsonx.Marshal(next)
	if err != nil {
		return false, fmt.Errorf("encode facts: %w", err)
	}
	return bytes.Equal(a, b), nil
}

// readT3Snapshot sets facts' providers and projects to what the computer's T3 Code lists through its runner; a computer
// with no live session, or a T3 Code that does not answer, keeps the last ones read.
func (s *Service) readT3Snapshot(ctx context.Context, userID, computerID string, facts *Facts) {
	session, err := s.sessionComputer(ctx, userID, computerID)
	if errors.Is(err, apperrs.ErrInvalid) {
		return
	}
	log := logging.FromCtx(ctx)
	if err != nil {
		log.Warn("computer facts: no session to read T3 Code with", "computer_id", computerID, "error", err)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, snapshotTimeout)
	defer cancel()
	providers, err := s.harnessProviders(ctx, session)
	if err != nil {
		log.Warn("computer facts: T3 Code listed no providers", "computer_id", computerID, "error", err)
		return
	}
	client, err := s.client(session.Kind)
	if err != nil {
		return
	}
	projects, err := client.ListProjects(ctx, session.Session())
	if err != nil {
		log.Warn("computer facts: T3 Code listed no projects", "computer_id", computerID, "error", err)
		return
	}
	facts.Providers, facts.Projects = providerFacts(providers), projects
}

// pairThroughRunner pairs c through its runner; a failure leaves c as it was and stays its row's reason until one succeeds.
func (s *Service) pairThroughRunner(ctx context.Context, c Computer) (*Computer, error) {
	paired, err := s.mintAndPair(ctx, c)
	s.pairMu.Lock()
	if err != nil {
		s.pairFailures[c.ID] = err.Error()
		s.pairMu.Unlock()
		s.publishPairFailed(ctx, c)
		return nil, err
	}
	delete(s.pairFailures, c.ID)
	s.pairMu.Unlock()
	return paired, nil
}

// publishPairFailed tells the owner's open views to read the computer's new pair_error.
func (s *Service) publishPairFailed(ctx context.Context, c Computer) {
	if s.bus == nil {
		return
	}
	err := s.bus.Publish(ctx, TopicPairFailed, PairFailedEvent{ComputerID: c.ID, UserID: c.UserID, MembersOnly: true})
	if err != nil {
		logging.FromCtx(ctx).Warn("publish computer pair failure", "computer_id", c.ID, "error", err)
	}
}

func (s *Service) mintAndPair(ctx context.Context, c Computer) (*Computer, error) {
	runners, err := s.runnerSeam()
	if err != nil {
		return nil, err
	}
	r, err := runners.ComputerRunner(ctx, c.ID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("%w: %s has no Nexul app yet; run the command from Add a computer on it", apperrs.ErrInvalid, c.Name)
	}
	if err != nil {
		return nil, fmt.Errorf("get runner of computer %s: %w", c.ID, err)
	}
	if !r.Connected {
		return nil, fmt.Errorf("%w: %s isn't connected to Nexul", apperrs.ErrConflict, c.Name)
	}
	token, err := runners.PairingToken(ctx, c.ID)
	if err != nil {
		return nil, fmt.Errorf("T3 Code on %s made no pairing token: %w", c.Name, err)
	}
	return s.pair(ctx, c, runnerAddress(c.ID), token)
}

// pairFailure is why the computer's last pairing through its runner failed, "" after a success.
func (s *Service) pairFailure(computerID string) string {
	s.pairMu.Lock()
	defer s.pairMu.Unlock()
	return s.pairFailures[computerID]
}

// withRunners adds each computer's runner state, none before one enrolls, and why its last pairing failed.
func (s *Service) withRunners(ctx context.Context, cs []Computer) ([]Computer, error) {
	for i := range cs {
		cs[i].PairError = s.pairFailure(cs[i].ID)
	}
	if s.runners == nil {
		return cs, nil
	}
	for i := range cs {
		r, err := s.runners.ComputerRunner(ctx, cs[i].ID)
		if errors.Is(err, apperrs.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("get runner of computer %s: %w", cs[i].ID, err)
		}
		cs[i].Runner = &r
	}
	return cs, nil
}

func (s *Service) runnerSeam() (Runners, error) {
	if s.runners == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: personal runners are not wired", apperrs.ErrFatal))
	}
	return s.runners, nil
}
