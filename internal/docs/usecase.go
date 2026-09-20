package docs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// AccessChecker is the slice of access docs consumes, defined consumer-side so docs never imports it (ADR 0017).
type AccessChecker interface {
	Can(ctx context.Context, userID, docID string, action permissions.Action) (bool, error)
	GrantCreator(ctx context.Context, docID, creatorID string) error
	// DeleteByDoc removes every permission overwrite on docID; called explicitly, permission_overwrites has no FK to docs.
	DeleteByDoc(ctx context.Context, docID string) error
}

// Service is the docs use-case layer (ADR 0019); permission checks run here so every adapter inherits them.
type Service struct {
	repo   Repo
	access AccessChecker
	now    func() time.Time
}

// NewService wires the docs use-cases over the given repo and access checker.
func NewService(repo Repo, access AccessChecker) *Service {
	return &Service{repo: repo, access: access, now: time.Now}
}

// Create validates and persists a new doc (v1), enqueuing doc.created and granting the creator full permissions.
func (s *Service) Create(ctx context.Context, projectID, title, body string) (*Doc, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required — create a project before creating docs", apperrs.ErrInvalid)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", apperrs.ErrInvalid)
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	body, err := richtext.Normalize(body)
	if err != nil {
		return nil, fmt.Errorf("%w: body is not valid document content", apperrs.ErrInvalid)
	}
	now := s.now().UTC()
	d := &Doc{
		ID:        ids.New(),
		ProjectID: projectID,
		Title:     title,
		Body:      body,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, d, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicCreated, Payload: CreatedEvent{Doc: *d}}); err != nil {
		return nil, fmt.Errorf("create doc: %w", err)
	}
	if s.access != nil {
		if err := s.access.GrantCreator(ctx, d.ID, actor.ID); err != nil {
			return nil, fmt.Errorf("grant creator on doc %s: %w", d.ID, err)
		}
	}
	return d, nil
}

// Get returns a doc by id; a user without the read bit gets ErrForbidden, though list surfaces may disclose its title.
func (s *Service) Get(ctx context.Context, id string) (*Doc, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get doc %s: %w", id, err)
	}
	if err := s.require(ctx, d.ID, permissions.DocsRead); err != nil {
		return nil, err
	}
	return d, nil
}

// List returns all docs, oldest first; docs the actor can't open come back as entries with can_open=false, no body.
func (s *Service) List(ctx context.Context) ([]*DocListItem, error) {
	ds, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list docs: %w", err)
	}
	return s.toListItems(ctx, ds), nil
}

// ListByProject returns the docs in a project, oldest first; the list stays flat, no sub-grouping (ADR 0025).
func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*DocListItem, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	ds, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list docs for project %s: %w", projectID, err)
	}
	return s.toListItems(ctx, ds), nil
}

func (s *Service) toListItems(ctx context.Context, ds []*Doc) []*DocListItem {
	out := make([]*DocListItem, 0, len(ds))
	for _, d := range ds {
		item := &DocListItem{
			ID:        d.ID,
			ProjectID: d.ProjectID,
			Title:     d.Title,
			Version:   d.Version,
			Archived:  d.Archived,
			UpdatedAt: d.UpdatedAt,
		}
		item.CanOpen = s.can(ctx, d.ID, permissions.DocsRead)
		out = append(out, item)
	}
	return out
}

// Update replaces the title/body, bumps the version, appends a row, and enqueues doc.updated; requires the edit bit.
func (s *Service) Update(ctx context.Context, id, title, body string) (*Doc, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", apperrs.ErrInvalid)
	}
	body, err := richtext.Normalize(body)
	if err != nil {
		return nil, fmt.Errorf("%w: body is not valid document content", apperrs.ErrInvalid)
	}
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update doc %s: %w", id, err)
	}
	if err := s.require(ctx, current.ID, permissions.DocsWrite); err != nil {
		return nil, err
	}
	current.Title = title
	current.Body = body
	current.Version++
	current.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, current, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicUpdated, Payload: UpdatedEvent{Doc: *current}}); err != nil {
		return nil, fmt.Errorf("update doc %s: %w", id, err)
	}
	return current, nil
}

// Archive marks a doc archived (hidden from search), requiring the archive bit; Restore is the inverse.
func (s *Service) Archive(ctx context.Context, id string) (*Doc, error) {
	return s.setArchived(ctx, id, true)
}

func (s *Service) Restore(ctx context.Context, id string) (*Doc, error) {
	return s.setArchived(ctx, id, false)
}

func (s *Service) setArchived(ctx context.Context, id string, archived bool) (*Doc, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get doc %s: %w", id, err)
	}
	if err := s.require(ctx, current.ID, permissions.DocsWrite); err != nil {
		return nil, err
	}
	current.Archived = archived
	current.UpdatedAt = s.now().UTC()
	if err := s.repo.SetArchived(ctx, current.ID, archived, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicUpdated, Payload: UpdatedEvent{Doc: *current}}); err != nil {
		return nil, fmt.Errorf("archive doc %s: %w", id, err)
	}
	return current, nil
}

