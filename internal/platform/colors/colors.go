// Package colors defines the fixed suggested color palette shared by every owner-configurable color field.
package colors

// Color is a curated hue name; Valid("") is deliberately false so callers check for "unset" before validating.
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

// Valid reports whether c is one of the suggested palette hues.
func Valid(c Color) bool {
	return valid[c]
}

// All returns the suggested palette in a stable order, for API/MCP schemas
// that hand the frontend or an agent the picker's option list.
func All() []Color {
	return []Color{Cyan, Emerald, Orange, Fuchsia, Lime}
}
