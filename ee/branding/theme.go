package branding

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Appearance is the mode a VS Code theme is written for.
type Appearance string

const (
	Light Appearance = "light"
	Dark  Appearance = "dark"
)

// MaxThemeBytes bounds an imported theme file; real ones run to a few hundred kilobytes at most.
const MaxThemeBytes = 1 << 20

const (
	textFloor   = 7.0
	mutedFloor  = 5.0
	accentFloor = 4.5
	labelFloor  = 4.5
	minApart    = 1.1
)

// Theme is what an imported VS Code theme contributes: its canvas, its text colours, its accent and its error colour.
// The surface steps between them are derived, so a theme that paints one flat background still gets raised panels.
type Theme struct {
	Name       string     `json:"name"`
	Appearance Appearance `json:"appearance"`
	Canvas     string     `json:"canvas"`
	Text       string     `json:"text"`
	TextMuted  string     `json:"text_muted,omitempty"`
	Accent     string     `json:"accent"`
	Error      string     `json:"error,omitempty"`
}

// ImportTheme reads a VS Code color theme file (JSONC) and keeps the colours Nexul's tokens are built from.
func ImportTheme(source string) (Theme, error) {
	if len(source) > MaxThemeBytes {
		return Theme{}, fmt.Errorf("%w: this theme file is larger than %d KB", apperrs.ErrInvalid, MaxThemeBytes>>10)
	}
	doc, err := parseJSONC(source)
	if err != nil {
		return Theme{}, err
	}
	colors, _ := doc["colors"].(map[string]any)
	pick := func(keys ...string) (rgb, float64, bool) {
		for _, k := range keys {
			if s, ok := colors[k].(string); ok {
				if c, a, ok := parseHex(s); ok {
					return c, a, true
				}
			}
		}
		return rgb{}, 0, false
	}
	canvas, canvasAlpha, ok := pick("editor.background", "editorPane.background")
	if !ok {
		hint := ""
		if _, includes := doc["include"]; includes {
			hint = " It includes another theme file; upload a single file that holds its own colors."
		}
		return Theme{}, fmt.Errorf(`%w: this theme has no "editor.background" color, so there is nothing to build the instance's colors from.%s`, apperrs.ErrInvalid, hint)
	}
	canvas = canvas.over(rgb{1, 1, 1}, canvasAlpha)
	t := Theme{Name: themeName(doc), Appearance: appearanceOf(doc, canvas), Canvas: canvas.hex()}
	flat := func(keys ...string) string {
		c, a, ok := pick(keys...)
		if !ok {
			return ""
		}
		return c.over(canvas, a).hex()
	}
	t.Text = flat("editor.foreground", "foreground")
	t.TextMuted = flat("descriptionForeground", "disabledForeground")
	t.Error = flat("editorError.foreground", "errorForeground")
	for _, key := range []string{"focusBorder", "button.background", "textLink.foreground", "activityBarBadge.background", "progressBar.background", "badge.background"} {
		if c := flat(key); c != "" && contrast(hexRGB(c), canvas) >= minApart {
			t.Accent = c
			break
		}
	}
	return t, nil
}

func hexRGB(h string) rgb {
	c, _, _ := parseHex(h)
	return c
}

func appearanceOf(doc map[string]any, canvas rgb) Appearance {
	switch strings.ToLower(fmt.Sprint(doc["type"])) {
	case "light", "hc-light":
		return Light
	case "dark", "hc-black":
		return Dark
	}
	if canvas.luminance() < 0.179 {
		return Dark
	}
	return Light
}

func themeName(doc map[string]any) string {
	for _, key := range []string{"displayName", "name"} {
		if raw, ok := doc[key].(string); ok {
			if n := humanize(raw); n != "" {
				return truncateRunes(n, 48)
			}
		}
	}
	return "VS Code theme"
}

// humanize turns a slug such as one-dark-pro into One Dark Pro and leaves a written name alone.
func humanize(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.ContainsFunc(s, unicode.IsSpace) || !strings.ContainsAny(s, "-_.") {
		return s
	}
	words := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '_' || r == '.' })
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// Surfaces are the steps a palette stacks from its canvas, in OKLCH, each a step from the one below.
type surfaces struct {
	background, surface2, card, popover, secondary, muted, border, input lch
}

func (s surfaces) all() []rgb {
	return []rgb{s.background.rgb(), s.surface2.rgb(), s.card.rgb(), s.popover.rgb(), s.secondary.rgb(), s.muted.rgb()}
}

// ladder builds the surface steps the shipped palettes use: dark ones climb from the canvas, light ones fall from a
// near-white card, so a card always stands a step above the panel it sits on.
func ladder(mode Appearance, canvas lch) surfaces {
	if mode == Dark {
		base := lch{math.Min(canvas.L, 0.40), math.Min(canvas.C, 0.03), canvas.H}
		at := func(d float64) lch { return base.withL(base.L + d) }
		return surfaces{base, at(0.025), at(0.07), at(0.092), at(0.11), at(0.092), at(0.16), at(0.21)}
	}
	card := lch{math.Min(1, math.Max(canvas.L, 0.93)), math.Min(canvas.C, 0.02), canvas.H}
	card.C *= 0.4
	at := func(d float64) lch {
		l := card.withL(card.L - d)
		l.C = math.Min(canvas.C, 0.02)
		return l
	}
	return surfaces{at(0.05), at(0.035), card, card, at(0.078), at(0.066), at(0.115), at(0.155)}
}

// Notes name the colours of a theme that were moved to stay readable: text, secondary text, accent, error color.
type Notes []string

