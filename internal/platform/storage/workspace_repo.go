package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/workspace"
)

var _ workspace.Repo = (*ProjectsRepo)(nil)

type ProjectsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *ProjectsRepo) Create(ctx context.Context, p *workspace.Project) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.CreateProject(ctx, sqlcgen.CreateProjectParams{
			ID: p.ID, Name: p.Name, Prefix: p.Prefix, Position: int64(p.Position),
			WorkspaceID: p.WorkspaceID, Icon: string(p.Icon), CreatedAt: p.CreatedAt.Unix(), UpdatedAt: p.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert project %s: %w", p.ID, classifyWriteErr(err))
		}
		return seedProjectDefaults(ctx, q, p.WorkspaceID, p.ID, p.CreatedAt.Unix())
	})
}

// seedProjectDefaults inserts defaults in the same transaction as the project, so a board is always usable.
func seedProjectDefaults(ctx context.Context, q *sqlcgen.Queries, workspaceID, projectID string, at int64) error {
	statuses := []struct {
		name string
		kind string
	}{
		{"Backlog", "backlog"},
		{"In progress", "progress"},
		{"In review", "review"},
		{"Testing", "testing"},
		{"Done", "done"},
	}
	for i, st := range statuses {
		if err := q.SeedProjectStatus(ctx, sqlcgen.SeedProjectStatusParams{
			ID: uuid.NewString(), ProjectID: projectID, Name: st.name, Position: int64(i), Kind: st.kind, CreatedAt: at, UpdatedAt: at,
		}); err != nil {
			return fmt.Errorf("seed status %q for project %s: %w", st.name, projectID, err)
		}
	}
	types := []string{"task", "bug", "feature"}
	for i, name := range types {
		if err := q.SeedProjectTicketType(ctx, sqlcgen.SeedProjectTicketTypeParams{
			ID: uuid.NewString(), ProjectID: projectID, Name: name, Position: int64(i), CreatedAt: at, UpdatedAt: at,
		}); err != nil {
			return fmt.Errorf("seed ticket type %q for project %s: %w", name, projectID, err)
		}
	}
	memoryID := uuid.NewString()
	if err := q.CreateMemory(ctx, sqlcgen.CreateMemoryParams{
		ID: memoryID, WorkspaceID: workspaceID, ProjectID: sql.NullString{String: projectID, Valid: true},
		Title: "Working in this project", WhenToUse: "Always included in this project's turns.",
		Body:           seedMemoryBody,
		AlwaysIncluded: 1, Version: 1, CreatedAt: at, UpdatedAt: at,
	}); err != nil {
		return fmt.Errorf("seed memory for project %s: %w", projectID, err)
	}
	// No author: this row seeds automatically, no user wrote it (mirrors 0139's seed for project-general).
	if err := q.InsertMemoryVersion(ctx, sqlcgen.InsertMemoryVersionParams{
		ID: uuid.NewString(), MemoryID: memoryID, Version: 1,
		Title: "Working in this project", WhenToUse: "Always included in this project's turns.",
		Body: seedMemoryBody, AlwaysIncluded: 1, CreatedAt: at,
	}); err != nil {
		return fmt.Errorf("seed memory version for project %s: %w", projectID, err)
	}
	return nil
}

// seedMemoryBody is legacy plain text, not structured Tiptap JSON — ADR 0026 lets a pre-conversion body pass
// through unchanged, and no user authored this row for richtext.Normalize to run against.
const seedMemoryBody = `This memory is Agent's shared context for this project — durable notes worth remembering across every chat turn.

Replace this starter text with anything the team wants Agent to always know: conventions, gotchas, or standing preferences.

Edit it any time; it stays always-included until someone turns that off.`

func (r *ProjectsRepo) Get(ctx context.Context, id string) (*workspace.Project, error) {
	row, err := r.q.GetProject(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get project %s: %w", id, notFoundIfNoRows(err))
	}
	return toProject(row), nil
}

// List returns a workspace's projects ordered by position; a project always nests under exactly one workspace.
func (r *ProjectsRepo) List(ctx context.Context, workspaceID string) ([]*workspace.Project, error) {
	rows, err := r.q.ListProjectsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return toProjects(rows), nil
}

