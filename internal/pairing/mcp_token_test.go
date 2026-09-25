package pairing

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newTokenService(t *testing.T) (*Service, *fakeRepo, *fakeTokens) {
	t.Helper()
	repo := newFakeRepo()
	repo.computers["c1"] = Computer{ID: "c1", UserID: "u1", Name: "Laptop"}
	tokens := newFakeTokens()
	svc := NewService(Config{Repo: repo, Harnesses: registry(&fakeExchanger{}), EncryptionKey: testEncKey, Tokens: tokens, Now: func() time.Time { return testNow }})
	return svc, repo, tokens
}

func TestMCPToken_ErrorPaths(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTokenService(t)
	ctx := t.Context()

	_, err := svc.MintMCPToken(ctx, "", "c1")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	_, err = svc.MintMCPToken(ctx, "u1", " ")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = svc.MintMCPToken(ctx, "u2", "c1")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "another user's computer is never minted a token")
	_, err = svc.GetMCPToken(ctx, "u2", "c1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.ErrorIs(t, svc.RevokeMCPToken(ctx, "u2", "c1"), apperrs.ErrNotFound)

	unwired := NewService(Config{Repo: svc.repo})
	_, err = unwired.MintMCPToken(ctx, "u1", "c1")
	require.ErrorIs(t, err, apperrs.ErrFatal)
	_, err = unwired.GetMCPToken(ctx, "u1", "c1")
	require.ErrorIs(t, err, apperrs.ErrFatal)
	require.ErrorIs(t, unwired.RevokeMCPToken(ctx, "u1", "c1"), apperrs.ErrFatal)
}

func TestMCPToken_MintReplacesAndRevokeRemoves(t *testing.T) {
	t.Parallel()
	svc, _, tokens := newTokenService(t)
	ctx := t.Context()

	none, err := svc.GetMCPToken(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.Nil(t, none)

	first, err := svc.MintMCPToken(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, "Nexul MCP on Laptop", first.Name)
	assert.NotEmpty(t, first.Token)

	second, err := svc.MintMCPToken(ctx, "u1", "c1")
	require.NoError(t, err)
	got, err := svc.GetMCPToken(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, second.ID, got.ID)
	assert.Equal(t, []string{first.ID}, tokens.revoked)

	require.NoError(t, svc.RevokeMCPToken(ctx, "u1", "c1"))
	got, err = svc.GetMCPToken(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.Nil(t, got)

	tokens.mintErr = errors.New("store down")
	_, err = svc.MintMCPToken(ctx, "u1", "c1")
	require.ErrorContains(t, err, "store down")
}

func TestUnconfirmSetup_RevokesTheMCPTokenFirst(t *testing.T) {
	t.Parallel()
	svc, repo, tokens := newTokenService(t)
	ctx := t.Context()
	_, err := svc.ConfirmSetup(ctx, "u1", "c1")
	require.NoError(t, err)
	minted, err := svc.MintMCPToken(ctx, "u1", "c1")
	require.NoError(t, err)

	tokens.revokeErr = errors.New("store down")
	_, err = svc.UnconfirmSetup(ctx, "u1", "c1")
	require.ErrorContains(t, err, "store down")
	assert.NotNil(t, repo.computers["c1"].SetupConfirmedAt, "a failed revoke keeps the confirmation, so un-confirming again retries it")

	tokens.revokeErr = nil
	setup, err := svc.UnconfirmSetup(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.Nil(t, setup.ConfirmedAt)
	assert.Equal(t, []string{minted.ID}, tokens.revoked)
}

func TestDeleteComputer_RevokesTheMCPTokenFirst(t *testing.T) {
	t.Parallel()
	svc, repo, tokens := newTokenService(t)
	ctx := t.Context()
	minted, err := svc.MintMCPToken(ctx, "u1", "c1")
	require.NoError(t, err)

	tokens.revokeErr = errors.New("store down")
	require.ErrorContains(t, svc.DeleteComputer(ctx, "u1", "c1"), "store down")
	assert.Contains(t, repo.computers, "c1", "a failed revoke keeps the computer, so removing it again retries")

	tokens.revokeErr = nil
	require.NoError(t, svc.DeleteComputer(ctx, "u1", "c1"))
	assert.NotContains(t, repo.computers, "c1")
	assert.Equal(t, []string{minted.ID}, tokens.revoked)
}

func TestHandler_MCPToken(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTokenService(t)
	routes := NewHandler(svc).Routes()

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers/c1/mcp-token", "u1", nil)
	require.Equal(t, http.StatusCreated, rec.Code)
	var minted MintedMCPToken
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &minted))
	assert.NotEmpty(t, minted.Token)

	rec = doRequest(routes, http.MethodGet, "/api/pairing/computers/c1/mcp-token", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), minted.Token)
	assert.Contains(t, rec.Body.String(), minted.ID)

	rec = doRequest(routes, http.MethodDelete, "/api/pairing/computers/c1/mcp-token", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	rec = doRequest(routes, http.MethodGet, "/api/pairing/computers/c1/mcp-token", "u1", nil)
	assert.JSONEq(t, `{"mcp_token": null}`, rec.Body.String())

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		rec = doRequest(routes, method, "/api/pairing/computers/c1/mcp-token", "u2", nil)
		assert.Equal(t, http.StatusNotFound, rec.Code, method)
	}
}
