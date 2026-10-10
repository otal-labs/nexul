package plays

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Facts is a ticket or doc as an auto play's conditions read it now; type and category are names, developer and
// tester user ids, status a column id.
type Facts struct {
	ProjectID string
	FolderID  string
	Type      string
	Stage     Stage
	Status    string
	Category  string
	Labels    []string
	Developer string
	Tester    string
	SourceDoc bool
	LinkedPR  bool
	Blocked   bool
	Archived  bool
}

// FactReader reads a ticket's or doc's facts without an access check, since matching runs in the background as
// nobody; a deleted target is ErrNotFound.
type FactReader interface {
	Facts(ctx context.Context, targetType TargetType, id string) (Facts, error)
}

var factFields = map[Field]func(Facts) []string{
	FieldType:      func(f Facts) []string { return present(f.Type) },
	FieldProject:   func(f Facts) []string { return present(f.ProjectID) },
	FieldStage:     func(f Facts) []string { return present(string(f.Stage)) },
	FieldStatus:    func(f Facts) []string { return present(f.Status) },
	FieldCategory:  func(f Facts) []string { return present(f.Category) },
	FieldLabel:     func(f Facts) []string { return f.Labels },
	FieldDeveloper: func(f Facts) []string { return present(f.Developer) },
	FieldTester:    func(f Facts) []string { return present(f.Tester) },
	FieldSourceDoc: func(f Facts) []string { return flag(f.SourceDoc) },
	FieldLinkedPR:  func(f Facts) []string { return flag(f.LinkedPR) },
	FieldBlocked:   func(f Facts) []string { return flag(f.Blocked) },
	FieldFolder:    func(f Facts) []string { return present(f.FolderID) },
}

func present(v string) []string {
	if v == "" {
		return nil
	}
	return []string{v}
}

func flag(set bool) []string {
	if !set {
		return nil
	}
	return []string{"set"}
}

// holds compares names without case, as a person typing a label or picking a type expects.
func (r Rule) holds(f Facts) bool {
	read, ok := factFields[r.Field]
	if !ok {
		return false
	}
	got := read(f)
	switch r.Op {
	case OpSet:
		return len(got) > 0
	case OpUnset:
		return len(got) == 0
	case OpIsNot:
		return !anyEqual(got, r.Values)
	}
	return anyEqual(got, r.Values)
}

func anyEqual(got, want []string) bool {
	return slices.ContainsFunc(got, func(g string) bool {
		return slices.ContainsFunc(want, func(w string) bool { return strings.EqualFold(g, w) })
	})
}

func (g Group) holds(f Facts) bool {
	return matches(g.Match, len(g.Rules), func(i int) bool { return g.Rules[i].holds(f) })
}

// holds is true for no groups at all: such an auto play fires on every match of its moment.
func (c Conditions) holds(f Facts) bool {
	if len(c.Groups) == 0 {
		return true
	}
	return matches(c.Match, len(c.Groups), func(i int) bool { return c.Groups[i].holds(f) })
}

func matches(m Match, n int, holds func(int) bool) bool {
	anyOf := m == MatchAny
	for i := range n {
		if holds(i) == anyOf {
			return anyOf
		}
	}
	return !anyOf
}

// level is the first rule's level whose group holds, else Otherwise.
func (p Priority) level(f Facts) Level {
	for _, r := range p.Rules {
		if r.When.holds(f) {
			return r.Level
		}
	}
	return p.Otherwise
}

// moment is one thing that happened to a ticket or doc, as the matcher reads it off an event.
type moment struct {
	name       Moment
	stage      Stage
	targetType TargetType
	targetID   string
	projectID  string
	causerID   string
	// automation marks a moment an automation caused; it runs on the developer, as nobody pressed anything.
	automation bool
	via        Via
}

// person is whose computer a's run lands on.
func (a *AutoPlay) person(f Facts, m moment) string {
	if a.RunOn == RunOnDeveloper {
		return f.Developer
	}
	if a.RunOn == RunOnTester {
		return f.Tester
	}
	if m.automation {
		return f.Developer
	}
	return m.causerID
}

// momentPayload is the union of the event payloads moments come from, declared here so plays never imports tickets or docs.
type momentPayload struct {
	TicketID  string `json:"ticket_id"`
	ProjectID string `json:"project_id"`
	Ticket    struct {
		ID        string `json:"id"`
		ProjectID string `json:"project_id"`
		Reporter  struct {
			Kind  string `json:"kind"`
			Login string `json:"login"`
		} `json:"reporter"`
	} `json:"ticket"`
	Doc struct {
		ID        string `json:"id"`
		ProjectID string `json:"project_id"`
	} `json:"doc"`
	From    string `json:"from"`
	To      string `json:"to"`
	Tester  string `json:"tester"`
	First   bool   `json:"first"`
	ActorID string `json:"actor_id"`
	Actor   struct {
		Kind   string `json:"kind"`
		UserID string `json:"user_id"`
	} `json:"actor"`
}

