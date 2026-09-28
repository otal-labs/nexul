package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// sessionPrefix marks a session token so the gateway can tell it from a personal access token or a setup pass.
const sessionPrefix = "ses_"

const (
	browserSessionTTL = 30 * 24 * time.Hour
	phoneSessionTTL   = 90 * 24 * time.Hour
	// sessionTouchInterval bounds the last-active writes so a busy tab does not turn every request into a write.
	sessionTouchInterval = time.Hour
)

const deviceCtxKey ctxKey = "device"

// WithDevice records what is signing in, so the mint sites can label the session without a signature change.
func WithDevice(ctx context.Context, d Device) context.Context {
	return context.WithValue(ctx, deviceCtxKey, d)
}

func deviceFromCtx(ctx context.Context) Device {
	d, _ := ctx.Value(deviceCtxKey).(Device)
	return d
}

// DeviceFromRequest reads the signing-in device off the request: a few lines of user-agent matching, no dependency.
func DeviceFromRequest(r *http.Request) Device {
	ua := r.UserAgent()
	d := Device{Client: ClientBrowser, Platform: uaPlatform(ua), IP: clientAddr(r)}
	if strings.Contains(ua, "Electron/") {
		d.Client = ClientDesktop
		d.Label = "Nexul desktop"
		return d
	}
	d.Label = uaBrowser(ua)
	return d
}

func uaPlatform(ua string) string {
	for _, m := range []struct{ needle, name string }{
		{"Windows", "Windows"}, {"Android", "Android"}, {"iPhone", "iOS"}, {"iPad", "iOS"},
		{"Mac OS X", "macOS"}, {"Macintosh", "macOS"}, {"CrOS", "ChromeOS"}, {"Linux", "Linux"},
	} {
		if strings.Contains(ua, m.needle) {
			return m.name
		}
	}
	return "Unknown"
}

// uaBrowser checks the forks first: Edge and Opera also advertise Chrome, and Chrome also advertises Safari.
func uaBrowser(ua string) string {
	for _, m := range []struct{ needle, name string }{
		{"Edg/", "Edge"}, {"OPR/", "Opera"}, {"Firefox/", "Firefox"}, {"Chrome/", "Chrome"}, {"Safari/", "Safari"},
	} {
		if strings.Contains(ua, m.needle) {
			return m.name
		}
	}
	return "Browser"
}

func sessionTTL(c SessionClient) time.Duration {
	if c == ClientPhone {
		return phoneSessionTTL
	}
	return browserSessionTTL
}

// CreateSession is the one mint site: every sign-in path calls it, and it sweeps the user's expired rows first.
func (s *Service) CreateSession(ctx context.Context, userID string) (string, error) {
	if s.cfg.Sessions == nil {
		return "", fmt.Errorf("%w: sessions are not configured", apperrs.ErrUnauthorized)
	}
	now := s.cfg.Now()
	if err := s.cfg.Sessions.DeleteExpiredSessions(ctx, userID, now); err != nil {
		return "", fmt.Errorf("sweep expired sessions: %w", err)
	}
	raw, err := newToken(sessionPrefix)
	if err != nil {
		return "", err
	}
	d := deviceFromCtx(ctx)
	if d.Client == "" {
		d.Client = ClientBrowser
	}
	ses := &Session{
		ID: newUserID(), UserID: userID, Client: d.Client, Platform: d.Platform, Label: d.Label, IP: d.IP,
		CreatedAt: now, LastActiveAt: now, ExpiresAt: now.Add(sessionTTL(d.Client)), TokenHash: hashToken(raw),
	}
	if err := s.cfg.Sessions.CreateSession(ctx, ses, sessionEvent(TopicSessionCreated, *ses)); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return raw, nil
}

// AuthenticateSession resolves a ses_ token to its user and session; a missing or expired row fails the request.
func (s *Service) AuthenticateSession(ctx context.Context, raw, ip string) (*User, *Session, error) {
	if s.cfg.Sessions == nil || !strings.HasPrefix(raw, sessionPrefix) {
		return nil, nil, apperrs.ErrUnauthorized
	}
	ses, err := s.cfg.Sessions.GetSessionByHash(ctx, hashToken(raw))
	if err != nil {
		return nil, nil, apperrs.ErrUnauthorized
	}
	now := s.cfg.Now()
	if !now.Before(ses.ExpiresAt) {
		return nil, nil, apperrs.ErrUnauthorized
	}
	user, err := s.cfg.Users.GetUserByID(ctx, ses.UserID)
	if err != nil {
		return nil, nil, apperrs.ErrUnauthorized
	}
	if !accountIsActive(user.AccountStatus) {
		return nil, nil, apperrs.ErrUnauthorized
	}
	s.touchSession(ctx, ses, now, ip)
	return user, ses, nil
}

// touchSession slides the expiry at most once an hour; a write failure must not reject an otherwise valid request.
func (s *Service) touchSession(ctx context.Context, ses *Session, now time.Time, ip string) {
	if now.Sub(ses.LastActiveAt) < sessionTouchInterval {
		return
	}
	ses.LastActiveAt, ses.IP, ses.ExpiresAt = now, ip, now.Add(sessionTTL(ses.Client))
	if err := s.cfg.Sessions.TouchSession(ctx, ses.ID, now, ip, ses.ExpiresAt); err != nil {
		logging.FromCtx(ctx).Warn("touch session", "session_id", ses.ID, "error", err)
	}
}

// ListSessions returns the user's signed-in devices with the calling one flagged; metadata only, never a token.
func (s *Service) ListSessions(ctx context.Context, userID, currentID string) ([]Session, error) {
	sessions, err := s.cfg.Sessions.ListSessionsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	for i := range sessions {
		sessions[i].Current = sessions[i].ID == currentID
	}
	return sessions, nil
}

// SignOutSession deletes one of the user's sessions; that token's next request fails.
func (s *Service) SignOutSession(ctx context.Context, userID, id string) error {
	sessions, err := s.cfg.Sessions.ListSessionsByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}
	for _, ses := range sessions {
		if ses.ID != id {
			continue
		}
		return s.cfg.Sessions.DeleteSession(ctx, id, userID, sessionEvent(TopicSessionRevoked, ses))
	}
	return fmt.Errorf("sign out session %s: %w", id, apperrs.ErrNotFound)
}

// SignOutOtherSessions keeps only the calling session; personal access tokens are a separate table and untouched.
func (s *Service) SignOutOtherSessions(ctx context.Context, userID, keepID string) error {
	sessions, err := s.cfg.Sessions.ListSessionsByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}
	evts := make([]eventbus.OutboxEvent, 0, len(sessions))
	for _, ses := range sessions {
		if ses.ID == keepID {
			continue
		}
		evts = append(evts, sessionEvent(TopicSessionRevoked, ses))
	}
	if len(evts) == 0 {
		return nil
	}
	return s.cfg.Sessions.DeleteOtherSessions(ctx, userID, keepID, evts...)
}

func sessionEvent(topic string, ses Session) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: SessionChangedEvent{
		SessionID: ses.ID, UserID: ses.UserID, Client: ses.Client, Platform: ses.Platform, Label: ses.Label,
	}}
}
