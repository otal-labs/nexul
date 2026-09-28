package auth

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newSessionHarness(t *testing.T) (*Service, *fakeSessionStore, string) {
	t.Helper()
	users := newFakeUserStore()
	s := newTestService(&fakeGitHub{user: ghUser("1", "owner")}, users, newFakeAllowlist(), newFakeSettings())
	sessions := newFakeSessionStore()
	s.cfg.Sessions = sessions
	return s, sessions, seedPATUser(t, users)
}

func TestDeviceFromRequest(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want Device
	}{
		{"chrome on linux", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0 Safari/537.36", Device{Client: ClientBrowser, Platform: "Linux", Label: "Chrome"}},
		{"edge on windows", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/128.0 Safari/537.36 Edg/128.0", Device{Client: ClientBrowser, Platform: "Windows", Label: "Edge"}},
		{"safari on mac", "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15 Version/17.5 Safari/605.1.15", Device{Client: ClientBrowser, Platform: "macOS", Label: "Safari"}},
		{"firefox on android", "Mozilla/5.0 (Android 14; Mobile; rv:129.0) Gecko/129.0 Firefox/129.0", Device{Client: ClientBrowser, Platform: "Android", Label: "Firefox"}},
		{"electron is the desktop app", "Mozilla/5.0 (X11; Linux x86_64) nexul/1.0 Chrome/128.0 Electron/32.0 Safari/537.36", Device{Client: ClientDesktop, Platform: "Linux", Label: "Nexul desktop"}},
		{"unknown agent", "curl/8.0", Device{Client: ClientBrowser, Platform: "Unknown", Label: "Browser"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("User-Agent", tt.ua)
			r.RemoteAddr = "203.0.113.9:4321"
			tt.want.IP = "203.0.113.9"
			assert.Equal(t, tt.want, DeviceFromRequest(r))
		})
	}
}

func TestCreateSession_RecordsDeviceAndPublishes(t *testing.T) {
	s, sessions, u := newSessionHarness(t)
	ctx := WithDevice(context.Background(), Device{Client: ClientPhone, Platform: "Android", Label: "Pixel 8", IP: "10.0.0.5"})
	raw, err := s.CreateSession(ctx, u)
	require.NoError(t, err)

	list, err := s.ListSessions(context.Background(), u, "")
	require.NoError(t, err)
	require.Len(t, list, 1)
	ses := list[0]
	assert.Equal(t, ClientPhone, ses.Client)
	assert.Equal(t, "Android", ses.Platform)
	assert.Equal(t, "Pixel 8", ses.Label)
	assert.Equal(t, "10.0.0.5", ses.IP)
	assert.Equal(t, hashToken(raw), ses.TokenHash)
	assert.Equal(t, s.cfg.Now().Add(phoneSessionTTL), ses.ExpiresAt, "a phone slides over 90 days")
	assert.Equal(t, []string{TopicSessionCreated}, sessions.topics())

	s.cfg.Sessions = nil
	_, err = s.CreateSession(context.Background(), u)
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestAuthenticateSession_TouchesAtMostOnceAnHour(t *testing.T) {
	s, sessions, u := newSessionHarness(t)
	start := s.cfg.Now()
	raw, err := s.CreateSession(context.Background(), u)
	require.NoError(t, err)

	clock := start
	s.cfg.Now = func() time.Time { return clock }
	for _, step := range []time.Duration{time.Minute, 30 * time.Minute, 59 * time.Minute} {
		clock = start.Add(step)
		_, ses, err := s.AuthenticateSession(context.Background(), raw, "10.0.0.2")
		require.NoError(t, err)
		assert.Equal(t, start, ses.LastActiveAt, "no write inside the hour")
	}
	assert.Equal(t, 0, sessions.touches)

	clock = start.Add(time.Hour)
	_, ses, err := s.AuthenticateSession(context.Background(), raw, "10.0.0.2")
	require.NoError(t, err)
	assert.Equal(t, 1, sessions.touches)
	assert.Equal(t, clock, ses.LastActiveAt)
	assert.Equal(t, "10.0.0.2", ses.IP)
	assert.Equal(t, clock.Add(browserSessionTTL), ses.ExpiresAt, "expiry slides from the last use")

	clock = start.Add(90 * time.Minute)
	_, _, err = s.AuthenticateSession(context.Background(), raw, "10.0.0.3")
	require.NoError(t, err)
	assert.Equal(t, 1, sessions.touches, "the second write waits for the next hour")

	clock = start.Add(2 * time.Hour)
	sessions.touchErr = errors.New("disk gone")
	user, _, err := s.AuthenticateSession(context.Background(), raw, "10.0.0.3")
	require.NoError(t, err, "a failed touch never rejects a valid request")
	assert.Equal(t, u, user.ID)
}

func TestAuthenticateSession_ExpiredIsRejectedAndSweptAtNextSignIn(t *testing.T) {
	s, _, u := newSessionHarness(t)
	start := s.cfg.Now()
	raw, err := s.CreateSession(context.Background(), u)
	require.NoError(t, err)

	s.cfg.Now = func() time.Time { return start.Add(browserSessionTTL) }
	_, _, err = s.AuthenticateSession(context.Background(), raw, "")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	list, err := s.ListSessions(context.Background(), u, "")
	require.NoError(t, err)
	assert.Len(t, list, 1, "an expired row waits for the next sign-in")

	_, err = s.CreateSession(context.Background(), u)
	require.NoError(t, err)
	list, err = s.ListSessions(context.Background(), u, "")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.NotEqual(t, hashToken(raw), list[0].TokenHash, "only the fresh session remains")
}

func TestAuthenticateSession_RejectsInactiveOrUnknownUser(t *testing.T) {
	s, _, u := newSessionHarness(t)
	users := s.cfg.Users.(*fakeUserStore)
	raw, err := s.CreateSession(context.Background(), u)
	require.NoError(t, err)
	ghost, err := s.CreateSession(context.Background(), "ghost")
	require.NoError(t, err)

	_, _, err = s.AuthenticateSession(context.Background(), ghost, "")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	require.NoError(t, users.SetAccountStatus(context.Background(), u, AccountDisabled))
	_, _, err = s.AuthenticateSession(context.Background(), raw, "")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestSignOutSession_OneAndOthers(t *testing.T) {
	s, sessions, u := newSessionHarness(t)
	pats := newFakePATStore()
	s.cfg.PATs = pats
	patRaw, _, err := s.MintPAT(context.Background(), u, "ci")
	require.NoError(t, err)
	current, err := s.CreateSession(context.Background(), u)
	require.NoError(t, err)
	phone, err := s.CreateSession(WithDevice(context.Background(), Device{Client: ClientPhone}), u)
	require.NoError(t, err)
	laptop, err := s.CreateSession(context.Background(), u)
	require.NoError(t, err)
	_, currentSes, err := s.AuthenticateSession(context.Background(), current, "")
	require.NoError(t, err)
	_, phoneSes, err := s.AuthenticateSession(context.Background(), phone, "")
	require.NoError(t, err)

	list, err := s.ListSessions(context.Background(), u, currentSes.ID)
	require.NoError(t, err)
	require.Len(t, list, 3)
	var flagged int
	for _, ses := range list {
		if ses.Current {
			flagged++
			assert.Equal(t, currentSes.ID, ses.ID)
		}
	}
	assert.Equal(t, 1, flagged, "exactly the calling session is flagged")

	require.ErrorIs(t, s.SignOutSession(context.Background(), "someone-else", phoneSes.ID), apperrs.ErrNotFound)
	require.NoError(t, s.SignOutSession(context.Background(), u, phoneSes.ID))
	_, _, err = s.AuthenticateSession(context.Background(), phone, "")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized, "a signed-out token fails its next request")
	require.ErrorIs(t, s.SignOutSession(context.Background(), u, phoneSes.ID), apperrs.ErrNotFound)

	require.NoError(t, s.SignOutOtherSessions(context.Background(), u, currentSes.ID))
	_, _, err = s.AuthenticateSession(context.Background(), laptop, "")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	_, _, err = s.AuthenticateSession(context.Background(), current, "")
	require.NoError(t, err, "the calling session survives")
	_, err = s.AuthenticatePAT(context.Background(), patRaw)
	require.NoError(t, err, "personal access tokens are a separate table")
	require.NoError(t, s.SignOutOtherSessions(context.Background(), u, currentSes.ID), "nothing left to sign out is fine")

	assert.Equal(t, []string{TopicSessionCreated, TopicSessionCreated, TopicSessionCreated, TopicSessionRevoked, TopicSessionRevoked}, sessions.topics())
}

func TestSetPushToken_StoresAndClears(t *testing.T) {
	s, sessions, u := newSessionHarness(t)
	phone, err := s.CreateSession(WithDevice(context.Background(), Device{Client: ClientPhone}), u)
	require.NoError(t, err)
	_, ses, err := s.AuthenticateSession(context.Background(), phone, "")
	require.NoError(t, err)

	require.NoError(t, s.SetPushToken(context.Background(), u, ses.ID, " ExponentPushToken[abc] "))
	assert.Equal(t, "ExponentPushToken[abc]", sessions.pushTokens[ses.ID], "whitespace is trimmed")
	require.NoError(t, s.SetPushToken(context.Background(), u, ses.ID, ""))
	assert.Equal(t, "", sessions.pushTokens[ses.ID], "an empty token clears the registration")
	require.ErrorIs(t, s.SetPushToken(context.Background(), "someone-else", ses.ID, "tok"), apperrs.ErrNotFound)
	require.ErrorIs(t, s.SetPushToken(context.Background(), u, ses.ID, strings.Repeat("x", maxPushTokenLen+1)), apperrs.ErrInvalid)
}
