package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/imports"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ imports.Repo = (*ImportsRepo)(nil)

// ImportsRepo backs imports.Repo: the imports, import_items and import_source_ids tables, the credential sealed with
// the instance key.
type ImportsRepo struct {
	db     *sql.DB
	w      *Serializer
	q      *sqlcgen.Queries
	encKey []byte
}

func (r *ImportsRepo) Create(ctx context.Context, imp *imports.Import, evts ...eventbus.OutboxEvent) error {
	sealed, err := r.seal(imp.Credential)
	if err != nil {
		return err
	}
	report, err := json.Marshal(imp.Report)
	if err != nil {
		return fmt.Errorf("marshal import report: %w", err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateImport(ctx, sqlcgen.CreateImportParams{
			ID: imp.ID, WorkspaceID: imp.WorkspaceID, ProjectID: imp.ProjectID, Source: imp.Source, SourceFrom: imp.From,
			State: string(imp.State), CreatedBy: imp.CreatedBy, CredentialEncrypted: sealed, Total: int64(imp.Total),
			Done: int64(imp.Done), Failed: int64(imp.Failed), ReportJson: string(report),
			CreatedAt: imp.CreatedAt.Unix(), UpdatedAt: imp.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert import %s: %w", imp.ID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *ImportsRepo) Get(ctx context.Context, id string) (*imports.Import, error) {
	row, err := r.q.GetImport(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get import %s: %w", id, notFoundIfNoRows(err))
	}
	return r.toImport(row)
}

func (r *ImportsRepo) ListByWorkspace(ctx context.Context, workspaceID string, limit int) ([]*imports.Import, error) {
	rows, err := r.q.ListImportsByWorkspace(ctx, sqlcgen.ListImportsByWorkspaceParams{WorkspaceID: workspaceID, Limit: int64(limit)})
	if err != nil {
		return nil, fmt.Errorf("list imports of workspace %s: %w", workspaceID, err)
	}
	return r.toImports(rows)
}

func (r *ImportsRepo) Open(ctx context.Context) ([]*imports.Import, error) {
	rows, err := r.q.ListOpenImports(ctx)
	if err != nil {
		return nil, fmt.Errorf("list open imports: %w", err)
	}
	return r.toImports(rows)
}

func (r *ImportsRepo) Update(ctx context.Context, imp *imports.Import, from imports.State, evts ...eventbus.OutboxEvent) (bool, error) {
	report, err := json.Marshal(imp.Report)
	if err != nil {
		return false, fmt.Errorf("marshal import report: %w", err)
	}
	var changed bool
	err = r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if imp.State.Final() {
			if err := q.ClearImportCredential(ctx, sqlcgen.ClearImportCredentialParams{ID: imp.ID, State: string(from)}); err != nil {
				return fmt.Errorf("clear import %s credential: %w", imp.ID, err)
			}
		}
		n, err := q.UpdateImport(ctx, sqlcgen.UpdateImportParams{
			State: string(imp.State), SourceFrom: imp.From, Total: int64(imp.Total), Done: int64(imp.Done),
			Failed: int64(imp.Failed), ReportJson: string(report), UpdatedAt: imp.UpdatedAt.Unix(), ID: imp.ID, FromState: string(from),
		})
		if err != nil {
			return fmt.Errorf("update import %s: %w", imp.ID, classifyWriteErr(err))
		}
		if changed = n == 1; !changed {
			return nil
		}
		return insertOutboxRows(ctx, tx, evts)
	})
	return changed, err
}

func (r *ImportsRepo) Move(ctx context.Context, id string, from, to imports.State, reason string, at time.Time, event func(*imports.Import) eventbus.OutboxEvent) (*imports.Import, bool, error) {
	var moved *imports.Import
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if to.Final() {
			if err := q.ClearImportCredential(ctx, sqlcgen.ClearImportCredentialParams{ID: id, State: string(from)}); err != nil {
				return fmt.Errorf("clear import %s credential: %w", id, err)
			}
		}
		n, err := q.MoveImport(ctx, sqlcgen.MoveImportParams{State: string(to), UpdatedAt: at.Unix(), ID: id, FromState: string(from)})
		if err != nil {
			return fmt.Errorf("move import %s to %s: %w", id, to, classifyWriteErr(err))
		}
		if n == 0 {
			return nil
		}
		if reason != "" {
			if err := q.SetImportError(ctx, sqlcgen.SetImportErrorParams{JsonSet: reason, ID: id}); err != nil {
				return fmt.Errorf("record why import %s stopped: %w", id, err)
			}
		}
		row, err := q.GetImport(ctx, id)
		if err != nil {
			return fmt.Errorf("read moved import %s: %w", id, err)
		}
		if moved, err = r.toImport(row); err != nil {
			return err
		}
		return insertOutboxRows(ctx, tx, []eventbus.OutboxEvent{event(moved)})
	})
	if err != nil {
		return nil, false, err
	}
	return moved, moved != nil, nil
}

func (r *ImportsRepo) Item(ctx context.Context, importID, kind, sourceID string) (imports.Item, bool, error) {
	row, err := r.q.GetImportItem(ctx, sqlcgen.GetImportItemParams{ImportID: importID, Kind: kind, SourceID: sourceID})
	if errors.Is(err, sql.ErrNoRows) {
		return imports.Item{}, false, nil
	}
	if err != nil {
		return imports.Item{}, false, fmt.Errorf("get import item %s/%s: %w", kind, sourceID, err)
	}
	return imports.Item{
		ImportID: row.ImportID, Kind: row.Kind, SourceID: row.SourceID, NexulID: row.NexulID,
		State: imports.ItemState(row.State), Error: row.Error,
	}, true, nil
}

