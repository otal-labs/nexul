package livekit

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestHTTPBase(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"wss to https", "wss://lk.example.com/", "https://lk.example.com"},
		{"ws to http", "ws://localhost:7880", "http://localhost:7880"},
		{"already http passthrough", "http://lk.internal/", "http://lk.internal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := httpBase(tt.in); got != tt.want {
				t.Errorf("httpBase(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestMintToken_ClaimStructure asserts the exact claim shape the research
// doc (.scratch/integrations/research/08-livekit-facts.md) pins down:
// standard iss/sub/name/nbf/exp fields plus a video grant with roomJoin,
// roomCreate, canSubscribe, and canPublishSources limited to
// microphone/camera/screen_share/screen_share_audio.
func TestMintToken_ClaimStructure(t *testing.T) {
	c := &Client{WSURL: "wss://lk.example.com", APIKey: "APIkey1", APISecret: "secret1"}
	token, err := c.MintToken("user-1", "Ada Lovelace", "room-42")
	if err != nil {
		t.Fatalf("MintToken: %v", err)
	}

	claims, err := parseJWT(token, "secret1")
	if err != nil {
		t.Fatalf("parseJWT(minted token): %v", err)
	}
	assertStandardClaims(t, claims)
	assertVideoGrant(t, claims)
}

// assertStandardClaims checks the non-video claims MintToken sets: iss/sub/name plus an exp-nbf window
// matching tokenTTL.
func assertStandardClaims(t *testing.T, claims map[string]any) {
	t.Helper()
	if claims["iss"] != "APIkey1" {
		t.Errorf("iss = %v, want APIkey1", claims["iss"])
	}
	if claims["sub"] != "user-1" {
		t.Errorf("sub = %v, want user-1", claims["sub"])
	}
	if claims["name"] != "Ada Lovelace" {
		t.Errorf("name = %v, want Ada Lovelace", claims["name"])
	}
	exp, ok := claims["exp"].(float64)
	nbf, ok2 := claims["nbf"].(float64)
	if !ok || !ok2 {
		t.Fatalf("exp/nbf missing or not numeric: %+v", claims)
	}
	if got := exp - nbf; got != tokenTTL.Seconds() {
		t.Errorf("exp-nbf = %v seconds, want %v (tokenTTL)", got, tokenTTL.Seconds())
	}
}

// assertVideoGrant checks the video grant's room, roomJoin/roomCreate/canSubscribe flags, and the exact
// canPublishSources list the research doc pins down.
func assertVideoGrant(t *testing.T, claims map[string]any) {
	t.Helper()
	video, ok := claims["video"].(map[string]any)
	if !ok {
		t.Fatalf("video claim missing or wrong type: %+v", claims)
	}
	if video["room"] != "room-42" {
		t.Errorf("video.room = %v, want room-42", video["room"])
	}
	if video["roomJoin"] != true {
		t.Errorf("video.roomJoin = %v, want true", video["roomJoin"])
	}
	if video["roomCreate"] != true {
		t.Errorf("video.roomCreate = %v, want true", video["roomCreate"])
	}
	if video["canSubscribe"] != true {
		t.Errorf("video.canSubscribe = %v, want true", video["canSubscribe"])
	}
	sourcesAny, ok := video["canPublishSources"].([]any)
	if !ok {
		t.Fatalf("video.canPublishSources missing or wrong type: %+v", video)
	}
	var sources []string
	for _, s := range sourcesAny {
		sources = append(sources, s.(string))
	}
	want := []string{"microphone", "camera", "screen_share", "screen_share_audio"}
	if !reflect.DeepEqual(sources, want) {
		t.Errorf("video.canPublishSources = %v, want %v", sources, want)
	}
}

func TestMintToken_WrongSecret_FailsVerification(t *testing.T) {
	c := &Client{WSURL: "wss://lk.example.com", APIKey: "k", APISecret: "right-secret"}
	token, err := c.MintToken("u1", "n1", "r1")
	if err != nil {
		t.Fatalf("MintToken: %v", err)
	}
	if _, err := parseJWT(token, "wrong-secret"); err == nil {
		t.Fatal("parseJWT with wrong secret succeeded, want signature mismatch error")
	}
}

// twirpStub is a table-driven stub RoomService server for ListRooms/
// ListParticipants tests: routes by the Twirp method path, decodes the JSON
// body, and asserts the bearer token verifies against apiSecret.
func twirpStub(t *testing.T, apiSecret string, handlers map[string]func(t *testing.T, body map[string]any) any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if len(auth) < len(prefix) || auth[:len(prefix)] != prefix {
			t.Fatalf("request missing bearer token: %q", auth)
		}
		if _, err := parseJWT(auth[len(prefix):], apiSecret); err != nil {
			t.Fatalf("request token failed verification: %v", err)
		}
		method := r.URL.Path[len("/twirp/livekit.RoomService/"):]
		h, ok := handlers[method]
		if !ok {
			t.Fatalf("unexpected twirp method %q", method)
		}
		var body map[string]any
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
		}
		resp := h(t, body)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode stub response: %v", err)
		}
	}))
}

