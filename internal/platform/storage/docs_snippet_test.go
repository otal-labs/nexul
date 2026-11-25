package storage

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/docs/richtext"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

// snippetBodies are the shapes a list snippet is built from: headings skipped, runs joined, a long line cut, markdown.
var snippetBodies = map[string]string{
	"heading then paragraph": `{"type":"doc","content":[{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Plan"}]},{"type":"paragraph","content":[{"type":"text","text":"Ship the "},{"type":"text","marks":[{"type":"bold"}],"text":"list"},{"type":"text","text":" first."}]}]}`,
	"only headings":          `{"type":"doc","content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Just a title"}]}]}`,
	"mention and break":      `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"mention","attrs":{"label":"@alice"}},{"type":"hardBreak"},{"type":"text","text":"to review"}]}]}`,
	"bullet list":            `{"type":"doc","content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"one"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"two"}]}]}]}]}`,
	"long multibyte line":    `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"` + strings.Repeat("é", 300) + `"}]}]}`,
	"markdown":               "# Runbook\n- restart the runner",
	"empty":                  "",
}

func TestDocsRepo_ListByProject_SnippetMatchesTheBuilderOnEveryWrite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	writes := map[string]func(d *docs.Doc, body string) error{
		"create": func(d *docs.Doc, body string) error { d.Body = body; return s.Docs.Create(ctx, d) },
		"update": func(d *docs.Doc, body string) error {
			if err := s.Docs.Create(ctx, d); err != nil {
				return err
			}
			d.Body = body
			return s.Docs.Update(ctx, d, "", "")
		},
		"commit": func(d *docs.Doc, body string) error {
			if err := s.Docs.Create(ctx, d); err != nil {
				return err
			}
			d.Body = body
			return s.Docs.CommitBody(ctx, d, "")
		},
	}
	want := map[string]string{}
	for write, apply := range writes {
		for shape, body := range snippetBodies {
			id := write + "/" + shape
			require.NoError(t, apply(newTestDoc(id), body), id)
			want[id] = richtext.Snippet(body, docs.SnippetRunes)
		}
	}

	listed, err := s.Docs.ListByProject(ctx, "project-general")
	require.NoError(t, err)
	got := map[string]string{}
	for _, d := range listed {
		got[d.ID] = d.Snippet
		assert.Empty(t, d.Body, "%s: the list reads no body", d.ID)
	}
	assert.Equal(t, want, got)
}

// TestDocsRepo_ListByProject_ReadsSnippetsNotBodies counts the text one project list reads at 1,000 docs of 20 KB, about
// 40 MB were it to read each body and its search text, and only the short columns as it should.
func TestDocsRepo_ListByProject_ReadsSnippetsNotBodies(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := newTestDB(t)
	const n, size = 1000, 20 << 10
	seedLargeDocs(t, db, n, size)

	rows, err := sqlcgen.New(db).ListDocsByProject(ctx, sql.NullString{String: "project-general", Valid: true})
	require.NoError(t, err)
	require.Len(t, rows, n)
	read := 0
	for _, row := range rows {
		read += textBytes(reflect.ValueOf(row))
	}
	t.Logf("text read by one list of %d docs of %d bytes: %d bytes", n, size, read)
	assert.Less(t, read, n*(4*docs.SnippetRunes+200), "the list reads each doc's short columns, never its body")
}

// seedLargeDocs writes n docs whose body and search text are each about size bytes, as a save would.
func seedLargeDocs(t *testing.T, db *sql.DB, n, size int) {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	word := "storage "
	text := strings.Repeat(word, size/len(word))
	body := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"` + text + `"}]}]}`
	for i := range n {
		_, err := tx.Exec(`INSERT INTO docs (id, project_id, folder_id, title, body, body_md, snippet, version, created_at, updated_at)
			VALUES (?, 'project-general', '', ?, ?, ?, ?, 1, ?, ?)`,
			fmt.Sprintf("doc-%04d", i), fmt.Sprintf("Doc %d", i), body, richtext.SearchText(body), docSnippet(body), i, i)
		require.NoError(t, err)
	}
	require.NoError(t, tx.Commit())
}

// textBytes sums the strings a row holds, nullable ones included, so a column added to the list counts.
func textBytes(v reflect.Value) int {
	total := 0
	for i := range v.NumField() {
		f := v.Field(i)
		if f.Kind() == reflect.String {
			total += f.Len()
		}
		if ns, ok := f.Interface().(sql.NullString); ok {
			total += len(ns.String)
		}
	}
	return total
}
