package pairing

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/redact"
)

// setupSessionTimeout bounds each setup session; an install still running after it is stuck, not slow.
const setupSessionTimeout = 15 * time.Minute

// InstanceSettings is pairing's seam onto the instance URL the setup turns point each provider's MCP entry at.
type InstanceSettings interface {
	GetInstanceURL(ctx context.Context) (string, error)
}

// StartSetup runs one setup turn per provider the computer's harness lists, one after another, in the background.
func (s *Service) StartSetup(ctx context.Context, userID, computerID string) (*SetupRun, error) {
	return s.startSetup(ctx, userID, computerID, "")
}

// RetrySetupProvider runs one provider's setup turn alone; on a confirmed provider it re-verifies.
func (s *Service) RetrySetupProvider(ctx context.Context, userID, computerID, provider string) (*SetupRun, error) {
	provider, err := validateProvider(provider)
	if err != nil {
		return nil, err
	}
	return s.startSetup(ctx, userID, computerID, provider)
}

func (s *Service) startSetup(ctx context.Context, userID, computerID, only string) (*SetupRun, error) {
	session, err := s.sessionComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	mcpURL, err := s.setupMCPURL(ctx)
	if err != nil {
		return nil, err
	}
	providers, err := s.harnessProviders(ctx, session)
	if err != nil {
		return nil, err
	}
	providers, err = setupProviders(providers, only)
	if err != nil {
		return nil, err
	}
	if !s.claimSetup(session.ID) {
		return nil, fmt.Errorf("%w: setup is already running on %s", apperrs.ErrConflict, session.Name)
	}
	token, err := s.setupToken(ctx, session)
	if err != nil {
		s.releaseSetup(session.ID)
		return nil, err
	}
	run := &SetupRun{RunID: ids.New(), ComputerID: session.ID, Providers: make([]SetupProvider, 0, len(providers))}
	for _, p := range providers {
		run.Providers = append(run.Providers, SetupProvider{Provider: strings.ToLower(p.Driver), Name: p.Name})
	}
	prompt := setupPrompt{ComputerID: session.ID, ComputerName: session.Name, MCPURL: mcpURL, Token: token}
	runCtx := context.WithoutCancel(ctx)
	s.setupRuns.Go(func() {
		defer s.releaseSetup(session.ID)
		s.runSetup(runCtx, userID, run.RunID, session.ID, providers, prompt)
	})
	return run, nil
}

// setupProviders keeps one instance per driver, since a confirmation is per driver; only narrows it to one driver or instance.
func setupProviders(providers []harness.Provider, only string) ([]harness.Provider, error) {
	var out []harness.Provider
	for _, p := range providers {
		driver := strings.ToLower(p.Driver)
		if only != "" && driver != only && strings.ToLower(p.ID) != only {
			continue
		}
		if slices.ContainsFunc(out, func(o harness.Provider) bool { return strings.ToLower(o.Driver) == driver }) {
			continue
		}
		out = append(out, p)
	}
	if len(out) > 0 {
		return out, nil
	}
	if only != "" {
		return nil, fmt.Errorf("%w: provider %s is not available on this computer", apperrs.ErrInvalid, only)
	}
	return nil, fmt.Errorf("%w: the harness lists no usable provider to set up", apperrs.ErrInvalid)
}

func (s *Service) setupMCPURL(ctx context.Context) (string, error) {
	if s.instance == nil {
		return "", apperrs.Fatal(fmt.Errorf("%w: the instance URL reader is not wired", apperrs.ErrFatal))
	}
	instanceURL, err := s.instance.GetInstanceURL(ctx)
	if err != nil {
		return "", fmt.Errorf("read instance url: %w", err)
	}
	mcpURL := mcpURLFor(instanceURL)
	if mcpURL == "" {
		return "", fmt.Errorf("%w: set the instance URL first, the providers reach Nexul's MCP server through it", apperrs.ErrInvalid)
	}
	return mcpURL, nil
}

// setupToken reuses the token earlier setup turns wrote while it is still the computer's active one, else mints its replacement.
func (s *Service) setupToken(ctx context.Context, computer Computer) (string, error) {
	tokens, err := s.tokenSeam()
	if err != nil {
		return "", err
	}
	active, err := tokens.ComputerToken(ctx, computer.UserID, computer.ID)
	if err != nil {
		return "", fmt.Errorf("get mcp token for computer %s: %w", computer.ID, err)
	}
	if raw := s.storedSetupToken(computer); active != nil && raw != "" && strings.HasSuffix(raw, active.Prefix) {
		return raw, nil
	}
	raw, _, err := tokens.MintComputerToken(ctx, computer.UserID, computer.ID, computer.Name)
	if err != nil {
		return "", fmt.Errorf("mint mcp token for computer %s: %w", computer.ID, err)
	}
	sealed, err := crypto.Encrypt(s.key, []byte(raw))
	if err != nil {
		return "", fmt.Errorf("encrypt setup mcp token: %w", err)
	}
	if err := s.repo.SetSetupMCPToken(ctx, computer.UserID, computer.ID, sealed); err != nil {
		return "", fmt.Errorf("store setup mcp token for %s: %w", computer.ID, err)
	}
	return raw, nil
}

