package plays

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// KeptInstructionVersions and KeptChanges cap a play's history; a save prunes the oldest rows past them.
const (
	KeptInstructionVersions = 10
	KeptChanges             = 100
)

// Changed names the field of a Change.
const (
	ChangedLabel         = "label"
	ChangedDescription   = "description"
	ChangedEnabled       = "enabled"
	ChangedShowWhenStage = "show_when_stage"
	ChangedExcluded      = "excluded_project_ids"
)

// Actor kinds of a Change: a person, a person's agent over MCP, or an automation.
const (
	ActorKindUser       = "user"
	ActorKindUserMCP    = ActorKindUser + ":mcp"
	ActorKindAutomation = "automation"
)

// Actor is who made a change; it has the shape of a ticket's change actor, so one history dialog reads both.
type Actor struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id,omitempty"`
	AutomationID   string `json:"automation_id,omitempty"`
	AutomationName string `json:"automation_name,omitempty"`
	PlayLabel      string `json:"play_label,omitempty"`
	TrailID        string `json:"trail_id,omitempty"`
}

// VersionInfo is one kept version of a play's instructions, without its text.
type VersionInfo struct {
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	// Name is empty: plays keep no named versions.
	Name           string   `json:"name"`
	EditorIDs      []string `json:"editor_ids"`
	Via            string   `json:"via"`
	Current        bool     `json:"current"`
	BuiltinDefault bool     `json:"builtin_default"`
	// WordsAdded and WordsRemoved size the version against the kept one before it; nil on the oldest kept.
	WordsAdded   *int `json:"words_added"`
	WordsRemoved *int `json:"words_removed"`
	// RestoredFrom is the version number a restore copied this version from; nil for any other version.
	RestoredFrom *int `json:"restored_from"`
}

// VersionBody is a kept version with its text.
type VersionBody struct {
	VersionInfo
	Body string `json:"body"`
}

// Change is one changed field of a play: its old and new value as text.
type Change struct {
	ID        string    `json:"id"`
	Field     string    `json:"field"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Actor     Actor     `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
}

// History is a play's kept versions and changes, newest first, with no version text.
type History struct {
	Versions []VersionInfo `json:"versions"`
	Changes  []Change      `json:"changes"`
}

// NewVersion is the instructions a save adds to a play's history; the repo numbers it.
type NewVersion struct {
	Body           string
	EditorIDs      []string
	Via            string
	BuiltinDefault bool
	// RestoredFrom is the version number a restore copied the text from; zero for an ordinary save.
	RestoredFrom int
}

// Revision is what a save adds to a play's history, written in the save's own transaction.
type Revision struct {
	// Instructions is nil when the save left the instructions as they were.
	Instructions *NewVersion
	Changes      []Change
}

// HistoryRepo reads a play's history; Create and Update on Repo write a Revision and prune past the kept caps.
type HistoryRepo interface {
	// ListHistory returns the kept versions and changes newest first, no version text.
	ListHistory(ctx context.Context, playID string) (History, error)
	// GetVersion returns one kept version with its text, ErrNotFound when it was never written or has been pruned.
	GetVersion(ctx context.Context, playID string, version int) (*VersionBody, error)
}

// viaMCP is the via of every version saved through an MCP tool call, as for memories (ADR 0049).
const viaMCP = "mcp"

type viaMCPKey struct{}

// WithViaMCP marks a call as a person's over MCP, so the changes it makes are recorded as user:mcp.
func WithViaMCP(ctx context.Context) context.Context {
	return context.WithValue(ctx, viaMCPKey{}, true)
}

// historyActor is the request's actor, the user ids a version credits, and the automation name or "mcp" it came through.
func historyActor(ctx context.Context) (actor Actor, editors []string, via string) {
	a, _ := identity.ActorFromCtx(ctx)
	if a.Automation != nil {
		return Actor{Kind: ActorKindAutomation, AutomationID: a.Automation.ID, AutomationName: a.Automation.Name}, []string{}, a.Automation.Name
	}
	editors = []string{}
	if a.ID != "" {
		editors = append(editors, a.ID)
	}
	if ctx.Value(viaMCPKey{}) != nil {
		return Actor{Kind: ActorKindUserMCP, UserID: a.ID}, editors, viaMCP
	}
	return Actor{Kind: ActorKindUser, UserID: a.ID}, editors, ""
}

