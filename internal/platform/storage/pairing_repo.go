package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/jsonx"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/plays"
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
		q := r.q.WithTx(tx)
		next, err := q.NextPairingComputerPosition(ctx, c.UserID)
		if err != nil {
			return fmt.Errorf("place computer %s: %w", c.ID, err)
		}
		err = q.SavePairingComputer(ctx, sqlcgen.SavePairingComputerParams{
			ID: c.ID, UserID: c.UserID, Kind: string(c.Kind), Name: c.Name, ServerUrl: c.ServerURL, BearerToken: c.BearerToken,
			TokenExpiresAt: c.TokenExpiresAt.Unix(), HarnessVersion: c.HarnessVersion,
			CreatedAt: c.CreatedAt.Unix(), UpdatedAt: c.UpdatedAt.Unix(),
			TunnelID: t.TunnelID, TunnelHostname: t.Hostname, TunnelZoneID: t.ZoneID, TunnelRecordID: t.RecordID,
			TunnelAccessAppID: t.AccessAppID, Position: next,
		})
		if err != nil {
			return fmt.Errorf("save computer %s: %w", c.ID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *PairingRepo) ListTunnelComputers(ctx context.Context) ([]pairing.Computer, error) {
	rows, err := r.q.ListPairingTunnelComputers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list computers on a tunnel: %w", err)
	}
	out := make([]pairing.Computer, 0, len(rows))
	for _, row := range rows {
		out = append(out, toPairingComputer(row))
	}
	return out, nil
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

func (r *PairingRepo) ListComputersFor(ctx context.Context, userID string) ([]pairing.Computer, error) {
	rows, err := r.q.ListPairingComputersFor(ctx, nullString(userID))
	if err != nil {
		return nil, fmt.Errorf("list computers for %s: %w", userID, err)
	}
	out := make([]pairing.Computer, 0, len(rows))
	for _, row := range rows {
		c := toPairingComputer(row.PairingComputer)
		if c.UserID != userID {
			allow, err := parseSet(row.RuleAllow)
			if err != nil {
				return nil, fmt.Errorf("decode the rule on computer %s for %s: %w", c.ID, userID, err)
			}
			c.Shared = &pairing.SharedComputer{OwnerID: c.UserID, OwnerName: row.OwnerName, Permissions: allow}
		}
		out = append(out, c)
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

func (r *PairingRepo) SetFacts(ctx context.Context, userID, computerID string, facts pairing.Facts, at time.Time, evt eventbus.OutboxEvent) error {
	raw, err := jsonx.Marshal(facts)
	if err != nil {
		return fmt.Errorf("encode facts of computer %s: %w", computerID, err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetPairingComputerFacts(ctx, sqlcgen.SetPairingComputerFactsParams{
			Facts: string(raw), FactsAt: sql.NullInt64{Int64: at.Unix(), Valid: true}, ID: computerID, UserID: userID,
		})
		if err != nil {
			return fmt.Errorf("set facts of computer %s: %w", computerID, err)
		}
		if n == 0 {
			return fmt.Errorf("set facts of computer %s: %w", computerID, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, []eventbus.OutboxEvent{evt})
	})
}

func (r *PairingRepo) ClearTunnel(ctx context.Context, userID, computerID string, at time.Time, evt eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).ClearPairingComputerTunnel(ctx, sqlcgen.ClearPairingComputerTunnelParams{UpdatedAt: at.Unix(), ID: computerID, UserID: userID})
		if err != nil {
			return fmt.Errorf("clear tunnel of computer %s: %w", computerID, err)
		}
		if n == 0 {
			return fmt.Errorf("clear tunnel of computer %s: %w", computerID, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, []eventbus.OutboxEvent{evt})
	})
}

func (r *PairingRepo) ComputersToMove(ctx context.Context) ([]pairing.Computer, error) {
	rows, err := r.q.ListPairingComputersToMove(ctx, "%"+pairing.ComputerHostSuffix)
	if err != nil {
		return nil, fmt.Errorf("list computers to move: %w", err)
	}
	var out []pairing.Computer
	for _, row := range rows {
		if len(out) > 0 && out[len(out)-1].UserID == row.UserID {
			continue
		}
		out = append(out, pairing.Computer{ID: row.ID, UserID: row.UserID})
	}
	return out, nil
}

func (r *PairingRepo) RecordMoveNotice(ctx context.Context, userID string, at time.Time, evt eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).InsertPairingMoveNotice(ctx, sqlcgen.InsertPairingMoveNoticeParams{UserID: userID, NotifiedAt: at.Unix()})
		if err != nil {
			return fmt.Errorf("record move notice of %s: %w", userID, err)
		}
		if n == 0 {
			return nil
		}
		return insertOutboxRows(ctx, tx, []eventbus.OutboxEvent{evt})
	})
}

