package branding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/ee/license"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var testNow = time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

// pngBytes is the 8-byte PNG signature, which is all the type is judged by.
var pngBytes = []byte("\x89PNG\r\n\x1a\n" + "rest of an image")

type fakeRepo struct {
	mu     sync.Mutex
	rec    Record
	found  bool
	logo   []byte
	logoCT string
	events []eventbus.OutboxEvent
}

func (f *fakeRepo) Load(context.Context) (Record, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rec, f.found, nil
}

func (f *fakeRepo) Save(_ context.Context, r Record, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r.Logo = f.rec.Logo
	f.rec, f.found = r, true
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) Logo(context.Context) (string, []byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.logo == nil {
		return "", nil, apperrs.ErrNotFound
	}
	return f.logoCT, f.logo, nil
}

func (f *fakeRepo) SetLogo(_ context.Context, ct string, data []byte, version, _ string, _ time.Time, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logo, f.logoCT, f.found = data, ct, true
	f.rec.Logo = &Logo{ContentType: ct, Size: len(data), Version: version}
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) ClearLogo(_ context.Context, _ string, _ time.Time, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logo, f.rec.Logo = nil, nil
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rec, f.found, f.logo = Record{}, false, nil
	f.events = append(f.events, evts...)
	return nil
}

type fakeLicenses struct{ on bool }

func (f *fakeLicenses) Enabled(_ context.Context, feat license.Feature) bool {
	return f.on && feat == license.FeatureBranding
}

type fakeGate struct{}

func (fakeGate) RequireAnywhere(ctx context.Context, action permissions.Action) error {
	a, _ := identity.ActorFromCtx(ctx)
	if a.ID == "owner" || action == permissions.InstanceRead {
		return nil
	}
	return apperrs.ErrForbidden
}

func asOwner(t *testing.T) context.Context {
	return identity.WithActor(t.Context(), identity.Actor{ID: "owner"})
}

func asMember(t *testing.T) context.Context {
	return identity.WithActor(t.Context(), identity.Actor{ID: "member"})
}

func newService(unlocked bool) (*Service, *fakeRepo, *fakeLicenses) {
	repo, lic := &fakeRepo{}, &fakeLicenses{on: unlocked}
	s := NewService(repo, lic).WithGate(fakeGate{})
	s.now = func() time.Time { return testNow }
	return s, repo, lic
}

func ptr(s string) *string { return &s }