// MomentTopics are the topics auto play moments come from; HandleAutoPlayMoment is subscribed to each.
var MomentTopics = []string{
	"ticket.unblocked", "ticket.created", "ticket.status_changed", "ticket.developer_changed", "ticket.tester_changed",
	"ticket.test_failed", "doc.settled",
}

// HandleAutoPlayMoment queues a run for every switched-on auto play the event's moment matches (ADR 0132).
func (r *Runner) HandleAutoPlayMoment(ctx context.Context, ev eventbus.Event) error {
	if r.queue == nil || r.facts == nil {
		return nil
	}
	var p momentPayload
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	m, ok, err := r.momentOf(ctx, ev.Topic, p)
	if err != nil {
		return apperrs.Retryable(err)
	}
	if !ok {
		return nil
	}
	if err := r.match(ctx, m); err != nil {
		return apperrs.Retryable(err)
	}
	return nil
}

// momentOf reads the moment an event carries; false when it carries none, such as a developer cleared or a move
// within one stage.
func (r *Runner) momentOf(ctx context.Context, topic string, p momentPayload) (moment, bool, error) {
	ticket := moment{targetType: TargetTicket, targetID: p.Ticket.ID, projectID: p.Ticket.ProjectID, via: ViaWeb}
	ticket.actor(p.Actor.Kind, p.Actor.UserID)
	switch topic {
	case "ticket.unblocked":
		ticket.name, ticket.targetID, ticket.projectID = MomentTicketUnblocked, p.TicketID, p.ProjectID
	case "ticket.created":
		ticket.name = MomentTicketCreated
		ticket.actor(p.Ticket.Reporter.Kind, "")
		return r.withLogin(ctx, ticket, p.Ticket.Reporter.Login)
	case "ticket.status_changed":
		stage, entered, err := r.enteredStage(ctx, p.From, p.To)
		ticket.name, ticket.stage = MomentTicketEnteredStage, stage
		return ticket, entered, err
	case "ticket.developer_changed", "ticket.tester_changed":
		ticket.name = MomentTicketDeveloperSet
		if topic == "ticket.tester_changed" {
			ticket.name = MomentTicketTesterSet
		}
		return ticket, p.To != "", nil
	case "ticket.test_failed":
		ticket.name = MomentTicketTestFailed
		return r.withLogin(ctx, ticket, p.Tester)
	case "doc.settled":
		doc := moment{name: MomentDocChanged, targetType: TargetDoc, targetID: p.Doc.ID, projectID: p.Doc.ProjectID, causerID: p.ActorID, via: ViaWeb}
		if p.First {
			doc.name = MomentDocCreated
		}
		return doc, doc.targetID != "", nil
	}
	return ticket, ticket.name != "" && ticket.targetID != "", nil
}

// actor records who caused the moment: an automation, or a person and whether they acted through MCP.
func (m *moment) actor(kind, userID string) {
	m.causerID, m.automation = userID, kind == "automation"
	if strings.HasSuffix(kind, ":mcp") {
		m.via = ViaMCP
	}
}

// withLogin sets the causer from a login, as ticket.created and ticket.test_failed name people.
func (r *Runner) withLogin(ctx context.Context, m moment, login string) (moment, bool, error) {
	if login == "" || r.users == nil || m.automation {
		return m, m.targetID != "", nil
	}
	id, err := r.users.UserID(ctx, login)
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return m, false, fmt.Errorf("resolve %s: %w", login, err)
	}
	m.causerID = id
	return m, m.targetID != "", nil
}

// enteredStage is the stage a move lands in, and whether it entered it rather than moving within it.
func (r *Runner) enteredStage(ctx context.Context, from, to string) (Stage, bool, error) {
	toStatus, err := r.targets.GetStatus(ctx, to)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get status %s: %w", to, err)
	}
	if from == "" {
		return toStatus.Stage, true, nil
	}
	fromStatus, err := r.targets.GetStatus(ctx, from)
	if errors.Is(err, apperrs.ErrNotFound) {
		return toStatus.Stage, true, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get status %s: %w", from, err)
	}
	return toStatus.Stage, fromStatus.Stage != toStatus.Stage, nil
}