func (r *ProjectsRepo) Update(ctx context.Context, p *workspace.Project) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateProject(ctx, sqlcgen.UpdateProjectParams{
			Name: p.Name, Prefix: p.Prefix, Icon: string(p.Icon), UpdatedAt: p.UpdatedAt.Unix(), ID: p.ID,
		})
		if err != nil {
			return fmt.Errorf("update project %s: %w", p.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update project %s: %w", p.ID, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *ProjectsRepo) Delete(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteProject(ctx, id)
		if err != nil {
			return fmt.Errorf("delete project %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete project %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

// Reorder assigns each project's position by its index in ids, in one transaction so a reorder is atomic.
func (r *ProjectsRepo) Reorder(ctx context.Context, ids []string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for i, id := range ids {
			if err := q.ReorderProjectPosition(ctx, sqlcgen.ReorderProjectPositionParams{
				Position: int64(i), UpdatedAt: time.Now().Unix(), ID: id,
			}); err != nil {
				return fmt.Errorf("reorder project %s: %w", id, err)
			}
		}
		return nil
	})
}

func (r *ProjectsRepo) CountTickets(ctx context.Context, projectID string) (int, error) {
	return r.count(ctx, projectID, "tickets")
}

func (r *ProjectsRepo) CountRepos(ctx context.Context, projectID string) (int, error) {
	return r.count(ctx, projectID, "project_repos")
}

// CountServices reports how many deploy stacks a project owns; deletion must be blocked while any exist.
// Counts the "stacks" table; the "services" table holds observed containers, which have no project_id of their own.
func (r *ProjectsRepo) CountServices(ctx context.Context, projectID string) (int, error) {
	return r.count(ctx, projectID, "stacks")
}

func (r *ProjectsRepo) count(ctx context.Context, projectID, table string) (int, error) {
	var n int
	// hand-written: sqlc cannot express a table name chosen at runtime
	if err := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE project_id = ?`, table), projectID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count %s for project %s: %w", table, projectID, err)
	}
	return n, nil
}

func (r *ProjectsRepo) AddRepo(ctx context.Context, projectID string, ref workspace.RepoRef) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).AddProjectRepo(ctx, sqlcgen.AddProjectRepoParams{
			ProjectID: projectID, Owner: ref.Owner, Name: ref.Name, FullName: ref.FullName,
			ConnectorID: ref.ConnectorID, AddedAt: time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("add repo %s/%s to project %s: %w", ref.Owner, ref.Name, projectID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *ProjectsRepo) RemoveRepo(ctx context.Context, owner, name string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).RemoveProjectRepo(ctx, sqlcgen.RemoveProjectRepoParams{Owner: owner, Name: name})
		if err != nil {
			return fmt.Errorf("remove repo %s/%s: %w", owner, name, err)
		}
		if n == 0 {
			return fmt.Errorf("remove repo %s/%s: %w", owner, name, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *ProjectsRepo) ListRepos(ctx context.Context, projectID string) ([]workspace.RepoRef, error) {
	rows, err := r.q.ListProjectRepos(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list repos for project %s: %w", projectID, err)
	}
	var out []workspace.RepoRef
	for _, row := range rows {
		out = append(out, workspace.RepoRef{Owner: row.Owner, Name: row.Name, FullName: row.FullName, ConnectorID: row.ConnectorID})
	}
	return out, nil
}

// GetRepoByFullName reverse-looks-up the RepoRef linked under owner/name, across all projects, for the git router.
func (r *ProjectsRepo) GetRepoByFullName(ctx context.Context, owner, name string) (workspace.RepoRef, error) {
	row, err := r.q.GetProjectRepoByOwnerAndName(ctx, sqlcgen.GetProjectRepoByOwnerAndNameParams{Owner: owner, Name: name})
	if err != nil {
		return workspace.RepoRef{}, fmt.Errorf("get repo %s/%s: %w", owner, name, notFoundIfNoRows(err))
	}
	return workspace.RepoRef{Owner: row.Owner, Name: row.Name, FullName: row.FullName, ConnectorID: row.ConnectorID}, nil
}

// MoveTicket updates a ticket's project in place; the FK guarantees only an existing project can be referenced.
func (r *ProjectsRepo) MoveTicket(ctx context.Context, ticketID, projectID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).MoveTicketProject(ctx, sqlcgen.MoveTicketProjectParams{
			ProjectID: nullString(projectID), ID: ticketID,
		})
		if err != nil {
			return fmt.Errorf("move ticket %s to project %s: %w", ticketID, projectID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("move ticket %s to project %s: %w", ticketID, projectID, apperrs.ErrNotFound)
		}
		return nil
	})
}

func toProject(row sqlcgen.Project) *workspace.Project {
	return &workspace.Project{
		ID:          row.ID,
		Name:        row.Name,
		Prefix:      row.Prefix,
		Position:    int(row.Position),
		WorkspaceID: row.WorkspaceID,
		Icon:        workspace.ProjectIcon(row.Icon),
		CreatedAt:   time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:   time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

func toProjects(rows []sqlcgen.Project) []*workspace.Project {
	var out []*workspace.Project
	for _, row := range rows {
		out = append(out, toProject(row))
	}
	return out
}
