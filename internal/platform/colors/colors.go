// Package colors defines what every owner-configurable color field accepts: a suggested hue or any hex color.
package colors

import "regexp"

// Color is a suggested hue name or a #rrggbb hex; Valid("") is deliberately false so callers check for "unset" before validating.
type Color string

// The five hues match web/src/components/board/ticketTypeColor.tsx's TYPE_FALLBACK_PALETTE.
const (
	Cyan    Color = "cyan"
	Emerald Color = "emerald"
	Orange  Color = "orange"
	Fuchsia Color = "fuchsia"
	Lime    Color = "lime"
)

var valid = map[Color]bool{
	Cyan:    true,
	Emerald: true,
	Orange:  true,
	Fuchsia: true,
	Lime:    true,
}

var hex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Valid reports whether c is one of the suggested hues or a #rrggbb hex color.
func Valid(c Color) bool {
	return valid[c] || hex.MatchString(string(c))
}

// All returns the suggested palette in a stable order, for API/MCP schemas
// that hand the frontend or an agent the picker's option list.
func All() []Color {
	return []Color{Cyan, Emerald, Orange, Fuchsia, Lime}
}
