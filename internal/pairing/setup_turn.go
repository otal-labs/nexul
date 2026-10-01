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

// StartSetup runs one setup turn per provider in the background, each on its models entry (by driver) with that entry's options,
// else the provider's default; only narrows the run to those drivers or instance ids, empty for every provider, and the drivers
// it leaves out are remembered as skipped. folder picks the harness project the turns run in, empty for the default.
func (s *Service) StartSetup(ctx context.Context, userID, computerID string, models map[string]string, options map[string][]harness.OptionSetting, folder string, only []string) (*SetupRun, error) {
	return s.startSetup(ctx, userID, computerID, only, models, options, folder, true)
}

// RetrySetupProvider runs one provider's setup turn alone on model and its options, empty for its default; on a confirmed provider it re-verifies.
func (s *Service) RetrySetupProvider(ctx context.Context, userID, computerID, provider, model string, options []harness.OptionSetting, folder string) (*SetupRun, error) {
	provider, err := validateProvider(provider)
	if err != nil {
		return nil, err
	}
	return s.startSetup(ctx, userID, computerID, []string{provider}, map[string]string{provider: model}, map[string][]harness.OptionSetting{provider: options}, folder, false)
}

func (s *Service) startSetup(ctx context.Context, userID, computerID string, only []string, models map[string]string, options map[string][]harness.OptionSetting, folder string, remember bool) (*SetupRun, error) {
	choices, err := SetupChoices{Models: models, ModelOptions: options, Folder: folder}.clean()
	if err != nil {
		return nil, err
	}
	session, err := s.sessionComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	folder = choices.Folder
	projectID, err := s.setupProject(ctx, session, folder)
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
	detected := detectedDrivers(providers)
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
	if remember {
		choices.Skipped = skippedDrivers(detected, providers)
		if err := s.repo.SaveSetupChoices(ctx, userID, session.ID, choices); err != nil {
			s.releaseSetup(session.ID)
			return nil, err
		}
	}
	run := &SetupRun{RunID: ids.New(), ComputerID: session.ID, Folder: folder, Providers: make([]SetupProvider, 0, len(providers)), projectID: projectID}
	for _, p := range providers {
		model := setupPick(choices.Models, p)
		pick := SetupProvider{Provider: strings.ToLower(p.Driver), Name: p.Name, Model: model}
		if model != "" {
			pick.ModelOptions = setupPick(choices.ModelOptions, p)
		}
		run.Providers = append(run.Providers, pick)
	}
	prompt := setupPrompt{ComputerID: session.ID, ComputerName: session.Name, MCPURL: mcpURL, Token: token}
	runCtx := context.WithoutCancel(ctx)
	s.setupRuns.Go(func() {
		defer s.releaseSetup(session.ID)
		s.runSetup(runCtx, userID, *run, providers, prompt)
	})
	return run, nil
}

// setupProject finds the harness project that opens folder, so setup never runs in a project whose folder is gone.
func (s *Service) setupProject(ctx context.Context, session Computer, folder string) (string, error) {
	if folder == "" {
		return "", nil
	}
	client, err := s.client(session.Kind)
	if err != nil {
		return "", err
	}
	projects, err := client.ListProjects(ctx, session.Session())
	if err != nil {
		return "", fmt.Errorf("list projects on %s: %w", session.Name, err)
	}
	folders := make([]string, 0, len(projects))
	for _, p := range projects {
		if p.Path == folder {
			return p.ID, nil
		}
		folders = append(folders, p.Path)
	}
	return "", fmt.Errorf("%w: no T3 Code project on %s opens %s; add the folder in T3 Code first, or pick one it opens: %s",
		apperrs.ErrInvalid, session.Name, folder, strings.Join(folders, ", "))
}

// setupPick is the entry picked for p by driver kind or instance id, any case; the zero value leaves the provider on its own default.
func setupPick[T any](picks map[string]T, p harness.Provider) T {
	for key, pick := range picks {
		if strings.EqualFold(key, p.Driver) || strings.EqualFold(key, p.ID) {
			return pick
		}
	}
	var zero T
	return zero
}

