package issuesync

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestHandler_Routes(t *testing.T) {
	tests := []struct {
		name   string
		gate   error
		method string
		body   string
		want   int
	}{
		{"a body that is not JSON", nil, http.MethodPut, "{", http.StatusBadRequest},
		{"a caller who is not an owner", apperrs.ErrForbidden, http.MethodPut, `{"owner":"acme","repo":"app","enabled":true}`, http.StatusForbidden},
		{"a repository the project doesn't have", nil, http.MethodPut, `{"owner":"globex","repo":"site","enabled":true}`, http.StatusBadRequest},
		{"turning it on", nil, http.MethodPut, `{"owner":"acme","repo":"app","enabled":true}`, http.StatusOK},
		{"a reader who may not open the project", apperrs.ErrForbidden, http.MethodGet, "", http.StatusForbidden},
		{"listing", nil, http.MethodGet, "", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.svc.Gate = fakeGate{err: tt.gate}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/api/projects/p1/issue-sync", strings.NewReader(tt.body))
			NewHandler(f.svc).Routes().ServeHTTP(rec, req)
			assert.Equal(t, tt.want, rec.Code, rec.Body.String())
		})
	}
}
