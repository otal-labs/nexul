package tickets

import (
	"context"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// A ticket keeps its newest ten description versions and its newest hundred field changes.
const (
	KeptDescriptionVersions = 10
	KeptChanges             = 100
)

// The fields a Change names; position and internal fields are never recorded.
const (
	FieldCreated    = "created"
	FieldTitle      = "title"
	FieldStatus     = "status"
	FieldCategory   = "category"
	FieldType       = "type"
	FieldDeveloper  = "developer"
	FieldTester     = "tester"
	FieldDueOn      = "due_on"
	FieldLabels     = "labels"
	FieldBlockedBy  = "blocked_by"
	FieldFoundIn    = "found_in"
	FieldDoc        = "doc"
	FieldFinishedAt = "finished_at"
)

// OriginUnknown is what a found-in change shows for a bug whose origin nobody knows.
const OriginUnknown = "unknown"

// viaMCP is the via of a description version saved through an MCP tool call (ADR 0049).
const viaMCP = "mcp"

// Change is one field a save changed, as the history lists it: From and To are display-ready, empty for nothing.
type Change struct {
	ID        string    `json:"id"`
	Field     string    `json:"field"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Actor     Actor     `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
}

// VersionSummary is a description version in a ticket's history, without its body; Name is always empty, as tickets
// keep no named versions.
type VersionSummary struct {
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	EditorIDs []string  `json:"editor_ids"`
	Via       string    `json:"via"`
	// WordsAdded and WordsRemoved size the version against the kept one before it; nil on the oldest kept.
	WordsAdded   *int `json:"words_added"`
	WordsRemoved *int `json:"words_removed"`
	// RestoredFrom is the version number a restore copied this version from; nil for any other version.
	RestoredFrom *int `json:"restored_from"`
	// Current marks the newest version.
	Current bool `json:"current"`
}

// DescriptionVersion is one kept text of a ticket's description; EditorIDs is who it credits and Via the automation
// it came through, "mcp" for an agent's save, empty for the browser.
type DescriptionVersion struct {
	TicketID  string    `json:"ticket_id"`
	Version   int       `json:"version"`
	Body      string    `json:"body"`
	EditorIDs []string  `json:"editor_ids"`
	Via       string    `json:"via"`
	CreatedAt time.Time `json:"created_at"`
}

// History is what a ticket's history route answers, newest first, with no bodies.
type History struct {
	Versions []VersionSummary `json:"versions"`
	Changes  []Change         `json:"changes"`
}

// Credit asks a title and body write to leave a description version crediting EditorIDs, unless the newest version
// already holds the body.
type Credit struct {
	EditorIDs []string
	Via       string
	// RestoredFrom is the version number a restore copied the body from; zero for an ordinary save.
	RestoredFrom int
}

// HistoryRepo reads a ticket's description versions and field changes; the writes ride with the ticket's own, in the
// same transaction.
type HistoryRepo interface {
	ListDescriptionVersions(ctx context.Context, id string) ([]VersionSummary, error)
	GetDescriptionVersion(ctx context.Context, id string, version int) (*DescriptionVersion, error)
	ListChanges(ctx context.Context, id string) ([]Change, error)
	// AppendSittingVersion writes the ticket's current body as a version crediting editorIDs, unless the newest
	// version already holds it, then prunes like a save does.
	AppendSittingVersion(ctx context.Context, id string, editorIDs []string, at time.Time) error
}

// ActorOf is the actor a save is recorded under: an automation, a person, or a person's agent over MCP.
func ActorOf(ctx context.Context) Actor {
	actor, _ := identity.ActorFromCtx(ctx)
	if actor.Automation != nil {
		return Actor{Kind: ActorKindAutomation, AutomationID: actor.Automation.ID, AutomationName: actor.Automation.Name}
	}
	if ctx.Value(viaMCPKey{}) != nil {
		return Actor{Kind: ActorKindUserMCP, UserID: actor.ID}
	}
	return Actor{Kind: ActorKindUser, UserID: actor.ID}
}

// ViaOf is what a description version records as the way a save came: the automation's name, "mcp" for an agent, else "".
func ViaOf(ctx context.Context) string {
	actor, _ := identity.ActorFromCtx(ctx)
	if actor.Automation != nil {
		return actor.Automation.Name
	}
	if ctx.Value(viaMCPKey{}) != nil {
		return viaMCP
	}
	return ""
}

// creditOf credits the save's person; an automation or a call with nobody behind it credits no one.
func creditOf(ctx context.Context) *Credit {
	editors := []string{}
	if id := ActorOf(ctx).UserID; id != "" {
		editors = append(editors, id)
	}
	return &Credit{EditorIDs: editors, Via: ViaOf(ctx)}
}

// changed is the one change a field's move from one value to another makes; the same value makes none.
func (s *Service) changed(actor Actor, field, from, to string) []Change {
	if from == to {
		return nil
	}
	return []Change{{ID: ids.New(), Field: field, From: from, To: to, Actor: actor, CreatedAt: s.now().UTC()}}
}

// statusName shows a status by its column's name; an id the store no longer knows shows as itself.
func (s *Service) statusName(ctx context.Context, id Status) string {
	name, err := s.statuses.Name(ctx, string(id))
	if err != nil || name == "" {
		return string(id)
	}
	return name
}

// typeName shows a ticket type by its name; without the lookup, or for an id it no longer knows, it shows the id.
func (s *Service) typeName(ctx context.Context, typeID string) string {
	if typeID == "" || s.types == nil {
		return typeID
	}
	name, err := s.types.TypeName(ctx, typeID)
	if err != nil || name == "" {
		return typeID
	}
	return name
}

// History returns the ticket's description versions and field changes newest first, without bodies; needs tickets:read.
func (s *Service) History(ctx context.Context, id string) (*History, error) {
	if _, err := s.load(ctx, id, permissions.TicketsRead); err != nil {
		return nil, fmt.Errorf("history of ticket %s: %w", id, err)
	}
	versions, err := s.repo.ListDescriptionVersions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("history of ticket %s: %w", id, err)
	}
	changes, err := s.repo.ListChanges(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("history of ticket %s: %w", id, err)
	}
	if len(versions) > 0 {
		versions[0].Current = true
	}
	return &History{Versions: versions, Changes: changes}, nil
}