// storedSetupToken is "" when there is no readable copy; an unreadable one is replaced by a fresh mint, not a failure.
func (s *Service) storedSetupToken(computer Computer) string {
	if computer.SetupMCPToken == "" {
		return ""
	}
	raw, err := crypto.Decrypt(s.key, computer.SetupMCPToken)
	if err != nil {
		return ""
	}
	return string(raw)
}

func (s *Service) claimSetup(computerID string) bool {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	if s.settingUp[computerID] {
		return false
	}
	s.settingUp[computerID] = true
	return true
}

func (s *Service) releaseSetup(computerID string) {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	delete(s.settingUp, computerID)
}

// runSetup runs the turns in order, so two providers never install into the shared skill locations at once.
func (s *Service) runSetup(ctx context.Context, userID, runID, computerID string, providers []harness.Provider, prompt setupPrompt) {
	outcomes := make([]SetupTurnOutcome, 0, len(providers))
	for i, p := range providers {
		last := i == len(providers)-1
		prompt.Driver, prompt.ProviderName = p.Driver, p.Name
		turn := s.runSetupTurn(ctx, userID, runID, computerID, p, prompt)
		outcomes = append(outcomes, SetupTurnOutcome{Provider: turn.Provider, State: turn.State, Status: turn.Status})
		if !last {
			s.saveSetupTurn(ctx, turn)
			continue
		}
		s.saveSetupTurn(ctx, turn, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicSetupFinished, Payload: SetupFinishedEvent{
			ComputerID: computerID, UserID: userID, RunID: runID, Confirmed: s.computerConfirmed(ctx, userID, computerID), Providers: outcomes,
		}})
	}
}

// runSetupTurn is a prepare session that connects MCP and installs skills, then a fresh session that loads them and confirms.
func (s *Service) runSetupTurn(ctx context.Context, userID, runID, computerID string, p harness.Provider, prompt setupPrompt) SetupTurn {
	turn := SetupTurn{
		ID: ids.New(), RunID: runID, ComputerID: computerID, UserID: userID, Provider: strings.ToLower(p.Driver), ProviderName: p.Name,
		State: SetupTurnRunning, Status: "Connecting Nexul and installing skills", Transcript: []harness.Activity{}, StartedAt: s.now().UTC(),
	}
	s.saveSetupTurn(ctx, turn)
	target, err := s.ResolveSetupTurnTarget(ctx, userID, computerID, p.ID)
	if err != nil {
		return s.endSetupTurn(turn, SetupTurnFailed, err.Error())
	}
	client, err := s.client(target.Computer.Kind)
	if err != nil {
		return s.endSetupTurn(turn, SetupTurnFailed, err.Error())
	}
	ht := harness.Target{Session: target.Computer.Session(), ProjectID: target.HarnessProjectID, Provider: target.Provider, Model: target.Model}
	if reason := s.runSetupSession(ctx, client, ht, &turn, "Nexul setup: "+p.Name, prepareInstructions(prompt)); reason != "" {
		return s.endSetupTurn(turn, SetupTurnFailed, reason)
	}
	turn.Status = "Checking the skills " + p.Name + " discovered"
	s.saveSetupTurn(ctx, turn)
	if reason := s.runSetupSession(ctx, client, ht, &turn, "Nexul setup check: "+p.Name, confirmInstructions(prompt)); reason != "" {
		return s.endSetupTurn(turn, SetupTurnFailed, reason)
	}
	skills, err := s.confirmedSkillsSince(ctx, computerID, turn.Provider, turn.StartedAt)
	if err != nil {
		return s.endSetupTurn(turn, SetupTurnFailed, err.Error())
	}
	if skills == nil {
		return s.endSetupTurn(turn, SetupTurnFailed, "The turn ended without confirming "+p.Name)
	}
	return s.endSetupTurn(turn, SetupTurnConfirmed, fmt.Sprintf("Confirmed with %d skills", len(skills)))
}

// runSetupSession runs one fresh harness session to its end; the returned reason is empty only when it finished.
func (s *Service) runSetupSession(ctx context.Context, client harness.Client, target harness.Target, turn *SetupTurn, title, prompt string) string {
	ctx, cancel := context.WithTimeout(ctx, setupSessionTimeout)
	defer cancel()
	started, err := client.StartTurn(ctx, target, title, harness.TurnPrompts{Full: prompt, Incremental: prompt})
	if err != nil {
		return "Could not start the setup turn: " + err.Error()
	}
	target.SessionID = started.SessionID
	reply := ""
	for {
		select {
		case u, ok := <-started.Updates:
			if !ok {
				return "The harness closed the setup turn without a result"
			}
			reason, done := s.setupUpdate(ctx, turn, u, &reply)
			if !done {
				continue
			}
			if u.Question != nil {
				s.interruptSetup(ctx, client, target)
			}
			return reason
		case <-ctx.Done():
			s.interruptSetup(ctx, client, target)
			return fmt.Sprintf("No result within %s", setupSessionTimeout)
		}
	}
}

