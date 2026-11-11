package branding

// The default palette's surfaces, which the accent is fitted against when no theme is imported.
var defaultSurfaces = map[Appearance]surfaces{
	Light: {
		background: lch{0.95, 0.004, 265}, surface2: lch{0.965, 0.004, 265}, card: lch{L: 1}, popover: lch{L: 1},
		secondary: lch{0.92, 0.004, 265}, muted: lch{0.93, 0.004, 265},
	},
	Dark: {
		background: lch{0.13, 0.005, 265}, surface2: lch{0.155, 0.005, 265}, card: lch{0.21, 0.005, 265}, popover: lch{0.225, 0.006, 265},
		secondary: lch{0.25, 0.005, 265}, muted: lch{0.25, 0.005, 265},
	},
}

// accentRoles is the accent alone, as the one token value the default palette changes: brand, and the label on it.
func accentRoles(accent string, mode Appearance) map[string]string {
	fitted, label, _ := fitAccent(hexRGB(accent).lch(), mode, defaultSurfaces[mode].all())
	return map[string]string{"brand": fitted.css(), "brand-foreground": label.css()}
}
