package branding

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const darkThemeFile = `{
  // A dark theme with comments, like the ones editors ship
  "name": "acme-night",
  "type": "dark",
  "colors": {
    "editor.background": "#1b1e28",
    "editor.foreground": "#6c7a96", /* too dim to read on the panels */
    "descriptionForeground": "#4a5268",
    "focusBorder": "#1b1e2833",
    "button.background": "#7c5cff",
    "errorForeground": "#8a1f2d",
  },
}`

const lightThemeFile = `{
  "name": "Acme Day",
  "colors": {
    "editor.background": "#fdf6e3",
    "editor.foreground": "#93a1a1",
    "textLink.foreground": "#268bd2"
  }
}`

func TestImportTheme_ReadsWhatNexulBuildsItsColoursFrom(t *testing.T) {
	t.Parallel()
	dark, err := ImportTheme(darkThemeFile)
	require.NoError(t, err)
	assert.Equal(t, Theme{
		Name: "Acme Night", Appearance: Dark, Canvas: "#1b1e28", Text: "#6c7a96", TextMuted: "#4a5268", Accent: "#7c5cff", Error: "#8a1f2d",
	}, dark, "comments and a trailing comma are read, and a focus border that is mostly transparent is flattened onto the canvas and skipped for standing too close to it")

	light, err := ImportTheme(lightThemeFile)
	require.NoError(t, err)
	assert.Equal(t, Light, light.Appearance, "a theme with no type is judged by its canvas")
	assert.Equal(t, "#268bd2", light.Accent)
}

func TestImportTheme_RefusesAFileItCannotBuildFrom(t *testing.T) {
	t.Parallel()
	for name, source := range map[string]string{
		"not json":            `colors: nope`,
		"an array":            `[1, 2]`,
		"no canvas":           `{"colors": {"editor.foreground": "#ffffff"}}`,
		"includes another":    `{"include": "./base.json", "colors": {}}`,
		"larger than allowed": `{"colors": {"editor.background": "#000"}, "pad": "` + string(make([]byte, MaxThemeBytes)) + `"}`,
	} {
		_, err := ImportTheme(source)
		require.ErrorIs(t, err, apperrs.ErrInvalid, name)
	}
	_, err := ImportTheme(`{"include": "./base.json"}`)
	assert.ErrorContains(t, err, "upload a single file")
}

// oklab turns an emitted `oklch(L C H [/ A])` value back into its colour so the checks below read what a browser paints.
var oklchValue = regexp.MustCompile(`^oklch\(([\d.]+) ([\d.]+) ([\d.]+)(?: / ([\d.]+))?\)$`)

func parseRole(t *testing.T, value string) (lch, float64) {
	t.Helper()
	m := oklchValue.FindStringSubmatch(value)
	require.NotNil(t, m, value)
	f := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
	alpha := 1.0
	if m[4] != "" {
		alpha = f(m[4])
	}
	return lch{f(m[1]), f(m[2]), f(m[3])}, alpha
}

func role(t *testing.T, roles map[string]string, name string) rgb {
	t.Helper()
	v, ok := roles[name]
	require.True(t, ok, "no %s role", name)
	c, _ := parseRole(t, v)
	return c.rgb()
}

func lab(c lch) [3]float64 {
	h := c.H * math.Pi / 180
	return [3]float64{c.L, c.C * math.Cos(h), c.C * math.Sin(h)}
}

// failing mirrors the pairs web/src/lib/themeContrast.test.ts holds every palette to: text on each surface, the label on
// the accent, the accent as text, the current row's tint, and a dark card standing a step above its panel.
func failing(t *testing.T, mode Appearance, roles map[string]string) []string {
	t.Helper()
	bg, _ := parseRole(t, roles["background"])
	card, _ := parseRole(t, roles["card"])
	panelFill := card
	if mode == Dark {
		a, b := lab(card), lab(bg)
		var mix [3]float64
		for i := range mix {
			mix[i] = a[i]*0.6 + b[i]*0.4
		}
		panelFill = lch{mix[0], math.Hypot(mix[1], mix[2]), math.Atan2(mix[2], mix[1]) * 180 / math.Pi}
	}
	canvas, fill := bg.rgb(), panelFill.rgb()
	panel := rgb{}
	for i := range panel {
		panel[i] = fill[i]*0.85 + canvas[i]*0.15
	}
	surfaces := map[string]rgb{"background": canvas, "surface-2": role(t, roles, "surface-2"), "card": role(t, roles, "card"), "popover": role(t, roles, "popover"), "panel": panel}
	tint := 0.12
	if mode == Dark {
		tint = 0.16
	}
	var out []string
	check := func(name string, got, floor float64) {
		if got < floor {
			out = append(out, fmt.Sprintf("%s: %.2f < %.1f", name, got, floor))
		}
	}
	for name, s := range surfaces {
		check("foreground on "+name, contrast(role(t, roles, "foreground"), s), 4.5)
		check("muted-foreground on "+name, contrast(role(t, roles, "muted-foreground"), s), 4.5)
		check("brand as text on "+name, contrast(role(t, roles, "brand"), s), 4.5)
		brand := role(t, roles, "brand")
		check("foreground on the current row over "+name, contrast(role(t, roles, "foreground"), rgb{brand[0]*tint + s[0]*(1-tint), brand[1]*tint + s[1]*(1-tint), brand[2]*tint + s[2]*(1-tint)}), 4.5)
	}
	check("brand-foreground on brand", contrast(role(t, roles, "brand-foreground"), role(t, roles, "brand")), 4.5)
	check("destructive on card", contrast(role(t, roles, "destructive"), surfaces["card"]), 4.5)
	if mode == Dark {
		check("card above panel", (card.L-lightnessOf(panel))/0.029, 1)
	}
	return out
}

