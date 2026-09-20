package voice

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookHandler_ValidDeliveryUpdatesOccupancy(t *testing.T) {
	cred := &fakeCredentials{client: livekitTestClient("secret1")}
	bus := &fakePublisher{}
	svc := NewService(Config{
		Conversations: &fakeConversations{},
		Credentials:   cred,
		Users:         &fakeUserNames{},
		Bus:           bus,
	})
	h := NewWebhookHandler(svc, cred, nil)

	body := []byte(`{"event":"participant_joined","room":{"name":"conv-1"},"participant":{"identity":"u-1","name":"Ada"}}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/livekit", bytes.NewReader(body))
	req.Header.Set("Authorization", signTestWebhook(t, "secret1", body, time.Minute))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	got := svc.Occupancy()
	if len(got["conv-1"]) != 1 || got["conv-1"][0] != (Occupant{Identity: "u-1", Name: "Ada"}) {
		t.Errorf("Occupancy() = %+v, want conv-1 to hold u-1/Ada", got)
	}
}

func TestWebhookHandler_WrongSignature_Rejected(t *testing.T) {
	cred := &fakeCredentials{client: livekitTestClient("secret1")}
	svc := NewService(Config{Conversations: &fakeConversations{}, Credentials: cred, Users: &fakeUserNames{}, Bus: &fakePublisher{}})
	h := NewWebhookHandler(svc, cred, nil)

	body := []byte(`{"event":"participant_joined","room":{"name":"conv-1"},"participant":{"identity":"u-1","name":"Ada"}}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/livekit", bytes.NewReader(body))
	req.Header.Set("Authorization", signTestWebhook(t, "wrong-secret", body, time.Minute))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if got := svc.Occupancy(); len(got) != 0 {
		t.Errorf("Occupancy() after a rejected delivery = %+v, want empty", got)
	}
}

func TestWebhookHandler_NoConnectorConfigured_AcceptsWithoutApplying(t *testing.T) {
	cred := &fakeCredentials{err: errNotConfigured}
	svc := NewService(Config{Conversations: &fakeConversations{}, Credentials: cred, Users: &fakeUserNames{}, Bus: &fakePublisher{}})
	h := NewWebhookHandler(svc, cred, nil)

	body := []byte(`{"event":"participant_joined","room":{"name":"conv-1"},"participant":{"identity":"u-1","name":"Ada"}}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/livekit", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer whatever")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// 200 so LiveKit doesn't burn its retry budget on a delivery this
	// instance has no credential to verify against.
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := svc.Occupancy(); len(got) != 0 {
		t.Errorf("Occupancy() = %+v, want empty (delivery was never applied)", got)
	}
}