func (r *PairingRepo) SwitchComputerKind(ctx context.Context, id string, from, to harness.Kind, harnessVersion string, at time.Time, evt func(userID string) eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		userID, err := r.q.WithTx(tx).SwitchPairingComputerKind(ctx, sqlcgen.SwitchPairingComputerKindParams{
			ToKind: string(to), HarnessVersion: harnessVersion, UpdatedAt: at.Unix(), ID: id, FromKind: string(from),
		})
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("switch computer %s to %s: %w", id, to, err)
		}
		return insertOutboxRows(ctx, tx, []eventbus.OutboxEvent{evt(userID)})
	})
}

func (r *PairingRepo) ReorderComputers(ctx context.Context, userID string, ids []string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for i, id := range ids {
			n, err := q.SetPairingComputerPosition(ctx, sqlcgen.SetPairingComputerPositionParams{Position: int64(i), ID: id, UserID: userID})
			if err != nil {
				return fmt.Errorf("move computer %s: %w", id, err)
			}
			if n == 0 {
				return fmt.Errorf("move computer %s: %w", id, apperrs.ErrNotFound)
			}
		}
		return nil
	})
}

func (r *PairingRepo) SetComputerPlacement(ctx context.Context, userID, computerID string, runsAtOnce int, chatProjectID string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetPairingComputerPlacement(ctx, sqlcgen.SetPairingComputerPlacementParams{
			RunsAtOnce: int64(runsAtOnce), ChatProjectID: chatProjectID, UpdatedAt: at.Unix(), ID: computerID, UserID: userID,
		})
		if err != nil {
			return fmt.Errorf("set runs at once of computer %s: %w", computerID, err)
		}
		if n == 0 {
			return fmt.Errorf("set runs at once of computer %s: %w", computerID, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *PairingRepo) CountRunning(ctx context.Context, computerIDs []string) (map[string]int, error) {
	out := map[string]int{}
	if len(computerIDs) == 0 {
		return out, nil
	}
	rows, err := r.q.CountPairingComputerRuns(ctx, idsJSON(computerIDs))
	if err != nil {
		return nil, fmt.Errorf("count the runs on computers: %w", err)
	}
	for _, row := range rows {
		out[row.ComputerID] = int(row.Running)
	}
	return out, nil
}

func (r *PairingRepo) GetProjectLink(ctx context.Context, userID, projectID string) (pairing.ProjectLink, error) {
	link := pairing.ProjectLink{UserID: userID, ProjectID: projectID}
	row, err := r.q.GetPairingProjectLink(ctx, sqlcgen.GetPairingProjectLinkParams{UserID: userID, ProjectID: projectID})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return pairing.ProjectLink{}, fmt.Errorf("get project link %s: %w", projectID, err)
	}
	if err == nil {
		if link, err = toProjectLink(row); err != nil {
			return pairing.ProjectLink{}, err
		}
	}
	computers, err := r.q.ListPairingProjectComputers(ctx, sqlcgen.ListPairingProjectComputersParams{UserID: userID, ProjectID: projectID})
	if err != nil {
		return pairing.ProjectLink{}, fmt.Errorf("list the computers of project link %s: %w", projectID, err)
	}
	for _, c := range computers {
		link = withProjectComputer(link, c)
	}
	return link, nil
}

func (r *PairingRepo) ListProjectLinks(ctx context.Context, userID string) ([]pairing.ProjectLink, error) {
	rows, err := r.q.ListPairingProjectLinks(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list project links: %w", err)
	}
	computers, err := r.q.ListPairingProjectComputersByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list the computers of project links: %w", err)
	}
	byProject := map[string]pairing.ProjectLink{}
	var order []string
	for _, row := range rows {
		link, err := toProjectLink(row)
		if err != nil {
			return nil, err
		}
		byProject[link.ProjectID] = link
		order = append(order, link.ProjectID)
	}
	for _, c := range computers {
		link, ok := byProject[c.ProjectID]
		if !ok {
			link = pairing.ProjectLink{UserID: userID, ProjectID: c.ProjectID}
			order = append(order, c.ProjectID)
		}
		byProject[c.ProjectID] = withProjectComputer(link, c)
	}
	slices.Sort(order)
	out := make([]pairing.ProjectLink, 0, len(order))
	for _, id := range order {
		out = append(out, byProject[id])
	}
	return out, nil
}

