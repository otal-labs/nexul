package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

func TestStart_RefusesWhatItCannotRun(t *testing.T) {
	automation := identity.WithActor(context.Background(), identity.Actor{ID: testOwner, Automation: &identity.AutomationRef{ID: "a-1"}})
	tests := []struct {
		name string
		ctx  context.Context
		in   Start
		want error
	}{
		{"nobody signed in", context.Background(), Start{Source: "fake", ProjectID: testProject}, apperrs.ErrUnauthorized},
		{"an automation token", automation, Start{Source: "fake", ProjectID: testProject}, apperrs.ErrUnauthorized},
		{"an unknown source", as(testOwner), Start{Source: "nope", ProjectID: testProject}, apperrs.ErrInvalid},
		{"no project", as(testOwner), Start{Source: "fake"}, apperrs.ErrInvalid},
		{"a missing project", as(testOwner), Start{Source: "fake", ProjectID: "gone"}, apperrs.ErrNotFound},
		{"someone who is not the Owner", as("u-writer"), Start{Source: "fake", ProjectID: testProject}, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(fakeSource{})
			_, err := svc.Start(tt.ctx, tt.in)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestStart_ScansThenWaitsToBeConfirmed(t *testing.T) {
	src := fakeSource{scan: Scan{From: "acme/app", Tickets: 3, Comments: 4, People: []Person{{Login: "zed", Name: "zed"}},
		Statuses: []StatusRef{{Kind: StatusBacklog}, {Kind: StatusDone}}}}
	svc, repo := newTestService(src)
	imp, err := svc.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject, Credential: "secret"})
	require.NoError(t, err)
	assert.Equal(t, StateScanning, imp.State)

	svc.Pass(t.Context())
	got, err := svc.Get(as(testOwner), imp.ID)
	require.NoError(t, err)
	assert.Equal(t, StateReady, got.State)
	assert.Equal(t, "acme/app", got.From)
	assert.Equal(t, 3, got.Total)
	assert.Equal(t, []PersonMatch{{Person: Person{Login: "zed", Name: "zed"}}}, got.Report.People)
	assert.Equal(t, []StatusMatch{
		{StatusRef: StatusRef{Kind: StatusBacklog}, Column: "Backlog", New: true},
		{StatusRef: StatusRef{Kind: StatusDone}, Column: "Done", New: true},
	}, got.Report.Statuses)
	assert.Equal(t, "secret", repo.stored(imp.ID).Credential, "the credential stays until the import ends")
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "secret", "the credential is never sent")

	confirmed, err := svc.Confirm(as(testOwner), imp.ID)
	require.NoError(t, err)
	assert.Equal(t, StateImporting, confirmed.State)
	svc.Pass(t.Context())
	assert.Equal(t, StateCompleted, repo.stored(imp.ID).State)
	assert.Empty(t, repo.stored(imp.ID).Credential)
}

func TestScan_OverTheLimitsFailsWithTheReason(t *testing.T) {
	tests := []struct {
		name string
		scan Scan
		want string
	}{
		{"too many tickets", Scan{Tickets: MaxTickets + 1}, "more than the 20000 one import takes"},
		{"too many docs", Scan{Docs: MaxDocs + 1}, "more than the 5000 one import takes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService(fakeSource{scan: tt.scan})
			imp, err := svc.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject, Credential: "secret"})
			require.NoError(t, err)
			svc.Pass(t.Context())
			got := repo.stored(imp.ID)
			assert.Equal(t, StateFailed, got.State)
			assert.Contains(t, got.Report.Error, tt.want)
			assert.Empty(t, got.Credential)
		})
	}
}

