package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/plays"

	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ plays.TrailRepo = (*PlayTrailsRepo)(nil)

// PlayTrailsRepo persists the plays domain's Trail entity (ADR 0055).
type PlayTrailsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *PlayTrailsRepo) CreateTrail(ctx context.Context, t *plays.Trail, evts ...eventbus.OutboxEvent) error {
	memories, err := marshalStringList(t.SelectedMemoryIDs)
	if err != nil {
		return fmt.Errorf("encode selected memories for trail %s: %w", t.ID, err)
	}
	activity, err := marshalActivity(t.Activity)
	if err != nil {
		return fmt.Errorf("encode activity for trail %s: %w", t.ID, err)
	}
	question, err := marshalQuestion(t.Question)
	if err != nil {
		return fmt.Errorf("encode question for trail %s: %w", t.ID, err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreatePlayTrail(ctx, sqlcgen.CreatePlayTrailParams{
			ID: t.ID, WorkspaceID: t.WorkspaceID, PlayID: t.PlayID, PlayLabel: t.PlayLabel,
			TargetType: string(t.TargetType), TargetID: t.TargetID, ProjectID: t.ProjectID, ConversationID: t.ConversationID,
			StarterID: t.StarterID, Via: string(t.Via), SelectedMemoryIds: memories, CustomInstructions: t.CustomInstructions,
			MoveToStatusID: nullStringOrNil(t.MoveToStatusID), HarnessSessionID: t.HarnessSessionID, State: string(t.State),
			StartedAt: t.StartedAt.Unix(), EndedAt: nullUnixPtr(t.EndedAt), LastError: t.LastError,
			ReplyMessageID: t.ReplyMessageID, Activity: activity, ComputerID: t.ComputerID, Provider: t.Provider, Model: t.Model, Question: question,
		})
		if err != nil {
			return fmt.Errorf("insert trail %s: %w", t.ID, classifyWriteErr(err))
		}
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
}

func (r *PlayTrailsRepo) GetTrail(ctx context.Context, id string) (*plays.Trail, error) {
	row, err := r.q.GetPlayTrail(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get trail %s: %w", id, notFoundIfNoRows(err))
	}
	return toTrail(row)
}

func (r *PlayTrailsRepo) UpdateTrail(ctx context.Context, t *plays.Trail, evts ...eventbus.OutboxEvent) error {
	activity, err := marshalActivity(t.Activity)
	if err != nil {
		return fmt.Errorf("encode activity for trail %s: %w", t.ID, err)
	}
	question, err := marshalQuestion(t.Question)
	if err != nil {
		return fmt.Errorf("encode question for trail %s: %w", t.ID, err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdatePlayTrail(ctx, sqlcgen.UpdatePlayTrailParams{
			ConversationID: t.ConversationID, HarnessSessionID: t.HarnessSessionID, State: string(t.State),
			EndedAt: nullUnixPtr(t.EndedAt), LastError: t.LastError, ReplyMessageID: t.ReplyMessageID, Activity: activity,
			Question: question, ID: t.ID,
		})
		if err != nil {
			return fmt.Errorf("update trail %s: %w", t.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update trail %s: %w", t.ID, apperrs.ErrNotFound)
		}
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
}

func (r *PlayTrailsRepo) ListTrailsByTarget(ctx context.Context, targetType plays.TargetType, targetID string) ([]*plays.Trail, error) {
	rows, err := r.q.ListPlayTrailsByTarget(ctx, sqlcgen.ListPlayTrailsByTargetParams{TargetType: string(targetType), TargetID: targetID})
	if err != nil {
		return nil, fmt.Errorf("list trails for %s %s: %w", targetType, targetID, err)
	}
	out := make([]*plays.Trail, 0, len(rows))
	for _, row := range rows {
		t, err := toTrail(row)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *PlayTrailsRepo) ListActiveTrailsByTargets(ctx context.Context, targetType plays.TargetType, targetIDs []string) ([]*plays.Trail, error) {
	if len(targetIDs) == 0 {
		return []*plays.Trail{}, nil
	}
	rows, err := r.q.ListActivePlayTrailsByTargets(ctx, sqlcgen.ListActivePlayTrailsByTargetsParams{TargetType: string(targetType), Ids: targetIDs})
	if err != nil {
		return nil, fmt.Errorf("list active trails for %s: %w", targetType, err)
	}
	out := make([]*plays.Trail, 0, len(rows))
	for _, row := range rows {
		t, err := toTrail(row)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *PlayTrailsRepo) LatestTrailForChoices(ctx context.Context, starterID, playID, projectID string) (*plays.Trail, error) {
	row, err := r.q.LatestPlayTrailForChoices(ctx, sqlcgen.LatestPlayTrailForChoicesParams{StarterID: starterID, PlayID: playID, ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("latest trail for play %s by %s: %w", playID, starterID, notFoundIfNoRows(err))
	}
	return toTrail(row)
}

func nullStringOrNil(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func marshalActivity(entries []plays.ActivityEntry) (string, error) {
	if len(entries) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(entries)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func marshalQuestion(q *plays.TrailQuestion) (sql.NullString, error) {
	if q == nil {
		return sql.NullString{}, nil
	}
	b, err := json.Marshal(q)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(b), Valid: true}, nil
}

func toTrail(row sqlcgen.PlayTrail) (*plays.Trail, error) {
	var memories []string
	if err := json.Unmarshal([]byte(row.SelectedMemoryIds), &memories); err != nil {
		return nil, fmt.Errorf("decode selected memories for trail %s: %w", row.ID, err)
	}
	activity, err := plays.DecodeActivity([]byte(row.Activity))
	if err != nil {
		return nil, fmt.Errorf("decode activity for trail %s: %w", row.ID, err)
	}
	var endedAt *time.Time
	if row.EndedAt.Valid {
		at := time.Unix(row.EndedAt.Int64, 0).UTC()
		endedAt = &at
	}
	var question *plays.TrailQuestion
	if row.Question.Valid {
		question = &plays.TrailQuestion{}
		if err := json.Unmarshal([]byte(row.Question.String), question); err != nil {
			return nil, fmt.Errorf("decode question for trail %s: %w", row.ID, err)
		}
	}
	return &plays.Trail{
		ID: row.ID, WorkspaceID: row.WorkspaceID, PlayID: row.PlayID, PlayLabel: row.PlayLabel,
		TargetType: plays.TargetType(row.TargetType), TargetID: row.TargetID, ProjectID: row.ProjectID,
		ConversationID: row.ConversationID, StarterID: row.StarterID, Via: plays.Via(row.Via),
		SelectedMemoryIDs: memories, CustomInstructions: row.CustomInstructions, MoveToStatusID: row.MoveToStatusID.String,
		ComputerID: row.ComputerID, Provider: row.Provider, Model: row.Model,
		HarnessSessionID: row.HarnessSessionID, State: plays.TrailState(row.State),
		StartedAt: time.Unix(row.StartedAt, 0).UTC(), EndedAt: endedAt, LastError: row.LastError,
		ReplyMessageID: row.ReplyMessageID, Activity: activity, Question: question,
	}, nil
}
