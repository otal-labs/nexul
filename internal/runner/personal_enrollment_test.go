package runner

import (
	"context"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
)

// claimsOf reads a computer token's claims, checking it was signed with the service's key.
func claimsOf(t *testing.T, token string) ComputerClaims {
	t.Helper()
	var c ComputerClaims
	require.NoError(t, hostcred.ParseToken(token, testTokenKey, &c))
	return c
}

func TestService_CreatePersonalEnrollment_SignsATokenBoundToItsPersonAndComputer(t *testing.T) {
	withVersion(t, "v0.3.0-beta-012")
	repo := newFakeRunnerRepo()

	e, err := newEnrollService(repo, &fakeDispatch{}).CreatePersonalEnrollment(asMember(), "u-alice", "c-laptop")

	require.NoError(t, err, "adding a computer takes no permission bit")
	assert.Equal(t, enrollNow.Add(time.Hour), e.ExpiresAt)
	assert.Empty(t, e.Code, "the code travels only inside the token")
	assert.Equal(t, "curl -fsSL https://nexul.io/computer.sh | sudo NEXUL_VERSION=v0.3.0-beta-012 sh -s -- "+e.Token, e.Commands.Unix)
	claims := claimsOf(t, e.Token)
	assert.Equal(t, ComputerClaims{Server: "https://nexul.example.com", Code: claims.Code, Computer: "c-laptop", Exp: enrollNow.Add(time.Hour).Unix()}, claims)
	stored := repo.codes[hostcred.Hash(claims.Code)]
	require.NotNil(t, stored)
	assert.Equal(t, "u-alice", stored.OwnerUserID)
	assert.Equal(t, "c-laptop", stored.ComputerID)
	assert.Regexp(t, regexp.MustCompile(`^computer-[a-z2-7]{8}$`), stored.Name)
}

func TestService_CreatePersonalEnrollment_RendersAnotherSiteAndRelease(t *testing.T) {
	withVersion(t, "dev")
	svc := newEnrollService(newFakeRunnerRepo(), &fakeDispatch{})
	svc.install.SiteURL, svc.install.ReleaseURL = "http://10.0.0.5:8000/", "http://10.0.0.5:8000"

	e, err := svc.CreatePersonalEnrollment(asMember(), "u-alice", "c-laptop")

	require.NoError(t, err)
	assert.Equal(t, "curl -fsSL http://10.0.0.5:8000/computer.sh | sudo NEXUL_INSTALL_URL=http://10.0.0.5:8000/install.sh NEXUL_RELEASE_URL=http://10.0.0.5:8000 sh -s -- "+e.Token, e.Commands.Unix)
}

func TestService_CreatePersonalEnrollment_Refusals(t *testing.T) {
	tests := []struct {
		name               string
		change             func(*Service)
		userID, computerID string
		want               error
	}{
		{"no person", func(*Service) {}, "", "c-laptop", apperrs.ErrInvalid},
		{"no computer", func(*Service) {}, "u-alice", "", apperrs.ErrInvalid},
		{"a computer that already has its runner", func(*Service) {}, "u-alice", "c-taken", apperrs.ErrConflict},
		{"no instance url yet", func(s *Service) { s.install.Settings = &fakeSettingsReader{} }, "u-alice", "c-laptop", apperrs.ErrConflict},
		{"no key to sign with", func(s *Service) { s.install.TokenKey = nil }, "u-alice", "c-laptop", apperrs.ErrFatal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRunnerRepo()
			repo.personal("r-1", "u-alice", "c-taken")
			svc := newEnrollService(repo, &fakeDispatch{})
			tt.change(svc)
			_, err := svc.CreatePersonalEnrollment(asMember(), tt.userID, tt.computerID)
			require.ErrorIs(t, err, tt.want)
			assert.Empty(t, repo.codes)
		})
	}
}

