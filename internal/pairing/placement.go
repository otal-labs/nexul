package pairing

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// ProjectRemotes reads the git remotes a project's checkout is found by on a computer (ADR 0165).
type ProjectRemotes interface {
	// ProjectRemotes returns each GitHub repository of projectID as T3 normalizes a remote: github.com/owner/name, lowercase.
	ProjectRemotes(ctx context.Context, projectID string) ([]string, error)
}

// Room is how a computer's runs at once bears on where a run lands (ADR 0166).
type Room int

const (
	// RoomIgnored is a chat's: runs at once counts play runs only.
	RoomIgnored Room = iota
	// RoomPreferred is a pressed run's: room picks the computer, but a run with every computer full still starts.
	RoomPreferred
	// RoomRequired is a queued run's: it waits while no ready computer has room.
	RoomRequired
)

// RunRequest is what a run asks of the person's computers.
type RunRequest struct {
	UserID    string
	ProjectID string
	// ComputerIDs is the run's own list; empty takes the project link's, else the person's order.
	ComputerIDs []string
	// HarnessProjectID is the checkout picked on ComputerIDs' only computer.
	HarnessProjectID string
	Provider         string
	Model            string
	ModelOptions     []harness.OptionSetting
	Room             Room
	// Person is a run someone pressed: with no computer holding the project it asks where instead of failing.
	Person bool
	// Chat runs in the computer's chat folder when no computer has a checkout of the project.
	Chat bool
}

// ResolveRun lands a run on the first ready computer, behind the setup gate; a pick on a pressed run is remembered.
func (s *Service) ResolveRun(ctx context.Context, req RunRequest) (*ResolvedTarget, error) {
	return s.place(ctx, req, true)
}

// PreviewRun is where ResolveRun would land now, from what Nexul last heard, without asking any computer.
func (s *Service) PreviewRun(ctx context.Context, req RunRequest) (*ResolvedTarget, error) {
	target, err := s.place(ctx, req, false)
	if err != nil {
		return nil, err
	}
	target.Computer.BearerToken = ""
	return target, nil
}

// plan is what a run carries to every computer it tries: its model, where its threads start, and how it picks.
type plan struct {
	req       RunRequest
	link      ProjectLink
	pick      modelPick
	worktree  bool
	placement Placement
	clone     bool
	remotes   []string
	readRepos bool
}

