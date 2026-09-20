package livekit

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// signWebhook builds the Authorization header LiveKit's server would send
// for body: an HS256 JWT whose sha256 claim is body's base64-std-encoded
// SHA-256 hash (docs.livekit.io/home/server/webhooks).
func signWebhook(t *testing.T, secret string, body []byte, ttl time.Duration) string {
	t.Helper()
	hash := sha256.Sum256(body)
	now := time.Now().UTC()
	claims := map[string]any{
		"iss":    "key1",
		"nbf":    now.Unix(),
		"exp":    now.Add(ttl).Unix(),
		"sha256": base64.StdEncoding.EncodeToString(hash[:]),
	}
	token, err := signJWT(claims, secret)
	if err != nil {
		t.Fatalf("sign test webhook token: %v", err)
	}
	return "Bearer " + token
}

func TestVerifyWebhook(t *testing.T) {
	c := &Client{APIKey: "key1", APISecret: "secret1"}
	body := []byte(`{"event":"participant_joined","room":{"name":"room-1"},"participant":{"identity":"u1","name":"Ada"}}`)

	tests := []struct {
		name       string
		authHeader func() string
		body       []byte
		wantErr    error
		want       Event
	}{
		{
			name:       "valid participant_joined",
			authHeader: func() string { return signWebhook(t, "secret1", body, time.Minute) },
			body:       body,
			want:       Event{Type: "participant_joined", Room: "room-1", ParticipantIdentity: "u1", ParticipantName: "Ada"},
		},
		{
			name: "room_started has no participant",
			authHeader: func() string {
				return signWebhook(t, "secret1", []byte(`{"event":"room_started","room":{"name":"room-2"}}`), time.Minute)
			},
			body: []byte(`{"event":"room_started","room":{"name":"room-2"}}`),
			want: Event{Type: "room_started", Room: "room-2"},
		},
		{
			name:       "wrong secret rejected",
			authHeader: func() string { return signWebhook(t, "other-secret", body, time.Minute) },
			body:       body,
			wantErr:    apperrs.ErrUnauthorized,
		},
		{
			name:       "tampered body rejected (hash mismatch)",
			authHeader: func() string { return signWebhook(t, "secret1", body, time.Minute) },
			body:       []byte(`{"event":"participant_joined","room":{"name":"room-1"},"participant":{"identity":"attacker","name":"Eve"}}`),
			wantErr:    apperrs.ErrUnauthorized,
		},
		{
			name:       "expired token rejected",
			authHeader: func() string { return signWebhook(t, "secret1", body, -time.Minute) },
			body:       body,
			wantErr:    apperrs.ErrUnauthorized,
		},
		{
			name:       "malformed header rejected",
			authHeader: func() string { return "Bearer not-a-jwt" },
			body:       body,
			wantErr:    apperrs.ErrUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.VerifyWebhook(tt.body, tt.authHeader())
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("VerifyWebhook() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("VerifyWebhook(): %v", err)
			}
			if got != tt.want {
				t.Errorf("VerifyWebhook() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