func TestService_Enroll_PersonalAndDeployCodesNeverCross(t *testing.T) {
	repo := newFakeRunnerRepo()
	svc := newEnrollService(repo, &fakeDispatch{})
	personal, err := svc.CreatePersonalEnrollment(asMember(), "u-alice", "c-laptop")
	require.NoError(t, err)
	deploy := mustEnrollment(t, svc, "build-box", "")
	forged, err := hostcred.SignToken(ComputerClaims{Server: "https://nexul.example.com", Code: deploy, Computer: "c-laptop", Exp: enrollNow.Add(time.Hour).Unix()}, testTokenKey)
	require.NoError(t, err)

	_, err = svc.Enroll(context.Background(), EnrollRequest{Code: claimsOf(t, personal.Token).Code, Name: "build-box", Machine: "box-1"})
	assertCoded(t, err, apperrs.ErrConflict, "computer_code")
	_, err = svc.Enroll(context.Background(), EnrollRequest{Token: forged, Machine: "box-1"})
	assertCoded(t, err, apperrs.ErrConflict, "runner_code")

	assert.Len(t, repo.codes, 2, "a refused enrollment leaves its code for the right installer")
	all, err := repo.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, all)
	_, err = repo.GetByComputer(context.Background(), "c-laptop")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestService_Enroll_RefusesATokenThisInstanceWouldNotHaveIssued(t *testing.T) {
	repo := newFakeRunnerRepo()
	svc := newEnrollService(repo, &fakeDispatch{})
	e, err := svc.CreatePersonalEnrollment(asMember(), "u-alice", "c-laptop")
	require.NoError(t, err)
	claims := claimsOf(t, e.Token)
	sign := func(c ComputerClaims, key []byte) string {
		token, err := hostcred.SignToken(c, key)
		require.NoError(t, err)
		return token
	}
	otherComputer, expired, otherKey := claims, claims, claims
	otherComputer.Computer = "c-bobs"
	expired.Exp = enrollNow.Unix()
	for name, token := range map[string]string{
		"signed by another instance":           sign(otherKey, []byte("another instance's key")),
		"naming another computer":              sign(otherComputer, testTokenKey),
		"past its expiry":                      sign(expired, testTokenKey),
		"with the code alone, no token at all": "",
	} {
		req := EnrollRequest{Token: token, Machine: "alice-laptop"}
		if token == "" {
			req.Code = claims.Code
		}
		_, err := svc.Enroll(context.Background(), req)
		require.Error(t, err, name)
	}
	_, err = repo.GetByComputer(context.Background(), "c-laptop")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "nothing enrolled")

	_, err = svc.Enroll(context.Background(), EnrollRequest{Token: e.Token, Machine: "alice-laptop"})
	require.NoError(t, err)
	_, err = svc.Enroll(context.Background(), EnrollRequest{Token: e.Token, Machine: "alice-laptop"})
	assertCoded(t, err, apperrs.ErrUnauthorized, "invalid_code")
}

func assertCoded(t *testing.T, err error, sentinel error, code string) {
	t.Helper()
	require.ErrorIs(t, err, sentinel)
	var coded *apperrs.Coded
	require.ErrorAs(t, err, &coded)
	assert.Equal(t, code, coded.Code)
}

func TestService_Enroll_APersonalCodeMakesTheComputersRunnerOnNoMachine(t *testing.T) {
	repo := newFakeRunnerRepo()
	bus := newFakeBus()
	svc := newEnrollService(repo, &fakeDispatch{}).WithBus(bus)
	e, err := svc.CreatePersonalEnrollment(asMember(), "u-alice", "c-laptop")
	require.NoError(t, err)

	got, err := svc.Enroll(context.Background(), EnrollRequest{Token: e.Token, Machine: "alice-laptop", Version: "v0.3.0"})

	require.NoError(t, err)
	r, err := repo.GetByComputer(context.Background(), "c-laptop")
	require.NoError(t, err)
	assert.Equal(t, got.ID, r.ID)
	assert.Equal(t, got.Name, r.Name, "the code names the runner, so the installer sends none")
	assert.Equal(t, "u-alice", r.OwnerUserID)
	assert.Empty(t, r.MachineID)
	assert.Empty(t, got.Machine)
	machines, err := repo.machines.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, machines, "a computer is no machine")
	evs := bus.topicEvents(TopicPersonalChanged)
	require.Len(t, evs, 1)
	assert.Equal(t, PersonalChangedEvent{RunnerID: r.ID, ComputerID: "c-laptop", UserID: "u-alice", State: PersonalEnrolled, Hostname: "alice-laptop", MembersOnly: true},
		decodeEvent[PersonalChangedEvent](t, evs[0]))
}

// TestHandler_PersonalRunnerChangesGoOnlyToItsOwnersTopic: a personal runner's connect, beats and disconnect never
// reach runner.connected, runner.heartbeat or runner.disconnected, which every runners:read holder hears.
func TestHandler_PersonalRunnerChangesGoOnlyToItsOwnersTopic(t *testing.T) {
	repo := newFakeRunnerRepo()
	bus := newFakeBus()
	h := newTestHandler(bus, repo)
	srv := httptest.NewServer(streamMux(h))
	t.Cleanup(srv.Close)
	credential := repo.personal("r-alice", "u-alice", "c-laptop")

	conn, _, err := dialRunner(t.Context(), srv, credential, "/ws/runner")
	require.NoError(t, err)
	eventually(t, 2*time.Second, func() bool { return len(h.Runners()) == 1 })
	require.NoError(t, wsjson.Write(t.Context(), conn, Frame{Type: FrameHeartbeat, TS: time.Now().Unix()}))
	eventually(t, 2*time.Second, func() bool { return repo.heartbeatCount() == 1 })
	require.NoError(t, conn.Close(websocket.StatusNormalClosure, ""))
	eventually(t, 2*time.Second, func() bool { return len(bus.topicEvents(TopicPersonalChanged)) == 2 })

	var states []string
	for _, ev := range bus.topicEvents(TopicPersonalChanged) {
		p := decodeEvent[PersonalChangedEvent](t, ev)
		assert.Equal(t, "u-alice", p.UserID)
		assert.True(t, p.MembersOnly)
		states = append(states, p.State)
	}
	assert.Equal(t, []string{PersonalConnected, PersonalDisconnected}, states)
	for _, topic := range []string{TopicRunnerConnected, TopicRunnerHeartbeat, TopicRunnerDisconnected} {
		assert.Empty(t, bus.topicEvents(topic), topic)
	}
}