func (s *Service) place(ctx context.Context, req RunRequest, live bool) (*ResolvedTarget, error) {
	if strings.TrimSpace(req.UserID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	if len(s.key) == 0 {
		return nil, apperrs.Fatal(fmt.Errorf("%w: pairing encryption key is not configured", apperrs.ErrFatal))
	}
	req.ProjectID, req.HarnessProjectID = strings.TrimSpace(req.ProjectID), strings.TrimSpace(req.HarnessProjectID)
	if req.HarnessProjectID != "" && len(req.ComputerIDs) != 1 {
		return nil, fmt.Errorf("%w: a T3 project needs the one computer it is on", apperrs.ErrInvalid)
	}
	p, err := s.planRun(ctx, req)
	if err != nil {
		return nil, err
	}
	computers, err := s.candidates(ctx, p)
	if err != nil {
		return nil, err
	}
	target, misses, err := s.land(ctx, p, computers, live)
	if err != nil || target != nil {
		return target, err
	}
	if p.req.Chat && p.req.ProjectID != "" && allMissingProject(misses) {
		p.req.ProjectID = ""
		if target, misses, err = s.land(ctx, p, computers, live); err != nil || target != nil {
			return target, err
		}
	}
	return nil, refusal(req, computers, misses)
}

func (s *Service) planRun(ctx context.Context, req RunRequest) (*plan, error) {
	defaults, err := s.repo.GetDefaults(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("get defaults: %w", err)
	}
	var link ProjectLink
	if req.ProjectID != "" {
		if link, err = s.repo.GetProjectLink(ctx, req.UserID, req.ProjectID); err != nil {
			return nil, fmt.Errorf("get project link %s: %w", req.ProjectID, err)
		}
	}
	pick := modelPick{provider: strings.TrimSpace(req.Provider), model: strings.TrimSpace(req.Model), options: req.ModelOptions}
	pick = fillPick(fillPick(pick, modelPick{link.Provider, link.Model, link.ModelOptions}), modelPick{defaults.Provider, defaults.Model, defaults.ModelOptions})
	startIn := cmp.Or(link.StartIn, defaults.StartIn)
	placement := defaults.Placement
	if len(link.Computers) > 0 && link.Placement != "" {
		placement = link.Placement
	}
	return &plan{
		req: req, link: link, pick: pick, worktree: startIn == StartInWorktree, placement: placement, clone: defaults.CloneAutomatically,
	}, nil
}

// candidates are the computers a run may land on, in order: its own list, else the project's, else the person's.
func (s *Service) candidates(ctx context.Context, p *plan) ([]Computer, error) {
	own, err := s.repo.ListComputers(ctx, p.req.UserID)
	if err != nil {
		return nil, fmt.Errorf("list computers: %w", err)
	}
	ids := p.req.ComputerIDs
	if len(ids) == 0 {
		ids = p.link.Computers
	}
	if len(ids) == 0 {
		return own, nil
	}
	out := make([]Computer, 0, len(ids))
	for _, id := range ids {
		i := slices.IndexFunc(own, func(c Computer) bool { return c.ID == strings.TrimSpace(id) })
		if i < 0 && len(p.req.ComputerIDs) > 0 {
			return nil, &NotConfiguredError{Reason: ReasonUnpaired}
		}
		if i >= 0 {
			out = append(out, own[i])
		}
	}
	return out, nil
}

// miss is one computer a run passed over: the line every client shows, and the refusal it would be on its own.
type miss struct {
	pass PassOver
	nc   *NotConfiguredError
}

// option is a computer that can take the run, with the checkout it runs in and how busy it is.
type option struct {
	computer Computer
	checkout string
	running  int
	full     bool
}

// land tries each ready computer in placement order; with none taking the run it returns no target and every miss.
func (s *Service) land(ctx context.Context, p *plan, computers []Computer, live bool) (*ResolvedTarget, []miss, error) {
	if len(computers) == 0 {
		return nil, nil, nil
	}
	ids := make([]string, 0, len(computers))
	for _, c := range computers {
		ids = append(ids, c.ID)
	}
	running, err := s.repo.CountRunning(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	var misses []miss
	var ready, full []option
	for _, c := range computers {
		o, m, err := s.check(ctx, p, c, running[c.ID])
		if err != nil {
			return nil, nil, err
		}
		if m != nil {
			misses = append(misses, *m)
		}
		if o != nil && o.full {
			full = append(full, *o)
		}
		if o != nil && !o.full {
			ready = append(ready, *o)
		}
	}
	tries := byPlacement(ready, p.placement)
	if p.req.Room == RoomPreferred {
		tries = append(tries, full...)
	}
	for _, o := range tries {
		target, m, err := s.tryOption(ctx, p, o, live)
		if err != nil {
			return nil, nil, err
		}
		if m != nil {
			misses = append(misses, *m)
			continue
		}
		target.PassedOver = passesBut(misses, o.computer.ID)
		return target, nil, nil
	}
	return nil, misses, nil
}

// byPlacement orders ready computers: as they are for in order, least busy for their size first for spread.
func byPlacement(ready []option, placement Placement) []option {
	if placement != PlacementSpread {
		return ready
	}
	out := slices.Clone(ready)
	slices.SortStableFunc(out, func(a, b option) int {
		return cmp.Compare(a.running*runsAtOnce(b.computer), b.running*runsAtOnce(a.computer))
	})
	return out
}

func runsAtOnce(c Computer) int {
	return max(c.RunsAtOnce, MinRunsAtOnce)
}

// tryOption makes the target on o's computer; live, it asks the computer through the setup gate, and a computer that
// does not answer or is not set up becomes a miss so the run moves on.
func (s *Service) tryOption(ctx context.Context, p *plan, o option, live bool) (*ResolvedTarget, *miss, error) {
	c := o.computer
	if live && p.req.Person && p.req.HarnessProjectID != "" {
		if err := s.requireListedProject(ctx, p.req.UserID, c.ID, p.req.HarnessProjectID); err != nil {
			return nil, nil, err
		}
		if err := s.repo.SaveCheckout(ctx, p.req.UserID, p.req.ProjectID, c.ID, p.req.HarnessProjectID); err != nil {
			return nil, nil, err
		}
	}
	decrypted, err := crypto.Decrypt(s.key, c.BearerToken)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt bearer token for computer %s: %w", c.ID, err)
	}
	c.BearerToken = string(decrypted)
	target := &ResolvedTarget{
		Computer: c, HarnessProjectID: o.checkout, Provider: p.pick.provider, Model: p.pick.model, ModelOptions: p.pick.options, Worktree: p.worktree,
	}
	if !live {
		return target, nil, nil
	}
	gated, err := s.requireSetup(ctx, target)
	var nc *NotConfiguredError
	if errors.As(err, &nc) && (nc.Reason == ReasonOffline || nc.Reason == ReasonSetupRequired) {
		return nil, &miss{pass: passOf(c, nc), nc: nc}, nil
	}
	return gated, nil, err
}

// check is what Nexul last heard about c, in the order a gap is named: reachable, set up for the run's provider,
// offering its model, holding the project's checkout, and with room.
func (s *Service) check(ctx context.Context, p *plan, c Computer, running int) (*option, *miss, error) {
	if m := s.reachable(ctx, c); m != nil {
		return nil, m, nil
	}
	if m, err := s.providerReady(ctx, p, c); m != nil || err != nil {
		return nil, m, err
	}
	checkout, m, err := s.checkout(ctx, p, c)
	if m != nil || err != nil {
		return nil, m, err
	}
	o := &option{computer: c, checkout: checkout, running: running}
	if p.req.Room == RoomIgnored || running < runsAtOnce(c) {
		return o, nil, nil
	}
	o.full = true
	text := fmt.Sprintf("%s is full (%d of %d running)", c.Name, running, runsAtOnce(c))
	return o, &miss{pass: PassOver{ComputerID: c.ID, Name: c.Name, Why: PassOverFull, Text: text}}, nil
}

// reachable is whether c is paired, its session current, and its runner and T3 Code answering, as last heard.
func (s *Service) reachable(ctx context.Context, c Computer) *miss {
	if !c.Paired() {
		nc := &NotConfiguredError{Reason: ReasonUnpaired, Computer: c.Name, ComputerID: c.ID}
		return &miss{pass: PassOver{ComputerID: c.ID, Name: c.Name, Why: PassOverOffline, Text: c.Name + " isn't paired yet"}, nc: nc}
	}
	if c.TokenExpiresAt.Before(s.now()) {
		nc := &NotConfiguredError{Reason: ReasonExpiredToken, Computer: c.Name, ComputerID: c.ID}
		return &miss{pass: PassOver{ComputerID: c.ID, Name: c.Name, Why: PassOverExpiredToken, Text: c.Name + " needs pairing again"}, nc: nc}
	}
	offline := func(why string) *miss {
		nc := &NotConfiguredError{Reason: ReasonOffline, Computer: c.Name, ComputerID: c.ID, Why: why}
		return &miss{pass: passOf(c, nc), nc: nc}
	}
	if onRunner(c) && s.runners != nil {
		r, err := s.runners.ComputerRunner(ctx, c.ID)
		if err != nil || !r.Connected {
			return offline("it isn't connected to Nexul")
		}
	}
	if c.Facts != nil && c.Facts.T3.State != "" && c.Facts.T3.State != t3Answering {
		return offline("T3 Code isn't answering there")
	}
	return nil
}

// providerReady checks the run's provider and model against c's facts; a computer that never reported is left to the
// live gate.
func (s *Service) providerReady(ctx context.Context, p *plan, c Computer) (*miss, error) {
	setupMiss := func(name, id, why string) *miss {
		nc := &NotConfiguredError{Reason: ReasonSetupRequired, Computer: c.Name, ComputerID: c.ID, Provider: name, ProviderID: id, Why: why}
		return &miss{pass: passOf(c, nc), nc: nc}
	}
	if c.Facts == nil || len(c.Facts.Providers) == 0 {
		return nil, nil
	}
	setups, err := s.repo.ListProviderSetups(ctx, c.ID)
	if err != nil {
		return nil, fmt.Errorf("list provider setups for %s: %w", c.ID, err)
	}
	usable := func(f ProviderFacts) bool { return setupConfirmed(c, setups, f.Driver) }
	i := max(slices.IndexFunc(c.Facts.Providers, usable), 0)
	if p.pick.provider != "" {
		i = slices.IndexFunc(c.Facts.Providers, func(f ProviderFacts) bool { return f.ID == p.pick.provider })
	}
	if i < 0 {
		return setupMiss(p.pick.provider, p.pick.provider, ""), nil
	}
	f := c.Facts.Providers[i]
	if !usable(f) {
		why := ""
		if switchedOff(c, f.Driver) {
			why = "it's switched off"
		}
		return setupMiss(f.Name, f.ID, why), nil
	}
	if p.pick.model == "" || len(f.Models) == 0 || slices.ContainsFunc(f.Models, func(m ModelFacts) bool { return m.Slug == p.pick.model }) {
		return nil, nil
	}
	text := fmt.Sprintf("%s doesn't offer %s", c.Name, p.pick.model)
	return &miss{pass: PassOver{ComputerID: c.ID, Name: c.Name, Why: PassOverNoModel, Text: text}}, nil
}

// checkout is the T3 project the run uses on c: the one picked for the run, the chat folder outside a project, else
// the one picked for the project there, else the one checkout whose remote is a repository of the project.
func (s *Service) checkout(ctx context.Context, p *plan, c Computer) (string, *miss, error) {
	if p.req.HarnessProjectID != "" {
		return p.req.HarnessProjectID, nil, nil
	}
	if p.req.ProjectID == "" {
		if c.ChatProjectID != "" {
			return c.ChatProjectID, nil, nil
		}
		return "", missing(c, PassOverNoChatFolder, c.Name+" has no chat folder"), nil
	}
	listed := func(id string) bool {
		return c.Facts == nil || slices.ContainsFunc(c.Facts.Projects, func(hp harness.Project) bool { return hp.ID == id })
	}
	if picked := p.link.checkoutOn(c.ID); picked != "" && listed(picked) {
		return picked, nil, nil
	}
	remotes, err := s.remotes(ctx, p)
	if err != nil {
		return "", nil, err
	}
	var matches []string
	if c.Facts != nil {
		for _, hp := range c.Facts.Projects {
			if hp.Repository != "" && slices.Contains(remotes, strings.ToLower(hp.Repository)) {
				matches = append(matches, hp.ID)
			}
		}
	}
	if len(matches) == 1 {
		return matches[0], nil, nil
	}
	if len(matches) > 1 {
		return "", missing(c, PassOverAmbiguousCheckout, c.Name+" has several checkouts of this project; pick one"), nil
	}
	return "", missing(c, PassOverNoCheckout, c.Name+" doesn't have this project"), nil
}

func missing(c Computer, why PassOverReason, text string) *miss {
	return &miss{pass: PassOver{ComputerID: c.ID, Name: c.Name, Why: why, Text: text}}
}

// remotes reads the project's repositories once per run, on the first computer that needs them.
func (s *Service) remotes(ctx context.Context, p *plan) ([]string, error) {
	if p.readRepos || s.repositories == nil {
		return p.remotes, nil
	}
	remotes, err := s.repositories.ProjectRemotes(ctx, p.req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("read the repositories of project %s: %w", p.req.ProjectID, err)
	}
	p.remotes, p.readRepos = remotes, true
	return remotes, nil
}

// CheckoutOn is the checkout of projectID that userID's runs use on computerID, "" when it has none; it asks no computer.
func (s *Service) CheckoutOn(ctx context.Context, userID, projectID, computerID string) (string, error) {
	c, err := s.repo.GetComputer(ctx, userID, computerID)
	if err != nil {
		return "", err
	}
	link, err := s.repo.GetProjectLink(ctx, userID, projectID)
	if err != nil {
		return "", fmt.Errorf("get project link %s: %w", projectID, err)
	}
	checkout, _, err := s.checkout(ctx, &plan{req: RunRequest{UserID: userID, ProjectID: projectID}, link: link}, *c)
	return checkout, err
}

// passOf turns a refusal for one computer into its line.
func passOf(c Computer, nc *NotConfiguredError) PassOver {
	pass := PassOver{ComputerID: c.ID, Name: c.Name}
	if nc.Reason == ReasonSetupRequired {
		pass.Why, pass.Text = PassOverSetupRequired, fmt.Sprintf("%s isn't set up for %s", c.Name, nc.Provider)
		if nc.Why != "" {
			pass.Text = fmt.Sprintf("%s can't use %s: %s", c.Name, nc.Provider, nc.Why)
		}
		return pass
	}
	pass.Why, pass.Text = PassOverOffline, c.Name+" is offline: "+cmp.Or(nc.Why, "T3 Code isn't answering there")
	return pass
}

func passesBut(misses []miss, computerID string) []PassOver {
	out := make([]PassOver, 0, len(misses))
	for _, m := range misses {
		if m.pass.ComputerID != computerID {
			out = append(out, m.pass)
		}
	}
	return out
}

func allMissingProject(misses []miss) bool {
	return len(misses) > 0 && !slices.ContainsFunc(misses, func(m miss) bool { return !m.pass.missingProject() })
}

// refusal is why no computer took the run: the question on a pressed run no computer has the project for, the one
// computer's own refusal when there is one, else every computer's line.
func refusal(req RunRequest, computers []Computer, misses []miss) error {
	if len(computers) == 0 {
		return &NotConfiguredError{Reason: ReasonUnpaired}
	}
	passes := passesBut(misses, "")
	if req.Person && allMissingProject(misses) {
		return &NotConfiguredError{Reason: ReasonNeedsLocation, PassedOver: passes}
	}
	if len(misses) == 1 && misses[0].nc != nil {
		nc := *misses[0].nc
		nc.PassedOver = passes
		return &nc
	}
	return &NotConfiguredError{Reason: ReasonNoComputerReady, PassedOver: passes}
}

// ReorderComputers puts the caller's own computers in ids' order, which must name each of them once.
func (s *Service) ReorderComputers(ctx context.Context, userID string, ids []string) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	own, err := s.repo.ListComputers(ctx, userID)
	if err != nil {
		return fmt.Errorf("list computers: %w", err)
	}
	trimmed := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed = append(trimmed, strings.TrimSpace(id))
	}
	sorted := slices.Sorted(slices.Values(trimmed))
	mine := make([]string, 0, len(own))
	for _, c := range own {
		mine = append(mine, c.ID)
	}
	slices.Sort(mine)
	if !slices.Equal(sorted, mine) {
		return fmt.Errorf("%w: the order must name each of your computers once", apperrs.ErrInvalid)
	}
	if err := s.repo.ReorderComputers(ctx, userID, trimmed, s.now().UTC()); err != nil {
		return err
	}
	s.placementChanged(ctx, userID)
	return nil
}