// match queues m on every switched-on auto play of the workspace waiting for it whose play and conditions hold now.
func (r *Runner) match(ctx context.Context, m moment) error {
	workspaceID, err := r.projects.WorkspaceForProject(ctx, m.projectID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve workspace for project %s: %w", m.projectID, err)
	}
	autoPlays, err := r.plays.ListEnabledAutoPlays(ctx, workspaceID, m.name)
	if err != nil || len(autoPlays) == 0 {
		return err
	}
	f, err := r.facts.Facts(ctx, m.targetType, m.targetID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s %s: %w", m.targetType, m.targetID, err)
	}
	queued := false
	for _, a := range autoPlays {
		ok, err := r.matchOne(ctx, workspaceID, a, m, f)
		if err != nil {
			return err
		}
		queued = queued || ok
	}
	if queued {
		r.Kick()
	}
	return nil
}

// matchOne queues a's run on m, or records that it didn't run when nobody may run it; true when a run now waits.
func (r *Runner) matchOne(ctx context.Context, workspaceID string, a *AutoPlay, m moment, f Facts) (bool, error) {
	if a.Moment == MomentTicketEnteredStage && (a.MomentStage == nil || *a.MomentStage != m.stage) {
		return false, nil
	}
	play, err := r.plays.Get(ctx, a.PlayID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get play %s: %w", a.PlayID, err)
	}
	if !play.Enabled || slices.Contains(play.ExcludedProjectIDs, f.ProjectID) || f.Archived || !a.Conditions.holds(f) {
		return false, nil
	}
	now := r.now().UTC()
	if a.OnceWithinMinutes > 0 {
		since := now.Add(-time.Duration(a.OnceWithinMinutes) * time.Minute)
		ran, err := r.queue.QueuedSince(ctx, a.ID, m.targetType, m.targetID, since)
		if err != nil || ran {
			return false, err
		}
	}
	it := &QueueItem{
		ID: ids.New(), WorkspaceID: workspaceID, ProjectID: f.ProjectID, TargetType: m.targetType, TargetID: m.targetID,
		PlayID: play.ID, PlayLabel: play.Label, AutoPlayID: a.ID, PersonID: a.person(f, m), RunOn: a.RunOn, Moment: m.name,
		Priority: a.Priority.level(f), Status: QueueQueued, Via: m.via, QueuedAt: now, NotBefore: now,
	}
	if reason := r.whyNobody(ctx, it); reason != "" {
		r.failedTrail(ctx, it, reason)
		it.Status, it.Reason, it.DecidedAt = QueueDidntRun, reason, &now
		_, err := r.queue.EnqueueRun(ctx, it, r.queueEvent(TopicQueued, it))
		return false, err
	}
	return r.queue.EnqueueRun(ctx, it, r.queueEvent(TopicQueued, it))
}

// whyNobody says why a run has nobody to land on: no such person, or one the play's permissions leave out.
func (r *Runner) whyNobody(ctx context.Context, it *QueueItem) string {
	if it.PersonID == "" {
		return fmt.Sprintf("nobody to run it on: the %s has no %s", it.TargetType, it.RunOn)
	}
	if !r.perm.HasPermission(ctx, it.PersonID, it.WorkspaceID, permissions.Member, resourceTypeProject, it.ProjectID) ||
		!r.perm.HasPermission(ctx, it.PersonID, it.WorkspaceID, permissions.PlaysRun, resourceTypePlay, it.PlayID) {
		return fmt.Sprintf("%s may not run %s here", r.login(ctx, it.PersonID), it.PlayLabel)
	}
	return ""
}

// failedTrail keeps a run that didn't run as a failed trail on its target, as the decisions check's refusals are.
func (r *Runner) failedTrail(ctx context.Context, it *QueueItem, reason string) {
	trail := &Trail{
		ID: ids.New(), WorkspaceID: it.WorkspaceID, PlayID: it.PlayID, PlayLabel: it.PlayLabel, TargetType: it.TargetType,
		TargetID: it.TargetID, ProjectID: it.ProjectID, StarterID: it.PersonID, Via: it.Via, SelectedMemoryIDs: []string{},
		StartedAt: r.now().UTC(),
	}
	title := ""
	if it.PersonID != "" {
		if tgt, err := r.readTarget(identity.WithActor(ctx, identity.Actor{ID: it.PersonID}), it.TargetType, it.TargetID); err == nil {
			title = tgt.title
		}
	}
	r.createFailed(ctx, trail, title, reason)
	it.TrailID = trail.ID
}
