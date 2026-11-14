package branding

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestValidateCSS_RefusesEveryWayToLoadAFile(t *testing.T) {
	t.Parallel()
	refused := map[string]string{
		"a remote url":                  `body{background:url(https://evil.example/x.png)}`,
		"a quoted url":                  `body{background:url("https://evil.example/x.png")}`,
		"a protocol-relative url":       `body{background:url(//evil.example/x.png)}`,
		"a same-site path":              `body{background:url(/api/users)}`,
		"a url hidden in a token":       `:root{--x:URL( 'https://evil.example')}`,
		"a url between string comments": `a{content:"/*"}b{background:url(https://evil.example)}c{content:"*/"}`,
		"an import":                     `@import "https://evil.example/x.css";`,
		"an image set":                  `a{background:image-set("https://evil.example/x.png" 1x)}`,
		"a prefixed image set":          `a{background:-webkit-image-set("x.png" 1x)}`,
		"image()":                       `a{background:image("x.png")}`,
		"src()":                         `a{background:src("x.png")}`,
		"an escape":                     `a{background:u\72l(https://evil.example)}`,
		"a closing tag":                 `a{}</style><script>alert(1)</script>`,
		"too long":                      strings.Repeat("a{}", MaxCSSBytes),
	}
	for name, css := range refused {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := validateCSS(css)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
}

func TestValidateCSS_KeepsInlineImagesAndPlainRules(t *testing.T) {
	t.Parallel()
	for _, css := range []string{
		`:root.dark{--brand:#7c5cff !important}`,
		`.light-field{background:url( "data:image/png;base64,iVBORw0KGgo=") center/cover}`,
		`a{mask:url(#m)}`,
		`.app-frame{background-image:linear-gradient(#000,#111)}`,
		`[data-slot="button"]{border-radius:0}`,
	} {
		got, err := validateCSS("  " + css + "\n")
		require.NoError(t, err, css)
		assert.Equal(t, css, got)
	}
}

func TestService_CSS_ShowsOnEveryPageAndARefusalKeepsWhatWasSaved(t *testing.T) {
	t.Parallel()
	s, _, _ := newService(true)
	_, err := s.Update(asOwner(t), Patch{CSS: ptr(`[data-app-sidebar]{font-family:monospace}`)})
	require.NoError(t, err)

	_, err = s.Update(asOwner(t), Patch{CSS: ptr(`a{background:url(https://evil.example)}`)})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	got, err := s.Get(asOwner(t))
	require.NoError(t, err)
	assert.Equal(t, `[data-app-sidebar]{font-family:monospace}`, got.CSS)

	look, err := s.Look(t.Context())
	require.NoError(t, err)
	assert.Equal(t, got.CSS, look.CSS)
	page := string(s.Index(t.Context(), []byte("<head></head>")))
	assert.Contains(t, page, `<style id="branding-css">[data-app-sidebar]{font-family:monospace}</style>`)

	_, err = s.Update(asOwner(t), Patch{CSS: ptr("")})
	require.NoError(t, err)
	assert.NotContains(t, string(s.Index(t.Context(), []byte("<head></head>"))), "branding-css")
}

func TestStableSelectors_StayInTheWebSource(t *testing.T) {
	t.Parallel()
	var source strings.Builder
	err := filepath.WalkDir(filepath.Join("..", "..", "web", "src"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".tsx") || strings.Contains(path, ".test.") {
			return err
		}
		b, err := os.ReadFile(path)
		source.Write(b)
		return err
	})
	require.NoError(t, err)
	for _, s := range StableSelectors {
		assert.Contains(t, source.String(), s.Probe, "%s is promised to branding css; keep it or drop it from the guide", s.Selector)
		assert.Contains(t, StylingGuide(), s.Selector)
	}
}
