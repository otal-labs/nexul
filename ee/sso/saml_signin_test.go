package sso

import (
	"encoding/base64"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/testsaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// signInHarness is a harness whose SAML provider is the returned test IdP, with Nexul's key pair made.
func signInHarness(t *testing.T, licensed bool) (*harness, *testIdP) {
	t.Helper()
	h := samlHarness(t, licensed, nil)
	idp := newTestIdP(t, idpEntity)
	key, cert, err := newSPKeyPair("nexul.example.com", time.Now())
	require.NoError(t, err)
	h.saml.st = SAMLSettings{
		IdP: idp.metadata(), EmailAttribute: "email", NameAttribute: "name", DisplayName: "Acme SSO", JoinRoleID: "member",
		SPKey: key, SPCertificate: cert,
	}
	return h, idp
}

// started starts a sign-in and returns the id of the AuthnRequest it sent.
func started(t *testing.T, h *harness) string {
	t.Helper()
	target, err := h.svc.StartSAML(t.Context())
	require.NoError(t, err)
	u, err := url.Parse(target)
	require.NoError(t, err)
	raw, err := testsaml.ParseRedirectRequest(u)
	require.NoError(t, err)
	var req saml.AuthnRequest
	require.NoError(t, xml.Unmarshal(raw, &req))
	return req.ID
}

func TestStartSAML_SendsASignedRequestForThisInstanceAndKeepsItsID(t *testing.T) {
	t.Parallel()
	h, _ := signInHarness(t, true)

	target, err := h.svc.StartSAML(t.Context())

	require.NoError(t, err)
	u, err := url.Parse(target)
	require.NoError(t, err)
	assert.Equal(t, "https://idp.example.com/sso", u.Scheme+"://"+u.Host+u.Path)
	for _, param := range []string{"SAMLRequest", "RelayState", "SigAlg", "Signature"} {
		assert.NotEmpty(t, u.Query().Get(param), param)
	}
	raw, err := testsaml.ParseRedirectRequest(u)
	require.NoError(t, err)
	var req saml.AuthnRequest
	require.NoError(t, xml.Unmarshal(raw, &req))
	assert.Equal(t, "https://nexul.example.com/auth/saml/acs", req.AssertionConsumerServiceURL)
	assert.Equal(t, "https://nexul.example.com/auth/saml/metadata", req.Issuer.Value)
	assert.Contains(t, h.saml.requests, req.ID)
}

func TestStartSAML_WithNoSAMLProvider_IsRefused(t *testing.T) {
	t.Parallel()
	h := samlHarness(t, true, nil)
	_, err := h.svc.StartSAML(t.Context())
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestCompleteSAML_AResponseThatDoesNotProveItself_IsRefused(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	tests := []struct {
		name string
		// respond builds the posted response for the request id; idp is the instance's provider.
		respond func(t *testing.T, idp *testIdP, id string) string
		want    error
	}{
		{"unsigned assertion", func(t *testing.T, idp *testIdP, id string) string {
			r, _ := idp.response(t, assertionOpts{requestID: id, unsigned: true})
			return r
		}, errSAMLRefused},
		{"signed by a certificate not in the metadata", func(t *testing.T, _ *testIdP, id string) string {
			r, _ := newTestIdP(t, idpEntity).response(t, assertionOpts{requestID: id})
			return r
		}, errSAMLRefused},
		{"tampered after signing", func(t *testing.T, idp *testIdP, id string) string {
			_, el := idp.response(t, assertionOpts{requestID: id})
			for _, v := range el.FindElements(".//AttributeValue") {
				if v.Text() == "alice@example.com" {
					v.SetText("mallory@example.com")
				}
			}
			return encodeResponse(t, el)
		}, errSAMLRefused},
		{"a second unsigned assertion beside the signed one", func(t *testing.T, idp *testIdP, id string) string {
			_, el := idp.response(t, assertionOpts{requestID: id})
			_, forged := idp.response(t, assertionOpts{requestID: id, nameID: "mallory", email: "mallory@example.com", unsigned: true})
			el.InsertChildAt(2, forged.SelectElement("Assertion"))
			return encodeResponse(t, el)
		}, errSAMLRefused},
		{"expired", func(t *testing.T, idp *testIdP, id string) string {
			r, _ := idp.response(t, assertionOpts{requestID: id, issued: now.Add(-time.Minute), notOnOrAfter: now.Add(-70 * time.Second)})
			return r
		}, errSAMLRefused},
		{"not valid yet, past 60 seconds of skew", func(t *testing.T, idp *testIdP, id string) string {
			r, _ := idp.response(t, assertionOpts{requestID: id, issued: now, notBefore: now.Add(2 * time.Minute)})
			return r
		}, errSAMLRefused},
		{"for another service provider", func(t *testing.T, idp *testIdP, id string) string {
			r, _ := idp.response(t, assertionOpts{requestID: id, audience: "https://other.example.com/saml"})
			return r
		}, errSAMLRefused},
		{"from another issuer", func(t *testing.T, idp *testIdP, id string) string {
			r, _ := idp.response(t, assertionOpts{requestID: id, issuer: "https://other.example.com"})
			return r
		}, errSAMLRefused},
		{"an unknown request", func(t *testing.T, idp *testIdP, _ string) string {
			r, _ := idp.response(t, assertionOpts{requestID: "id-nobody-sent"})
			return r
		}, errSAMLRefused},
		{"no request at all, started at the provider", func(t *testing.T, idp *testIdP, _ string) string {
			_, el := idp.response(t, assertionOpts{requestID: "x"})
			el.RemoveAttr("InResponseTo")
			return encodeResponse(t, el)
		}, errSAMLRefused},
		{"an encrypted assertion", func(t *testing.T, idp *testIdP, id string) string {
			_, el := idp.response(t, assertionOpts{requestID: id})
			el.CreateElement("saml:EncryptedAssertion")
			return encodeResponse(t, el)
		}, errSAMLRefused},
		{"a transient NameID", func(t *testing.T, idp *testIdP, id string) string {
			r, _ := idp.response(t, assertionOpts{requestID: id, nameIDFormat: string(saml.TransientNameIDFormat)})
			return r
		}, errSAMLTransient},
		{"no email attribute", func(t *testing.T, idp *testIdP, id string) string {
			r, _ := idp.response(t, assertionOpts{requestID: id, email: "-"})
			return r
		}, errUnverified},
		{"not base64", func(*testing.T, *testIdP, string) string { return "%%%" }, errSAMLRefused},
		{"not XML", func(*testing.T, *testIdP, string) string { return base64.StdEncoding.EncodeToString([]byte("nope")) }, errSAMLRefused},
		{"larger than the cap", func(*testing.T, *testIdP, string) string { return strings.Repeat("A", maxSAMLResponse+4) }, errSAMLRefused},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h, idp := signInHarness(t, true)
			id := started(t, h)

			_, err := h.svc.CompleteSAML(t.Context(), tt.respond(t, idp, id))

			require.ErrorIs(t, err, tt.want)
			assert.Empty(t, h.accounts.created)
			assert.Empty(t, h.accounts.linked)
		})
	}
}