// setupUpdate records one update; done means the session is over, with an empty reason only when it finished.
func (s *Service) setupUpdate(ctx context.Context, turn *SetupTurn, u harness.Update, reply *string) (string, bool) {
	if u.Activity != nil {
		s.recordSetupActivity(ctx, turn, *u.Activity)
		return "", false
	}
	if u.Snapshot != nil && u.Snapshot.Text != "" {
		*reply = u.Snapshot.Text
		return "", false
	}
	if u.Question != nil {
		return "The agent stopped to ask a question; setup runs unattended", true
	}
	if u.Terminal == nil {
		return "", false
	}
	if *reply != "" {
		s.recordSetupActivity(ctx, turn, harness.Activity{Kind: harness.ActivityText, Summary: harness.Preview(*reply, 160), Detail: harness.CapDetail(*reply), At: s.now().UTC()})
	}
	if u.Terminal.State == harness.TurnDone {
		return "", true
	}
	if u.Terminal.LastError != "" {
		return u.Terminal.LastError, true
	}
	return "The setup turn ended " + string(u.Terminal.State), true
}

// recordSetupActivity hides tokens as the step enters the transcript, so neither the saved turn nor the live line carries one.
func (s *Service) recordSetupActivity(ctx context.Context, turn *SetupTurn, a harness.Activity) {
	a.Summary, a.Detail = redact.Tokens(a.Summary), redact.Tokens(a.Detail)
	turn.Transcript = append(turn.Transcript, a)
	if s.bus == nil || a.Summary == "" {
		return
	}
	err := s.bus.Publish(ctx, TopicSetupTurnActivity, SetupTurnActivityEvent{
		ComputerID: turn.ComputerID, UserID: turn.UserID, RunID: turn.RunID, TurnID: turn.ID, Provider: turn.Provider, Status: harness.Preview(a.Summary, 120),
	})
	if err != nil {
		logging.FromCtx(ctx).Warn("publish setup turn activity", "computer_id", turn.ComputerID, "error", err)
	}
}

func (s *Service) interruptSetup(ctx context.Context, client harness.Client, target harness.Target) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := client.Interrupt(ctx, target); err != nil {
		logging.FromCtx(ctx).Warn("interrupt setup turn", "session", target.SessionID, "error", err)
	}
}

// confirmedSkillsSince returns the skills a confirmation made during this turn recorded, nil when none was made.
func (s *Service) confirmedSkillsSince(ctx context.Context, computerID, driver string, since time.Time) ([]string, error) {
	setups, err := s.repo.ListProviderSetups(ctx, computerID)
	if err != nil {
		return nil, fmt.Errorf("list provider setups for %s: %w", computerID, err)
	}
	for _, p := range setups {
		if p.Provider == driver && p.ConfirmedAt != nil && !p.ConfirmedAt.Before(since.Truncate(time.Second)) {
			return p.Skills, nil
		}
	}
	return nil, nil
}

func (s *Service) computerConfirmed(ctx context.Context, userID, computerID string) bool {
	computer, err := s.repo.GetComputer(ctx, userID, computerID)
	if err != nil {
		logging.FromCtx(ctx).Error("read computer after setup", "computer_id", computerID, "error", err)
		return false
	}
	return computer.SetupConfirmedAt != nil
}

func (s *Service) endSetupTurn(turn SetupTurn, state SetupTurnState, status string) SetupTurn {
	now := s.now().UTC()
	turn.State, turn.Status, turn.EndedAt = state, redact.Tokens(status), &now
	return turn
}

// saveSetupTurn writes the turn with its changed event; a failed write is logged, the run carries on.
func (s *Service) saveSetupTurn(ctx context.Context, turn SetupTurn, evts ...eventbus.OutboxEvent) {
	turn.Status, turn.UpdatedAt = redact.Tokens(turn.Status), s.now().UTC()
	changed := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicSetupTurnChanged, Payload: SetupTurnChangedEvent{
		ComputerID: turn.ComputerID, UserID: turn.UserID, RunID: turn.RunID, TurnID: turn.ID, Provider: turn.Provider,
		ProviderName: turn.ProviderName, State: turn.State, Status: turn.Status, StartedAt: turn.StartedAt, EndedAt: turn.EndedAt,
	}}
	if err := s.repo.SaveSetupTurn(ctx, turn, append([]eventbus.OutboxEvent{changed}, evts...)...); err != nil {
		logging.FromCtx(ctx).Error("save setup turn", "computer_id", turn.ComputerID, "provider", turn.Provider, "error", err)
	}
}
