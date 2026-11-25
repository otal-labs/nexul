package sso

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const metadataURL = "https://idp.example.com/metadata"

// refreshHarness is a sign-in harness whose provider's metadata comes from metadataURL, last read two days ago, served
// by serve.
func refreshHarness(t *testing.T, serve func() ([]byte, error)) (*harness, *testIdP) {
	t.Helper()
	h, idp := signInHarness(t, true)
	h.saml.st.MetadataURL = metadataURL
	h.saml.st.RefreshedAt = time.Now().Add(-48 * time.Hour)
	h.svc.cfg.FetchMetadata = func(context.Context, string) ([]byte, error) { return serve() }
	return h, idp
}

func serving(xml string) func() ([]byte, error) {
	return func() ([]byte, error) { return []byte(xml), nil }
}

func TestRefreshMetadata_AReadThatFails_KeepsTheLastGoodCertificatesAndSaysWhy(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		serve func() ([]byte, error)
		want  string
	}{
		"an error":                     {func() ([]byte, error) { return nil, errors.New("connection refused") }, "connection refused"},
		"an empty body":                {serving(""), "not an identity provider"},
		"metadata with no certificate": {serving(idpMetadata(idpEntity, "https://idp.example.com/sso")), "no signing certificate"},
		"another provider's metadata":  {serving(idpMetadata("https://other.example.com", "https://other.example.com/sso", testCertificate(t))), "another provider"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h, idp := refreshHarness(t, c.serve)

			require.NoError(t, h.svc.refreshMetadata(t.Context()))

			assert.Equal(t, idp.metadata(), h.saml.st.IdP, "the last good certificates stay")
			assert.Contains(t, h.saml.st.RefreshError, c.want)
			assert.NotRegexp(t, "^invalid: ", h.saml.st.RefreshError, "the card shows the reason, not the error kind")
			require.Len(t, h.saml.events, 1, "the card learns the refresh is failing")
			v, err := h.svc.SAMLView(asActor(t))
			require.NoError(t, err)
			assert.Contains(t, v.MetadataRefreshError, c.want)

			h.saml.st.RefreshedAt = time.Now().Add(-48 * time.Hour)
			require.NoError(t, h.svc.refreshMetadata(t.Context()))
			assert.Len(t, h.saml.events, 1, "the same failure again tells nobody anything new")
		})
	}
}

func TestRefreshMetadata_ACertificateTheProviderAdds_VerifiesAtOnceAndOneItDropsStops(t *testing.T) {
	t.Parallel()
	served := ""
	h, oldIdP := refreshHarness(t, func() ([]byte, error) { return []byte(served), nil })
	newIdP := newTestIdP(t, idpEntity)
	served = idpMetadata(idpEntity, "https://idp.example.com/sso", oldIdP.cert, newIdP.cert)

	require.NoError(t, h.svc.refreshMetadata(t.Context()))

	assert.Empty(t, h.saml.st.RefreshError)
	for name, idp := range map[string]*testIdP{"old": oldIdP, "new": newIdP} {
		posted, _ := idp.response(t, assertionOpts{requestID: started(t, h)})
		_, err := h.svc.CompleteSAML(t.Context(), posted)
		require.NoError(t, err, "while the metadata publishes both, the %s certificate verifies", name)
	}

	served = idpMetadata(idpEntity, "https://idp.example.com/sso", newIdP.cert)
	h.saml.st.RefreshedAt = time.Now().Add(-48 * time.Hour)
	require.NoError(t, h.svc.refreshMetadata(t.Context()))
	posted, _ := oldIdP.response(t, assertionOpts{requestID: started(t, h)})
	_, err := h.svc.CompleteSAML(t.Context(), posted)
	require.ErrorIs(t, err, errSAMLRefused, "once the metadata drops the old certificate, it stops verifying")
}

