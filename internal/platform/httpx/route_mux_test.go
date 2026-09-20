package httpx

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServeMuxTracksMethodRoutes(t *testing.T) {
	mux := NewServeMux()
	mux.HandleFunc("GET /api/tickets", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("POST /api/tickets", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("/api/", func(http.ResponseWriter, *http.Request) {})

	assert.Equal(t, []Route{
		{Method: http.MethodGet, Path: "/api/tickets"},
		{Method: http.MethodPost, Path: "/api/tickets"},
	}, mux.Routes())
}

func TestServeMuxMountPropagatesRoutes(t *testing.T) {
	child := NewServeMux()
	child.HandleFunc("GET /api/new-route", func(http.ResponseWriter, *http.Request) {})

	parent := NewServeMux()
	parent.Mount("/api", child)

	assert.Equal(t, []Route{{Method: http.MethodGet, Path: "/api/new-route"}}, parent.Routes())
}

func TestServeMuxMountIgnoresRoutesOutsideAPIGateway(t *testing.T) {
	child := NewServeMux()
	child.HandleFunc("GET /api/new-route", func(http.ResponseWriter, *http.Request) {})

	parent := NewServeMux()
	parent.Mount("/auth", child)

	assert.Empty(t, parent.Routes())
}