func (r *ImportsRepo) Imported(ctx context.Context, workspaceID, source, kind, sourceID string) (string, bool, error) {
	id, err := r.q.GetImportSourceID(ctx, sqlcgen.GetImportSourceIDParams{WorkspaceID: workspaceID, Source: source, Kind: kind, SourceID: sourceID})
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get imported %s %s: %w", kind, sourceID, err)
	}
	return id, true, nil
}

func (r *ImportsRepo) Record(ctx context.Context, workspaceID, source string, item imports.Item) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.PutImportItem(ctx, sqlcgen.PutImportItemParams{
			ImportID: item.ImportID, Kind: item.Kind, SourceID: item.SourceID, NexulID: item.NexulID, State: string(item.State), Error: item.Error,
		})
		if err != nil {
			return fmt.Errorf("record import item %s/%s: %w", item.Kind, item.SourceID, classifyWriteErr(err))
		}
		if item.State == imports.ItemFailed {
			return nil
		}
		err = q.PutImportSourceID(ctx, sqlcgen.PutImportSourceIDParams{
			WorkspaceID: workspaceID, Source: source, Kind: item.Kind, SourceID: item.SourceID, NexulID: item.NexulID,
		})
		if err != nil {
			return fmt.Errorf("record imported %s %s: %w", item.Kind, item.SourceID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *ImportsRepo) Unfilled(ctx context.Context, importID, kind string) ([]imports.Item, error) {
	rows, err := r.q.ListImportItemsInState(ctx, sqlcgen.ListImportItemsInStateParams{ImportID: importID, Kind: kind, State: string(imports.ItemCreated)})
	if err != nil {
		return nil, fmt.Errorf("list unfilled %s items: %w", kind, err)
	}
	out := make([]imports.Item, 0, len(rows))
	for _, row := range rows {
		out = append(out, imports.Item{
			ImportID: row.ImportID, Kind: row.Kind, SourceID: row.SourceID, NexulID: row.NexulID,
			State: imports.ItemState(row.State), Error: row.Error,
		})
	}
	return out, nil
}

func (r *ImportsRepo) Forget(ctx context.Context, workspaceID, source string, item imports.Item) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.PutImportItem(ctx, sqlcgen.PutImportItemParams{
			ImportID: item.ImportID, Kind: item.Kind, SourceID: item.SourceID, NexulID: item.NexulID, State: string(item.State), Error: item.Error,
		})
		if err != nil {
			return fmt.Errorf("record import item %s/%s: %w", item.Kind, item.SourceID, classifyWriteErr(err))
		}
		err = q.DeleteImportSourceID(ctx, sqlcgen.DeleteImportSourceIDParams{WorkspaceID: workspaceID, Source: source, Kind: item.Kind, SourceID: item.SourceID})
		if err != nil {
			return fmt.Errorf("forget imported %s %s: %w", item.Kind, item.SourceID, err)
		}
		return nil
	})
}

// People is every member of the workspace with the GitHub logins and emails their linked sign-ins carry, lowercased;
// an email is the login of a Google, Discord or single sign-on identity.
func (r *ImportsRepo) People(ctx context.Context, workspaceID string) ([]imports.Member, error) {
	rows, err := r.q.ListImportPeople(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list people of workspace %s: %w", workspaceID, err)
	}
	var out []imports.Member
	for _, row := range rows {
		if len(out) == 0 || out[len(out)-1].UserID != row.UserID {
			out = append(out, imports.Member{UserID: row.UserID, Login: row.UserLogin})
		}
		m := &out[len(out)-1]
		login := strings.ToLower(strings.TrimSpace(row.IdentityLogin))
		if row.Provider == "github" {
			m.GitHubLogins = append(m.GitHubLogins, login)
			continue
		}
		if strings.Contains(login, "@") && !slices.Contains(m.Emails, login) {
			m.Emails = append(m.Emails, login)
		}
	}
	return out, nil
}

func (r *ImportsRepo) seal(credential string) (sql.NullString, error) {
	if credential == "" {
		return sql.NullString{}, nil
	}
	sealed, err := crypto.Encrypt(r.encKey, []byte(credential))
	if err != nil {
		return sql.NullString{}, fmt.Errorf("seal import credential: %w", err)
	}
	return sql.NullString{String: sealed, Valid: true}, nil
}

func (r *ImportsRepo) toImports(rows []sqlcgen.Import) ([]*imports.Import, error) {
	out := make([]*imports.Import, 0, len(rows))
	for _, row := range rows {
		imp, err := r.toImport(row)
		if err != nil {
			return nil, err
		}
		out = append(out, imp)
	}
	return out, nil
}

func (r *ImportsRepo) toImport(row sqlcgen.Import) (*imports.Import, error) {
	imp := &imports.Import{
		ID: row.ID, WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, Source: row.Source, From: row.SourceFrom,
		State: imports.State(row.State), CreatedBy: row.CreatedBy, Total: int(row.Total), Done: int(row.Done), Failed: int(row.Failed),
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
	if err := json.Unmarshal([]byte(row.ReportJson), &imp.Report); err != nil {
		return nil, fmt.Errorf("read import %s report: %w", row.ID, err)
	}
	if !row.CredentialEncrypted.Valid {
		return imp, nil
	}
	plain, err := crypto.Decrypt(r.encKey, row.CredentialEncrypted.String)
	if err != nil {
		return nil, fmt.Errorf("unseal import %s credential: %w", row.ID, apperrs.Fatal(err))
	}
	imp.Credential = string(plain)
	return imp, nil
}
