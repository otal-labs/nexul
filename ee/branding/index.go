package branding

import (
	"bytes"
	"context"
	"encoding/json"
	"html"
	"regexp"
	"slices"
	"strings"
)

var (
	titleTag  = regexp.MustCompile(`<title>[^<]*</title>`)
	iconLinks = regexp.MustCompile(`[ \t]*<link rel="(?:icon|apple-touch-icon)"[^>]*>\n?`)
)

// Index puts the instance's own look into the page head before the web app loads, so the first paint already wears the
// brand: the tab title, the icon, the colours and the data the app would otherwise fetch. The page is left as it is while
// no key unlocks branding.
func (s *Service) Index(ctx context.Context, index []byte) []byte {
	l, err := s.Look(ctx)
	if err != nil || !l.Enabled {
		return index
	}
	out := index
	if l.ProductName != "" {
		out = titleTag.ReplaceAll(out, []byte("<title>"+html.EscapeString(l.ProductName)+"</title>"))
	}
	var head strings.Builder
	if l.LogoURL != "" {
		out = iconLinks.ReplaceAll(out, nil)
		head.WriteString(`<link rel="icon" href="` + html.EscapeString(l.LogoURL) + `" />` + "\n")
	}
	if css := l.css(); css != "" {
		head.WriteString(`<style id="branding-style">` + css + "</style>\n")
	}
	data, err := json.Marshal(l)
	if err != nil {
		return index
	}
	head.WriteString(`<script id="branding-data" type="application/json">` + string(data) + "</script>\n")
	return bytes.Replace(out, []byte("</head>"), []byte(head.String()+"</head>"), 1)
}

// css writes each mode's roles as custom properties; the light block names :not(.dark) so it never leaks into dark mode.
func (l Look) css() string {
	var b strings.Builder
	block := func(selector string, m ModeRoles) {
		if len(m.Roles) == 0 {
			return
		}
		names := make([]string, 0, len(m.Roles))
		for name := range m.Roles {
			names = append(names, name)
		}
		slices.Sort(names)
		b.WriteString(selector + "{")
		for _, name := range names {
			b.WriteString("--" + name + ":" + m.Roles[name] + ";")
		}
		b.WriteString("}")
	}
	block(":root:not(.dark)", l.Light)
	block(":root.dark", l.Dark)
	return b.String()
}
