package tenancy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvitationHandler_CreateAndList_UseInvitationService(t *testing.T) {
	t.Parallel()
	svc, _ := newInvitationServiceFixture()
	h := NewInvitationHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/invitations", bytes.NewBufferString(`{"grants":[{"workspace_id":"ws-1","role_id":"role-editor"}],"expires_in_days":7}`))
	req = req.WithContext(WithUserID(req.Context(), "actor"))
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), "/invite#")

	req = httptest.NewRequest(http.MethodGet, "/api/invitations", nil).WithContext(WithUserID(t.Context(), "actor"))
	rec = httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestInvitationHandler_Preview_ReturnsMetadataOnly(t *testing.T) {
	t.Parallel()
	svc, repo := newInvitationServiceFixture()
	raw := "4d2f3f29-2a43-4ae7-b2d4-0b6f1a7f4c44"
	hash, err := HashInvitationToken(raw)
	require.NoError(t, err)
	repo.byToken[hash] = &Invitation{ID: "inv-1", ExpiresAt: time.Now().Add(time.Hour), Grants: []*InvitationGrant{{WorkspaceID: "ws-1", WorkspaceName: "Acme", RoleID: "role-editor", RoleName: "Editor", Allow: nil}}}
	h := NewInvitationHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/invitations/preview", bytes.NewBufferString(`{"token":"`+raw+`"}`))
	rec := httptest.NewRecorder()
	h.PublicRoutes().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var body InvitationPreview
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Grants, 1)
	assert.Equal(t, "Acme", body.Grants[0].WorkspaceName)
}
