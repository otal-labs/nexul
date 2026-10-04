package storage

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TestMigration0066_ClarificationsLiveAndDieWithTheirDoc upgrades a database holding a doc, then stores a round with
// its questions: the last pending answer writes its event once, a removed round takes its questions, and deleting
// the doc takes the rest.
func TestMigration0066_ClarificationsLiveAndDieWithTheirDoc(t *testing.T) {
	db := migrateBefore(t, "0066")
	_, err := db.Exec(`INSERT INTO docs (id, title, body, version, created_at, updated_at) VALUES ('d-spec', 'Spec', '{}', 1, 1, 1)`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0066 applies on top, as an upgrade would")

	ctx := t.Context()
	repo := New(db, testEncKey).Docs
	at := time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC)
	round := &docs.ClarificationRound{DocID: "d-spec", Round: 1, StartedBy: "u-dev", TrailID: "trail-1", StartedAt: at, Running: true, TookLock: true}
	require.NoError(t, repo.CreateClarificationRound(ctx, round))
	require.ErrorIs(t, repo.CreateClarificationRound(ctx, round), apperrs.ErrConflict, "a doc numbers each round once")
	round.Running, round.NoGapsAt = false, nil
	questions := []*docs.ClarificationQuestion{
		{ID: "q-1", DocID: "d-spec", Round: 1, Position: 1, Question: "Who signs in?", Options: []docs.QuestionOption{{Label: "Staff"}}, MultiSelect: true},
		{ID: "q-2", DocID: "d-spec", Round: 1, Position: 2, Question: "Which devices?"},
	}
	require.NoError(t, repo.SaveClarification(ctx, []*docs.ClarificationRound{round}, questions))

	rounds, err := repo.ListClarificationRounds(ctx, "d-spec")
	require.NoError(t, err)
	require.Len(t, rounds, 1)
	assert.False(t, rounds[0].Running)
	assert.True(t, rounds[0].TookLock)
	assert.Nil(t, rounds[0].ClosedAt)
	got, err := repo.GetClarificationQuestion(ctx, "q-1")
	require.NoError(t, err)
	assert.True(t, got.Pending())
	assert.Equal(t, []docs.QuestionOption{{Label: "Staff"}}, got.Options)

	answered := eventbus.OutboxEvent{ID: "evt-answered", Topic: docs.TopicClarificationRoundAnswered, Payload: map[string]string{}}
	got.Selected, got.AnsweredBy, got.AnsweredAt = []string{"Staff"}, "u-client", &at
	require.NoError(t, repo.SaveClarificationAnswer(ctx, got, &answered))
	assert.Zero(t, topicCount(t, db, docs.TopicClarificationRoundAnswered), "q-2 is still pending")
	second, err := repo.GetClarificationQuestion(ctx, "q-2")
	require.NoError(t, err)
	second.Skipped = true
	answered.ID = "evt-answered-2"
	require.NoError(t, repo.SaveClarificationAnswer(ctx, second, &answered))
	assert.Equal(t, 1, topicCount(t, db, docs.TopicClarificationRoundAnswered), "the last pending answer writes it")

	got, err = repo.GetClarificationQuestion(ctx, "q-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"Staff"}, got.Selected)
	assert.Equal(t, at, *got.AnsweredAt)

	require.NoError(t, repo.CreateClarificationRound(ctx, &docs.ClarificationRound{DocID: "d-spec", Round: 2, StartedBy: "u-dev", StartedAt: at}))
	require.NoError(t, repo.SaveClarification(ctx, nil, []*docs.ClarificationQuestion{{ID: "q-3", DocID: "d-spec", Round: 2, Position: 1, Question: "Who signs in?"}}))
	require.NoError(t, repo.DeleteClarificationRound(ctx, "d-spec", 2))
	_, err = repo.GetClarificationQuestion(ctx, "q-3")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a removed round takes its questions")
	require.ErrorIs(t, repo.DeleteClarificationRound(ctx, "d-spec", 2), apperrs.ErrNotFound)

	_, err = db.Exec(`DELETE FROM docs WHERE id = 'd-spec'`)
	require.NoError(t, err)
	rounds, err = repo.ListClarificationRounds(ctx, "d-spec")
	require.NoError(t, err)
	assert.Empty(t, rounds, "deleting the doc deletes its clarification")
	qs, err := repo.ListClarificationQuestions(ctx, "d-spec")
	require.NoError(t, err)
	assert.Empty(t, qs)
}

func topicCount(t *testing.T, db *sql.DB, topic string) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM outbox WHERE topic = ?`, topic).Scan(&n))
	return n
}
