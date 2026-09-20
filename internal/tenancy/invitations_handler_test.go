package tenancy

import (
	"bytes"
	"encoding/json"
	"errors"
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

func TestInvitationHandler_Create_OmittedLifetimeDefaultsToSevenDays(t *testing.T) {
	t.Parallel()
	svc, repo := newInvitationServiceFixture()
	h := NewInvitationHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/invitations", bytes.NewBufferString(`{"grants":[{"workspace_id":"ws-1","role_id":"role-editor"}]}`)).WithContext(WithUserID(t.Context(), "actor"))
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Len(t, repo.created, 1)
	assert.Equal(t, 7*24*time.Hour, repo.created[0].ExpiresAt.Sub(repo.created[0].CreatedAt))
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

func TestInvitationHandler_DecodeAndServiceErrors_ReturnJSONErrors(t *testing.T) {
	t.Parallel()
	svc, repo := newInvitationServiceFixture()
	h := NewInvitationHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/invitations", bytes.NewBufferString("{"))
	req = req.WithContext(WithUserID(req.Context(), "actor"))
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	repo.listErr = errors.New("list failed")
	req = httptest.NewRequest(http.MethodGet, "/api/invitations", nil).WithContext(WithUserID(t.Context(), "actor"))
	rec = httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	repo.revokeErr = errors.New("revoke failed")
	req = httptest.NewRequest(http.MethodDelete, "/api/invitations/inv-1", nil).WithContext(WithUserID(t.Context(), "actor"))
	rec = httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