// Delete removes a doc, requiring the delete bit, and publishes doc.deleted; referencing tickets reject via FK.
func (s *Service) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete doc %s: %w", id, err)
	}
	if err := s.require(ctx, d.ID, permissions.DocsDelete); err != nil {
		return err
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicDeleted, Payload: DeletedEvent{ID: d.ID, Title: d.Title}}
	if err := s.repo.Delete(ctx, id, evt); err != nil {
		return fmt.Errorf("delete doc %s: %w", id, err)
	}
	if s.access != nil {
		if err := s.access.DeleteByDoc(ctx, id); err != nil {
			return fmt.Errorf("delete permissions for doc %s: %w", id, err)
		}
	}
	return nil
}

// ExportMarkdown renders a doc's body as markdown for LLM/import/export/integrations; requires the read bit.
func (s *Service) ExportMarkdown(ctx context.Context, id string) (string, error) {
	d, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}
	md, err := richtext.ToMarkdown(d.Body)
	if err != nil {
		return "", fmt.Errorf("export doc %s: %w", id, err)
	}
	return md, nil
}

// ImportMarkdown creates a doc from a markdown body, converted to the canonical structured representation.
func (s *Service) ImportMarkdown(ctx context.Context, projectID, title, body string) (*Doc, error) {
	return s.Create(ctx, projectID, title, body)
}

// Search runs an FTS5 query over doc titles and bodies; archived and inaccessible docs are excluded entirely, never disclosed (ADR 0028).
func (s *Service) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("%w: query is required", apperrs.ErrInvalid)
	}
	if limit < 1 {
		limit = 20
	}
	results, err := s.repo.Search(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search docs: %w", err)
	}
	out := results[:0]
	for _, r := range results {
		if s.can(ctx, r.ID, permissions.DocsRead) {
			out = append(out, r)
		}
	}
	return out, nil
}

// ListVersions returns the version history for a doc, newest first; requires the read bit.
func (s *Service) ListVersions(ctx context.Context, docID string) ([]*DocVersion, error) {
	if strings.TrimSpace(docID) == "" {
		return nil, fmt.Errorf("%w: doc id is required", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, docID, permissions.DocsRead); err != nil {
		return nil, err
	}
	vs, err := s.repo.ListVersions(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("list doc versions %s: %w", docID, err)
	}
	return vs, nil
}

// GetVersion returns a specific historical version of a doc; requires the read bit.
func (s *Service) GetVersion(ctx context.Context, docID string, version int) (*DocVersion, error) {
	if strings.TrimSpace(docID) == "" {
		return nil, fmt.Errorf("%w: doc id is required", apperrs.ErrInvalid)
	}
	if version < 1 {
		return nil, fmt.Errorf("%w: version must be positive", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, docID, permissions.DocsRead); err != nil {
		return nil, err
	}
	v, err := s.repo.GetVersion(ctx, docID, version)
	if err != nil {
		return nil, fmt.Errorf("get doc %s version %d: %w", docID, version, err)
	}
	return v, nil
}

// CommitCollab persists a converged collaboration state to the canonical doc without a heavyweight version row.
func (s *Service) CommitCollab(ctx context.Context, id, title, body string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	body, err := richtext.Normalize(body)
	if err != nil {
		return fmt.Errorf("%w: body is not valid document content", apperrs.ErrInvalid)
	}
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("commit doc %s: %w", id, err)
	}
	// An empty title means "unchanged": commits carry the title only from the
	// client that actually renamed, so a peer's body commit never resets it.
	if title = strings.TrimSpace(title); title != "" {
		current.Title = title
	}
	current.Body = body
	current.Version++
	current.UpdatedAt = s.now().UTC()
	if err := s.repo.CommitBody(ctx, current, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicUpdated, Payload: UpdatedEvent{Doc: *current}}); err != nil {
		return fmt.Errorf("commit doc %s: %w", id, err)
	}
	return nil
}

// CreateNamedVersion snapshots the doc's current state as a named milestone version; requires the edit bit.
func (s *Service) CreateNamedVersion(ctx context.Context, docID, name string) (*DocVersion, error) {
	if strings.TrimSpace(docID) == "" {
		return nil, fmt.Errorf("%w: doc id is required", apperrs.ErrInvalid)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: version name is required", apperrs.ErrInvalid)
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	current, err := s.repo.GetByID(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("version doc %s: %w", docID, err)
	}
	if err := s.require(ctx, current.ID, permissions.DocsWrite); err != nil {
		return nil, err
	}
	v := &DocVersion{
		DocID:     current.ID,
		Version:   current.Version + 1,
		Title:     current.Title,
		Body:      current.Body,
		Name:      name,
		AuthorID:  actor.ID,
		CreatedAt: s.now().UTC(),
	}
	if err := s.repo.CreateNamedVersion(ctx, v); err != nil {
		return nil, fmt.Errorf("version doc %s: %w", docID, err)
	}
	return v, nil
}

func (s *Service) require(ctx context.Context, docID string, action permissions.Action) error {
	ok := s.can(ctx, docID, action)
	if !ok {
		return fmt.Errorf("%w: no %s permission on doc %s", apperrs.ErrForbidden, action, docID)
	}
	return nil
}

func (s *Service) can(ctx context.Context, docID string, action permissions.Action) bool {
	// Fail closed: a service without a wired access checker (misconfiguration) must deny, never silently allow.
	if s.access == nil {
		return false
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return false
	}
	allowed, err := s.access.Can(ctx, actor.ID, docID, action)
	if err != nil {
		return false
	}
	return allowed
}
