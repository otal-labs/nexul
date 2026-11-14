package branding

import (
	"fmt"
	"regexp"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// MaxCSSBytes bounds the custom CSS: enough for dozens of rules and one small image written as a data: address.
const MaxCSSBytes = 20_000

var (
	// cssLoaders are the ways CSS fetches a file other than url(): each takes a bare string as an address.
	cssLoaders = regexp.MustCompile(`(?i)@import|(?:^|[^a-z0-9_-])(?:-webkit-)?(?:image-set|image|src)\s*\(`)
	cssURL     = regexp.MustCompile(`(?i)url\s*\(\s*['"]?\s*`)
)

// validateCSS refuses CSS that could fetch anything from another address, so a theme can never send what a page shows
// elsewhere. It refuses rather than strips, so the author learns which rule to change. A backslash is refused outright,
// since an escape can spell any of the checked words.
func validateCSS(css string) (string, error) {
	css = strings.TrimSpace(css)
	if len(css) > MaxCSSBytes {
		return "", fmt.Errorf("%w: the CSS is at most %d characters", apperrs.ErrInvalid, MaxCSSBytes)
	}
	if strings.Contains(css, `\`) {
		return "", fmt.Errorf("%w: the CSS cannot hold a backslash; write characters as themselves", apperrs.ErrInvalid)
	}
	if strings.Contains(css, "<") {
		return "", fmt.Errorf("%w: the CSS cannot hold <, since it sits in the page as a style element; write an SVG image as base64 or with %%3C", apperrs.ErrInvalid)
	}
	if m := cssLoaders.FindString(css); m != "" {
		return "", fmt.Errorf("%w: the CSS cannot load files, so %s is refused; write an image inline as url(data:...)", apperrs.ErrInvalid, strings.TrimSpace(m))
	}
	for _, loc := range cssURL.FindAllStringIndex(css, -1) {
		rest := strings.ToLower(css[loc[1]:])
		if strings.HasPrefix(rest, "data:") || strings.HasPrefix(rest, "#") {
			continue
		}
		return "", fmt.Errorf("%w: url() takes only a data: address or an #id, never a file from another address", apperrs.ErrInvalid)
	}
	return css, nil
}