func TestCompleteSAML_AResponseIsTakenOnce(t *testing.T) {
	t.Parallel()
	h, idp := signInHarness(t, true)
	posted, _ := idp.response(t, assertionOpts{requestID: started(t, h)})

	_, err := h.svc.CompleteSAML(t.Context(), posted)
	require.NoError(t, err)
	_, err = h.svc.CompleteSAML(t.Context(), posted)

	require.ErrorIs(t, err, errSAMLRefused, "a replayed response finds its request gone")
}

func TestCompleteSAML_ARequestOlderThanTenMinutes_IsRefused(t *testing.T) {
	t.Parallel()
	h, idp := signInHarness(t, true)
	id := started(t, h)
	h.saml.requests[id] = time.Now().Add(-11 * time.Minute)
	posted, _ := idp.response(t, assertionOpts{requestID: id})

	_, err := h.svc.CompleteSAML(t.Context(), posted)

	require.ErrorIs(t, err, errSAMLRefused)
}

func TestCompleteSAML_AnAccountThatMayNotSignInOrJoin_IsRefused(t *testing.T) {
	t.Parallel()
	key := idpEntity + "#alice-1"
	for name, c := range map[string]struct {
		setup func(*harness)
		want  error
	}{
		"disabled":                 {func(h *harness) { h.accounts.bySubject[key] = Account{UserID: "u-1", Status: "disabled"} }, errDisabled},
		"removed, found by email":  {func(h *harness) { h.accounts.byEmail["alice@example.com"] = Account{UserID: "u-1", Status: "removed"} }, errRemoved},
		"the join role is Owner's": {func(h *harness) { h.accounts.roles["member"] = false }, errNotReady},
		"a lapsed key and no link": {func(h *harness) { h.svc.cfg.Licenses = fakeLicenses{} }, errLapsed},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h, idp := signInHarness(t, true)
			c.setup(h)
			posted, _ := idp.response(t, assertionOpts{requestID: started(t, h)})

			_, err := h.svc.CompleteSAML(t.Context(), posted)

			require.ErrorIs(t, err, c.want)
			assert.Empty(t, h.accounts.created)
		})
	}
}