func TestService_WithoutAKey_TheLookIsTheDefaultAndTheSavedBrandingStays(t *testing.T) {
	t.Parallel()
	s, _, lic := newService(true)
	_, err := s.Update(asOwner(t), Patch{ProductName: ptr("Acme Hub"), Accent: ptr("#2563eb"), Theme: ptr(darkThemeFile)})
	require.NoError(t, err)
	_, err = s.SetLogo(asOwner(t), pngBytes)
	require.NoError(t, err)

	look, err := s.Look(t.Context())
	require.NoError(t, err)
	assert.True(t, look.Enabled)
	assert.Equal(t, "Acme Hub", look.ProductName)
	assert.Contains(t, look.LogoURL, "/api/branding/logo?v=")
	assert.True(t, look.Dark.Themed)
	assert.False(t, look.Light.Themed)
	assert.Contains(t, look.Light.Roles, "brand", "the accent alone still paints the other mode")

	lic.on = false
	look, err = s.Look(t.Context())
	require.NoError(t, err)
	assert.Equal(t, Look{}, look, "an expired or removed key shows the default look")
	assert.Equal(t, DefaultName, s.ProductName(t.Context()))
	_, _, err = s.Logo(t.Context())
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	saved, err := s.Get(asOwner(t))
	require.NoError(t, err)
	assert.False(t, saved.Unlocked)
	assert.Equal(t, "Acme Hub", saved.ProductName, "the saved branding stays for when a key comes back")
	assert.NotNil(t, saved.Dark)
	assert.NotNil(t, saved.Logo)

	_, err = s.Update(asOwner(t), Patch{ProductName: ptr("Other")})
	require.ErrorIs(t, err, apperrs.ErrForbidden, "without a key nothing new is saved")
	_, err = s.SetLogo(asOwner(t), pngBytes)
	require.ErrorIs(t, err, apperrs.ErrForbidden)

	lic.on = true
	look, err = s.Look(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Acme Hub", look.ProductName, "a renewed key shows it again")
}

func TestService_OnlyTheOwnerChangesIt_AndEveryoneSeesTheLook(t *testing.T) {
	t.Parallel()
	s, _, _ := newService(true)
	_, err := s.Update(asMember(t), Patch{ProductName: ptr("Acme")})
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = s.SetLogo(asMember(t), pngBytes)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = s.RemoveLogo(asMember(t))
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = s.Clear(asMember(t))
	require.ErrorIs(t, err, apperrs.ErrForbidden)

	_, err = s.Get(asMember(t))
	require.NoError(t, err, "a reader sees what is saved")
	_, err = s.Look(t.Context())
	require.NoError(t, err, "the look needs no sign-in")
}

func TestService_Update_ValidatesEachFieldAndKeepsTheRest(t *testing.T) {
	t.Parallel()
	s, repo, _ := newService(true)
	owner := asOwner(t)
	for name, patch := range map[string]Patch{
		"a name over the limit":  {ProductName: ptr(strings.Repeat("a", MaxNameRunes+1))},
		"a name with markup":     {ProductName: ptr("<b>Acme</b>")},
		"an accent not in hex":   {Accent: ptr("blue")},
		"a theme with no canvas": {Theme: ptr(`{"colors":{}}`)},
		"an unknown removal":     {RemoveTheme: "sepia"},
	} {
		_, err := s.Update(owner, patch)
		require.ErrorIs(t, err, apperrs.ErrInvalid, name)
	}
	assert.Empty(t, repo.events, "a refused change publishes nothing")

	_, err := s.Update(owner, Patch{ProductName: ptr("Acme"), Accent: ptr("#2563EB")})
	require.NoError(t, err)
	st, err := s.Update(owner, Patch{Theme: ptr(darkThemeFile)})
	require.NoError(t, err)
	assert.Equal(t, "Acme", st.ProductName, "a patch leaves the fields it omits")
	assert.Equal(t, "#2563eb", st.Accent, "an accent is kept in lower case")
	require.NotNil(t, st.Dark)
	assert.Nil(t, st.Light)
	assert.NotEmpty(t, st.Dark.Adjusted)

	st, err = s.Update(owner, Patch{Theme: ptr(lightThemeFile)})
	require.NoError(t, err)
	assert.NotNil(t, st.Dark, "a light theme fills the light slot and leaves the dark one")
	assert.NotNil(t, st.Light)

	st, err = s.Update(owner, Patch{RemoveTheme: "dark", ProductName: ptr("")})
	require.NoError(t, err)
	assert.Nil(t, st.Dark)
	assert.NotNil(t, st.Light)
	assert.Empty(t, st.ProductName, "an empty name goes back to Nexul's")
	st, err = s.Update(owner, Patch{RemoveTheme: "all"})
	require.NoError(t, err)
	assert.Nil(t, st.Light)
}

func TestService_Logo_IsJudgedByItsBytesAndBounded(t *testing.T) {
	t.Parallel()
	s, repo, _ := newService(true)
	owner := asOwner(t)
	for name, data := range map[string][]byte{
		"a script":                []byte("<script>alert(1)</script>"),
		"text called an image":    []byte("hello"),
		"an empty file":           {},
		"larger than the limit":   append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, MaxLogoBytes)...),
		"html wearing an svg tag": []byte("<html><svg></svg></html>"),
	} {
		_, err := s.SetLogo(owner, data)
		require.ErrorIs(t, err, apperrs.ErrInvalid, name)
	}

	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><circle r="4"/></svg>`)
	st, err := s.SetLogo(owner, svg)
	require.NoError(t, err)
	assert.Equal(t, "image/svg+xml", st.Logo.ContentType)
	ct, data, err := s.Logo(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "image/svg+xml", ct)
	assert.Equal(t, svg, data)

	first := st.Logo.Version
	st, err = s.SetLogo(owner, pngBytes)
	require.NoError(t, err)
	assert.NotEqual(t, first, st.Logo.Version, "new bytes get a new version, so a cached copy never goes stale")

	st, err = s.RemoveLogo(owner)
	require.NoError(t, err)
	assert.Nil(t, st.Logo)
	assert.Len(t, repo.events, 3)
}

func TestService_Clear_WorksWithoutAKeyAndPublishesIt(t *testing.T) {
	t.Parallel()
	s, repo, lic := newService(true)
	_, err := s.Update(asOwner(t), Patch{ProductName: ptr("Acme")})
	require.NoError(t, err)
	lic.on = false
	st, err := s.Clear(asOwner(t))
	require.NoError(t, err, "an instance whose key ran out can still wipe what it no longer shows")
	assert.Empty(t, st.ProductName)

	last := repo.events[len(repo.events)-1]
	assert.Equal(t, TopicChanged, last.Topic)
	assert.True(t, last.Payload.(ChangedEvent).Cleared)
	assert.Equal(t, "owner", last.Payload.(ChangedEvent).ChangedBy)
}

func TestService_ProductName_FallsBackToNexul(t *testing.T) {
	t.Parallel()
	s, _, _ := newService(true)
	assert.Equal(t, DefaultName, s.ProductName(t.Context()), "no branding saved")
	_, err := s.Update(asOwner(t), Patch{Accent: ptr("#2563eb")})
	require.NoError(t, err)
	assert.Equal(t, DefaultName, s.ProductName(t.Context()), "branding with no name")
	_, err = s.Update(asOwner(t), Patch{ProductName: ptr("Acme Hub")})
	require.NoError(t, err)
	assert.Equal(t, "Acme Hub", s.ProductName(t.Context()))
}

func TestIndex_PutsTheBrandInTheHeadAndLeavesTheDefaultPageAlone(t *testing.T) {
	t.Parallel()
	page := []byte(`<head>
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <link rel="icon" type="image/png" sizes="32x32" href="/favicon-32.png" />
    <link rel="apple-touch-icon" href="/apple-touch-icon.png" />
    <link rel="manifest" href="/manifest.webmanifest" />
    <title>Nexul</title>
  </head>`)
	s, _, lic := newService(true)
	assert.Equal(t, page, s.Index(t.Context(), page), "no branding saved")

	_, err := s.Update(asOwner(t), Patch{ProductName: ptr(`Acme "Hub" & Sons`), Accent: ptr("#2563eb")})
	require.NoError(t, err)
	_, err = s.SetLogo(asOwner(t), pngBytes)
	require.NoError(t, err)
	got := string(s.Index(t.Context(), page))
	assert.Contains(t, got, "<title>Acme &#34;Hub&#34; &amp; Sons</title>")
	assert.NotContains(t, got, "/favicon", "the instance's own mark replaces every Nexul icon")
	assert.NotContains(t, got, "apple-touch-icon")
	assert.Contains(t, got, `<link rel="manifest"`)
	assert.Contains(t, got, `<link rel="icon" href="/api/branding/logo?v=`)
	assert.Contains(t, got, `:root:not(.dark){--brand:`)
	assert.Contains(t, got, `:root.dark{--brand:`)
	assert.Contains(t, got, `Acme \"Hub\" \u0026 Sons`, "the name sits in the data block as escaped JSON")
	assert.Equal(t, 1, strings.Count(got, `id="branding-data"`))

	lic.on = false
	assert.Equal(t, page, s.Index(t.Context(), page), "an expired key puts nothing in the head")
}

