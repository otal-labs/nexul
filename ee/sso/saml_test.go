package sso

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

type fakeSAMLRepo struct {
	st     SAMLSettings
	events []eventbus.OutboxEvent
	writes int
	// kept is the users TurnOnSAMLOnly was told to keep signed in; nil until it runs.
	kept []string
	// requests is each waiting sign-in request by id, with when it was sent.
	requests map[string]time.Time
}

func (r *fakeSAMLRepo) GetSAML(context.Context) (SAMLSettings, error) { return r.st, nil }
func (r *fakeSAMLRepo) SetSAML(_ context.Context, s SAMLSettings, e ...eventbus.OutboxEvent) error {
	r.st, r.writes = s, r.writes+1
	r.events = append(r.events, e...)
	return nil
}
func (r *fakeSAMLRepo) TurnOnSAMLOnly(_ context.Context, s SAMLSettings, keep []string, e ...eventbus.OutboxEvent) error {
	r.kept = append([]string{}, keep...)
	return r.SetSAML(context.Background(), s, e...)
}
func (r *fakeSAMLRepo) SaveSAMLRequest(_ context.Context, id string, at, dropBefore time.Time) error {
	if r.requests == nil {
		r.requests = map[string]time.Time{}
	}
	for k, sent := range r.requests {
		if sent.Before(dropBefore) {
			delete(r.requests, k)
		}
	}
	r.requests[id] = at
	return nil
}
func (r *fakeSAMLRepo) TakeSAMLRequest(_ context.Context, id string, since time.Time) (bool, error) {
	sent, ok := r.requests[id]
	delete(r.requests, id)
	return ok && !sent.Before(since), nil
}
func (r *fakeSAMLRepo) RefreshSAML(_ context.Context, metadataURL string, idp IdPMetadata, at time.Time, refreshErr string, e ...eventbus.OutboxEvent) error {
	if r.st.MetadataURL != metadataURL || r.st.IdP.EntityID != idp.EntityID {
		return nil
	}
	r.st.IdP, r.st.RefreshedAt, r.st.RefreshError, r.writes = idp, at, refreshErr, r.writes+1
	r.events = append(r.events, e...)
	return nil
}
func (r *fakeSAMLRepo) DeleteSAML(_ context.Context, e ...eventbus.OutboxEvent) error {
	r.st, r.writes = SAMLSettings{}, r.writes+1
	r.events = append(r.events, e...)
	return nil
}

// testCertificate is a fresh self-signed certificate as metadata carries it, base64 DER.
func testCertificate(t *testing.T) string {
	t.Helper()
	_, certPEM, err := newSPKeyPair("idp.example.com", time.Now())
	require.NoError(t, err)
	block, _ := pem.Decode([]byte(certPEM))
	return base64.StdEncoding.EncodeToString(block.Bytes)
}

// idpMetadata is an identity provider's metadata naming entityID, a redirect sign-in URL when ssoURL is set, and certs.
func idpMetadata(entityID, ssoURL string, certs ...string) string {
	var keys, sso string
	for _, c := range certs {
		keys += fmt.Sprintf(`<md:KeyDescriptor use="signing"><ds:KeyInfo><ds:X509Data><ds:X509Certificate>%s</ds:X509Certificate></ds:X509Data></ds:KeyInfo></md:KeyDescriptor>`, c)
	}
	if ssoURL != "" {
		sso = fmt.Sprintf(`<md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="%s"/>`, ssoURL)
	}
	return fmt.Sprintf(`<?xml version="1.0"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" entityID="%s">
<md:IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">%s%s</md:IDPSSODescriptor>
</md:EntityDescriptor>`, entityID, keys, sso)
}

const idpEntity = "https://idp.example.com/saml"

// samlHarness is a harness with no OIDC provider saved and the metadata URL served by an https test server.
func samlHarness(t *testing.T, licensed bool, metadata http.HandlerFunc) *harness {
	t.Helper()
	h := newHarness(t, licensed)
	h.repo.st = Settings{}
	srv := httptest.NewTLSServer(metadata)
	t.Cleanup(srv.Close)
	h.svc.cfg.FetchMetadata = fetchOver(srv.Client())
	return h
}

func serveText(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }
}

func ptr[T any](v T) *T { return &v }

func validPatch(xml string) SAMLPatch {
	return SAMLPatch{MetadataXML: ptr(xml), DisplayName: ptr("Acme SSO"), JoinRoleID: ptr("member")}
}

