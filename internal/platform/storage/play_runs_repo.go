package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/plays"
)

// playRunsFrom joins each run to its doc, so readClause can say whether a doc run's doc is reached.
const playRunsFrom = "play_trails pt LEFT JOIN docs d ON pt.target_type = 'doc' AND d.id = pt.target_id"

// whereRunScope adds the conditions that keep the runs scope shows: in its projects, and a doc run only on a reached doc.
func whereRunScope(where func(cond string, args ...any), scope plays.RunScope) {
	if !scope.All {
		where("pt.project_id IN (SELECT value FROM json_each(?))", idsJSON(scope.ProjectIDs))
	}
	if scope.Docs != nil {
		cond, args := readClause(scope.Docs)
		where("(pt.target_type <> 'doc' OR "+cond+")", args...)
	}
}

func (r *PlayTrailsRepo) ListPlayRuns(ctx context.Context, playID string, scope plays.RunScope, waiting bool, w paging.Window) ([]plays.PlayRun, int, error) {
	q := &pageQuery{from: playRunsFrom, id: "pt.id", order: "pt.started_at DESC, pt.id DESC"}
	q.where("pt.play_id = ?", playID)
	if waiting {
		q.where("pt.state = ?", string(plays.TrailWaiting))
		q.order = "pt.started_at, pt.id"
	}
	whereRunScope(q.where, scope)
	ids, total, err := q.page(ctx, r.db, w)
	if err != nil {
		return nil, 0, fmt.Errorf("page the runs of play %s: %w", playID, err)
	}
	if len(ids) == 0 {
		return []plays.PlayRun{}, total, nil
	}
	rows, err := r.q.ListPlayRunsByIDs(ctx, idsJSON(ids))
	if err != nil {
		return nil, 0, fmt.Errorf("read the runs of play %s: %w", playID, err)
	}
	runs := make([]plays.PlayRun, 0, len(rows))
	for _, row := range rows {
		t, err := toTrail(row.PlayTrail)
		if err != nil {
			return nil, 0, err
		}
		runs = append(runs, plays.PlayRun{Trail: *t, TargetKey: row.TargetKey, TargetTitle: row.TargetTitle, AutoPlayID: row.AutoPlayID})
	}
	return inOrder(ids, runs, func(run plays.PlayRun) string { return run.ID }), total, nil
}

func (r *PlaysRepo) RunHealth(ctx context.Context, playIDs []string, scope plays.RunScope, since time.Time) (out map[string]plays.RunHealth, err error) {
	out = make(map[string]plays.RunHealth, len(playIDs))
	if len(playIDs) == 0 {
		return out, nil
	}
	where, args := "pt.play_id IN (SELECT value FROM json_each(?))", []any{idsJSON(playIDs)}
	whereRunScope(func(cond string, condArgs ...any) {
		where += " AND " + cond
		args = append(args, condArgs...)
	}, scope)
	// hand-written: sqlc cannot express the scope's optional conditions.
	stmt := `WITH v AS (SELECT pt.play_id, pt.state, pt.started_at, pt.id FROM ` + playRunsFrom + ` WHERE ` + where + `)
SELECT play_id,
       SUM(started_at >= ?), SUM(started_at >= ? AND state = 'failed'), SUM(state = 'waiting'), SUM(state IN ('starting', 'running')),
       (SELECT l.state FROM v l WHERE l.play_id = v.play_id ORDER BY l.started_at DESC, l.id DESC LIMIT 1),
       MAX(started_at)
FROM v GROUP BY play_id`
	rows, err := r.db.QueryContext(ctx, stmt, append(args, since.Unix(), since.Unix())...)
	if err != nil {
		return nil, fmt.Errorf("read the health of %d plays: %w", len(playIDs), err)
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var (
			playID, state                     string
			runs, failed, waitingNow, running int
			last                              sql.NullInt64
		)
		if err := rows.Scan(&playID, &runs, &failed, &waitingNow, &running, &state, &last); err != nil {
			return nil, fmt.Errorf("scan play health: %w", err)
		}
		out[playID] = plays.RunHealth{
			LastRun:  &plays.LastRun{State: plays.TrailState(state), StartedAt: time.Unix(last.Int64, 0).UTC()},
			Runs7d:   runs,
			Failed7d: failed,
			Waiting:  waitingNow,
			Running:  running,
		}
	}
	return out, rows.Err()
}