func TestRefreshMetadata_UploadedOrFreshMetadata_IsNotRead(t *testing.T) {
	t.Parallel()
	var reads atomic.Int32
	h, _ := refreshHarness(t, func() ([]byte, error) { reads.Add(1); return nil, errors.New("not expected") })
	h.saml.st.RefreshedAt = time.Now().Add(-time.Hour)
	require.NoError(t, h.svc.refreshMetadata(t.Context()))

	h.saml.st.MetadataURL, h.saml.st.MetadataXML = "", "<md:EntityDescriptor/>"
	require.NoError(t, h.svc.refreshMetadata(t.Context()))

	assert.Zero(t, reads.Load())
}

func TestRunSAMLRefreshLoop_ReadsTheURLOnceADayUntilStopped(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var reads atomic.Int32
		h := newHarness(t, true)
		h.repo.st = Settings{}
		h.svc.cfg.Now = time.Now
		h.saml.st = SAMLSettings{MetadataURL: metadataURL, IdP: IdPMetadata{EntityID: idpEntity}, RefreshedAt: time.Now()}
		h.svc.cfg.FetchMetadata = func(context.Context, string) ([]byte, error) {
			reads.Add(1)
			return nil, errors.New("down")
		}
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			h.svc.RunSAMLRefreshLoop(ctx, time.Hour, slog.New(slog.DiscardHandler))
			close(done)
		}()

		time.Sleep(23*time.Hour + 30*time.Minute)
		synctest.Wait()
		assert.Zero(t, reads.Load(), "metadata read under a day ago is not read again")
		time.Sleep(time.Hour)
		synctest.Wait()
		assert.Equal(t, int32(1), reads.Load())
		time.Sleep(12 * time.Hour)
		synctest.Wait()
		assert.Equal(t, int32(1), reads.Load(), "a failed read waits a day too")

		cancel()
		<-done
	})
}

func TestUpdateSAML_TheSavedURLAgain_RefreshesItButNeverSwapsProviders(t *testing.T) {
	t.Parallel()
	served := ""
	h, idp := refreshHarness(t, func() ([]byte, error) { return []byte(served), nil })
	h.saml.st.RefreshError = "was down"
	newCert := testCertificate(t)
	served = idpMetadata(idpEntity, "https://idp.example.com/sso", idp.cert, newCert)

	v, err := h.svc.UpdateSAML(asActor(t), SAMLPatch{MetadataURL: ptr(metadataURL)})

	require.NoError(t, err)
	assert.Equal(t, 2, v.IdPCertificates)
	assert.Empty(t, v.MetadataRefreshError)
	require.NotNil(t, v.MetadataRefreshedAt)

	served = idpMetadata("https://other.example.com", "https://other.example.com/sso", newCert)
	_, err = h.svc.UpdateSAML(asActor(t), SAMLPatch{MetadataURL: ptr(metadataURL)})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "another provider")
}

// verifiesRequest checks the redirect-binding signature of a sign-in URL against a PEM certificate.
func verifiesRequest(t *testing.T, target, certPEM string) bool {
	t.Helper()
	u, err := url.Parse(target)
	require.NoError(t, err)
	q := u.RawQuery
	signed := q[:strings.Index(q, "&Signature=")]
	sig, err := base64.StdEncoding.DecodeString(u.Query().Get("Signature"))
	require.NoError(t, err)
	block, _ := pem.Decode([]byte(certPEM))
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	digest := sha256.Sum256([]byte(signed))
	return rsa.VerifyPKCS1v15(cert.PublicKey.(*rsa.PublicKey), crypto.SHA256, digest[:], sig) == nil
}

