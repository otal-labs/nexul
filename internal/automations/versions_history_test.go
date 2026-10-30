package automations

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_History_ListsEveryVersionInTheSharedShapeWithoutCode(t *testing.T) {
	h, id := newTestVersionsHandler(t)
	push := func(body string) Version {
		rec := serve(t, h, "owner", http.MethodPost, "/api/automations/"+id+"/versions", body)
		require.Equal(t, http.StatusCreated, rec.Code)
		var v Version
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v))
		return v
	}
	first := push(`{"code":"one\ntwo","message":"first cut"}`)
	require.Equal(t, http.StatusOK, serve(t, h, "owner", http.MethodPost, "/api/automations/"+id+"/versions/"+first.ID+"/merge", "").Code)
	push(`{"code":"one\nthree"}`)

	rec := serve(t, h, "owner", http.MethodGet, "/api/automations/"+id+"/history", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var got struct {
		Versions []map[string]any `json:"versions"`
		Changes  []any            `json:"changes"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.NotNil(t, got.Changes)
	assert.Empty(t, got.Changes)
	require.Len(t, got.Versions, 2)

	pending, active := got.Versions[0], got.Versions[1]
	assert.EqualValues(t, 2, pending["version"], "newest first, the sequence is the version number")
	assert.Equal(t, "pending", pending["status"])
	assert.Equal(t, false, pending["current"])
	assert.Equal(t, "", pending["name"])
	assert.EqualValues(t, 1, active["version"])
	assert.Equal(t, "active", active["status"])
	assert.Equal(t, true, active["current"], "the active version is the current one")
	assert.Equal(t, "first cut", active["name"], "the push message is the version's name")
	assert.Equal(t, []any{"owner"}, active["editor_ids"])
	assert.Nil(t, active["words_added"])
	assert.NotContains(t, active, "code")
	assert.NotContains(t, active, "body")

	t.Run("one version by its sequence carries its code as the body", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodGet, "/api/automations/"+id+"/versions/2", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var v map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v))
		assert.Equal(t, "one\nthree", v["body"])
		assert.EqualValues(t, 2, v["version"])
	})

	t.Run("a sequence the automation never had is 404", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, serve(t, h, "owner", http.MethodGet, "/api/automations/"+id+"/versions/9", "").Code)
	})

	t.Run("reading the history needs automations:read", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, serve(t, h, "alice", http.MethodGet, "/api/automations/"+id+"/history", "").Code)
		assert.Equal(t, http.StatusForbidden, serve(t, h, "alice", http.MethodGet, "/api/automations/"+id+"/versions/1", "").Code)
	})
}

func TestVersionsService_History_CreditsASeededVersionToNexulNotAPerson(t *testing.T) {
	autoRepo := newFakeRepo()
	seedAutomation(t, autoRepo, "a1")
	s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), allowAll("owner"))
	_, err := s.SeedVersion(context.Background(), "a1", "code", "shipped")
	require.NoError(t, err)

	got, err := s.History(context.Background(), "owner", "a1")
	require.NoError(t, err)
	require.Len(t, got.Versions, 1)
	assert.Empty(t, got.Versions[0].EditorIDs)
	assert.Equal(t, "Nexul", got.Versions[0].Via)
}
