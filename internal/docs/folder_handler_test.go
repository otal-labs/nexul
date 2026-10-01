package docs

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsHandler_FolderRoundTrip(t *testing.T) {
	h, repo := newDocsHandler()

	rec := serve(t, h, http.MethodPost, "/api/docs/folders", `{"project_id":"project-1","name":"GetSource"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	var folder Folder
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &folder))

	rec = serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","folder_id":"`+folder.ID+`","title":"EP01","body":""}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	doc := decodeDoc(t, rec)
	assert.Equal(t, folder.ID, doc.FolderID, "a doc is created in the folder the request names")

	rec = serve(t, h, http.MethodGet, "/api/docs/folders?project_id=project-1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var folders []Folder
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &folders))
	assert.Equal(t, []string{"Main", "GetSource"}, folderNames(ptrs(folders)))

	rec = serve(t, h, http.MethodPut, "/api/docs/folders/"+folder.ID, `{"name":"Episodes"}`)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = serve(t, h, http.MethodPost, "/api/docs/"+doc.ID+"/move", `{"folder_id":"main-project-1"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "main-project-1", decodeDoc(t, rec).FolderID)

	rec = serve(t, h, http.MethodDelete, "/api/docs/folders/main-project-1", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, "the default folder is never deleted")

	rec = serve(t, h, http.MethodDelete, "/api/docs/folders/"+folder.ID, "")
	require.Equal(t, http.StatusNoContent, rec.Code)
	_, err := repo.GetFolder(t.Context(), folder.ID)
	require.Error(t, err)
}

func TestDocsHandler_FolderErrors(t *testing.T) {
	h, _ := newDocsHandler()
	for name, tt := range map[string]struct {
		method, path, body string
		want               int
	}{
		"listing needs a project":     {http.MethodGet, "/api/docs/folders", "", http.StatusBadRequest},
		"a blank name is 400":         {http.MethodPost, "/api/docs/folders", `{"project_id":"project-1","name":" "}`, http.StatusBadRequest},
		"a malformed create is 400":   {http.MethodPost, "/api/docs/folders", `{"name":`, http.StatusBadRequest},
		"a malformed rename is 400":   {http.MethodPut, "/api/docs/folders/main-project-1", `{"name":`, http.StatusBadRequest},
		"renaming nothing is 404":     {http.MethodPut, "/api/docs/folders/nope", `{"name":"x"}`, http.StatusNotFound},
		"a malformed move is 400":     {http.MethodPost, "/api/docs/doc-1/move", `{"folder_id":`, http.StatusBadRequest},
		"moving a missing doc is 404": {http.MethodPost, "/api/docs/nope/move", `{"folder_id":"main-project-1"}`, http.StatusNotFound},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, serve(t, h, tt.method, tt.path, tt.body).Code)
		})
	}
}

func ptrs(folders []Folder) []*Folder {
	out := make([]*Folder, len(folders))
	for i := range folders {
		out[i] = &folders[i]
	}
	return out
}