// creation is the first version of a new play.
func creation(ctx context.Context, p *Play) Revision {
	_, editors, via := historyActor(ctx)
	return Revision{Instructions: &NewVersion{Body: p.Instructions, EditorIDs: editors, Via: via, BuiltinDefault: p.BuiltinKey != ""}}
}

// revision is what a save from before to after adds to the history: a version when the instructions changed, a
// change per other changed field, and nothing when the save changed nothing.
func revision(ctx context.Context, before, after *Play, restoredFrom int) Revision {
	actor, editors, via := historyActor(ctx)
	var rev Revision
	if before.Instructions != after.Instructions {
		rev.Instructions = &NewVersion{Body: after.Instructions, EditorIDs: editors, Via: via, RestoredFrom: restoredFrom}
	}
	add := func(field, from, to string) {
		if from != to {
			rev.Changes = append(rev.Changes, Change{ID: ids.New(), Field: field, From: from, To: to, Actor: actor, CreatedAt: after.UpdatedAt})
		}
	}
	add(ChangedLabel, before.Label, after.Label)
	add(ChangedDescription, before.Description, after.Description)
	add(ChangedEnabled, strconv.FormatBool(before.Enabled), strconv.FormatBool(after.Enabled))
	add(ChangedShowWhenStage, stageText(before.ShowWhenStage), stageText(after.ShowWhenStage))
	add(ChangedExcluded, projectsText(before.ExcludedProjectIDs), projectsText(after.ExcludedProjectIDs))
	return rev
}

func stageText(s *Stage) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

// projectsText is the sorted ids as one comma-separated line, so two saves of the same set read as equal.
func projectsText(projectIDs []string) string {
	return strings.Join(normalizeIDs(projectIDs), ",")
}

// History returns a play's kept versions and changes, newest first; plays:read on the play.
func (s *Service) History(ctx context.Context, id string) (*History, error) {
	p, err := s.seenByID(ctx, id)
	if err != nil {
		return nil, err
	}
	h, err := s.repo.ListHistory(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("list history of play %s: %w", p.ID, err)
	}
	for i := range h.Versions {
		h.Versions[i].Current = i == 0
	}
	return &h, nil
}

// Version returns one kept version of a play's instructions with its text; plays:read on the play.
func (s *Service) Version(ctx context.Context, id string, version int) (*VersionBody, error) {
	p, err := s.seenByID(ctx, id)
	if err != nil {
		return nil, err
	}
	v, err := s.version(ctx, p, version)
	if err != nil {
		return nil, err
	}
	current, err := s.repo.ListHistory(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("list history of play %s: %w", p.ID, err)
	}
	v.Current = len(current.Versions) > 0 && current.Versions[0].Version == v.Version
	return v, nil
}

func (s *Service) version(ctx context.Context, p *Play, version int) (*VersionBody, error) {
	if version < 1 {
		return nil, fmt.Errorf("%w: version must be positive", apperrs.ErrInvalid)
	}
	v, err := s.repo.GetVersion(ctx, p.ID, version)
	if err != nil {
		return nil, fmt.Errorf("get version %d of play %s: %w", version, p.ID, err)
	}
	return v, nil
}

// Restore saves a kept version's text as the play's instructions through Update, so it lands as a new version on
// top and nothing is lost; plays:write on the play.
func (s *Service) Restore(ctx context.Context, id string, version int) (*Play, error) {
	p, err := s.seenByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.requireOn(ctx, p, permissions.PlaysWrite); err != nil {
		return nil, err
	}
	v, err := s.version(ctx, p, version)
	if err != nil {
		return nil, err
	}
	return s.update(ctx, p.WorkspaceID, p.ID, UpdateInput{
		Label: p.Label, Description: p.Description, Instructions: v.Body, Enabled: p.Enabled,
		ShowWhenStage: p.ShowWhenStage, ExcludedProjectIDs: slices.Clone(p.ExcludedProjectIDs),
	}, version)
}

// seenByID is the play id's row for a caller who can see it, found without naming its workspace.
func (s *Service) seenByID(ctx context.Context, id string) (*Play, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: play id is required", apperrs.ErrInvalid)
	}
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get play %s: %w", id, err)
	}
	if !s.sees(ctx, p) {
		return nil, fmt.Errorf("get play %s: %w", p.ID, apperrs.ErrNotFound)
	}
	return p, nil
}