func TestScan_APassingFailureRetriesAndAnyOtherFails(t *testing.T) {
	retry, _ := newTestService(fakeSource{scanErr: apperrs.Retryable(errors.New("rate limited"))})
	imp, err := retry.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject})
	require.NoError(t, err)
	assert.Equal(t, retryAfter, retry.Pass(t.Context()))
	got, err := retry.Get(as(testOwner), imp.ID)
	require.NoError(t, err)
	assert.Equal(t, StateScanning, got.State)

	for _, cause := range []error{errors.New("socket closed: 10.0.0.1"), fmt.Errorf("get credentials github: %w", apperrs.ErrNotFound)} {
		broken, repo := newTestService(fakeSource{scanErr: cause})
		imp, err := broken.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject})
		require.NoError(t, err)
		broken.Pass(t.Context())
		assert.Equal(t, StateFailed, repo.stored(imp.ID).State)
		assert.Equal(t, "the source could not be read; the server log has the details", repo.stored(imp.ID).Report.Error, cause.Error())
	}
}

func TestReady_ExpiresAfterADay(t *testing.T) {
	svc, repo := newTestService(fakeSource{})
	imp, err := svc.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject, Credential: "secret"})
	require.NoError(t, err)
	svc.Pass(t.Context())
	wait := svc.Pass(t.Context())
	assert.InDelta(t, ReadyFor, wait, float64(time.Minute), "a ready import wakes the loop when it expires")

	svc.now = func() time.Time { return time.Now().Add(ReadyFor + time.Minute) }
	svc.Pass(t.Context())
	got := repo.stored(imp.ID)
	assert.Equal(t, StateCancelled, got.State)
	assert.Empty(t, got.Credential)
	assert.Contains(t, got.Report.Error, "24 hours")
}

func TestConfirmAndCancel_RefuseTheWrongState(t *testing.T) {
	svc, _ := newTestService(fakeSource{})
	imp, err := svc.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject})
	require.NoError(t, err)
	_, err = svc.Confirm(as(testOwner), imp.ID)
	assert.ErrorIs(t, err, apperrs.ErrInvalid, "a scanning import is not ready")

	cancelled, err := svc.Cancel(as(testOwner), imp.ID)
	require.NoError(t, err)
	assert.Equal(t, StateCancelled, cancelled.State)
	_, err = svc.Cancel(as(testOwner), imp.ID)
	assert.ErrorIs(t, err, apperrs.ErrInvalid, "an ended import cannot be cancelled again")

	_, err = svc.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject})
	assert.NoError(t, err, "a cancelled import frees the workspace for the next")
}

func TestGet_HidesAnImportFromAnyoneButTheOwner(t *testing.T) {
	svc, _ := newTestService(fakeSource{})
	imp, err := svc.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject})
	require.NoError(t, err)
	_, err = svc.Get(as("u-writer"), imp.ID)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = svc.List(as("u-writer"), testWorkspace)
	assert.ErrorIs(t, err, apperrs.ErrForbidden)
	list, err := svc.List(as(testOwner), testWorkspace)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestImportRun_ReadsStartsConfirmsAndCancels(t *testing.T) {
	svc, _ := newTestService(fakeSource{})
	tool := MCPTools(svc)[0]
	call := func(args string) (*Import, error) {
		out, err := tool.Call(as(testOwner), json.RawMessage(args))
		if err != nil {
			return nil, err
		}
		return out.(*Import), nil
	}
	_, err := call(`{"confirm": true}`)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)

	started, err := call(`{"source": "fake", "project_id": "p-1"}`)
	require.NoError(t, err)
	_, err = call(`{"id": "` + started.ID + `", "confirm": true, "cancel": true}`)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)

	svc.Pass(t.Context())
	read, err := call(`{"id": "` + started.ID + `"}`)
	require.NoError(t, err)
	assert.Equal(t, StateReady, read.State)
	confirmed, err := call(`{"id": "` + started.ID + `", "confirm": true}`)
	require.NoError(t, err)
	assert.Equal(t, StateImporting, confirmed.State)
	cancelled, err := call(`{"id": "` + started.ID + `", "cancel": true}`)
	require.NoError(t, err)
	assert.Equal(t, StateCancelled, cancelled.State)
}