func TestRotateSAMLCertificate_SignsWithTheNewAndPublishesBothUntilRetired(t *testing.T) {
	t.Parallel()
	h, _ := signInHarness(t, true)
	old := h.saml.st.SPCertificate

	v, err := h.svc.RotateSAMLCertificate(asActor(t))

	require.NoError(t, err)
	assert.Equal(t, old, v.SP.PreviousCertificate)
	assert.NotEqual(t, old, v.SP.Certificate)
	target, err := h.svc.StartSAML(t.Context())
	require.NoError(t, err)
	assert.True(t, verifiesRequest(t, target, v.SP.Certificate), "requests are signed with the new key")
	assert.False(t, verifiesRequest(t, target, old))
	meta, err := h.svc.SPMetadata(t.Context())
	require.NoError(t, err)
	for _, c := range []string{old, v.SP.Certificate} {
		block, _ := pem.Decode([]byte(c))
		assert.Contains(t, string(meta), base64.StdEncoding.EncodeToString(block.Bytes), "an IdP that cached either certificate still finds it")
	}
	require.Len(t, h.saml.events, 1)

	_, err = h.svc.RotateSAMLCertificate(asActor(t))
	require.ErrorIs(t, err, apperrs.ErrConflict, "a second rotation would drop a certificate the provider may still use")

	v, err = h.svc.RetireSAMLCertificate(asActor(t))
	require.NoError(t, err)
	assert.Empty(t, v.SP.PreviousCertificate)
	meta, err = h.svc.SPMetadata(t.Context())
	require.NoError(t, err)
	block, _ := pem.Decode([]byte(old))
	assert.NotContains(t, string(meta), base64.StdEncoding.EncodeToString(block.Bytes))
	assert.Len(t, h.saml.events, 2)
}

func TestRetireSAMLCertificate_WithNothingToRetire_ChangesNothing(t *testing.T) {
	t.Parallel()
	h, _ := signInHarness(t, true)
	before := h.saml.st

	_, err := h.svc.RetireSAMLCertificate(asActor(t))

	require.NoError(t, err)
	assert.Equal(t, before, h.saml.st)
	assert.Empty(t, h.saml.events)
}

func TestCertificateChanges_NeedASavedProviderAndInstanceWrite(t *testing.T) {
	t.Parallel()
	h := samlHarness(t, true, nil)
	_, err := h.svc.RotateSAMLCertificate(asActor(t))
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = h.svc.RetireSAMLCertificate(asActor(t))
	require.ErrorIs(t, err, apperrs.ErrInvalid)

	h.svc.cfg.Gate = fakeGate{err: apperrs.ErrForbidden}
	_, err = h.svc.RotateSAMLCertificate(asActor(t))
	require.ErrorIs(t, err, apperrs.ErrForbidden)

	routes := NewHandler(h.svc, &fakeBrowser{}).Routes()
	assert.Equal(t, http.StatusForbidden, serve(t, routes, http.MethodPost, "/api/instance/saml/rotate", "").Code)
	assert.Equal(t, http.StatusForbidden, serve(t, routes, http.MethodPost, "/api/instance/saml/retire", "").Code)
}

func TestCertificateRoutes_AnswerWithTheView(t *testing.T) {
	t.Parallel()
	h, _ := signInHarness(t, false)
	routes := NewHandler(h.svc, &fakeBrowser{}).Routes()

	rec := serve(t, routes, http.MethodPost, "/api/instance/saml/rotate", "")
	require.Equal(t, http.StatusOK, rec.Code, "rotating works without a key: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"use":"sp_previous"`)
	rec = serve(t, routes, http.MethodPost, "/api/instance/saml/retire", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), `"use":"sp_previous"`)
}

// certificateExpiring is a base64 DER certificate that expires at notAfter.
func certificateExpiring(t *testing.T, notAfter time.Time) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(7), Subject: pkix.Name{CommonName: "idp"}, NotBefore: notAfter.Add(-time.Hour * 24 * 365), NotAfter: notAfter}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(der)
}

func TestSAMLView_FlagsEachCertificateWithinThirtyDaysOfExpiry(t *testing.T) {
	t.Parallel()
	h, _ := signInHarness(t, true)
	now := time.Now()
	h.saml.st.IdP.Certificates = []string{certificateExpiring(t, now.Add(10*24*time.Hour)), certificateExpiring(t, now.Add(90*24*time.Hour)), "bm90IGEgY2VydA=="}

	v, err := h.svc.SAMLView(asActor(t))

	require.NoError(t, err)
	require.Len(t, v.Certificates, 3, "Nexul's and the two that parse")
	assert.Equal(t, CertificateSP, v.Certificates[0].Use)
	assert.False(t, v.Certificates[0].ExpiresSoon, "a new certificate runs for years")
	assert.True(t, v.Certificates[1].ExpiresSoon)
	assert.False(t, v.Certificates[2].ExpiresSoon)
	assert.Equal(t, CertificateIdP, v.Certificates[2].Use)
}