func TestUpdateSAML_MetadataThatNamesNoProvider_IsRefusedAndNothingIsSaved(t *testing.T) {
	t.Parallel()
	cert := testCertificate(t)
	for name, xml := range map[string]string{
		"not xml":                           "this is not metadata",
		"no signing certificate":            idpMetadata(idpEntity, "https://idp.example.com/sso"),
		"no single sign-on url":             idpMetadata(idpEntity, "", cert),
		"an http sign-in url":               idpMetadata(idpEntity, "http://idp.example.com/sso", cert),
		"no entity id":                      idpMetadata("", "https://idp.example.com/sso", cert),
		"a certificate that does not parse": idpMetadata(idpEntity, "https://idp.example.com/sso", "bm90IGEgY2VydA=="),
		"larger than the cap":               strings.Repeat(" ", maxMetadata+1),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := samlHarness(t, true, nil)

			_, err := h.svc.UpdateSAML(asActor(t), validPatch(xml))

			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Zero(t, h.saml.writes)
		})
	}
}

func TestUpdateSAML_AMetadataURLThatCannotBeRead_IsRefusedWithWhy(t *testing.T) {
	t.Parallel()
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()
	tests := []struct {
		name    string
		url     func(srv string) string
		handler http.HandlerFunc
		want    string
	}{
		{"http", func(string) string { return "http://idp.example.com/metadata" }, nil, "must use https"},
		{"resolves nowhere", func(string) string { return closed.URL }, nil, "cannot reach"},
		{"answers 404", func(srv string) string { return srv }, http.NotFound, "404"},
		{"larger than the cap", func(srv string) string { return srv }, serveText(strings.Repeat("x", maxMetadata+1)), "larger than"},
		{"not metadata", func(srv string) string { return srv }, serveText("<html></html>"), "not an identity provider"},
		{"times out", func(srv string) string { return srv }, func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() }, "cannot reach"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewTLSServer(tt.handler)
			t.Cleanup(srv.Close)
			h := newHarness(t, true)
			h.repo.st = Settings{}
			client := srv.Client()
			client.Timeout = 200 * time.Millisecond
			h.svc.cfg.FetchMetadata = fetchOver(client)

			_, err := h.svc.UpdateSAML(asActor(t), SAMLPatch{MetadataURL: ptr(tt.url(srv.URL)), DisplayName: ptr("Acme SSO"), JoinRoleID: ptr("member")})

			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Contains(t, err.Error(), tt.want)
			assert.Zero(t, h.saml.writes)
		})
	}
}

func TestUpdateSAML_RefusedWithoutWhatTheSaveNeeds(t *testing.T) {
	t.Parallel()
	xml := idpMetadata(idpEntity, "https://idp.example.com/sso", testCertificate(t))
	tests := []struct {
		name     string
		licensed bool
		setup    func(*harness)
		patch    SAMLPatch
		want     error
	}{
		{"the key is locked", false, nil, validPatch(xml), errNeedsLicense},
		{"an OIDC provider is saved", true, func(h *harness) { h.repo.st = Settings{IssuerURL: "https://login.example.com"} }, validPatch(xml), errOIDCSaved},
		{"no metadata", true, nil, SAMLPatch{DisplayName: ptr("Acme SSO"), JoinRoleID: ptr("member")}, apperrs.ErrInvalid},
		{"no display name", true, nil, SAMLPatch{MetadataXML: ptr(xml), JoinRoleID: ptr("member")}, apperrs.ErrInvalid},
		{"a name too long", true, nil, SAMLPatch{MetadataXML: ptr(xml), DisplayName: ptr(strings.Repeat("a", 41)), JoinRoleID: ptr("member")}, apperrs.ErrInvalid},
		{"an attribute too long", true, nil, SAMLPatch{MetadataXML: ptr(xml), DisplayName: ptr("Acme"), JoinRoleID: ptr("member"), EmailAttribute: ptr(strings.Repeat("a", 257))}, apperrs.ErrInvalid},
		{"a role nobody may join as", true, nil, SAMLPatch{MetadataXML: ptr(xml), DisplayName: ptr("Acme"), JoinRoleID: ptr("owner")}, apperrs.ErrInvalid},
		{"both metadata kinds", true, nil, SAMLPatch{MetadataXML: ptr(xml), MetadataURL: ptr("https://idp.example.com/m"), DisplayName: ptr("Acme"), JoinRoleID: ptr("member")}, apperrs.ErrInvalid},
		{"SSO-only with no Owner passkey", true, func(h *harness) {
			h.svc.cfg.OwnerHasPasskey = func(context.Context) (bool, error) { return false, nil }
		}, SAMLPatch{MetadataXML: ptr(xml), DisplayName: ptr("Acme"), JoinRoleID: ptr("member"), Only: ptr(true)}, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := samlHarness(t, tt.licensed, nil)
			if tt.setup != nil {
				tt.setup(h)
			}

			_, err := h.svc.UpdateSAML(asActor(t), tt.patch)

			require.ErrorIs(t, err, tt.want)
			assert.Zero(t, h.saml.writes)
		})
	}
}