// SetComputerPlacement sets one of the caller's own computers' runs at once (1 to 5) and chat folder.
func (s *Service) SetComputerPlacement(ctx context.Context, userID, computerID string, runsAtOnce int, chatProjectID string) (*Computer, error) {
	c, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if runsAtOnce < MinRunsAtOnce || runsAtOnce > MaxRunsAtOnce {
		return nil, fmt.Errorf("%w: runs at once must be %d to %d", apperrs.ErrInvalid, MinRunsAtOnce, MaxRunsAtOnce)
	}
	chatProjectID = strings.TrimSpace(chatProjectID)
	if err := s.repo.SetComputerPlacement(ctx, userID, c.ID, runsAtOnce, chatProjectID, s.now().UTC()); err != nil {
		return nil, err
	}
	s.placementChanged(ctx, userID)
	c.RunsAtOnce, c.ChatProjectID = runsAtOnce, chatProjectID
	c.BearerToken = ""
	return c, nil
}

// placementChanged tells the queue a person's runs may land somewhere new; a failed publish waits out the queue's retry.
func (s *Service) placementChanged(ctx context.Context, userID string) {
	if s.bus == nil {
		return
	}
	if err := s.bus.Publish(ctx, TopicPlacementChanged, PlacementChangedEvent{UserID: userID, MembersOnly: true}); err != nil {
		logging.FromCtx(ctx).Warn("publish computer placement change", "user_id", userID, "error", err)
	}
}