// setupProviders keeps one instance per driver, since a confirmation is per driver; only narrows it to the named drivers or instances.
func setupProviders(providers []harness.Provider, only []string) ([]harness.Provider, error) {
	for _, name := range only {
		if !slices.ContainsFunc(providers, func(p harness.Provider) bool { return namesProvider(p, name) }) {
			return nil, fmt.Errorf("%w: provider %s is not available on this computer", apperrs.ErrInvalid, name)
		}
	}
	var out []harness.Provider
	for _, p := range providers {
		driver := strings.ToLower(p.Driver)
		if len(only) > 0 && !slices.ContainsFunc(only, func(name string) bool { return namesProvider(p, name) }) {
			continue
		}
		if slices.ContainsFunc(out, func(o harness.Provider) bool { return strings.ToLower(o.Driver) == driver }) {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: the harness lists no usable provider to set up", apperrs.ErrInvalid)
	}
	return out, nil
}

// namesProvider is whether name, in any case, is p's driver kind or instance id.
func namesProvider(p harness.Provider, name string) bool {
	return strings.EqualFold(name, p.Driver) || strings.EqualFold(name, p.ID)
}

// detectedDrivers is the distinct driver kinds the harness lists, lowercase, in listed order.
func detectedDrivers(providers []harness.Provider) []string {
	var out []string
	for _, p := range providers {
		if driver := strings.ToLower(p.Driver); !slices.Contains(out, driver) {
			out = append(out, driver)
		}
	}
	return out
}

// skippedDrivers is every detected driver the run does not cover; never nil, so it stores as an empty list.
func skippedDrivers(detected []string, run []harness.Provider) []string {
	skipped := []string{}
	for _, driver := range detected {
		if !slices.ContainsFunc(run, func(p harness.Provider) bool { return strings.ToLower(p.Driver) == driver }) {
			skipped = append(skipped, driver)
		}
	}
	return skipped
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
func (s *Service) runSetup(ctx context.Context, userID string, run SetupRun, providers []harness.Provider, prompt setupPrompt) {
	runID, computerID := run.RunID, run.ComputerID
	outcomes := make([]SetupTurnOutcome, 0, len(providers))
	for i, p := range providers {
		last := i == len(providers)-1
		prompt.Driver, prompt.ProviderName = p.Driver, p.Name
		turn := s.runSetupTurn(ctx, userID, run, p, run.Providers[i], prompt)
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
func (s *Service) runSetupTurn(ctx context.Context, userID string, run SetupRun, p harness.Provider, pick SetupProvider, prompt setupPrompt) SetupTurn {
	computerID := run.ComputerID
	turn := SetupTurn{
		ID: ids.New(), RunID: run.RunID, ComputerID: computerID, UserID: userID, Provider: strings.ToLower(p.Driver), ProviderName: p.Name, Model: pick.Model,
		State: SetupTurnRunning, Status: "Connecting Nexul and installing skills", Transcript: []harness.Activity{}, StartedAt: s.now().UTC(),
	}
	s.saveSetupTurn(ctx, turn)
	target, err := s.ResolveSetupTurnTarget(ctx, userID, computerID, p.ID, run.projectID)
	if err != nil {
		return s.endSetupTurn(turn, SetupTurnFailed, err.Error())
	}
	client, err := s.client(target.Computer.Kind)
	if err != nil {
		return s.endSetupTurn(turn, SetupTurnFailed, err.Error())
	}
	// Both sessions run on the picked model only: the pairing defaults' model belongs to one provider, not every provider.
	ht := harness.Target{Session: target.Computer.Session(), ProjectID: target.HarnessProjectID, Provider: target.Provider, Model: pick.Model, ModelOptions: pick.ModelOptions}
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
		if !u.Snapshot.Streaming {
			s.recordSetupText(ctx, turn, reply)
		}
		return "", false
	}
	if u.Question != nil {
		return "The agent stopped to ask a question; setup runs unattended", true
	}
	if u.Terminal == nil {
		return "", false
	}
	s.recordSetupText(ctx, turn, reply)
	if u.Terminal.State == harness.TurnDone {
		return "", true
	}
	if u.Terminal.LastError != "" {
		return u.Terminal.LastError, true
	}
	return "The setup turn ended " + string(u.Terminal.State), true
}

// recordSetupText records a closed message as a text step where it was said, then clears it so it is never recorded twice.
func (s *Service) recordSetupText(ctx context.Context, turn *SetupTurn, reply *string) {
	if *reply == "" {
		return
	}
	s.recordSetupActivity(ctx, turn, harness.Activity{Kind: harness.ActivityText, Summary: harness.Preview(*reply, 160), Detail: harness.CapDetail(*reply), At: s.now().UTC()})
	*reply = ""
}

// recordSetupActivity hides tokens as the step enters the transcript, so neither the saved turn nor the live line carries one.
func (s *Service) recordSetupActivity(ctx context.Context, turn *SetupTurn, a harness.Activity) {
	a.Summary, a.Detail = redact.Tokens(a.Summary), redact.Tokens(a.Detail)
	turn.appendStep(a)
	if s.bus == nil || a.Summary == "" {
		return
	}
	frame := SetupTurnActivityEvent{
		ComputerID: turn.ComputerID, UserID: turn.UserID, RunID: turn.RunID, TurnID: turn.ID, Provider: turn.Provider,
		Status: harness.Preview(a.Summary, 120), CallID: a.CallID, Kind: string(a.Kind), Tool: a.Tool, At: a.At,
	}
	if a.Kind == harness.ActivityText {
		frame.Text = a.Detail
	}
	err := s.bus.Publish(ctx, TopicSetupTurnActivity, frame)
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
		ProviderName: turn.ProviderName, Model: turn.Model, State: turn.State, Status: turn.Status, StartedAt: turn.StartedAt, EndedAt: turn.EndedAt,
	}}
	if err := s.repo.SaveSetupTurn(ctx, turn, append([]eventbus.OutboxEvent{changed}, evts...)...); err != nil {
		logging.FromCtx(ctx).Error("save setup turn", "computer_id", turn.ComputerID, "provider", turn.Provider, "error", err)
	}
}
