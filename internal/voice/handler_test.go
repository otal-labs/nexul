package voice

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
