package automations

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestVersionsHandler(t *testing.T) (http.Handler, string) {
	t.Helper()
	autoRepo := newFakeRepo()
	seedAutomation(t, autoRepo, "a1")
	versionsSvc := newTestVersionsService(autoRepo, newFakeVersionsRepo(), allowAll("owner"))
	h := NewHandler(newTestService(autoRepo, allowAll("owner"))).WithVersions(versionsSvc).Routes()
	return h, "a1"
}

func TestHandler_PushListDiffGetMergeRollback(t *testing.T) {
	h, id := newTestVersionsHandler(t)

	rec := serve(t, h, "owner", http.MethodPost, "/api/automations/"+id+"/versions", `{"code":"v1","message":"first"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	var v1 Version
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v1))
	assert.Equal(t, VersionPending, v1.Status)

	t.Run("list shows the pending version", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodGet, "/api/automations/"+id+"/versions", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []Version
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list, 1)
	})

	t.Run("get pulls the version's code", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodGet, "/api/automations/"+id+"/versions/"+v1.ID, "")
		require.Equal(t, http.StatusOK, rec.Code)
		var got Version
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, "v1", got.Code)
	})

	t.Run("diff before any merge has no active side", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodGet, "/api/automations/"+id+"/versions/diff", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var diff diffResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &diff))
		assert.Nil(t, diff.Active)
		require.NotNil(t, diff.Pending)
		assert.Equal(t, v1.ID, diff.Pending.ID)
	})

	t.Run("merge activates the pending version", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodPost, "/api/automations/"+id+"/versions/"+v1.ID+"/merge", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var merged Version
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &merged))
		assert.Equal(t, VersionActive, merged.Status)
	})

	t.Run("a second push and merge, then rollback to v1", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodPost, "/api/automations/"+id+"/versions", `{"code":"v2"}`)
		require.Equal(t, http.StatusCreated, rec.Code)
		var v2 Version
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v2))
		rec = serve(t, h, "owner", http.MethodPost, "/api/automations/"+id+"/versions/"+v2.ID+"/merge", "")
		require.Equal(t, http.StatusOK, rec.Code)

		rec = serve(t, h, "owner", http.MethodPost, "/api/automations/"+id+"/versions/"+v1.ID+"/rollback", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var rolledBack Version
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rolledBack))
		assert.Equal(t, v1.ID, rolledBack.ID)
		assert.Equal(t, VersionActive, rolledBack.Status)
	})

	t.Run("no actor is 401", func(t *testing.T) {
		rec := serve(t, h, "", http.MethodGet, "/api/automations/"+id+"/versions", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("non-privileged actor is 403 on push", func(t *testing.T) {
		rec := serve(t, h, "alice", http.MethodPost, "/api/automations/"+id+"/versions", `{"code":"v3"}`)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("pushing to an unknown automation is 404", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodPost, "/api/automations/missing/versions", `{"code":"v1"}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("malformed push body is 400", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodPost, "/api/automations/"+id+"/versions", `{not json`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("getting an unknown version is 404", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodGet, "/api/automations/"+id+"/versions/missing", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("diff on an unknown automation is 404", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodGet, "/api/automations/missing/versions/diff", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("merging an unknown version is 404", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodPost, "/api/automations/"+id+"/versions/missing/merge", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("rolling back to an unknown version is 404", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodPost, "/api/automations/"+id+"/versions/missing/rollback", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("listing versions for an unknown automation is 404", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodGet, "/api/automations/missing/versions", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
