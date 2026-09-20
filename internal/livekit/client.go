// Package livekit is a minimal, stdlib-only LiveKit client; deliberately skips the server-sdk-go dependency.
package livekit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// tokenTTL is a join token's lifetime ("short-lived").
const tokenTTL = 6 * time.Hour

// serviceTokenTTL is a Twirp API token's lifetime; used once and discarded, never handed to a browser.
const serviceTokenTTL = 5 * time.Minute

// httpClient is a package var, not a Client field, since Client is a plain value constructed fresh per call.
var httpClient = &http.Client{Timeout: 15 * time.Second}

// Client talks to one LiveKit server, identified by its WebSocket URL and API key/secret pair.
type Client struct {
	WSURL     string
	APIKey    string
	APISecret string
}

// MintToken builds a short-lived room-join token; grants are a fixed shape for a Discord-style channel (ADR 0030).
func (c *Client) MintToken(identity, name, room string) (string, error) {
	now := time.Now().UTC()
	claims := map[string]any{
		"iss":  c.APIKey,
		"sub":  identity,
		"name": name,
		"nbf":  now.Unix(),
		"exp":  now.Add(tokenTTL).Unix(),
		"video": map[string]any{
			"room":              room,
			"roomJoin":          true,
			"roomCreate":        true,
			"canSubscribe":      true,
			"canPublishSources": []string{"microphone", "camera", "screen_share", "screen_share_audio"},
		},
	}
	return signJWT(claims, c.APISecret)
}

// Room is one active LiveKit room and who's in it.
type Room struct {
	Name         string
	Participants []Participant
}

// Participant is one room occupant's identity and display name.
type Participant struct {
	Identity string
	Name     string
}

// ListRooms lists every active room, fetching participants for any room that reports at least one.
func (c *Client) ListRooms(ctx context.Context) ([]Room, error) {
	// Accept both num_participants and numParticipants; mis-parsing as 0 silently empties everyone's presence.
	var resp struct {
		Rooms []struct {
			Name                 string `json:"name"`
			NumParticipants      int    `json:"num_participants"`
			NumParticipantsCamel int    `json:"numParticipants"`
		} `json:"rooms"`
	}
	if err := c.call(ctx, "ListRooms", map[string]any{"roomList": true}, map[string]any{}, &resp); err != nil {
		return nil, err
	}
	rooms := make([]Room, 0, len(resp.Rooms))
	for _, r := range resp.Rooms {
		room := Room{Name: r.Name}
		if r.NumParticipants > 0 || r.NumParticipantsCamel > 0 {
			participants, err := c.ListParticipants(ctx, r.Name)
			if err != nil {
				return nil, err
			}
			room.Participants = participants
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

// ListParticipants lists who's currently in room.
func (c *Client) ListParticipants(ctx context.Context, room string) ([]Participant, error) {
	var resp struct {
		Participants []struct {
			Identity string `json:"identity"`
			Name     string `json:"name"`
		} `json:"participants"`
	}
	if err := c.call(ctx, "ListParticipants", map[string]any{"roomAdmin": true, "room": room}, map[string]any{"room": room}, &resp); err != nil {
		return nil, err
	}
	out := make([]Participant, 0, len(resp.Participants))
	for _, p := range resp.Participants {
		out = append(out, Participant{Identity: p.Identity, Name: p.Name})
	}
	return out, nil
}

// call issues one Twirp RoomService request; grants are per-method (roomList vs roomAdmin).
func (c *Client) call(ctx context.Context, method string, grant map[string]any, body any, out any) (err error) {
	now := time.Now().UTC()
	token, err := signJWT(map[string]any{
		"iss":   c.APIKey,
		"nbf":   now.Unix(),
		"exp":   now.Add(serviceTokenTTL).Unix(),
		"video": grant,
	}, c.APISecret)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal livekit %s request: %w", method, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		httpBase(c.WSURL)+"/twirp/livekit.RoomService/"+method, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("livekit %s: %w", method, apperrs.Retryable(err))
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read livekit %s response: %w", method, apperrs.Retryable(err))
	}
	if resp.StatusCode >= 500 {
		return apperrs.Retryable(fmt.Errorf("livekit %s: status %d", method, resp.StatusCode))
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: livekit %s: status %d: %s", apperrs.ErrUnauthorized, method, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode livekit %s response: %w", method, apperrs.Retryable(err))
	}
	return nil
}

// httpBase converts a ws(s):// URL to the http(s) base the Twirp API lives at; an http(s) URL passes through.
func httpBase(wsURL string) string {
	switch {
	case strings.HasPrefix(wsURL, "wss://"):
		wsURL = "https://" + strings.TrimPrefix(wsURL, "wss://")
	case strings.HasPrefix(wsURL, "ws://"):
		wsURL = "http://" + strings.TrimPrefix(wsURL, "ws://")
	}
	return strings.TrimRight(wsURL, "/")
}