// withRunning sets how many play runs are going on each of the caller's own computers.
func (s *Service) withRunning(ctx context.Context, cs []Computer) ([]Computer, error) {
	ids := make([]string, 0, len(cs))
	for _, c := range cs {
		if c.Shared == nil {
			ids = append(ids, c.ID)
		}
	}
	running, err := s.repo.CountRunning(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range cs {
		cs[i].Running = running[cs[i].ID]
	}
	return cs, nil
}

// ComputerPatch is a change to one of the caller's own computers; a nil field keeps its value.
type ComputerPatch struct {
	Name          *string
	RunsAtOnce    *int
	ChatProjectID *string
}

// UpdateComputer renames the caller's own computer and sets its runs at once and chat folder, each only when given.
func (s *Service) UpdateComputer(ctx context.Context, userID, computerID string, patch ComputerPatch) (*Computer, error) {
	c, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if patch.Name != nil {
		if c, err = s.RenameComputer(ctx, userID, computerID, *patch.Name); err != nil {
			return nil, err
		}
	}
	if patch.RunsAtOnce == nil && patch.ChatProjectID == nil {
		c.BearerToken = ""
		return c, nil
	}
	runs, chat := runsAtOnce(*c), c.ChatProjectID
	if patch.RunsAtOnce != nil {
		runs = *patch.RunsAtOnce
	}
	if patch.ChatProjectID != nil {
		chat = *patch.ChatProjectID
	}
	return s.SetComputerPlacement(ctx, userID, computerID, runs, chat)
}

// MoveComputer puts one of the caller's own computers at position in their order, 0 first, clamped to the order's ends.
func (s *Service) MoveComputer(ctx context.Context, userID, computerID string, position int) error {
	c, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return err
	}
	own, err := s.repo.ListComputers(ctx, userID)
	if err != nil {
		return fmt.Errorf("list computers: %w", err)
	}
	ids := make([]string, 0, len(own))
	for _, o := range own {
		if o.ID != c.ID {
			ids = append(ids, o.ID)
		}
	}
	at := min(max(position, 0), len(ids))
	return s.ReorderComputers(ctx, userID, slices.Insert(ids, at, c.ID))
}