func lightnessOf(c rgb) float64 { return c.lch().L }

func TestTheme_Roles_KeepTextReadable(t *testing.T) {
	t.Parallel()
	dark, err := ImportTheme(darkThemeFile)
	require.NoError(t, err)
	light, err := ImportTheme(lightThemeFile)
	require.NoError(t, err)
	themes := map[string]Theme{
		"dim text on a dark canvas":         dark,
		"pale text on a cream canvas":       light,
		"pure black canvas":                 {Name: "Black", Appearance: Dark, Canvas: "#000000", Text: "#ffffff", Accent: "#ff3b3b"},
		"pure white canvas":                 {Name: "White", Appearance: Light, Canvas: "#ffffff", Text: "#000000", Accent: "#ffee00"},
		"a bright dark canvas":              {Name: "Slate", Appearance: Dark, Canvas: "#5a5f70", Text: "#9aa0b4", Accent: "#3a3a40"},
		"no text or accent given":           {Name: "Bare", Appearance: Dark, Canvas: "#101820"},
		"an error colour on the accent hue": {Name: "Red", Appearance: Light, Canvas: "#f4f4f4", Text: "#333333", Accent: "#d62d3a", Error: "#d62d3a"},
	}
	for name, theme := range themes {
		roles, _ := theme.Roles("")
		assert.Empty(t, failing(t, theme.Appearance, roles), name)
	}
}

func TestTheme_Roles_SaysWhatItMoved(t *testing.T) {
	t.Parallel()
	dark, err := ImportTheme(darkThemeFile)
	require.NoError(t, err)
	_, notes := dark.Roles("")
	assert.Contains(t, notes, "text")
	assert.Contains(t, notes, "error color")

	fine := Theme{Name: "Fine", Appearance: Dark, Canvas: "#101010", Text: "#ffffff", Accent: "#4a9eff"}
	_, notes = fine.Roles("")
	assert.Empty(t, notes, "a readable theme is taken as it is")
}

func TestTheme_Roles_InstanceAccentReplacesTheThemesOwn(t *testing.T) {
	t.Parallel()
	theme := Theme{Name: "Teal", Appearance: Dark, Canvas: "#10181c", Text: "#e8f1f2", Accent: "#2bd9c4"}
	own, _ := theme.Roles("")
	set, _ := theme.Roles("#ff7a00")
	assert.NotEqual(t, own["brand"], set["brand"])
	assert.Equal(t, set["brand"], set["primary"], "a palette's primary is its accent")
	assert.Equal(t, set["brand"], set["ring"])
	assert.Equal(t, own["card"], set["card"], "the accent touches no surface")
	assert.Empty(t, failing(t, Dark, set))
}

func TestAccentRoles_StayReadableOnTheDefaultPalette(t *testing.T) {
	t.Parallel()
	for _, accent := range []string{"#ffffff", "#000000", "#ffee00", "#1d3557", "#ff7a00"} {
		for _, mode := range []Appearance{Light, Dark} {
			roles := accentRoles(accent, mode)
			require.Len(t, roles, 2, "the accent changes brand and its label, nothing else")
			s := defaultSurfaces[mode].all()
			assert.GreaterOrEqual(t, contrast(role(t, roles, "brand-foreground"), role(t, roles, "brand")), 4.5, "%s %s label", accent, mode)
			for _, surface := range s {
				assert.GreaterOrEqual(t, contrast(role(t, roles, "brand"), surface), 4.5, "%s %s as text", accent, mode)
			}
		}
	}
}
