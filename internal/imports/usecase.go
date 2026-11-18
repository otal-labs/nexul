package imports

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// Owners answers who may run an import: the workspace's Owner alone.
type Owners interface {
	IsOwner(ctx context.Context, userID, workspaceID string) (bool, error)
}

// Projects resolves a project's workspace.
type Projects interface {
	WorkspaceOf(ctx context.Context, projectID string) (string, error)
}

// Column is one of a project's status columns.
type Column struct {
	ID   string
	Name string
	Kind StatusKind
}

// Board is the project's columns, which statuses map onto (ADR 0017: imports never imports workspace).
type Board interface {
	Columns(ctx context.Context, projectID string) ([]Column, error)
	AddColumn(ctx context.Context, projectID, name string, kind StatusKind) (Column, error)
}

// Member is someone who can open the project, and the logins and emails their linked sign-ins carry, lowercased.
type Member struct {
	UserID       string
	Login        string
	GitHubLogins []string
	Emails       []string
}

// People lists who can open a project, whom a source's people are matched against.
type People interface {
	Members(ctx context.Context, projectID string) ([]Member, error)
}

// NewTicket is one ticket the import files; ID is derived, so filing it again is ErrConflict.
type NewTicket struct {
	ID, ProjectID, Title, Body, Status, Developer, Source string
	Labels                                                []string
}

// NewComment is one comment copied into a ticket's thread; ID is derived, so posting it again is ErrConflict.
type NewComment struct {
	ID, AuthorKey, Name, AvatarURL, Body, Via string
	At                                        time.Time
}

// Desk writes the records, as the person who started the import.
type Desk interface {
	FileTicket(ctx context.Context, t NewTicket) error
	SetBody(ctx context.Context, ticketID, title, body string) error
	Comment(ctx context.Context, ticketID string, c NewComment) error
	// Attach stores a file on the ticket and returns the attachment's id.
	Attach(ctx context.Context, ticketID, name string, data []byte) (string, error)
}

// Service runs imports; HTTP and MCP both go through it (ADR 0019).
type Service struct {
	repo    Repo
	owners  Owners
	project Projects
	board   Board
	people  People
	desk    Desk
	files   Files
	sources map[string]Source
	log     *slog.Logger
	now     func() time.Time
}

// Deps are the Service's collaborators.
type Deps struct {
	Repo     Repo
	Owners   Owners
	Projects Projects
	Board    Board
	People   People
	Desk     Desk
	// Files downloads source files; nil uses a client that carries no credential and keeps to each source's hosts.
	Files Files
	// Sources is every tool an import can read, by the name import_run takes.
	Sources map[string]Source
	Log     *slog.Logger
}

// NewService wires the import service.
func NewService(d Deps) *Service {
	s := &Service{
		repo: d.Repo, owners: d.Owners, project: d.Projects, board: d.Board, people: d.People, desk: d.Desk,
		files: d.Files, sources: d.Sources, log: d.Log, now: time.Now,
	}
	if s.files == nil {
		s.files = NewFetcher()
	}
	if s.log == nil {
		s.log = slog.Default()
	}
	return s
}

// SourceNames lists the sources an import can read, sorted.
func (s *Service) SourceNames() []string { return slices.Sorted(maps.Keys(s.sources)) }

