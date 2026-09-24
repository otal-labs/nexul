package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ pairing.Repo = (*PairingRepo)(nil)

// PairingRepo persists pairing state: computers and per-user
// defaults. Bearer tokens arrive already encrypted from the use-case layer.
type PairingRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *PairingRepo) SaveComputer(ctx context.Context, c pairing.Computer, evts ...eventbus.OutboxEvent) error {
	var t pairing.ComputerTunnel
	if c.Tunnel != nil {
		t = *c.Tunnel
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SavePairingComputer(ctx, sqlcgen.SavePairingComputerParams{
			ID: c.ID, UserID: c.UserID, Kind: string(c.Kind), Name: c.Name, ServerUrl: c.ServerURL, BearerToken: c.BearerToken,
			TokenExpiresAt: c.TokenExpiresAt.Unix(), HarnessVersion: c.HarnessVersion,
			CreatedAt: c.CreatedAt.Unix(), UpdatedAt: c.UpdatedAt.Unix(),
			TunnelID: t.TunnelID, TunnelHostname: t.Hostname, TunnelZoneID: t.ZoneID, TunnelRecordID: t.RecordID,
			TunnelAccessAppID: t.AccessAppID,
		})
		if err != nil {
			return fmt.Errorf("save computer %s: %w", c.ID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

// ComputerTunnelHostnameExists reports whether host is a computer tunnel's hostname, the only hosts the Access headers go to.
func (r *PairingRepo) ComputerTunnelHostnameExists(ctx context.Context, host string) (bool, error) {
	exists, err := r.q.PairingComputerTunnelHostnameExists(ctx, host)
	if err != nil {
		return false, fmt.Errorf("look up computer tunnel hostname: %w", err)
	}
	return exists, nil
}

func (r *PairingRepo) GetComputer(ctx context.Context, userID, id string) (*pairing.Computer, error) {
	row, err := r.q.GetPairingComputer(ctx, sqlcgen.GetPairingComputerParams{ID: id, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("get computer %s: %w", id, notFoundIfNoRows(err))
	}
	c := toPairingComputer(row)
	return &c, nil
}

func (r *PairingRepo) ListComputers(ctx context.Context, userID string) ([]pairing.Computer, error) {
	rows, err := r.q.ListPairingComputers(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list computers: %w", err)
	}
	out := make([]pairing.Computer, 0, len(rows))
	for _, row := range rows {
		out = append(out, toPairingComputer(row))
	}
	return out, nil
}

func (r *PairingRepo) DeleteComputer(ctx context.Context, userID, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeletePairingComputer(ctx, sqlcgen.DeletePairingComputerParams{ID: id, UserID: userID})
		if err != nil {
			return fmt.Errorf("delete computer %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete computer %s: %w", id, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *PairingRepo) GetComputerByID(ctx context.Context, id string) (*pairing.Computer, error) {
	row, err := r.q.GetPairingComputerByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get computer %s: %w", id, notFoundIfNoRows(err))
	}
	c := toPairingComputer(row)
	return &c, nil
}

func (r *PairingRepo) GetProjectLink(ctx context.Context, projectID string) (pairing.ProjectLink, error) {
	row, err := r.q.GetPairingProjectLink(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pairing.ProjectLink{}, nil
		}
		return pairing.ProjectLink{}, fmt.Errorf("get project link %s: %w", projectID, err)
	}
	return pairing.ProjectLink{
		ProjectID: row.ProjectID, ComputerID: row.ComputerID, HarnessProjectID: row.HarnessProjectID,
		Provider: row.Provider, Model: row.Model, UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}, nil
}

func (r *PairingRepo) SaveProjectLink(ctx context.Context, l pairing.ProjectLink) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SavePairingProjectLink(ctx, sqlcgen.SavePairingProjectLinkParams{
			ProjectID: l.ProjectID, ComputerID: l.ComputerID, HarnessProjectID: l.HarnessProjectID,
			Provider: l.Provider, Model: l.Model, UpdatedAt: l.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save project link %s: %w", l.ProjectID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *PairingRepo) DeleteProjectLink(ctx context.Context, projectID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeletePairingProjectLink(ctx, projectID); err != nil {
			return fmt.Errorf("clear project link %s: %w", projectID, err)
		}
		return nil
	})
}

func (r *PairingRepo) GetDefaults(ctx context.Context, userID string) (pairing.Defaults, error) {
	row, err := r.q.GetPairingDefaults(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pairing.Defaults{}, nil
		}
		return pairing.Defaults{}, fmt.Errorf("get defaults: %w", err)
	}
	return pairing.Defaults{
		DefaultComputerID: row.DefaultComputerID.String,
		FallbackProjectID: row.FallbackProjectID,
		Provider:          row.Provider,
		Model:             row.Model,
	}, nil
}

func (r *PairingRepo) SaveDefaults(ctx context.Context, d pairing.Defaults) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SavePairingDefaults(ctx, sqlcgen.SavePairingDefaultsParams{
			UserID:            d.UserID,
			DefaultComputerID: sql.NullString{String: d.DefaultComputerID, Valid: d.DefaultComputerID != ""},
			FallbackProjectID: d.FallbackProjectID,
			Provider:          d.Provider,
			Model:             d.Model,
		})
		if err != nil {
			return fmt.Errorf("save defaults for %s: %w", d.UserID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *PairingRepo) SetSetupConfirmedAt(ctx context.Context, userID, computerID string, at *time.Time, evt eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetPairingComputerSetupConfirmedAt(ctx, sqlcgen.SetPairingComputerSetupConfirmedAtParams{
			SetupConfirmedAt: nullUnixPtr(at), ID: computerID, UserID: userID,
		})
		if err != nil {
			return fmt.Errorf("set setup confirmation for %s: %w", computerID, err)
		}
		if n == 0 {
			return fmt.Errorf("set setup confirmation for %s: %w", computerID, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, []eventbus.OutboxEvent{evt})
	})
}

func (r *PairingRepo) ListProviderSetups(ctx context.Context, computerID string) ([]pairing.ProviderSetup, error) {
	rows, err := r.q.ListPairingProviderSetups(ctx, computerID)
	if err != nil {
		return nil, fmt.Errorf("list provider setups for %s: %w", computerID, err)
	}
	out := make([]pairing.ProviderSetup, 0, len(rows))
	for _, row := range rows {
		var skills []string
		if err := json.Unmarshal([]byte(row.SkillsJson), &skills); err != nil {
			return nil, fmt.Errorf("decode %s skills for %s: %w", row.Provider, computerID, err)
		}
		out = append(out, pairing.ProviderSetup{Provider: row.Provider, ConfirmedAt: unixPtrFromNull(row.ConfirmedAt), Skills: skills})
	}
	return out, nil
}

func (r *PairingRepo) SaveProviderSetup(ctx context.Context, computerID string, p pairing.ProviderSetup, updatedAt time.Time, evt eventbus.OutboxEvent) error {
	skills, err := json.Marshal(p.Skills)
	if err != nil {
		return fmt.Errorf("encode %s skills: %w", p.Provider, err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SavePairingProviderSetup(ctx, sqlcgen.SavePairingProviderSetupParams{
			ComputerID: computerID, Provider: p.Provider, ConfirmedAt: nullUnixPtr(p.ConfirmedAt),
			SkillsJson: string(skills), UpdatedAt: updatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save %s setup for %s: %w", p.Provider, computerID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, []eventbus.OutboxEvent{evt})
	})
}

func (r *PairingRepo) SaveSetupTurn(ctx context.Context, t pairing.SetupTurn, evts ...eventbus.OutboxEvent) error {
	transcript, err := json.Marshal(t.Transcript)
	if err != nil {
		return fmt.Errorf("encode setup turn %s transcript: %w", t.ID, err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SavePairingSetupTurn(ctx, sqlcgen.SavePairingSetupTurnParams{
			ID: t.ID, RunID: t.RunID, ComputerID: t.ComputerID, Provider: t.Provider, ProviderName: t.ProviderName,
			State: string(t.State), Status: t.Status, Transcript: string(transcript), StartedAt: t.StartedAt.Unix(),
			UpdatedAt: t.UpdatedAt.Unix(), EndedAt: nullUnixPtr(t.EndedAt),
		})
		if err != nil {
			return fmt.Errorf("save setup turn %s: %w", t.ID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *PairingRepo) ListLatestSetupTurns(ctx context.Context, computerID string) ([]pairing.SetupTurnSummary, error) {
	rows, err := r.q.ListPairingSetupTurnsLatest(ctx, computerID)
	if err != nil {
		return nil, fmt.Errorf("list setup turns for %s: %w", computerID, err)
	}
	out := make([]pairing.SetupTurnSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, pairing.SetupTurnSummary{
			RunID: row.RunID, TurnID: row.ID, Provider: row.Provider, ProviderName: row.ProviderName,
			State: pairing.SetupTurnState(row.State), Status: row.Status, UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
		})
	}
	return out, nil
}

func (r *PairingRepo) SetSetupMCPToken(ctx context.Context, userID, computerID, sealed string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetPairingComputerSetupMCPToken(ctx, sqlcgen.SetPairingComputerSetupMCPTokenParams{
			SetupMcpToken: sealed, ID: computerID, UserID: userID,
		})
		if err != nil {
			return fmt.Errorf("set setup mcp token for %s: %w", computerID, err)
		}
		if n == 0 {
			return fmt.Errorf("set setup mcp token for %s: %w", computerID, apperrs.ErrNotFound)
		}
		return nil
	})
}

func toPairingComputer(row sqlcgen.PairingComputer) pairing.Computer {
	return pairing.Computer{
		ID: row.ID, UserID: row.UserID, Kind: harness.Kind(row.Kind), Name: row.Name, ServerURL: row.ServerUrl, BearerToken: row.BearerToken,
		TokenExpiresAt: time.Unix(row.TokenExpiresAt, 0).UTC(), HarnessVersion: row.HarnessVersion,
		SetupConfirmedAt: unixPtrFromNull(row.SetupConfirmedAt), SetupMCPToken: row.SetupMcpToken,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
		Tunnel: toComputerTunnel(row),
	}
}

func toComputerTunnel(row sqlcgen.PairingComputer) *pairing.ComputerTunnel {
	if row.TunnelID == "" {
		return nil
	}
	return &pairing.ComputerTunnel{
		TunnelID: row.TunnelID, Hostname: row.TunnelHostname, ZoneID: row.TunnelZoneID, RecordID: row.TunnelRecordID,
		AccessAppID: row.TunnelAccessAppID,
	}
}

// unixPtrFromNull is nullUnixPtr's inverse: SQL NULL reads back as nil.
func unixPtrFromNull(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := time.Unix(v.Int64, 0).UTC()
	return &t
}
