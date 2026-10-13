package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

var updateAccessMatrix = flag.Bool("update-access-matrix", false, "rewrite the access matrix golden file from the list parity fixture")

const accessMatrixGolden = "testdata/access_matrix.json"

// matrixRow is what one member holds on one entity: each catalog action the check lets through, plus "open" for a
// project they may open and channels:read for a channel they may read.
type matrixRow struct {
	User   string   `json:"user"`
	Entity string   `json:"entity"`
	Held   []string `json:"held"`
}

// accessMatrix is every member of each workspace × every catalog action × the workspace and each of its projects,
// docs, plays and channels, and every member × each instance area, answered by the checks enforcement uses, so two
// runs over one database diff to exactly the access that changed between them.
func accessMatrix(t *testing.T, s *coreServices, db *sql.DB) []matrixRow {
	t.Helper()
	members := map[string][]string{}
	scan(t, db, `SELECT workspace_id, user_id FROM workspace_members`, func(workspaceID, userID string) {
		members[workspaceID] = append(members[workspaceID], userID)
	})
	type entity struct{ kind, id, workspaceID string }
	var entities []entity
	scan(t, db, `SELECT 'workspace', id FROM workspaces`, func(kind, id string) { entities = append(entities, entity{kind, id, id}) })
	for _, q := range []struct{ kind, query string }{
		{"project", `SELECT id, workspace_id FROM projects`},
		{"doc", `SELECT d.id, p.workspace_id FROM docs d JOIN projects p ON p.id = d.project_id`},
		{"play", `SELECT id, workspace_id FROM plays`},
		{"channel", `SELECT id, workspace_id FROM conversations WHERE kind IN ('` + string(chat.KindChannel) + `', '` + string(chat.KindVoiceChannel) + `')`},
	} {
		scan(t, db, q.query, func(id, workspaceID string) { entities = append(entities, entity{q.kind, id, workspaceID}) })
	}
	a := s.accessSvc
	var rows []matrixRow
	for _, e := range entities {
		for _, user := range members[e.workspaceID] {
			ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: user}), a.NewMemo())
			held := heldOn(ctx, s, user, e.kind, e.id, e.workspaceID)
			rows = append(rows, matrixRow{User: user, Entity: e.kind + ":" + e.id, Held: held})
		}
	}
	people := map[string]bool{}
	for _, users := range members {
		for _, user := range users {
			people[user] = true
		}
	}
	for user := range people {
		ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: user}), a.NewMemo())
		for _, area := range a.InstanceAreas() {
			held := []string{}
			for _, action := range area.Actions {
				if ok, err := a.HoldsArea(ctx, user, area.Key, action); err == nil && ok {
					held = append(held, string(action))
				}
			}
			rows = append(rows, matrixRow{User: user, Entity: "instance_area:" + area.Key, Held: held})
		}
	}
	return rows
}

// heldOn is what user holds on one entity, through the check each kind of entity is enforced with.
func heldOn(ctx context.Context, s *coreServices, user, kind, id, workspaceID string) []string {
	a := s.accessSvc
	held := []string{}
	if kind == "channel" {
		if _, err := s.chatSvc.GetConversation(ctx, id); err == nil {
			held = append(held, "channels:read")
		}
		return held
	}
	if kind == "project" && a.CanInProject(ctx, user, id, permissions.Member) {
		held = append(held, "open")
	}
	for _, action := range permissions.AllActions() {
		var ok bool
		switch kind {
		case "workspace":
			ok = a.HasPermission(ctx, user, workspaceID, action, "", "")
		case "project":
			ok = a.CanInProject(ctx, user, id, action)
		case "doc":
			ok, _ = a.Can(ctx, user, id, action)
		case "play":
			ok = a.HasPermission(ctx, user, workspaceID, action, "play", id)
		}
		if ok {
			held = append(held, string(action))
		}
	}
	return held
}

func scan(t *testing.T, db *sql.DB, query string, row func(a, b string)) {
	t.Helper()
	rows, err := db.Query(query)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	for rows.Next() {
		var a, b string
		require.NoError(t, rows.Scan(&a, &b))
		row(a, b)
	}
	require.NoError(t, rows.Err())
}

// marshalMatrix is the matrix as a JSON array sorted by person and entity, one row per line so a diff names each.
func marshalMatrix(t *testing.T, rows []matrixRow) []byte {
	t.Helper()
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		line, err := json.Marshal(r)
		require.NoError(t, err)
		lines = append(lines, string(line))
	}
	slices.Sort(lines)
	return []byte("[\n" + strings.Join(lines, ",\n") + "\n]\n")
}

// TestAccessMatrix_OfTheListParityFixture pins what every member holds on everything in the list parity fixture, so a
// change to the evaluator that moves an answer shows up as a diff of the golden file.
func TestAccessMatrix_OfTheListParityFixture(t *testing.T) {
	db := mentionsTestDB(t)
	svc, store := wiredOver(t, db)
	withListParityRows(t, seedPermFixture(t, db, svc, store))
	rows := accessMatrix(t, svc, db)
	names := map[string]string{}
	// Channels and some seeded plays take random ids, so the golden file names them.
	scan(t, db, `SELECT id, '#' || name FROM conversations UNION ALL SELECT id, label FROM plays`, func(id, name string) { names[id] = name })
	for i, r := range rows {
		kind, id, _ := strings.Cut(r.Entity, ":")
		if name, ok := names[id]; ok {
			rows[i].Entity = kind + ":" + name
		}
	}
	got := marshalMatrix(t, rows)

	if *updateAccessMatrix {
		require.NoError(t, os.MkdirAll(filepath.Dir(accessMatrixGolden), 0o755))
		require.NoError(t, os.WriteFile(accessMatrixGolden, got, 0o644))
	}
	want, err := os.ReadFile(accessMatrixGolden)
	require.NoError(t, err, "go test ./server/cmd -run TestAccessMatrix_OfTheListParityFixture -update-access-matrix writes it")
	assert.Equal(t, string(want), string(got))
}

// TestAccessMatrix_OfACopy writes the matrix of the database copy NEXUL_ACCESS_MATRIX_DB names, migrated in a
// temporary copy of its own, to NEXUL_ACCESS_MATRIX_OUT (the copy's path plus .matrix.json by default). Run it on the
// base and the head of a change and diff the two files; before the switch every diff is empty.
func TestAccessMatrix_OfACopy(t *testing.T) {
	source := os.Getenv("NEXUL_ACCESS_MATRIX_DB")
	if source == "" {
		t.Skip("NEXUL_ACCESS_MATRIX_DB names no database copy")
	}
	out := os.Getenv("NEXUL_ACCESS_MATRIX_OUT")
	if out == "" {
		out = source + ".matrix.json"
	}
	path := filepath.Join(t.TempDir(), "copy.db")
	copyFile(t, source, path)
	db, err := storage.OpenDB(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, storage.Migrate(db))
	svc, _ := wiredOver(t, db)

	require.NoError(t, os.WriteFile(out, marshalMatrix(t, accessMatrix(t, svc, db)), 0o644))
	t.Logf("access matrix written to %s", out)
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	in, err := os.Open(from)
	require.NoError(t, err)
	defer func() { require.NoError(t, in.Close()) }()
	dst, err := os.Create(to)
	require.NoError(t, err)
	_, err = io.Copy(dst, in)
	require.NoError(t, err)
	require.NoError(t, dst.Close())
}
