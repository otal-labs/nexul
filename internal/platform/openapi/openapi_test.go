package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

func TestRegisterAndJSON(t *testing.T) {
	s := New(Info{Title: "Nexul API", Version: "v1"})
	s.AddSecuritySchemes()
	s.Register("GET", "/api/tickets", "List tickets", "tickets")
	s.Register("POST", "/api/tickets", "Create a ticket", "tickets")
	s.Register("GET", "/api/integrations/{id}", "Get an install", "integrations")

	doc, err := s.JSON()
	require.NoError(t, err)

	var parsed struct {
		OpenAPI    string               `json:"openapi"`
		Info       Info                 `json:"info"`
		Paths      map[string]*PathItem `json:"paths"`
		Tags       []tagEntry           `json:"tags"`
		Components map[string]any       `json:"components"`
	}
	require.NoError(t, json.Unmarshal(doc, &parsed))
	assert.Equal(t, "3.0.3", parsed.OpenAPI)
	assert.Equal(t, "Nexul API", parsed.Info.Title)

	tickets := parsed.Paths["/api/tickets"]
	require.NotNil(t, tickets)
	require.NotNil(t, tickets.Get)
	assert.Equal(t, "List tickets", tickets.Get.Summary)
	require.NotNil(t, tickets.Post)
	require.NotNil(t, parsed.Paths["/api/integrations/{id}"].Get)

	assert.Equal(t, []tagEntry{{Name: "integrations"}, {Name: "tickets"}}, parsed.Tags)
	require.NotNil(t, parsed.Components)
}

func TestReplaceRegistration(t *testing.T) {
	s := New(Info{Title: "x", Version: "1"})
	s.Register("GET", "/api/x", "first")
	s.Register("GET", "/api/x", "second")
	doc, err := s.JSON()
	require.NoError(t, err)
	var parsed struct {
		Paths map[string]*PathItem `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(doc, &parsed))
	assert.Equal(t, "second", parsed.Paths["/api/x"].Get.Summary)
}

func TestRegisterMountedRoutes(t *testing.T) {
	s := New(Info{Title: "x", Version: "1"})
	s.RegisterMountedRoutes([]httpx.Route{{Method: http.MethodGet, Path: "/api/new-route"}})

	doc, err := s.JSON()
	require.NoError(t, err)
	var parsed struct {
		Paths map[string]*PathItem `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(doc, &parsed))

	require.NotNil(t, parsed.Paths["/api/new-route"])
	assert.Equal(t, "API operation", parsed.Paths["/api/new-route"].Get.Summary)
}

func TestDefaultResponses(t *testing.T) {
	s := New(Info{Title: "x", Version: "1"})
	s.Register("GET", "/api/x", "x")
	doc, err := s.JSON()
	require.NoError(t, err)
	var parsed struct {
		Paths map[string]*PathItem `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(doc, &parsed))
	require.NotNil(t, parsed.Paths["/api/x"].Get.Responses)
	assert.NotEmpty(t, parsed.Paths["/api/x"].Get.Responses["200"])
}

func TestHandler(t *testing.T) {
	s := New(Info{Title: "x", Version: "1"})
	s.Register("GET", "/api/x", "x")
	h := s.Handler()

	t.Run("serves openapi json", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		var doc map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
		assert.Equal(t, "3.0.3", doc["openapi"])
	})

	t.Run("serves swagger ui", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "swagger-ui")
	})

	t.Run("unknown path is 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