func TestUpdate_OIDCWhileASAMLProviderIsSaved_IsRefused(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.saml.st = SAMLSettings{IdP: IdPMetadata{EntityID: idpEntity}}

	_, err := h.svc.Update(asActor(t), Patch{DisplayName: ptr("Acme")})

	require.ErrorIs(t, err, errSAMLSaved)
	assert.Contains(t, err.Error(), "remove it first")
	assert.Empty(t, h.repo.events)
}

func TestUpdateSAML_UploadedMetadata_SavesTheProviderWithANewKeyPairAndOneEvent(t *testing.T) {
	t.Parallel()
	h := samlHarness(t, true, nil)
	cert := testCertificate(t)
	xml := idpMetadata(idpEntity, "https://idp.example.com/sso", cert, cert)

	v, err := h.svc.UpdateSAML(asActor(t), validPatch(xml))

	require.NoError(t, err)
	assert.True(t, v.Active)
	assert.Equal(t, idpEntity, v.IdPEntityID)
	assert.Equal(t, 1, v.IdPCertificates, "a certificate named twice counts once")
	assert.Equal(t, "email", v.EmailAttribute, "the attribute defaults to email")
	assert.True(t, v.MetadataXMLSet)
	assert.Equal(t, SPValues{
		EntityID: "https://nexul.example.com/auth/saml/metadata", ACSURL: "https://nexul.example.com/auth/saml/acs",
		MetadataURL: "https://nexul.example.com/auth/saml/metadata", Certificate: h.saml.st.SPCertificate,
	}, v.SP)
	assert.Contains(t, h.saml.st.SPKey, "PRIVATE KEY")
	assert.Equal(t, xml, h.saml.st.MetadataXML, "uploaded metadata is kept as it was")
	require.Len(t, h.saml.events, 1)
	assert.Equal(t, TopicSAMLChanged, h.saml.events[0].Topic)
	assert.Equal(t, SAMLChangedEvent{DisplayName: "Acme SSO", EntityID: idpEntity, ChangedBy: "owner-1", ChangedAt: h.saml.events[0].Payload.(SAMLChangedEvent).ChangedAt}, h.saml.events[0].Payload)
	assert.Equal(t, Public{Active: true, Name: "Acme SSO", Protocol: ProtocolSAML}, h.svc.Public(t.Context()))
}

func TestUpdateSAML_AMetadataURL_IsFetchedAndReplacesUploadedMetadata(t *testing.T) {
	t.Parallel()
	cert := testCertificate(t)
	h := samlHarness(t, true, serveText(idpMetadata("https://other.example.com", "https://other.example.com/sso", cert)))
	_, err := h.svc.UpdateSAML(asActor(t), validPatch(idpMetadata(idpEntity, "https://idp.example.com/sso", cert)))
	require.NoError(t, err)
	key := h.saml.st.SPKey
	srv := httptest.NewTLSServer(serveText(idpMetadata("https://other.example.com", "https://other.example.com/sso", cert)))
	t.Cleanup(srv.Close)
	h.svc.cfg.FetchMetadata = fetchOver(srv.Client())

	v, err := h.svc.UpdateSAML(asActor(t), SAMLPatch{MetadataURL: ptr(srv.URL + "/metadata")})

	require.NoError(t, err)
	assert.Equal(t, "https://other.example.com", v.IdPEntityID)
	assert.Equal(t, srv.URL+"/metadata", v.MetadataURL)
	assert.False(t, v.MetadataXMLSet)
	assert.Equal(t, key, h.saml.st.SPKey, "the key pair is made once")
}

