// Package paging is the stretch of a list a use-case reads in SQL, the same for the HTTP gateway and MCP (ADR 0140).
package paging

// DefaultLimit and MaxLimit bound every paged list.
const (
	DefaultLimit = 50
	MaxLimit     = 100
)

// Window is Limit items of a list's order after its first Offset.
type Window struct {
	Offset int
	Limit  int
}

// Clamped is the window a list reads: a limit below 1 is DefaultLimit, one above MaxLimit is MaxLimit, and a
// negative offset is 0, so a bad argument falls back instead of failing or reading the whole table.
func (w Window) Clamped() Window {
	if w.Limit < 1 {
		w.Limit = DefaultLimit
	}
	return Window{Offset: max(w.Offset, 0), Limit: min(w.Limit, MaxLimit)}
}
