package voice

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_Join(t *testing.T) {
	svc := NewService(Config{
		Conversations: &fakeConversations{voiceChannels: map[string]bool{"conv-1": true}},
		Credentials:   &fakeCredentials{client: livekitTestClient("secret1")},
		Users:         &fakeUserNames{names: map[string]string{"u-1": "Ada Lovelace"}},
		Bus:           &fakePublisher{},
	})
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/voice/conv-1/token", nil)
	req = req.WithContext(WithUserID(req.Context(), "u-1"))
	req.SetPathValue("conversationID", "conv-1")
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var got JoinToken
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Token == "" || got.WSURL == "" {
		t.Errorf("response = %+v, want both fields set", got)
	}
}

func TestHandler_Join_NotAVoiceChannel_Is4xx(t *testing.T) {
	svc := NewService(Config{
		Conversations: &fakeConversations{voiceChannels: map[string]bool{"conv-1": false}},
		Credentials:   &fakeCredentials{client: livekitTestClient("secret1")},
		Users:         &fakeUserNames{},
		Bus:           &fakePublisher{},
	})
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/voice/conv-1/token", nil)
	req = req.WithContext(WithUserID(req.Context(), "u-1"))
	req.SetPathValue("conversationID", "conv-1")
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code < 400 || rec.Code >= 500 {
		t.Fatalf("status = %d, want a 4xx", rec.Code)
	}
}

func TestHandler_Occupancy(t *testing.T) {
	svc := NewService(Config{
		Conversations: &fakeConversations{},
		Credentials:   &fakeCredentials{},
		Users:         &fakeUserNames{},
		Bus:           &fakePublisher{},
	})
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/voice/occupancy", nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got map[string][]Occupant
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("occupancy on a fresh service = %+v, want empty", got)
	}
}

func TestHandler_Listen(t *testing.T) {
	tests := []struct {
		name       string
		voice      bool
		userID     string
		wantStatus int
	}{
		{"a voice channel mints a listen token", true, "u-1", http.StatusOK},
		{"not a voice channel is a 4xx", false, "u-1", http.StatusBadRequest},
		{"no signed-in user is a 4xx", true, "", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(Config{
				Conversations: &fakeConversations{voiceChannels: map[string]bool{"conv-1": tt.voice}},
				Credentials:   &fakeCredentials{client: livekitTestClient("secret1")},
				Users:         &fakeUserNames{},
				Bus:           &fakePublisher{},
			})
			req := httptest.NewRequest(http.MethodPost, "/api/voice/conv-1/listen", nil)
			req = req.WithContext(WithUserID(req.Context(), tt.userID))
			rec := httptest.NewRecorder()
			NewHandler(svc).Routes().ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			if tt.wantStatus != http.StatusOK {
				return
			}
			var got JoinToken
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			claims, err := parseTestJWT(got.Token, "secret1")
			require.NoError(t, err)
			assert.Equal(t, "listen:u-1", claims["sub"])
		})
	}
}