func TestCompleteSAML_AFirstSignIn_MakesAnAccountAndTheNextFindsIt(t *testing.T) {
	t.Parallel()
	h, idp := signInHarness(t, true)
	posted, _ := idp.response(t, assertionOpts{requestID: started(t, h)})

	userID, err := h.svc.CompleteSAML(t.Context(), posted)

	require.NoError(t, err)
	assert.Equal(t, "member", h.accounts.created[userID])
	assert.Equal(t, Identity{Provider: ProtocolSAML, Key: idpEntity + "#alice-1", Email: "alice@example.com", Name: "Alice"}, h.accounts.linked[userID])

	h.accounts.bySubject[idpEntity+"#alice-1"] = Account{UserID: userID, Status: "active"}
	again, _ := idp.response(t, assertionOpts{requestID: started(t, h)})
	second, err := h.svc.CompleteSAML(t.Context(), again)
	require.NoError(t, err)
	assert.Equal(t, userID, second)
	assert.Len(t, h.accounts.created, 1)
}

func TestCompleteSAML_AnEmailOfAGoogleSignIn_LinksThatAccount(t *testing.T) {
	t.Parallel()
	h, idp := signInHarness(t, true)
	h.accounts.byEmail["alice@example.com"] = Account{UserID: "u-alice", Status: "active"}
	posted, _ := idp.response(t, assertionOpts{requestID: started(t, h)})

	userID, err := h.svc.CompleteSAML(t.Context(), posted)

	require.NoError(t, err)
	assert.Equal(t, "u-alice", userID)
	assert.Equal(t, ProtocolSAML, h.accounts.linked["u-alice"].Provider)
	assert.Empty(t, h.accounts.created)
}

func TestCompleteSAML_TheAttributeIsReadByNameOrFriendlyName(t *testing.T) {
	t.Parallel()
	h, idp := signInHarness(t, true)
	h.saml.st.EmailAttribute = "mail"
	posted, _ := idp.response(t, assertionOpts{requestID: started(t, h), edit: func(a *etree.Element) {
		for _, attr := range a.FindElements(".//Attribute") {
			if attr.SelectAttrValue("Name", "") == "email" {
				attr.CreateAttr("FriendlyName", "mail")
			}
		}
	}})

	userID, err := h.svc.CompleteSAML(t.Context(), posted)

	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", h.accounts.linked[userID].Email)
}

