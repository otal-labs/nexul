package docs

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// ListFolders returns a project's folders, the default first; a viewer without docs:write sees only the folders
// holding a doc they can open.
func (s *Service) ListFolders(ctx context.Context, projectID string) ([]*Folder, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.requireProject(ctx, projectID, permissions.Member); err != nil {
		return nil, err
	}
	folders, err := s.repo.ListFolders(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list folders for project %s: %w", projectID, err)
	}
	if s.requireProject(ctx, projectID, permissions.DocsWrite) == nil {
		return folders, nil
	}
	ds, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list docs for project %s: %w", projectID, err)
	}
	holding := map[string]bool{}
	for _, d := range ds {
		if !holding[d.FolderID] && s.can(ctx, d.ID, permissions.DocsRead) {
			holding[d.FolderID] = true
		}
	}
	return slices.DeleteFunc(folders, func(f *Folder) bool { return !holding[f.ID] }), nil
}

// CreateFolder adds a folder to a project (docs:write); a name another folder there has, ignoring case, conflicts.
func (s *Service) CreateFolder(ctx context.Context, projectID, name string) (*Folder, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	name, err := folderName(name)
	if err != nil {
		return nil, err
	}
	if err := s.requireProject(ctx, projectID, permissions.DocsWrite); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	f := &Folder{ID: ids.New(), ProjectID: projectID, Name: name, CreatedAt: now, UpdatedAt: now}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicFolderCreated, Payload: FolderEvent{Folder: *f, ActorID: actorID(ctx)}}
	if err := s.repo.CreateFolder(ctx, f, evt); err != nil {
		return nil, folderWriteErr(name, err)
	}
	return f, nil
}

// RenameFolder renames a folder (docs:write), the default one included.
func (s *Service) RenameFolder(ctx context.Context, id, name string) (*Folder, error) {
	name, err := folderName(name)
	if err != nil {
		return nil, err
	}
	f, err := s.writableFolder(ctx, id)
	if err != nil {
		return nil, err
	}
	previous := f.Name
	renamed := *f
	renamed.Name = name
	renamed.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicFolderUpdated, Payload: FolderUpdatedEvent{Folder: renamed, PreviousName: previous, ActorID: actorID(ctx)}}
	if err := s.repo.RenameFolder(ctx, &renamed, evt); err != nil {
		return nil, folderWriteErr(name, err)
	}
	return &renamed, nil
}

// DeleteFolder deletes a folder (docs:write) and moves its docs to the project's default folder, which is never deleted.
func (s *Service) DeleteFolder(ctx context.Context, id string) error {
	f, err := s.writableFolder(ctx, id)
	if err != nil {
		return err
	}
	if f.IsDefault {
		return fmt.Errorf("%w: %s is the project's default folder, which can be renamed but not deleted", apperrs.ErrInvalid, f.Name)
	}
	main, err := s.repo.DefaultFolder(ctx, f.ProjectID)
	if err != nil {
		return fmt.Errorf("default folder of project %s: %w", f.ProjectID, err)
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicFolderDeleted, Payload: FolderDeletedEvent{Folder: *f, MovedToFolderID: main.ID, ActorID: actorID(ctx)}}
	if err := s.repo.DeleteFolder(ctx, f.ID, main.ID, evt); err != nil {
		return fmt.Errorf("delete folder %s: %w", id, err)
	}
	return nil
}

// MoveToFolder moves a doc into another folder of its project, needing docs:write on the doc; a locked doc moves too.
func (s *Service) MoveToFolder(ctx context.Context, docID, folderID string) (*Doc, error) {
	if strings.TrimSpace(docID) == "" || strings.TrimSpace(folderID) == "" {
		return nil, fmt.Errorf("%w: doc id and folder id are required", apperrs.ErrInvalid)
	}
	d, err := s.repo.GetByID(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("move doc %s: %w", docID, err)
	}
	if err := s.require(ctx, d.ID, permissions.DocsWrite); err != nil {
		if memberErr := s.requireProject(ctx, d.ProjectID, permissions.Member); memberErr != nil {
			return nil, memberErr
		}
		return nil, err
	}
	to, err := s.folderIn(ctx, d.ProjectID, folderID)
	if err != nil {
		return nil, err
	}
	if d.FolderID == to {
		return d, nil
	}
	moved := *d
	moved.FolderID = to
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMoved, Payload: MovedEvent{Doc: moved, FromFolderID: d.FolderID, ActorID: actorID(ctx)}}
	if err := s.repo.SetDocFolder(ctx, d.ID, to, evt); err != nil {
		return nil, fmt.Errorf("move doc %s: %w", docID, err)
	}
	return &moved, nil
}

// folderIn resolves where a doc in projectID goes: folderID when it is one of the project's, its default when empty.
func (s *Service) folderIn(ctx context.Context, projectID, folderID string) (string, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		main, err := s.repo.DefaultFolder(ctx, projectID)
		if err != nil {
			return "", fmt.Errorf("default folder of project %s: %w", projectID, err)
		}
		return main.ID, nil
	}
	f, err := s.repo.GetFolder(ctx, folderID)
	if errors.Is(err, apperrs.ErrNotFound) || (err == nil && f.ProjectID != projectID) {
		return "", fmt.Errorf("%w: folder %s is not a folder of this doc's project", apperrs.ErrInvalid, folderID)
	}
	if err != nil {
		return "", fmt.Errorf("get folder %s: %w", folderID, err)
	}
	return f.ID, nil
}

func (s *Service) writableFolder(ctx context.Context, id string) (*Folder, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: folder id is required", apperrs.ErrInvalid)
	}
	f, err := s.repo.GetFolder(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get folder %s: %w", id, err)
	}
	if err := s.requireProject(ctx, f.ProjectID, permissions.DocsWrite); err != nil {
		return nil, err
	}
	return f, nil
}

func folderName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: folder name is required", apperrs.ErrInvalid)
	}
	return name, nil
}

func folderWriteErr(name string, err error) error {
	if errors.Is(err, apperrs.ErrConflict) {
		return fmt.Errorf("%w: this project already has a folder named %s", apperrs.ErrConflict, name)
	}
	return fmt.Errorf("save folder %s: %w", name, err)
}