// withProjectComputer adds one project computer row to link: a position places it in the project's order, and a T3
// project is the checkout picked there.
func withProjectComputer(link pairing.ProjectLink, c sqlcgen.PairingProjectComputer) pairing.ProjectLink {
	if c.Position.Valid {
		link.Computers = append(link.Computers, c.ComputerID)
	}
	if c.HarnessProjectID != "" {
		link.Checkouts = append(link.Checkouts, pairing.Checkout{ComputerID: c.ComputerID, HarnessProjectID: c.HarnessProjectID})
	}
	return link
}

func toProjectLink(row sqlcgen.PairingProjectLink) (pairing.ProjectLink, error) {
	options, err := unmarshalModelOptions(row.ModelOptions)
	if err != nil {
		return pairing.ProjectLink{}, fmt.Errorf("decode model options for project link %s: %w", row.ProjectID, err)
	}
	return pairing.ProjectLink{
		UserID: row.UserID, ProjectID: row.ProjectID, Provider: row.Provider, Model: row.Model, ModelOptions: options,
		StartIn: pairing.StartIn(row.StartIn), Placement: pairing.Placement(row.Placement), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}, nil
}

func (r *PairingRepo) SaveProjectLink(ctx context.Context, l pairing.ProjectLink) error {
	options, err := marshalModelOptions(l.ModelOptions)
	if err != nil {
		return fmt.Errorf("encode model options for project link %s: %w", l.ProjectID, err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.SavePairingProjectLink(ctx, sqlcgen.SavePairingProjectLinkParams{
			UserID: l.UserID, ProjectID: l.ProjectID, Provider: l.Provider, Model: l.Model, ModelOptions: options,
			StartIn: string(l.StartIn), Placement: string(l.Placement), UpdatedAt: l.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save project link %s: %w", l.ProjectID, classifyWriteErr(err))
		}
		if err := q.ClearPairingProjectOrder(ctx, sqlcgen.ClearPairingProjectOrderParams{UserID: l.UserID, ProjectID: l.ProjectID}); err != nil {
			return fmt.Errorf("clear the order of project link %s: %w", l.ProjectID, err)
		}
		for i, id := range l.Computers {
			err := q.SetPairingProjectComputerPosition(ctx, sqlcgen.SetPairingProjectComputerPositionParams{
				UserID: l.UserID, ProjectID: l.ProjectID, ComputerID: id, Position: sql.NullInt64{Int64: int64(i), Valid: true},
			})
			if err != nil {
				return fmt.Errorf("place computer %s in project link %s: %w", id, l.ProjectID, classifyWriteErr(err))
			}
		}
		for _, c := range l.Checkouts {
			err := q.SetPairingProjectCheckout(ctx, sqlcgen.SetPairingProjectCheckoutParams{
				UserID: l.UserID, ProjectID: l.ProjectID, ComputerID: c.ComputerID, HarnessProjectID: c.HarnessProjectID,
			})
			if err != nil {
				return fmt.Errorf("save the checkout on %s for project link %s: %w", c.ComputerID, l.ProjectID, classifyWriteErr(err))
			}
		}
		if err := q.PrunePairingProjectComputers(ctx, sqlcgen.PrunePairingProjectComputersParams{UserID: l.UserID, ProjectID: l.ProjectID}); err != nil {
			return fmt.Errorf("prune project link %s: %w", l.ProjectID, err)
		}
		return nil
	})
}

func (r *PairingRepo) DeleteProjectLink(ctx context.Context, userID, projectID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.DeletePairingProjectLink(ctx, sqlcgen.DeletePairingProjectLinkParams{UserID: userID, ProjectID: projectID}); err != nil {
			return fmt.Errorf("clear project link %s: %w", projectID, err)
		}
		if err := q.ClearPairingProjectOrder(ctx, sqlcgen.ClearPairingProjectOrderParams{UserID: userID, ProjectID: projectID}); err != nil {
			return fmt.Errorf("clear the order of project link %s: %w", projectID, err)
		}
		if err := q.PrunePairingProjectComputers(ctx, sqlcgen.PrunePairingProjectComputersParams{UserID: userID, ProjectID: projectID}); err != nil {
			return fmt.Errorf("prune project link %s: %w", projectID, err)
		}
		return nil
	})
}