func TestUpdateSAML_APatchThatChangesNothing_WritesNothing(t *testing.T) {
	t.Parallel()
	h := samlHarness(t, true, nil)
	xml := idpMetadata(idpEntity, "https://idp.example.com/sso", testCertificate(t))
	_, err := h.svc.UpdateSAML(asActor(t), validPatch(xml))
	require.NoError(t, err)

	_, err = h.svc.UpdateSAML(asActor(t), validPatch(xml))
	require.NoError(t, err)
	_, err = h.svc.UpdateSAML(asActor(t), SAMLPatch{})
	require.NoError(t, err)

	assert.Equal(t, 1, h.saml.writes)
	assert.Len(t, h.saml.events, 1)
}

func TestUpdateSAML_TurningSSOOnlyOn_KeepsOnlyTheOwnersSignedIn(t *testing.T) {
	t.Parallel()
	h := samlHarness(t, true, nil)
	p := validPatch(idpMetadata(idpEntity, "https://idp.example.com/sso", testCertificate(t)))
	p.Only = ptr(true)

	_, err := h.svc.UpdateSAML(asActor(t), p)

	require.NoError(t, err)
	assert.Equal(t, []string{"owner-1"}, h.saml.kept)
	assert.True(t, h.svc.OnlySSO(t.Context()))
}

func TestRemoveSAML_WorksWithoutAKeyAndBringsTheOtherSignInsBack(t *testing.T) {
	t.Parallel()
	h := samlHarness(t, false, nil)
	h.saml.st = SAMLSettings{IdP: IdPMetadata{EntityID: idpEntity}, Only: true}

	v, err := h.svc.RemoveSAML(asActor(t))

	require.NoError(t, err)
	assert.False(t, v.Configured)
	assert.False(t, h.saml.st.Saved())
	require.Len(t, h.saml.events, 1)
	assert.True(t, h.saml.events[0].Payload.(SAMLChangedEvent).Removed)

	_, err = h.svc.RemoveSAML(asActor(t))
	require.NoError(t, err)
	assert.Len(t, h.saml.events, 1, "removing nothing publishes nothing")
}

func TestSAMLRoutes_NeedTheirPermissions(t *testing.T) {
	t.Parallel()
	h := samlHarness(t, true, nil)
	h.svc.cfg.Gate = fakeGate{err: apperrs.ErrForbidden}
	for name, call := range map[string]func() error{
		"view":   func() error { _, err := h.svc.SAMLView(asActor(t)); return err },
		"update": func() error { _, err := h.svc.UpdateSAML(asActor(t), SAMLPatch{}); return err },
		"check":  func() error { _, err := h.svc.CheckSAML(asActor(t), "email", SAMLPatch{}); return err },
		"remove": func() error { _, err := h.svc.RemoveSAML(asActor(t)); return err },
	} {
		require.ErrorIs(t, call(), apperrs.ErrForbidden, name)
	}
}

func TestCheckSAML(t *testing.T) {
	t.Parallel()
	cert := testCertificate(t)
	xml := idpMetadata(idpEntity, "https://idp.example.com/sso", cert)
	tests := []struct {
		name       string
		licensed   bool
		check      string
		patch      SAMLPatch
		wantDetail string
		wantErr    error
	}{
		{"locked", false, "metadata", SAMLPatch{MetadataXML: ptr(xml)}, "", errNeedsLicense},
		{"an unknown check", true, "issuer", SAMLPatch{}, "", apperrs.ErrInvalid},
		{"no metadata given or saved", true, "metadata", SAMLPatch{}, "", apperrs.ErrInvalid},
		{"metadata without a certificate", true, "metadata", SAMLPatch{MetadataXML: ptr(idpMetadata(idpEntity, "https://idp.example.com/sso"))}, "", apperrs.ErrInvalid},
		{"an empty email attribute", true, "email", SAMLPatch{EmailAttribute: ptr(" ")}, "", apperrs.ErrInvalid},
		{"metadata that parses", true, "metadata", SAMLPatch{MetadataXML: ptr(xml)}, "Read " + idpEntity + " with 1 signing certificate", nil},
		{"an email attribute", true, "email", SAMLPatch{EmailAttribute: ptr("http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress")}, "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := samlHarness(t, tt.licensed, nil)

			detail, err := h.svc.CheckSAML(asActor(t), tt.check, tt.patch)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantDetail, detail)
			assert.Zero(t, h.saml.writes, "a check stores nothing")
		})
	}
}