func TestListRooms(t *testing.T) {
	tests := []struct {
		name    string
		handler map[string]func(t *testing.T, body map[string]any) any
		want    []Room
	}{
		{
			name: "empty room skips ListParticipants",
			handler: map[string]func(t *testing.T, body map[string]any) any{
				"ListRooms": func(t *testing.T, body map[string]any) any {
					return map[string]any{"rooms": []map[string]any{{"name": "empty-room", "numParticipants": 0}}}
				},
			},
			want: []Room{{Name: "empty-room"}},
		},
		{
			// LiveKit Cloud emits proto field names, not protojson camelCase —
			// the field mis-parsing as 0 silently emptied everyone's presence
			// (found live, 2026-08-28).
			name: "snake_case num_participants also triggers the participant fetch",
			handler: map[string]func(t *testing.T, body map[string]any) any{
				"ListRooms": func(t *testing.T, body map[string]any) any {
					return map[string]any{"rooms": []map[string]any{{"name": "cloud-room", "num_participants": 1}}}
				},
				"ListParticipants": func(t *testing.T, body map[string]any) any {
					return map[string]any{"participants": []map[string]any{{"identity": "u1", "name": "Ada"}}}
				},
			},
			want: []Room{{Name: "cloud-room", Participants: []Participant{{Identity: "u1", Name: "Ada"}}}},
		},
		{
			name: "occupied room fetches participants",
			handler: map[string]func(t *testing.T, body map[string]any) any{
				"ListRooms": func(t *testing.T, body map[string]any) any {
					return map[string]any{"rooms": []map[string]any{{"name": "busy-room", "numParticipants": 2}}}
				},
				"ListParticipants": func(t *testing.T, body map[string]any) any {
					if body["room"] != "busy-room" {
						t.Fatalf("ListParticipants body = %+v, want room=busy-room", body)
					}
					return map[string]any{"participants": []map[string]any{
						{"identity": "u1", "name": "Ada"},
						{"identity": "u2", "name": "Grace"},
					}}
				},
			},
			want: []Room{{Name: "busy-room", Participants: []Participant{
				{Identity: "u1", Name: "Ada"},
				{Identity: "u2", Name: "Grace"},
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := twirpStub(t, "secret1", tt.handler)
			defer srv.Close()
			c := &Client{WSURL: srv.URL, APIKey: "key1", APISecret: "secret1"}
			got, err := c.ListRooms(context.Background())
			if err != nil {
				t.Fatalf("ListRooms: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListRooms() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListRooms_ServerError_Retryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := &Client{WSURL: srv.URL, APIKey: "key1", APISecret: "secret1"}
	_, err := c.ListRooms(context.Background())
	if !errors.Is(err, apperrs.ErrRetryable) {
		t.Fatalf("ListRooms (500) = %v, want ErrRetryable", err)
	}
}

func TestListRooms_Unauthorized_NotRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"msg":"invalid api key"}`))
	}))
	defer srv.Close()
	c := &Client{WSURL: srv.URL, APIKey: "key1", APISecret: "secret1"}
	_, err := c.ListRooms(context.Background())
	if !errors.Is(err, apperrs.ErrUnauthorized) {
		t.Fatalf("ListRooms (401) = %v, want ErrUnauthorized", err)
	}
	if errors.Is(err, apperrs.ErrRetryable) {
		t.Fatalf("ListRooms (401) = %v, must not be retryable (bad credential, not transient)", err)
	}
}
