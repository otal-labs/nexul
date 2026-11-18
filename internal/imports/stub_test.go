package imports

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// stubAPI is a source's REST API as a test server: each path answers with its JSON, and every request is kept.
type stubAPI struct {
	mu       sync.Mutex
	srv      *httptest.Server
	requests []*http.Request
	// answer returns the status, headers and body for a request; nil body is a 404.
	answer func(r *http.Request) (int, http.Header, any)
}

func newStubAPI(t *testing.T, answer func(r *http.Request) (int, http.Header, any)) *stubAPI {
	t.Helper()
	s := &stubAPI{answer: answer}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.requests = append(s.requests, r)
		s.mu.Unlock()
		code, h, body := s.answer(r)
		for k, v := range h {
			w.Header()[k] = v
		}
		if body == nil && code == http.StatusOK {
			code = http.StatusNotFound
		}
		w.WriteHeader(code)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body) // test server: write errors are irrelevant
		}
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *stubAPI) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

// sleeps records each wait a source asks for instead of waiting.
type sleeps struct {
	mu  sync.Mutex
	got []time.Duration
}

func (s *sleeps) sleep(_ context.Context, d time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d > 0 {
		s.got = append(s.got, d)
	}
	return nil
}

func (s *sleeps) longest() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	var most time.Duration
	for _, d := range s.got {
		most = max(most, d)
	}
	return most
}