func (r *PairingRepo) SaveCheckout(ctx context.Context, userID, projectID, computerID, harnessProjectID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SetPairingProjectCheckout(ctx, sqlcgen.SetPairingProjectCheckoutParams{
			UserID: userID, ProjectID: projectID, ComputerID: computerID, HarnessProjectID: harnessProjectID,
		})
		if err != nil {
			return fmt.Errorf("save the checkout on %s for project %s: %w", computerID, projectID, classifyWriteErr(err))
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
	options, err := unmarshalModelOptions(row.ModelOptions)
	if err != nil {
		return pairing.Defaults{}, fmt.Errorf("decode model options for defaults: %w", err)
	}
	return pairing.Defaults{
		Provider: row.Provider, Model: row.Model, ModelOptions: options, StartIn: pairing.StartIn(row.StartIn),
		Placement: pairing.Placement(row.Placement), CloneAutomatically: row.CloneAutomatically != 0,
	}, nil
}

func (r *PairingRepo) SaveDefaults(ctx context.Context, d pairing.Defaults) error {
	options, err := marshalModelOptions(d.ModelOptions)
	if err != nil {
		return fmt.Errorf("encode model options for %s: %w", d.UserID, err)
	}
	clone := int64(0)
	if d.CloneAutomatically {
		clone = 1
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SavePairingDefaults(ctx, sqlcgen.SavePairingDefaultsParams{
			UserID: d.UserID, Provider: d.Provider, Model: d.Model, ModelOptions: options, StartIn: string(d.StartIn),
			Placement: string(d.Placement), CloneAutomatically: clone,
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

func (r *PairingRepo) SaveSetupChoices(ctx context.Context, userID, computerID string, c pairing.SetupChoices) error {
	skipped, err := json.Marshal(c.Skipped)
	if err != nil {
		return fmt.Errorf("encode skipped providers: %w", err)
	}
	models, err := json.Marshal(c.Models)
	if err != nil {
		return fmt.Errorf("encode setup models: %w", err)
	}
	options, err := json.Marshal(c.ModelOptions)
	if err != nil {
		return fmt.Errorf("encode setup model options: %w", err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetPairingComputerSetupChoices(ctx, sqlcgen.SetPairingComputerSetupChoicesParams{
			SetupSkippedProviders: string(skipped), SetupModels: string(models), SetupModelOptions: string(options),
			SetupFolder: c.Folder, ID: computerID, UserID: userID,
		})
		if err != nil {
			return fmt.Errorf("save setup choices for %s: %w", computerID, err)
		}
		if n == 0 {
			return fmt.Errorf("save setup choices for %s: %w", computerID, apperrs.ErrNotFound)
		}
		return nil
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
		out = append(out, pairing.ProviderSetup{
			Provider: row.Provider, ConfirmedAt: unixPtrFromNull(row.ConfirmedAt), Skills: skills, SkillsVersion: row.SkillsVersion,
		})
	}
	return out, nil
}

func (r *PairingRepo) ListOutdatedSkillsComputers(ctx context.Context, userID, version string) ([]string, error) {
	ids, err := r.q.ListPairingComputersWithOutdatedSkills(ctx, sqlcgen.ListPairingComputersWithOutdatedSkillsParams{UserID: userID, SkillsVersion: version})
	if err != nil {
		return nil, fmt.Errorf("list computers with outdated skills for %s: %w", userID, err)
	}
	return ids, nil
}

func (r *PairingRepo) SaveProviderSetup(ctx context.Context, computerID string, p pairing.ProviderSetup, updatedAt time.Time, evt eventbus.OutboxEvent) error {
	skills, err := json.Marshal(p.Skills)
	if err != nil {
		return fmt.Errorf("encode %s skills: %w", p.Provider, err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SavePairingProviderSetup(ctx, sqlcgen.SavePairingProviderSetupParams{
			ComputerID: computerID, Provider: p.Provider, ConfirmedAt: nullUnixPtr(p.ConfirmedAt),
			SkillsJson: string(skills), SkillsVersion: p.SkillsVersion, UpdatedAt: updatedAt.Unix(),
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
			ID: t.ID, RunID: t.RunID, ComputerID: t.ComputerID, Provider: t.Provider, ProviderName: t.ProviderName, Model: t.Model,
			Kind: string(t.Kind), State: string(t.State), Status: t.Status, Transcript: string(transcript), StartedAt: t.StartedAt.Unix(),
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
			RunID: row.RunID, TurnID: row.ID, Provider: row.Provider, ProviderName: row.ProviderName, Model: row.Model,
			Kind: pairing.SetupTurnKind(row.Kind), State: pairing.SetupTurnState(row.State), Status: row.Status, UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
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
		SetupChoices: pairing.SetupChoices{
			Skipped: decodeJSON(row.SetupSkippedProviders, []string{}), Models: decodeJSON(row.SetupModels, map[string]string{}),
			ModelOptions: decodeJSON(row.SetupModelOptions, map[string][]harness.OptionSetting{}), Folder: row.SetupFolder,
		},
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
		Tunnel: toComputerTunnel(row), Facts: toFacts(row), FactsAt: unixPtrFromNull(row.FactsAt),
		Position: int(row.Position), RunsAtOnce: int(row.RunsAtOnce), ChatProjectID: row.ChatProjectID,
	}
}

// toFacts is nil until the computer's runner first reported.
func toFacts(row sqlcgen.PairingComputer) *pairing.Facts {
	if !row.FactsAt.Valid {
		return nil
	}
	f := decodeJSON(row.Facts, pairing.Facts{})
	return &f
}

// decodeJSON reads a value this repo wrote; an unreadable one is empty, so setup opens on its own preselection.
func decodeJSON[T any](raw string, empty T) T {
	var v T
	if err := json.Unmarshal([]byte(raw), &v); err != nil || raw == "null" {
		return empty
	}
	return v
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

// marshalModelOptions stores no options as "[]", the column default.
func marshalModelOptions(options []harness.OptionSetting) (string, error) {
	if len(options) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(options)
	return string(b), err
}

// unmarshalModelOptions reads the column back; an empty list is nil, as the domain never sets one.
func unmarshalModelOptions(raw string) ([]harness.OptionSetting, error) {
	var options []harness.OptionSetting
	if err := json.Unmarshal([]byte(raw), &options); err != nil {
		return nil, err
	}
	if len(options) == 0 {
		return nil, nil
	}
	return options, nil
}

// decodePassedOver reads a stored list of passed-over computers; empty reads back as none.
func decodePassedOver(raw string) []plays.PassedOver {
	out := decodeJSON(raw, []plays.PassedOver(nil))
	if len(out) == 0 {
		return nil
	}
	return out
}
