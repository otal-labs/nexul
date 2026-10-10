package plays

import (
	"fmt"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Moment is what happens to a ticket or doc that an auto play waits for; the values are stored, so they never change.
type Moment string

const (
	MomentTicketUnblocked    Moment = "ticket.unblocked"
	MomentTicketEnteredStage Moment = "ticket.entered_stage"
	MomentTicketCreated      Moment = "ticket.created"
	MomentTicketDeveloperSet Moment = "ticket.developer_set"
	MomentTicketTesterSet    Moment = "ticket.tester_set"
	MomentTicketTestFailed   Moment = "ticket.test_failed"
	MomentDocCreated         Moment = "doc.created"
	MomentDocChanged         Moment = "doc.changed"
	// MomentAutomation marks a queued run an automation's runPlay asked for; no auto play waits for it.
	MomentAutomation Moment = "automation"
)

var momentPlayType = map[Moment]Type{
	MomentTicketUnblocked: TypeTicket, MomentTicketEnteredStage: TypeTicket, MomentTicketCreated: TypeTicket,
	MomentTicketDeveloperSet: TypeTicket, MomentTicketTesterSet: TypeTicket, MomentTicketTestFailed: TypeTicket,
	MomentDocCreated: TypeDoc, MomentDocChanged: TypeDoc,
}

// RunOn names whose computer an auto play's run lands on.
type RunOn string

const (
	RunOnDeveloper RunOn = "developer"
	RunOnTester    RunOn = "tester"
	RunOnCauser    RunOn = "causer"
)

// Level is an auto play run's place in its person's queue.
type Level string

const (
	LevelHigh   Level = "high"
	LevelNormal Level = "normal"
	LevelLow    Level = "low"
)

// Match says whether all or any of a group's members must hold.
type Match string

const (
	MatchAll Match = "all"
	MatchAny Match = "any"
)

// Field is a ticket's or doc's existing field a condition reads.
type Field string

const (
	FieldType      Field = "type"
	FieldProject   Field = "project"
	FieldStage     Field = "stage"
	FieldStatus    Field = "status"
	FieldCategory  Field = "category"
	FieldLabel     Field = "label"
	FieldDeveloper Field = "developer"
	FieldTester    Field = "tester"
	FieldSourceDoc Field = "source_doc"
	FieldLinkedPR  Field = "linked_pr"
	FieldBlocked   Field = "blocked"
	FieldFolder    Field = "folder"
)

// Op compares a field: is and is_not against any of the values, set and unset on the field alone.
type Op string

const (
	OpIs    Op = "is"
	OpIsNot Op = "is_not"
	OpSet   Op = "set"
	OpUnset Op = "unset"
)

var (
	valueOps    = []Op{OpIs, OpIsNot}
	presenceOps = []Op{OpSet, OpUnset}
	anyOp       = []Op{OpIs, OpIsNot, OpSet, OpUnset}
)

// fieldOps lists, per play type, the fields a condition may read and the ops each takes.
var fieldOps = map[Type]map[Field][]Op{
	TypeTicket: {
		FieldType: valueOps, FieldProject: valueOps, FieldStage: valueOps, FieldStatus: valueOps,
		FieldCategory: anyOp, FieldLabel: anyOp, FieldDeveloper: anyOp, FieldTester: anyOp,
		FieldSourceDoc: presenceOps, FieldLinkedPR: presenceOps, FieldBlocked: presenceOps,
	},
	TypeDoc: {FieldProject: valueOps, FieldFolder: valueOps},
}

// Bounds on what one auto play may hold, so a pasted tree cannot grow without limit.
const (
	MaxAutoPlayRules        = 20
	MaxAutoPlayRuleValues   = 50
	MaxOnceWithinMinutes    = 30 * 24 * 60
	MinAutoPlayDailyCap     = 1
	MaxAutoPlayDailyCap     = 50
	DefaultAutoPlayDailyCap = 5
)

// Conditions is a stack nested one level: the top matches all or any of its groups, each group all or any of its rules.
type Conditions struct {
	Match  Match   `json:"match" jsonschema:"all or any: whether every group must hold or one is enough." enum:"all,any"`
	Groups []Group `json:"groups" jsonschema:"The groups; none means the auto play fires on every match of its moment."`
}

// Group is one level of conditions.
type Group struct {
	Match Match  `json:"match" jsonschema:"all or any: whether every rule must hold or one is enough." enum:"all,any"`
	Rules []Rule `json:"rules" jsonschema:"The rules, at least one."`
}

// Rule compares one field; project, status and folder values are ids, type and category names, developer and tester user ids.
type Rule struct {
	Field  Field    `json:"field" jsonschema:"Ticket plays: type, project, stage, status, category, label, developer, tester, source_doc, linked_pr, or blocked. Doc plays: project or folder." enum:"type,project,stage,status,category,label,developer,tester,source_doc,linked_pr,blocked,folder"`
	Op     Op       `json:"op" jsonschema:"is or is_not compare with values; set or unset take no values (source_doc, linked_pr and blocked take only these)." enum:"is,is_not,set,unset"`
	Values []string `json:"values,omitempty" jsonschema:"For is and is_not, matched if the field is any of them: ids for project, status and folder, names for type, category and label, user ids for developer and tester, and backlog, progress, review, testing or done for stage."`
}

// Priority sets a run's level: the first rule whose group holds, else Otherwise.
type Priority struct {
	Rules     []PriorityRule `json:"rules" jsonschema:"Checked in order; the first that holds sets the level."`
	Otherwise Level          `json:"otherwise" jsonschema:"The level when no rule holds: high, normal, or low." enum:"high,normal,low"`
}

// PriorityRule is one "this level if this group holds".
type PriorityRule struct {
	Level Level `json:"level" jsonschema:"high, normal, or low." enum:"high,normal,low"`
	When  Group `json:"when" jsonschema:"The group that must hold, with the same rules as a condition group."`
}

// AutoPlay starts its play when a moment matches (ADR 0132); a play may have several.
type AutoPlay struct {
	ID          string `json:"id"`
	PlayID      string `json:"play_id"`
	WorkspaceID string `json:"workspace_id"`
	Enabled     bool   `json:"enabled"`
	Moment      Moment `json:"moment" enum:"ticket.unblocked,ticket.entered_stage,ticket.created,ticket.developer_set,ticket.tester_set,ticket.test_failed,doc.created,doc.changed"`
	// MomentStage is set only for ticket.entered_stage.
	MomentStage *Stage     `json:"moment_stage" enum:"backlog,progress,review,testing,done"`
	Conditions  Conditions `json:"conditions"`
	Priority    Priority   `json:"priority"`
	// OnceWithinMinutes allows one run per target per window after the moment; 0 is no limit.
	OnceWithinMinutes int       `json:"once_within_minutes"`
	RunOn             RunOn     `json:"run_on" enum:"developer,tester,causer"`
	CreatedBy         string    `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// normalize fills the defaults an omitted part means and tidies values, so equal auto plays store equal.
func (a *AutoPlay) normalize(playType Type) {
	if a.Conditions.Match == "" {
		a.Conditions.Match = MatchAll
	}
	if a.Conditions.Groups == nil {
		a.Conditions.Groups = []Group{}
	}
	for i := range a.Conditions.Groups {
		normalizeGroup(&a.Conditions.Groups[i])
	}
	if a.Priority.Otherwise == "" {
		a.Priority.Otherwise = LevelNormal
	}
	if a.Priority.Rules == nil {
		a.Priority.Rules = []PriorityRule{}
	}
	for i := range a.Priority.Rules {
		normalizeGroup(&a.Priority.Rules[i].When)
	}
	if a.RunOn == "" {
		a.RunOn = RunOnDeveloper
		if playType == TypeDoc {
			a.RunOn = RunOnCauser
		}
	}
	if a.Moment != MomentTicketEnteredStage {
		a.MomentStage = nil
	}
}

func normalizeGroup(g *Group) {
	if g.Match == "" {
		g.Match = MatchAll
	}
	for i := range g.Rules {
		values := make([]string, 0, len(g.Rules[i].Values))
		for _, v := range g.Rules[i].Values {
			if v = strings.TrimSpace(v); v != "" && !slices.Contains(values, v) {
				values = append(values, v)
			}
		}
		g.Rules[i].Values = values
	}
}

// Validate checks an auto play against its play's type; ids it names are not checked, since they can go stale anyway.
func (a *AutoPlay) Validate(playType Type) error {
	if playType == TypeInterview {
		return fmt.Errorf("%w: an interview play takes no auto plays; only ticket and doc plays do", apperrs.ErrInvalid)
	}
	if err := a.validateMoment(playType); err != nil {
		return err
	}
	if err := a.validateRunOn(playType); err != nil {
		return err
	}
	if a.OnceWithinMinutes < 0 || a.OnceWithinMinutes > MaxOnceWithinMinutes {
		return fmt.Errorf("%w: once_within_minutes must be 0 (no limit) to %d (30 days)", apperrs.ErrInvalid, MaxOnceWithinMinutes)
	}
	if !a.Conditions.Match.valid() {
		return fmt.Errorf("%w: conditions match must be all or any", apperrs.ErrInvalid)
	}
	rules := 0
	for i, g := range a.Conditions.Groups {
		if err := validateGroup(g, playType, fmt.Sprintf("conditions group %d", i+1)); err != nil {
			return err
		}
		rules += len(g.Rules)
	}
	if !a.Priority.Otherwise.valid() {
		return fmt.Errorf("%w: priority otherwise must be high, normal, or low", apperrs.ErrInvalid)
	}
	for i, r := range a.Priority.Rules {
		if !r.Level.valid() {
			return fmt.Errorf("%w: priority rule %d: level must be high, normal, or low", apperrs.ErrInvalid, i+1)
		}
		if err := validateGroup(r.When, playType, fmt.Sprintf("priority rule %d", i+1)); err != nil {
			return err
		}
		rules += len(r.When.Rules)
	}
	if rules > MaxAutoPlayRules {
		return fmt.Errorf("%w: an auto play holds at most %d rules across its conditions and priority; this one has %d", apperrs.ErrInvalid, MaxAutoPlayRules, rules)
	}
	return nil
}

func (a *AutoPlay) validateMoment(playType Type) error {
	want, ok := momentPlayType[a.Moment]
	if !ok {
		return fmt.Errorf("%w: moment must be one of ticket.unblocked, ticket.entered_stage, ticket.created, ticket.developer_set, ticket.tester_set, ticket.test_failed, doc.created, doc.changed", apperrs.ErrInvalid)
	}
	if want != playType {
		return fmt.Errorf("%w: moment %s belongs to %s plays; this is a %s play", apperrs.ErrInvalid, a.Moment, want, playType)
	}
	if a.Moment != MomentTicketEnteredStage {
		return nil
	}
	if a.MomentStage == nil || !a.MomentStage.valid() {
		return fmt.Errorf("%w: ticket.entered_stage needs moment_stage: backlog, progress, review, testing, or done", apperrs.ErrInvalid)
	}
	return nil
}

func (a *AutoPlay) validateRunOn(playType Type) error {
	if playType == TypeDoc && a.RunOn != RunOnCauser {
		return fmt.Errorf("%w: a doc play's auto play runs on causer, whoever created or changed the doc; a doc has no developer or tester", apperrs.ErrInvalid)
	}
	if a.RunOn != RunOnDeveloper && a.RunOn != RunOnTester && a.RunOn != RunOnCauser {
		return fmt.Errorf("%w: run_on must be developer, tester, or causer", apperrs.ErrInvalid)
	}
	return nil
}

func validateGroup(g Group, playType Type, where string) error {
	if !g.Match.valid() {
		return fmt.Errorf("%w: %s: match must be all or any", apperrs.ErrInvalid, where)
	}
	if len(g.Rules) == 0 {
		return fmt.Errorf("%w: %s has no rules; remove the group or give it one", apperrs.ErrInvalid, where)
	}
	for i, r := range g.Rules {
		if err := validateRule(r, playType); err != nil {
			return fmt.Errorf("%s, rule %d: %w", where, i+1, err)
		}
	}
	return nil
}

func validateRule(r Rule, playType Type) error {
	ops, ok := fieldOps[playType][r.Field]
	if !ok {
		names := make([]string, 0, len(fieldOps[playType]))
		for f := range fieldOps[playType] {
			names = append(names, string(f))
		}
		slices.Sort(names)
		return fmt.Errorf("%w: field %q is not one a %s play reads; use %s", apperrs.ErrInvalid, r.Field, playType, strings.Join(names, ", "))
	}
	if !slices.Contains(ops, r.Op) {
		return fmt.Errorf("%w: field %s takes the ops %s, not %q", apperrs.ErrInvalid, r.Field, joinOps(ops), r.Op)
	}
	if r.Op == OpSet || r.Op == OpUnset {
		if len(r.Values) > 0 {
			return fmt.Errorf("%w: %s takes no values", apperrs.ErrInvalid, r.Op)
		}
		return nil
	}
	if len(r.Values) == 0 {
		return fmt.Errorf("%w: %s needs at least one value", apperrs.ErrInvalid, r.Op)
	}
	if len(r.Values) > MaxAutoPlayRuleValues {
		return fmt.Errorf("%w: a rule takes at most %d values", apperrs.ErrInvalid, MaxAutoPlayRuleValues)
	}
	if r.Field != FieldStage {
		return nil
	}
	for _, v := range r.Values {
		if !Stage(v).valid() {
			return fmt.Errorf("%w: stage %q is not one of backlog, progress, review, testing, done", apperrs.ErrInvalid, v)
		}
	}
	return nil
}

func joinOps(ops []Op) string {
	names := make([]string, len(ops))
	for i, o := range ops {
		names[i] = string(o)
	}
	return strings.Join(names, ", ")
}

func (m Match) valid() bool { return m == MatchAll || m == MatchAny }

func (l Level) valid() bool { return l == LevelHigh || l == LevelNormal || l == LevelLow }
