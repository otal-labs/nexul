package livekit

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Event is the presence-relevant slice of a LiveKit WebhookEvent; join/leave events also carry who.
type Event struct {
	Type                string
	Room                string
	ParticipantIdentity string
	ParticipantName     string
}

// VerifyWebhook needs the untouched POST bytes as body, not re-marshaled.
func (c *Client) VerifyWebhook(body []byte, authHeader string) (Event, error) {
	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := parseJWT(token, c.APISecret)
	if err != nil {
		return Event{}, fmt.Errorf("%w: verify livekit webhook: %v", apperrs.ErrUnauthorized, err)
	}
	if exp, ok := claims["exp"].(float64); ok && time.Now().After(time.Unix(int64(exp), 0)) {
		return Event{}, fmt.Errorf("%w: livekit webhook token expired", apperrs.ErrUnauthorized)
	}
	claimHash, _ := claims["sha256"].(string)
	if claimHash == "" {
		return Event{}, fmt.Errorf("%w: livekit webhook token missing sha256 claim", apperrs.ErrUnauthorized)
	}
	wantHash, err := base64.StdEncoding.DecodeString(claimHash)
	if err != nil {
		return Event{}, fmt.Errorf("%w: livekit webhook sha256 claim is not valid base64", apperrs.ErrUnauthorized)
	}
	gotHash := sha256.Sum256(body)
	if !bytes.Equal(gotHash[:], wantHash) {
		return Event{}, fmt.Errorf("%w: livekit webhook body hash mismatch", apperrs.ErrUnauthorized)
	}

	var payload struct {
		Event string `json:"event"`
		Room  *struct {
			Name string `json:"name"`
		} `json:"room"`
		Participant *struct {
			Identity string `json:"identity"`
			Name     string `json:"name"`
		} `json:"participant"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Event{}, fmt.Errorf("%w: decode livekit webhook body: %v", apperrs.ErrInvalid, err)
	}
	ev := Event{Type: payload.Event}
	if payload.Room != nil {
		ev.Room = payload.Room.Name
	}
	if payload.Participant != nil {
		ev.ParticipantIdentity = payload.Participant.Identity
		ev.ParticipantName = payload.Participant.Name
	}
	return ev, nil
}