func (n *Notes) add(msg string) {
	for _, have := range *n {
		if have == msg {
			return
		}
	}
	*n = append(*n, msg)
}

// fitInk moves ink toward the readable end until it holds floor on every surface; it returns whether it moved.
func fitInk(seed lch, mode Appearance, on []rgb, floor float64) (lch, bool) {
	step := 0.01
	if mode == Light {
		step = -0.01
	}
	ink, moved := seed, false
	for range 100 {
		if holds(ink.rgb(), on, floor) {
			return ink, moved
		}
		ink = ink.withL(ink.L + step)
		moved = true
	}
	if mode == Dark {
		return lch{L: 1}, true
	}
	return lch{L: 0}, true
}

func holds(c rgb, on []rgb, floor float64) bool {
	for _, s := range on {
		if contrast(c, s) < floor {
			return false
		}
	}
	return true
}

// fitAccent moves an accent along its lightness until it reads as text on every surface and carries a readable label.
func fitAccent(seed lch, mode Appearance, on []rgb) (accent lch, label lch, moved bool) {
	step := 0.01
	if mode == Light {
		step = -0.01
	}
	dark := lch{0.16, 0.03, seed.H}
	white := lch{L: 1}
	prefer, other := white, dark
	if mode == Dark {
		prefer, other = dark, white
	}
	accent = seed
	for range 100 {
		if holds(accent.rgb(), on, accentFloor) {
			if contrast(prefer.rgb(), accent.rgb()) >= labelFloor {
				return accent, prefer, moved
			}
			if contrast(other.rgb(), accent.rgb()) >= labelFloor {
				return accent, other, moved
			}
		}
		accent = accent.withL(accent.L + step)
		moved = true
	}
	return accent, prefer, moved
}

var (
	defaultAccent = map[Appearance]lch{Light: {0.565, 0.19, 35}, Dark: {0.68, 0.2, 35}}
	defaultError  = map[Appearance]lch{Light: {0.52, 0.2, 15}, Dark: {0.7, 0.18, 15}}
)

// Roles derives every token this theme sets, for its own appearance. accent, when not empty, replaces the theme's own.
func (t Theme) Roles(accent string) (map[string]string, Notes) {
	var notes Notes
	canvas := hexRGB(t.Canvas).lch()
	st := ladder(t.Appearance, canvas)
	on := st.all()

	inkSeed := hexRGB(firstNonEmpty(t.Text, t.Canvas)).lch()
	if t.Text == "" {
		inkSeed = lch{L: map[Appearance]float64{Dark: 0.95, Light: 0.2}[t.Appearance], C: 0.01, H: canvas.H}
	}
	ink, moved := fitInk(inkSeed, t.Appearance, on, textFloor)
	if moved && t.Text != "" {
		notes.add("text")
	}
	mutedSeed := hexRGB(firstNonEmpty(t.TextMuted, t.Text, t.Canvas)).lch()
	if t.TextMuted == "" {
		mutedSeed = ink.withL(ink.L + map[Appearance]float64{Dark: -0.2, Light: 0.2}[t.Appearance])
	}
	muted, moved := fitInk(mutedSeed, t.Appearance, on, mutedFloor)
	if moved && t.TextMuted != "" {
		notes.add("secondary text")
	}

	fitted, label := t.accentRoles(accent, on, &notes)
	destructive := t.errorRole(fitted, on, &notes)

	roles := map[string]string{
		"background": st.background.css(), "surface-2": st.surface2.css(), "card": st.card.css(), "popover": st.popover.css(),
		"secondary": st.secondary.css(), "muted": st.muted.css(), "border": st.border.css(), "input": st.input.css(),
		"foreground": ink.css(), "card-foreground": ink.css(), "popover-foreground": ink.css(), "secondary-foreground": ink.css(),
		"muted-foreground": muted.css(), "accent-foreground": ink.css(),
		"accent":  inkOverlay(t.Appearance),
		"primary": fitted.css(), "primary-foreground": label.css(), "brand": fitted.css(), "brand-foreground": label.css(),
		"ring": fitted.css(), "destructive": destructive.css(),
	}
	if t.Appearance == Dark && st.background.L < 0.06 {
		roles["well-shade"] = "oklch(1 0 0 / 0.025)"
	}
	return roles, notes
}

func inkOverlay(mode Appearance) string {
	if mode == Dark {
		return "oklch(1 0 0 / 0.07)"
	}
	return "oklch(0.2 0.01 265 / 0.06)"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func hueApart(a, b float64) float64 {
	d := math.Abs(a - b)
	return math.Min(d, 360-d)
}

// accentRoles fits the accent, the instance's own when one is set, else the theme's, else Nexul's ember.
func (t Theme) accentRoles(accent string, on []rgb, notes *Notes) (lch, lch) {
	seed := firstNonEmpty(accent, t.Accent)
	start := defaultAccent[t.Appearance]
	if seed != "" {
		start = hexRGB(seed).lch()
	}
	fitted, label, moved := fitAccent(start, t.Appearance, on)
	if moved && seed != "" {
		notes.add("accent")
	}
	return fitted, label
}

// errorRole fits the error colour, turned away from the accent's hue so a failure never reads as the accent.
func (t Theme) errorRole(accent lch, on []rgb, notes *Notes) lch {
	seed := defaultError[t.Appearance]
	if t.Error != "" {
		seed = hexRGB(t.Error).lch()
	}
	if accent.C > 0.04 && hueApart(seed.H, accent.H) < 25 {
		seed.H = math.Mod(accent.H+35, 360)
	}
	fitted, moved := fitInk(seed, t.Appearance, on, accentFloor)
	if moved && t.Error != "" {
		notes.add("error color")
	}
	return fitted
}
