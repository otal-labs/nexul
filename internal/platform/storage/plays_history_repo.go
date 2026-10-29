package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/plays"

	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

// writeRevision adds a save's history to its transaction, then drops what falls past the caps.
func writeRevision(ctx context.Context, q *sqlcgen.Queries, p *plays.Play, rev plays.Revision) error {
	if v := rev.Instructions; v != nil {
		editors, err := marshalStringList(v.EditorIDs)
		if err != nil {
			return fmt.Errorf("encode editors of play %s: %w", p.ID, err)
		}
		sizer := playSizer(q, p.ID)
		size, err := sizer.againstNewest(ctx, v.Body)
		if err != nil {
			return fmt.Errorf("size instructions version of play %s: %w", p.ID, err)
		}
		err = q.InsertPlayInstructionVersion(ctx, sqlcgen.InsertPlayInstructionVersionParams{
			PlayID: p.ID, Body: v.Body, EditorIds: editors, Via: v.Via, BuiltinDefault: boolToInt(v.BuiltinDefault),
			WordsAdded: size.added, WordsRemoved: size.removed, RestoredFrom: restoredFromOf(v.RestoredFrom), CreatedAt: p.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert instructions version of play %s: %w", p.ID, classifyWriteErr(err))
		}
		err = sizer.pruned(ctx, func() error {
			return q.PrunePlayInstructionVersions(ctx, sqlcgen.PrunePlayInstructionVersionsParams{PlayID: p.ID, Keep: plays.KeptInstructionVersions})
		})
		if err != nil {
			return fmt.Errorf("prune instructions versions of play %s: %w", p.ID, err)
		}
	}
	if len(rev.Changes) == 0 {
		return nil
	}
	for _, c := range rev.Changes {
		actor, err := json.Marshal(c.Actor)
		if err != nil {
			return fmt.Errorf("encode actor of change %s: %w", c.ID, err)
		}
		err = q.InsertPlayChange(ctx, sqlcgen.InsertPlayChangeParams{
			ID: c.ID, PlayID: p.ID, Field: c.Field, FromValue: c.From, ToValue: c.To, Actor: string(actor), CreatedAt: c.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert change %s of play %s: %w", c.ID, p.ID, classifyWriteErr(err))
		}
	}
	if err := q.PrunePlayChanges(ctx, sqlcgen.PrunePlayChangesParams{PlayID: p.ID, Keep: plays.KeptChanges}); err != nil {
		return fmt.Errorf("prune changes of play %s: %w", p.ID, err)
	}
	return nil
}

func (r *PlaysRepo) ListHistory(ctx context.Context, playID string) (plays.History, error) {
	versions, err := r.q.ListPlayInstructionVersions(ctx, playID)
	if err != nil {
		return plays.History{}, fmt.Errorf("list instructions versions of play %s: %w", playID, err)
	}
	changes, err := r.q.ListPlayChanges(ctx, playID)
	if err != nil {
		return plays.History{}, fmt.Errorf("list changes of play %s: %w", playID, err)
	}
	out := plays.History{Versions: make([]plays.VersionInfo, 0, len(versions)), Changes: make([]plays.Change, 0, len(changes))}
	for _, row := range versions {
		info, err := toVersionInfo(row.PlayID, row.Version, row.EditorIds, row.Via, row.BuiltinDefault, row.CreatedAt)
		if err != nil {
			return plays.History{}, err
		}
		info.WordsAdded, info.WordsRemoved, info.RestoredFrom = nullableInt(row.WordsAdded), nullableInt(row.WordsRemoved), nullableInt(row.RestoredFrom)
		out.Versions = append(out.Versions, info)
	}
	for _, row := range changes {
		var actor plays.Actor
		if err := json.Unmarshal([]byte(row.Actor), &actor); err != nil {
			return plays.History{}, fmt.Errorf("decode actor of change %s: %w", row.ID, err)
		}
		out.Changes = append(out.Changes, plays.Change{
			ID: row.ID, Field: row.Field, From: row.FromValue, To: row.ToValue, Actor: actor, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		})
	}
	return out, nil
}

func (r *PlaysRepo) GetVersion(ctx context.Context, playID string, version int) (*plays.VersionBody, error) {
	row, err := r.q.GetPlayInstructionVersion(ctx, sqlcgen.GetPlayInstructionVersionParams{PlayID: playID, Version: int64(version)})
	if err != nil {
		return nil, fmt.Errorf("get version %d of play %s: %w", version, playID, notFoundIfNoRows(err))
	}
	info, err := toVersionInfo(row.PlayID, row.Version, row.EditorIds, row.Via, row.BuiltinDefault, row.CreatedAt)
	if err != nil {
		return nil, err
	}
	info.WordsAdded, info.WordsRemoved, info.RestoredFrom = nullableInt(row.WordsAdded), nullableInt(row.WordsRemoved), nullableInt(row.RestoredFrom)
	return &plays.VersionBody{VersionInfo: info, Body: row.Body}, nil
}

func toVersionInfo(playID string, version int64, editorIDs, via string, builtinDefault, createdAt int64) (plays.VersionInfo, error) {
	var editors []string
	if err := json.Unmarshal([]byte(editorIDs), &editors); err != nil {
		return plays.VersionInfo{}, fmt.Errorf("decode editors of play %s version %d: %w", playID, version, err)
	}
	return plays.VersionInfo{
		Version: int(version), CreatedAt: time.Unix(createdAt, 0).UTC(), EditorIDs: editors, Via: via, BuiltinDefault: builtinDefault != 0,
	}, nil
}
