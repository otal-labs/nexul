package pairing

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func pairedExchanger() *fakeExchanger {
	return &fakeExchanger{result: harness.PairResult{BearerToken: "bearer", ExpiresIn: 30 * 24 * time.Hour}, version: "0.0.40"}
}

func TestService_PairComputer_FillsTheTunnelComputersSessionOverItsHostname(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	exch := pairedExchanger()
	svc, changed := newTunnelService(repo, exch, &fakeTunnels{})
	id := tunnelComputer(t, svc)
	before, err := repo.GetComputer(t.Context(), "u1", id)
	require.NoError(t, err)
	require.False(t, before.Paired())

	c, err := svc.PairComputer(t.Context(), "u1", id, " t3-pair-token ")
	require.NoError(t, err)
	assert.Equal(t, "https://laptop-ab12cd34.example.com", exch.pairedURL)
	assert.Equal(t, id, c.ID)
	assert.True(t, c.Paired())
	assert.Equal(t, "0.0.40", c.HarnessVersion)
	assert.Equal(t, before.CreatedAt, c.CreatedAt)
	require.NotNil(t, c.Tunnel)
	assert.Empty(t, c.BearerToken)

	all, err := repo.ListComputers(t.Context(), "u1")
	require.NoError(t, err)
	require.Len(t, all, 1, "pairing fills in the Connect step's row, never a second one")
	assert.NotEmpty(t, all[0].BearerToken)
	assert.Equal(t, []string{"u1", "u1"}, *changed)

	last := repo.outbox[len(repo.outbox)-1]
	assert.Equal(t, TopicComputerPaired, last.Topic)
	assert.Equal(t, ComputerPairedEvent{
		ComputerID: id, UserID: "u1", ServerURL: "https://laptop-ab12cd34.example.com", HarnessVersion: "0.0.40", TokenExpiresAt: c.TokenExpiresAt,
	}, last.Payload)
}

func TestService_PairComputer_PairsAURLComputerAtItsServerURL(t *testing.T) {
	t.Parallel()
	exch := pairedExchanger()
	svc := newTestService(newFakeRepo(), exch)
	c, err := svc.Pair(t.Context(), "u1", harness.KindT3Code, "VPS", "https://vps.example.com", "tok1")
	require.NoError(t, err)

	_, err = svc.PairComputer(t.Context(), "u1", c.ID, "tok2")
	require.NoError(t, err)
	assert.Equal(t, "https://vps.example.com", exch.pairedURL)
}

func TestService_PairComputer_BlamesTheFieldThatCausedTheFailure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		token     string
		exchErr   error
		wantField string
		wantIs    error
	}{
		{"blank token", "  ", nil, "token", apperrs.ErrInvalid},
		{"harness refused the token", "tok", apperrs.ErrInvalid, "token", apperrs.ErrInvalid},
		{"harness unreachable", "tok", apperrs.Retryable(errBoom), "server_url", apperrs.ErrRetryable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newFakeRepo()
			exch := pairedExchanger()
			exch.exchangeErr = tt.exchErr
			svc, _ := newTunnelService(repo, exch, &fakeTunnels{})
			id := tunnelComputer(t, svc)

			_, err := svc.PairComputer(t.Context(), "u1", id, tt.token)
			var fe *FieldError
			require.ErrorAs(t, err, &fe)
			assert.Equal(t, tt.wantField, fe.Field)
			require.ErrorIs(t, err, tt.wantIs)
			stored, err := repo.GetComputer(t.Context(), "u1", id)
			require.NoError(t, err)
			assert.False(t, stored.Paired(), "a failed pairing leaves the row still pairing")
		})
	}
}

