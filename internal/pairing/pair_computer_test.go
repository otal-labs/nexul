package pairing

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestService_PairComputer_OnlyTheOwnersComputer(t *testing.T) {
	t.Parallel()
	f := newRunnerFixture(t, Computer{Name: "Laptop"})

	_, err := f.svc.PairComputer(t.Context(), "u2", "c-laptop")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = f.svc.PairComputer(t.Context(), "", "c-laptop")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	_, err = f.svc.PairComputer(t.Context(), "u-alice", " ")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestService_ComputerStillPairing_ReadsAsUnpairedNotExpired(t *testing.T) {
	t.Parallel()
	svc, _ := newTunnelService(newFakeRepo(), &fakeExchanger{}, &fakeTunnels{})
	id := tunnelComputer(t, svc)

	_, err := svc.ResolveTarget(t.Context(), "u1", "")
	var nc *NotConfiguredError
	require.ErrorAs(t, err, &nc)
	assert.Equal(t, ReasonUnpaired, nc.Reason)

	_, err = svc.ListProjects(t.Context(), "u1", id)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "still pairing")
}

func TestHandler_PairComputer_PairsThroughTheRunner(t *testing.T) {
	t.Parallel()
	f := newRunnerFixture(t, Computer{Name: "Laptop"})
	routes := NewHandler(f.svc).Routes()

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers/c-laptop/pair", "u-alice", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var paired Computer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &paired))
	assert.True(t, paired.Paired())
	assert.Equal(t, runnerAddress("c-laptop"), f.exch.pairedURL)
	assert.Equal(t, "pair-tok", f.exch.pairedToken)

	rec = doRequest(routes, http.MethodPost, "/api/pairing/computers/c-laptop/pair", "u2", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