func TestHandler_ServesTheLookToAnyoneAndTheLogoSandboxed(t *testing.T) {
	t.Parallel()
	s, _, _ := newService(true)
	_, err := s.Update(asOwner(t), Patch{ProductName: ptr("Acme Hub")})
	require.NoError(t, err)
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	_, err = s.SetLogo(asOwner(t), svg)
	require.NoError(t, err)
	h := NewHandler(s)

	rec := httptest.NewRecorder()
	h.PublicRoutes().ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/branding", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var look Look
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &look))
	assert.Equal(t, "Acme Hub", look.ProductName)

	rec = httptest.NewRecorder()
	h.PublicRoutes().ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, look.LogoURL, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/svg+xml", rec.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "sandbox", "an SVG logo never runs its script, even opened on its own")
	assert.Contains(t, rec.Header().Get("Cache-Control"), "immutable")
	assert.Equal(t, svg, rec.Body.Bytes())

	rec = httptest.NewRecorder()
	h.PublicRoutes().ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/branding/logo", nil))
	assert.NotContains(t, rec.Header().Get("Cache-Control"), "immutable", "an address without its version is never cached for good")
}

func TestHandler_OwnerRoutes_NameTheFieldARefusalBelongsTo(t *testing.T) {
	t.Parallel()
	s, _, _ := newService(true)
	h := NewHandler(s).Routes()
	do := func(ctx context.Context, method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, path, strings.NewReader(body)))
		return rec
	}

	rec := do(asOwner(t), http.MethodPatch, "/api/instance/branding", `{"theme_file":"{}"}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"theme_file":[`)
	rec = do(asOwner(t), http.MethodPatch, "/api/instance/branding", `{"accent":"blue"}`)
	assert.Contains(t, rec.Body.String(), `"accent":[`)
	rec = do(asOwner(t), http.MethodPut, "/api/instance/branding/logo", "not an image")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"logo":[`)
	rec = do(asOwner(t), http.MethodPut, "/api/instance/branding/logo", string(make([]byte, MaxLogoBytes+10)))
	require.Equal(t, http.StatusBadRequest, rec.Code, "a body past the limit is refused before it is read whole")

	rec = do(asMember(t), http.MethodPatch, "/api/instance/branding", `{"product_name":"Acme"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	rec = do(asOwner(t), http.MethodPatch, "/api/instance/branding", `{"product_name":"Acme Hub"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"product_name":"Acme Hub"`)
	rec = do(asOwner(t), http.MethodDelete, "/api/instance/branding", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"product_name":""`)
}