// GetVersion returns one kept description version with its body; needs tickets:read.
func (s *Service) GetVersion(ctx context.Context, id string, version int) (*DescriptionVersion, error) {
	if _, err := s.load(ctx, id, permissions.TicketsRead); err != nil {
		return nil, fmt.Errorf("version %d of ticket %s: %w", version, id, err)
	}
	v, err := s.repo.GetDescriptionVersion(ctx, id, version)
	if err != nil {
		return nil, fmt.Errorf("version %d of ticket %s: %w", version, id, err)
	}
	return v, nil
}

// RestoreVersion saves a kept description as the ticket's current one through the normal update, so it becomes a new
// version on top and an open editing room follows it; needs tickets:write.
func (s *Service) RestoreVersion(ctx context.Context, id string, version int) (*Ticket, error) {
	current, err := s.load(ctx, id, permissions.TicketsWrite)
	if err != nil {
		return nil, fmt.Errorf("restore version %d of ticket %s: %w", version, id, err)
	}
	v, err := s.repo.GetDescriptionVersion(ctx, id, version)
	if err != nil {
		return nil, fmt.Errorf("restore version %d of ticket %s: %w", version, id, err)
	}
	return s.updateTicket(ctx, id, current.Title, v.Body, version)
}

// CommitSitting writes the version a live editing sitting ends with, crediting everyone who edited; a body the
// newest version holds writes nothing.
func (s *Service) CommitSitting(ctx context.Context, id string, editorIDs []string) error {
	if len(editorIDs) == 0 {
		return nil
	}
	if err := s.repo.AppendSittingVersion(ctx, id, editorIDs, s.now().UTC()); err != nil {
		return fmt.Errorf("commit sitting of ticket %s: %w", id, err)
	}
	return nil
}