// The library's advisories of 2026-11-07 are pinned here, so a downgrade to a version without their fixes fails.
func TestCompleteSAML_TheFixedAdvisoriesStayFixed(t *testing.T) {
	t.Parallel()
	t.Run("a signature with two references is refused, though its value is good", func(t *testing.T) {
		t.Parallel()
		h, idp := signInHarness(t, true)
		_, el := idp.response(t, assertionOpts{requestID: started(t, h)})
		assertion := el.SelectElement("Assertion")
		info := assertion.FindElement("./Signature/SignedInfo")
		info.AddChild(info.SelectElement("Reference").Copy())
		idp.resign(t, assertion)

		_, err := h.svc.CompleteSAML(t.Context(), encodeResponse(t, el))

		require.ErrorIs(t, err, errSAMLRefused)
		assert.Contains(t, err.Error(), "Reference", "refused for the second reference, not for a broken signature")
	})
	t.Run("the same re-signing with one reference is accepted, so the case above is real", func(t *testing.T) {
		t.Parallel()
		h, idp := signInHarness(t, true)
		_, el := idp.response(t, assertionOpts{requestID: started(t, h)})
		idp.resign(t, el.SelectElement("Assertion"))

		_, err := h.svc.CompleteSAML(t.Context(), encodeResponse(t, el))

		require.NoError(t, err)
	})
	t.Run("an attribute in another namespace is not read as the email", func(t *testing.T) {
		t.Parallel()
		h, idp := signInHarness(t, true)
		posted, _ := idp.response(t, assertionOpts{requestID: started(t, h), edit: func(a *etree.Element) {
			stmt := a.FindElement(".//AttributeStatement")
			foreign := etree.NewElement("x:Attribute")
			foreign.CreateAttr("xmlns:x", "urn:example:foreign")
			foreign.CreateAttr("Name", "email")
			foreign.CreateElement("x:AttributeValue").SetText("mallory@example.com")
			stmt.InsertChildAt(0, foreign)
		}})

		userID, err := h.svc.CompleteSAML(t.Context(), posted)

		require.NoError(t, err)
		assert.Equal(t, "alice@example.com", h.accounts.linked[userID].Email)
	})
}

func TestSAMLHandler_SignInLandsOnLoginWithASessionOrTheReason(t *testing.T) {
	t.Parallel()
	h, idp := signInHarness(t, true)
	b := &fakeBrowser{}
	routes := NewHandler(h.svc, b).PublicRoutes()

	rec := serve(t, routes, http.MethodGet, "/auth/saml", "")
	require.Equal(t, http.StatusFound, rec.Code)
	u, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	raw, err := testsaml.ParseRedirectRequest(u)
	require.NoError(t, err)
	var req saml.AuthnRequest
	require.NoError(t, xml.Unmarshal(raw, &req))
	posted, _ := idp.response(t, assertionOpts{requestID: req.ID})

	rec = postForm(t, routes, url.Values{"SAMLResponse": {posted}})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Len(t, b.minted, 1, "a session is made for the account")
	assert.Equal(t, "https://nexul.example.com/login?token=ses_"+b.minted[0], rec.Header().Get("Location"))

	rec = postForm(t, routes, url.Values{"SAMLResponse": {posted}})
	assert.Equal(t, "https://nexul.example.com/login?error=sso_failed", rec.Header().Get("Location"), "the reason is logged, never sent")

	h.saml.st = SAMLSettings{}
	rec = serve(t, routes, http.MethodGet, "/auth/saml", "")
	assert.Equal(t, "https://nexul.example.com/login?error=not_configured", rec.Header().Get("Location"))
}

func TestSAMLHandler_AResponseWithoutAnEmail_LandsOnTheUnverifiedReason(t *testing.T) {
	t.Parallel()
	h, idp := signInHarness(t, true)
	posted, _ := idp.response(t, assertionOpts{requestID: started(t, h), email: "-"})

	rec := postForm(t, NewHandler(h.svc, &fakeBrowser{}).PublicRoutes(), url.Values{"SAMLResponse": {posted}})

	assert.Equal(t, "https://nexul.example.com/login?error=sso_email_unverified", rec.Header().Get("Location"))
}

func TestSSOOnly_WithSAMLAsTheProvider_RefusesTheOIDCCallback(t *testing.T) {
	t.Parallel()
	h, _ := signInHarness(t, true)
	h.saml.st.Only = true
	state := &http.Cookie{Name: stateCookie, Value: "s"}

	rec := serve(t, NewHandler(h.svc, &fakeBrowser{}).PublicRoutes(), http.MethodGet, "/auth/oidc/callback?state=s&code=c", "", state)

	assert.True(t, h.svc.OnlySSO(t.Context()), "every other method is off")
	assert.Equal(t, "https://nexul.example.com/login?error=not_configured", rec.Header().Get("Location"))
}

func postForm(t *testing.T, h http.Handler, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/saml/acs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