// Start begins an import by scanning; the loop counts what it would bring, then it waits to be confirmed.
func (s *Service) Start(ctx context.Context, in Start) (*Import, error) {
	userID, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	in.Source, in.ProjectID, in.From = strings.TrimSpace(in.Source), strings.TrimSpace(in.ProjectID), strings.TrimSpace(in.From)
	if _, ok := s.sources[in.Source]; !ok {
		return nil, fmt.Errorf("%w: source must be one of %s", apperrs.ErrInvalid, strings.Join(s.SourceNames(), ", "))
	}
	if in.ProjectID == "" {
		return nil, fmt.Errorf("%w: project_id is required; project_list lists them", apperrs.ErrInvalid)
	}
	workspaceID, err := s.project.WorkspaceOf(ctx, in.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("start import into project %s: %w", in.ProjectID, err)
	}
	if err := s.requireOwner(ctx, userID, workspaceID); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	imp := &Import{
		ID: ids.New(), WorkspaceID: workspaceID, ProjectID: in.ProjectID, Source: in.Source, From: in.From, State: StateScanning,
		CreatedBy: userID, Credential: strings.TrimSpace(in.Credential), CreatedAt: now, UpdatedAt: now,
	}
	err = s.repo.Create(ctx, imp, progressEvent(imp))
	if errors.Is(err, apperrs.ErrConflict) {
		return nil, fmt.Errorf("%w: this workspace already has an import under way; finish or cancel it first", apperrs.ErrConflict)
	}
	if err != nil {
		return nil, fmt.Errorf("start import: %w", err)
	}
	return imp, nil
}

// Confirm starts a scanned import's writes.
func (s *Service) Confirm(ctx context.Context, id string) (*Import, error) {
	imp, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if imp.State != StateReady {
		return nil, fmt.Errorf("%w: import %s is %s; only a scanned, ready import can be confirmed", apperrs.ErrInvalid, id, imp.State)
	}
	return s.move(ctx, imp, StateReady, StateImporting, "")
}

// Cancel stops an import that has not ended; what it already wrote stays, and a re-run skips it.
func (s *Service) Cancel(ctx context.Context, id string) (*Import, error) {
	imp, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if imp.State.Final() {
		return nil, fmt.Errorf("%w: import %s has already ended (%s)", apperrs.ErrInvalid, id, imp.State)
	}
	return s.move(ctx, imp, imp.State, StateCancelled, "")
}

// move changes imp's state from from to to, leaving its counts and report as stored, or says the import moved on
// meanwhile.
func (s *Service) move(ctx context.Context, imp *Import, from, to State, reason string) (*Import, error) {
	moved, ok, err := s.repo.Move(ctx, imp.ID, from, to, reason, s.now().UTC(), progressEvent)
	if err != nil {
		return nil, fmt.Errorf("move import %s to %s: %w", imp.ID, to, err)
	}
	if !ok {
		return nil, fmt.Errorf("%w: import %s changed meanwhile; read it again", apperrs.ErrConflict, imp.ID)
	}
	return moved, nil
}

// Get returns one import to an Owner of its workspace.
func (s *Service) Get(ctx context.Context, id string) (*Import, error) {
	userID, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	imp, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get import %s: %w", id, err)
	}
	if err := s.requireOwner(ctx, userID, imp.WorkspaceID); err != nil {
		if errors.Is(err, apperrs.ErrForbidden) {
			return nil, fmt.Errorf("%w: import %s", apperrs.ErrNotFound, id)
		}
		return nil, err
	}
	return imp, nil
}

// ListImportsLimit is how many of a workspace's imports List returns, newest first.
const ListImportsLimit = 20

// List returns a workspace's latest imports to its Owner.
func (s *Service) List(ctx context.Context, workspaceID string) ([]*Import, error) {
	userID, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.requireOwner(ctx, userID, strings.TrimSpace(workspaceID)); err != nil {
		return nil, err
	}
	return s.repo.ListByWorkspace(ctx, strings.TrimSpace(workspaceID), ListImportsLimit)
}

func (s *Service) requireOwner(ctx context.Context, userID, workspaceID string) error {
	owner, err := s.owners.IsOwner(ctx, userID, workspaceID)
	if err != nil {
		return fmt.Errorf("check owner of workspace %s: %w", workspaceID, err)
	}
	if !owner {
		return fmt.Errorf("%w: only the workspace's Owner can run an import", apperrs.ErrForbidden)
	}
	return nil
}

func callerID(ctx context.Context) (string, error) {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" || actor.Automation != nil {
		return "", fmt.Errorf("%w: an import is run by a signed-in person", apperrs.ErrUnauthorized)
	}
	return actor.ID, nil
}
