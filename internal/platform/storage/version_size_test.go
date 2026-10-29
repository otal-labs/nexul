package storage

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
)

func TestPlaysRepo_Update_StoresTheSizeAgainstThePreviousVersionAndAPruneClearsTheNewOldest(t *testing.T) {
	s, p := historyStore(t)
	save(t, s, p, p.CreatedAt.Add(time.Minute), plays.Revision{Instructions: &plays.NewVersion{Body: "First. Then more words."}})
	for i := 3; i <= 12; i++ {
		save(t, s, p, p.CreatedAt.Add(time.Duration(i)*time.Minute), plays.Revision{Instructions: &plays.NewVersion{Body: fmt.Sprintf("v%d only", i)}})
	}

	h, err := s.Plays.ListHistory(t.Context(), p.ID)
	require.NoError(t, err)
	require.Len(t, h.Versions, plays.KeptInstructionVersions)
	assert.Equal(t, "+1 -1", sizeLabel(h.Versions[0].WordsAdded, h.Versions[0].WordsRemoved), "v12 swaps one word of v11")
	oldest := h.Versions[9]
	assert.Equal(t, 3, oldest.Version)
	assert.Nil(t, oldest.WordsAdded, "version 3 lost its predecessor to the prune")
	assert.Nil(t, oldest.WordsRemoved)
}

func TestPlaysRepo_Update_TheFirstVersionHasNoSizeAndTheSecondIsSizedAgainstIt(t *testing.T) {
	s, p := historyStore(t)
	save(t, s, p, p.CreatedAt.Add(time.Minute), plays.Revision{Instructions: &plays.NewVersion{Body: "First. Then more words."}})

	h, err := s.Plays.ListHistory(t.Context(), p.ID)
	require.NoError(t, err)
	require.Len(t, h.Versions, 2)
	assert.Equal(t, "+3 -0", sizeLabel(h.Versions[0].WordsAdded, h.Versions[0].WordsRemoved))
	assert.Nil(t, h.Versions[1].WordsAdded)
	v, err := s.Plays.GetVersion(t.Context(), p.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, "+3 -0", sizeLabel(v.WordsAdded, v.WordsRemoved), "the version read carries its size too")
}

// TestMigration0142_VersionsAlreadyKeptGetTheirSizeAgainstTheirKeptPredecessor upgrades over each history table with
// gaps in its numbering: the oldest kept row of an item stays empty and the others are sized against the row below.
func TestMigration0142_VersionsAlreadyKeptGetTheirSizeAgainstTheirKeptPredecessor(t *testing.T) {
	db := migrateBefore(t, "0142")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-seed', 'Seed', 'seed', 1, 1);
INSERT INTO docs (id, title, body, version, created_at, updated_at) VALUES
    ('d-1', 'Plan', 'one two', 7, 1, 1), ('d-2', 'Other', 'solo', 1, 1, 1);
INSERT INTO doc_versions (doc_id, version, title, body, name, created_at) VALUES
    ('d-1', 3, 'Plan', 'one two', '', 10),
    ('d-1', 5, 'Plan', '{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"one two three"}]}]}', 'Draft', 20),
    ('d-1', 7, 'Plan v2', 'one three', '', 30),
    ('d-2', 1, 'Other', 'solo', '', 10);
INSERT INTO tickets (id, title, body, status, developer, created_at, updated_at) VALUES ('t-1', 'Fix login', '', 'open', 'alice', 1, 1);
INSERT INTO ticket_description_versions (ticket_id, version, body, created_at) VALUES ('t-1', 4, 'a b c', 10), ('t-1', 5, 'a x c d', 20);
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at) VALUES
    ('p-1', 'ws-seed', 'Play', 'doc', '', 'go now', 1, NULL, '[]', '', 'u-1', 1, 1);
INSERT INTO play_instruction_versions (play_id, version, body, created_at) VALUES ('p-1', 2, 'go', 10), ('p-1', 3, 'go now', 20);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0142 and every later migration apply on top, as an upgrade would")

	sizes := func(query string, args ...any) []string {
		rows, err := db.Query(query, args...)
		require.NoError(t, err)
		defer func() { require.NoError(t, rows.Close()) }()
		var out []string
		for rows.Next() {
			var added, removed sql.NullInt64
			require.NoError(t, rows.Scan(&added, &removed))
			out = append(out, fmt.Sprint(added, removed))
		}
		require.NoError(t, rows.Err())
		return out
	}
	none, plus := fmt.Sprint(sql.NullInt64{}, sql.NullInt64{}), func(a, r int64) string {
		return fmt.Sprint(sql.NullInt64{Int64: a, Valid: true}, sql.NullInt64{Int64: r, Valid: true})
	}
	assert.Equal(t, []string{none, plus(1, 0), plus(1, 1)},
		sizes(`SELECT words_added, words_removed FROM doc_versions WHERE doc_id = 'd-1' ORDER BY version`),
		"v5 adds a word to v3; v7 renames the title and drops one word from v5")
	assert.Equal(t, []string{none}, sizes(`SELECT words_added, words_removed FROM doc_versions WHERE doc_id = 'd-2'`))
	assert.Equal(t, []string{none, plus(2, 1)}, sizes(`SELECT words_added, words_removed FROM ticket_description_versions WHERE ticket_id = 't-1' ORDER BY version`))
	assert.Equal(t, []string{none, plus(1, 0)}, sizes(`SELECT words_added, words_removed FROM play_instruction_versions WHERE play_id = 'p-1' ORDER BY version`))
}

func sizeLabel(added, removed *int) string {
	if added == nil || removed == nil {
		return "none"
	}
	return fmt.Sprintf("+%d -%d", *added, *removed)
}