func TestService_PairComputer_OnlyTheOwnersComputer(t *testing.T) {
	t.Parallel()
	svc, _ := newTunnelService(newFakeRepo(), pairedExchanger(), &fakeTunnels{})
	id := tunnelComputer(t, svc)

	_, err := svc.PairComputer(t.Context(), "u2", id, "tok")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = svc.PairComputer(t.Context(), "", id, "tok")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	_, err = svc.PairComputer(t.Context(), "u1", " ", "tok")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestService_Repair_TunnelComputerStaysOnItsHostname(t *testing.T) {
	t.Parallel()
	exch := pairedExchanger()
	svc, _ := newTunnelService(newFakeRepo(), exch, &fakeTunnels{})
	id := tunnelComputer(t, svc)

	_, err := svc.Repair(t.Context(), "u1", id, "Laptop", "https://elsewhere.example.com", "tok")
	var fe *FieldError
	require.ErrorAs(t, err, &fe)
	assert.Equal(t, "server_url", fe.Field)
	assert.Empty(t, exch.pairedURL, "nothing is dialled at the wrong address")
}

func TestService_ComputerStillPairing_ReadsAsUnpairedNotExpired(t *testing.T) {
	t.Parallel()
	svc, _ := newTunnelService(newFakeRepo(), pairedExchanger(), &fakeTunnels{})
	id := tunnelComputer(t, svc)

	_, err := svc.ResolveTarget(t.Context(), "u1", "")
	var nc *NotConfiguredError
	require.ErrorAs(t, err, &nc)
	assert.Equal(t, ReasonUnpaired, nc.Reason)

	_, err = svc.ListProjects(t.Context(), "u1", id)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "still pairing")
}

func TestHandler_PairComputer(t *testing.T) {
	t.Parallel()
	exch := pairedExchanger()
	svc, _ := newTunnelService(newFakeRepo(), exch, &fakeTunnels{})
	id := tunnelComputer(t, svc)
	routes := NewHandler(svc).Routes()

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers/"+id+"/pair", "u1", pairComputerRequest{Token: "tok"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var paired Computer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &paired))
	assert.Equal(t, id, paired.ID)
	assert.Equal(t, "laptop-ab12cd34.example.com", paired.Tunnel.Hostname)

	rec = doRequest(routes, http.MethodPost, "/api/pairing/computers/"+id+"/pair", "u1", "not json")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	rec = doRequest(routes, http.MethodPost, "/api/pairing/computers/"+id+"/pair", "u2", pairComputerRequest{Token: "tok"})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_PairErrors_KeyTheMessageUnderTheField(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		path      func(id string) string
		body      any
		exchErr   error
		wantCode  int
		wantField string
	}{
		{"refused token over the tunnel", func(id string) string { return "/api/pairing/computers/" + id + "/pair" }, pairComputerRequest{Token: "tok"}, apperrs.ErrInvalid, http.StatusBadRequest, "token"},
		{"unreachable over the tunnel", func(id string) string { return "/api/pairing/computers/" + id + "/pair" }, pairComputerRequest{Token: "tok"}, apperrs.Retryable(errBoom), http.StatusServiceUnavailable, "server_url"},
		{"bad URL by URL", func(string) string { return "/api/pairing/computers" }, pairRequest{Name: "VPS", ServerURL: "ftp://x", Token: "tok"}, nil, http.StatusBadRequest, "server_url"},
		{"blank name by URL", func(string) string { return "/api/pairing/computers" }, pairRequest{Name: " ", ServerURL: "https://x.example.com", Token: "tok"}, nil, http.StatusBadRequest, "name"},
		{"re-pair off the tunnel", func(id string) string { return "/api/pairing/computers/" + id + "/repair" }, pairRequest{Name: "Laptop", ServerURL: "https://x.example.com", Token: "tok"}, nil, http.StatusBadRequest, "server_url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			exch := pairedExchanger()
			exch.exchangeErr = tt.exchErr
			svc, _ := newTunnelService(newFakeRepo(), exch, &fakeTunnels{})
			id := tunnelComputer(t, svc)

			rec := doRequest(NewHandler(svc).Routes(), http.MethodPost, tt.path(id), "u1", tt.body)
			require.Equal(t, tt.wantCode, rec.Code, rec.Body.String())
			var body struct {
				Message string              `json:"message"`
				Errors  map[string][]string `json:"errors"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, []string{body.Message}, body.Errors[tt.wantField])
		})
	}
}
