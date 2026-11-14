package branding

import (
	"fmt"
	"strings"
)

// StableSelector is a hook in the web app that branding CSS may rely on; Probe is the text the web source must hold
// for it, which a test checks so a rename cannot silently break a saved theme.
type StableSelector struct {
	Selector string
	What     string
	Probe    string
}

// StableSelectors are the selectors the styling guide promises to keep.
var StableSelectors = []StableSelector{
	{`.light-field`, "the backdrop behind every signed-in page", `"light-field"`},
	{`.app-frame`, "the panel each page floats in", `"app-frame"`},
	{`[data-app-sidebar]`, "the left sidebar", `data-app-sidebar`},
	{`nav[aria-label="Main"]`, "the sidebar's links", `aria-label="Main"`},
	{`[data-slot="button"]`, "every button", `data-slot="button"`},
	{`[data-slot="input"]`, "single-line text fields", `data-slot="input"`},
	{`[data-slot="textarea"]`, "multi-line text fields", `data-slot="textarea"`},
	{`[data-slot="bubble"]`, "a chat message's bubble", `data-slot="bubble"`},
	{`[data-slot="dialog-content"]`, "a dialog's panel", `data-slot="dialog-content"`},
	{`[data-slot="tabs-trigger"]`, "a tab", `data-slot="tabs-trigger"`},
	{`[data-slot="switch"]`, "an on and off switch", `data-slot="switch"`},
	{`[data-slot="label-chip"]`, "a ticket label", `data-slot="label-chip"`},
}

// StylingGuide is what an agent reads before writing branding CSS: the tokens to set, the selectors that stay, and
// what the CSS may not do.
func StylingGuide() string {
	var b strings.Builder
	b.WriteString(`Branding CSS is plain CSS added after Nexul's own styles, on every page for everyone, the sign-in page included. instance_update with branding.css saves it, and every open page repaints at once.

Set colour tokens rather than painting elements, so every component follows. Light mode is :root:not(.dark) and dark mode is :root.dark. Tokens: --background (the canvas), --surface-2, --card, --popover, --foreground, --muted-foreground, --border, --input, --ring, --brand (the one accent: buttons, focus, the active link) and --brand-foreground, --primary, --secondary, --accent, --muted. Corners: --shape-sm, --shape-md, --shape-lg, --shape-xl, --shape-2xl. Fonts: --font-sans-stack, --font-mono-stack, --font-display-stack, from fonts the device has. A person's own palette is set on the page element, so a token without !important gives way to it; add !important to hold a token for everyone. A whole palette can also come from a VS Code theme file (branding.theme_file).

Selectors that stay stable:
`)
	for _, s := range StableSelectors {
		fmt.Fprintf(&b, "- %s: %s\n", s.Selector, s.What)
	}
	fmt.Fprintf(&b, `
Nothing may load: url() takes only a data: address or an #id, and @import, image-set(), image(), src(), a backslash and < are refused. Write an image as a base64 data: address; the CSS is at most %d characters. Leave the status colours (success, warning, info, destructive) alone, since they carry meaning. Some app rules outweigh a plain selector, so a rule may need !important. You cannot see the result: change a little at a time and ask the person to look. An empty css removes it all.
`, MaxCSSBytes)
	return b.String()
}
