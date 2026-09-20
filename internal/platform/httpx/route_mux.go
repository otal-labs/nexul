package httpx

import (
	"net/http"
	"strings"
)

// Route is a method and path registered with a tracked ServeMux.
type Route struct {
	Method string
	Path   string
}

// ServeMux delegates routing to net/http while retaining its method-bearing registrations.
type ServeMux struct {
	*http.ServeMux
	routes []Route
}

// NewServeMux creates a ServeMux that records method and path registrations.
func NewServeMux() *ServeMux {
	return &ServeMux{ServeMux: http.NewServeMux()}
}

// Handle registers a pattern with the underlying net/http ServeMux.
func (m *ServeMux) Handle(pattern string, handler http.Handler) {
	m.ServeMux.Handle(pattern, handler)
	m.record(pattern)
}

// HandleFunc registers a handler function with the underlying net/http ServeMux.
func (m *ServeMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.Handle(pattern, http.HandlerFunc(handler))
}

// Mount registers a gateway handler and propagates its tracked routes.
func (m *ServeMux) Mount(prefix string, handler http.Handler) {
	m.Handle(prefix, handler)
	m.Handle(prefix+"/", handler)
	if prefix != "/api" && !strings.HasPrefix(prefix, "/api/") {
		return
	}
	m.routes = append(m.routes, RoutesOf(handler)...)
}

// Routes returns the unique method and path registrations in registration order.
func (m *ServeMux) Routes() []Route {
	seen := make(map[Route]struct{}, len(m.routes))
	routes := make([]Route, 0, len(m.routes))
	for _, route := range m.routes {
		if _, ok := seen[route]; ok {
			continue
		}
		seen[route] = struct{}{}
		routes = append(routes, route)
	}
	return routes
}

func (m *ServeMux) record(pattern string) {
	method, path, ok := strings.Cut(pattern, " ")
	method = strings.ToUpper(method)
	if !ok || !isOpenAPIMethod(method) || !strings.HasPrefix(path, "/api/") {
		return
	}
	m.routes = append(m.routes, Route{Method: method, Path: path})
}

func isOpenAPIMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

type routedHandler struct {
	h      http.Handler
	routes []Route
}

func (h routedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.h.ServeHTTP(w, r)
}

func (h routedHandler) Routes() []Route {
	return h.routes
}

// WithRoutes preserves route metadata through a middleware wrapper.
func WithRoutes(handler http.Handler, routes []Route) http.Handler {
	if len(routes) == 0 {
		return handler
	}
	return routedHandler{h: handler, routes: routes}
}

// RoutesOf returns route metadata exposed by a handler, if any.
func RoutesOf(handler http.Handler) []Route {
	provider, ok := handler.(interface{ Routes() []Route })
	if !ok {
		return nil
	}
	return provider.Routes()
}