func TestSPMetadata_NamesTheEntityIDTheACSURLAndTheCertificate(t *testing.T) {
	t.Parallel()
	h := samlHarness(t, true, nil)
	_, err := NewHandler(h.svc, &fakeBrowser{}).svc.SPMetadata(t.Context())
	require.ErrorIs(t, err, apperrs.ErrNotFound, "no key pair until the first save")
	_, err = h.svc.UpdateSAML(asActor(t), validPatch(idpMetadata(idpEntity, "https://idp.example.com/sso", testCertificate(t))))
	require.NoError(t, err)

	rec := serve(t, NewHandler(h.svc, &fakeBrowser{}).PublicRoutes(), http.MethodGet, "/auth/saml/metadata", "")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "application/samlmetadata+xml", rec.Header().Get("Content-Type"))
	body := rec.Body.String()
	assert.Contains(t, body, `entityID="https://nexul.example.com/auth/saml/metadata"`)
	assert.Contains(t, body, `Location="https://nexul.example.com/auth/saml/acs"`)
	block, _ := pem.Decode([]byte(h.saml.st.SPCertificate))
	assert.Contains(t, body, base64.StdEncoding.EncodeToString(block.Bytes))
	assert.NotContains(t, body, `use="encryption"`, "encrypted assertions are not taken, so no encryption key is offered")
	assert.NotContains(t, body, "PRIVATE KEY")
}

func TestSAMLHandlers_AnswerAsTheUseCasesDo(t *testing.T) {
	t.Parallel()
	h := samlHarness(t, true, nil)
	routes := NewHandler(h.svc, &fakeBrowser{}).Routes()
	xml := idpMetadata(idpEntity, "https://idp.example.com/sso", testCertificate(t))
	body := fmt.Sprintf(`{"metadata_xml":%q,"display_name":"Acme SSO","join_role_id":"member"}`, xml)

	assert.Equal(t, http.StatusBadRequest, serve(t, routes, http.MethodPut, "/api/instance/saml", "{").Code)
	assert.Equal(t, http.StatusBadRequest, serve(t, routes, http.MethodPost, "/api/instance/saml/verify?check=metadata", "{").Code)
	assert.Equal(t, http.StatusBadRequest, serve(t, routes, http.MethodPut, "/api/instance/saml", `{"metadata_xml":"`+strings.Repeat("a", maxSAMLBody)+`"}`).Code, "a body over the cap is refused")
	assert.Equal(t, http.StatusBadRequest, serve(t, routes, http.MethodPost, "/api/instance/saml/verify?check=email", `{"email_attribute":""}`).Code)

	rec := serve(t, routes, http.MethodPost, "/api/instance/saml/verify?check=metadata", body)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "1 signing certificate")
	assert.Equal(t, http.StatusNoContent, serve(t, routes, http.MethodPost, "/api/instance/saml/verify?check=email", `{"email_attribute":"email"}`).Code)

	rec = serve(t, routes, http.MethodPut, "/api/instance/saml", body)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "PRIVATE KEY")
	rec = serve(t, routes, http.MethodGet, "/api/instance/saml", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"active":true`)
	assert.Equal(t, http.StatusConflict, serve(t, routes, http.MethodPut, "/api/instance/sso", `{"display_name":"x"}`).Code)

	rec = serve(t, routes, http.MethodDelete, "/api/instance/saml", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"configured":false`)
	assert.Equal(t, http.StatusNotFound, serve(t, NewHandler(h.svc, &fakeBrowser{}).PublicRoutes(), http.MethodGet, "/auth/saml/metadata", "").Code)
}

func TestSAMLHandlers_AFailedReadIsAnError(t *testing.T) {
	t.Parallel()
	h := samlHarness(t, true, nil)
	h.svc.cfg.Gate = fakeGate{err: apperrs.ErrForbidden}
	routes := NewHandler(h.svc, &fakeBrowser{}).Routes()
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/instance/saml"}, {http.MethodPut, "/api/instance/saml"},
		{http.MethodDelete, "/api/instance/saml"}, {http.MethodPost, "/api/instance/saml/verify?check=email"},
	} {
		assert.Equal(t, http.StatusForbidden, serve(t, routes, c.method, c.path, "{}").Code, c.path)
	}
}
