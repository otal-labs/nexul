package sso

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// spCertificateLife is how long a service provider certificate Nexul makes is valid; Rotate certificate replaces it.
const spCertificateLife = 5 * 365 * 24 * time.Hour

// FetchMetadata reads the identity provider's metadata from its URL; it is a field so tests serve it from httptest.
type FetchMetadata func(ctx context.Context, rawURL string) ([]byte, error)

// fetchOver reads metadata over hc: a 2xx answer of at most maxMetadata bytes.
func fetchOver(hc *http.Client) FetchMetadata {
	return func(ctx context.Context, rawURL string) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("%w: metadata URL: %s", apperrs.ErrInvalid, err)
		}
		res, err := hc.Do(req)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot reach the metadata URL: %s", apperrs.ErrInvalid, err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode < 200 || res.StatusCode > 299 {
			return nil, fmt.Errorf("%w: the metadata URL answered %s", apperrs.ErrInvalid, res.Status)
		}
		data, err := io.ReadAll(io.LimitReader(res.Body, maxMetadata+1))
		if err != nil {
			return nil, fmt.Errorf("%w: read the metadata: %s", apperrs.ErrInvalid, err)
		}
		if len(data) > maxMetadata {
			return nil, errMetadataTooLarge
		}
		return data, nil
	}
}

var errMetadataTooLarge = fmt.Errorf("%w: the metadata is larger than %d KB", apperrs.ErrInvalid, maxMetadata>>10)

// parseIdPMetadata reads the entity id, the HTTP-Redirect sign-in URL and the signing certificates out of an identity
// provider's metadata, refusing metadata that lacks any of them.
func parseIdPMetadata(data []byte) (IdPMetadata, error) {
	if len(data) > maxMetadata {
		return IdPMetadata{}, errMetadataTooLarge
	}
	ed, err := samlsp.ParseMetadata(data)
	if err != nil {
		return IdPMetadata{}, fmt.Errorf("%w: the metadata is not an identity provider's SAML metadata: %s", apperrs.ErrInvalid, err)
	}
	out := IdPMetadata{EntityID: strings.TrimSpace(ed.EntityID)}
	if out.EntityID == "" || len(ed.IDPSSODescriptors) == 0 {
		return IdPMetadata{}, fmt.Errorf("%w: the metadata names no identity provider", apperrs.ErrInvalid)
	}
	for _, d := range ed.IDPSSODescriptors {
		for _, s := range d.SingleSignOnServices {
			if s.Binding == saml.HTTPRedirectBinding && out.SSOURL == "" {
				out.SSOURL = strings.TrimSpace(s.Location)
			}
		}
		if out.Certificates, err = signingCertificates(d.KeyDescriptors, out.Certificates); err != nil {
			return IdPMetadata{}, err
		}
	}
	if out.SSOURL == "" {
		return IdPMetadata{}, fmt.Errorf("%w: the metadata names no single sign-on URL for the HTTP-Redirect binding", apperrs.ErrInvalid)
	}
	if err := checkHTTPS("the metadata's single sign-on URL", "https://login.example.com/saml", out.SSOURL); err != nil {
		return IdPMetadata{}, err
	}
	if len(out.Certificates) == 0 {
		return IdPMetadata{}, fmt.Errorf("%w: the metadata names no signing certificate", apperrs.ErrInvalid)
	}
	return out, nil
}

// signingCertificates adds to certs each signing certificate of keys not already in it; a key with no use signs too.
func signingCertificates(keys []saml.KeyDescriptor, certs []string) ([]string, error) {
	for _, k := range keys {
		if k.Use != "signing" && k.Use != "" {
			continue
		}
		for _, c := range k.KeyInfo.X509Data.X509Certificates {
			der, err := certificateDER(c.Data)
			if err != nil {
				return nil, err
			}
			if !slices.Contains(certs, der) {
				certs = append(certs, der)
			}
		}
	}
	return certs, nil
}

// certificateDER checks a metadata certificate parses and returns it as base64 DER without whitespace.
func certificateDER(text string) (string, error) {
	compact := strings.Join(strings.Fields(text), "")
	der, err := base64.StdEncoding.DecodeString(compact)
	if err == nil {
		_, err = x509.ParseCertificate(der)
	}
	if err != nil {
		return "", fmt.Errorf("%w: a signing certificate in the metadata does not parse", apperrs.ErrInvalid)
	}
	return compact, nil
}

// newSPKeyPair makes Nexul's service provider key and a self-signed certificate for it, named after the instance host.
func newSPKeyPair(host string, now time.Time) (keyPEM, certPEM string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("generate the service provider key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return "", "", fmt.Errorf("generate a certificate serial: %w", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial, Subject: pkix.Name{CommonName: host},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(spCertificateLife),
		KeyUsage: x509.KeyUsageDigitalSignature, BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return "", "", fmt.Errorf("make the service provider certificate: %w", err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", "", fmt.Errorf("encode the service provider key: %w", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})),
		string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), nil
}

// serviceProvider builds the library's service provider for these settings and this instance address.
func serviceProvider(st SAMLSettings, sp SPValues) (*saml.ServiceProvider, error) {
	keyBlock, _ := pem.Decode([]byte(st.SPKey))
	certBlock, _ := pem.Decode([]byte(st.SPCertificate))
	if keyBlock == nil || certBlock == nil {
		return nil, errors.New("the service provider key pair does not decode")
	}
	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse the service provider key: %w", err)
	}
	signer, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("the service provider key is not RSA")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse the service provider certificate: %w", err)
	}
	acs, err := url.Parse(sp.ACSURL)
	if err != nil {
		return nil, fmt.Errorf("parse the assertion consumer service URL: %w", err)
	}
	meta, err := url.Parse(sp.MetadataURL)
	if err != nil {
		return nil, fmt.Errorf("parse the metadata URL: %w", err)
	}
	return &saml.ServiceProvider{
		EntityID: sp.EntityID, Key: signer, Certificate: cert, AcsURL: *acs, MetadataURL: *meta,
		AuthnNameIDFormat:                saml.PersistentNameIDFormat,
		SignatureMethod:                  "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256",
		AssertionConsumerServiceBindings: []string{saml.HTTPPostBinding},
		MetadataOmitValidUntil:           true,
		IDPMetadata:                      idpDescriptor(st.IdP),
	}, nil
}

// idpDescriptor rebuilds the slice of the identity provider's metadata a sign-in needs from what was read out of it.
func idpDescriptor(m IdPMetadata) *saml.EntityDescriptor {
	certs := make([]saml.X509Certificate, 0, len(m.Certificates))
	for _, c := range m.Certificates {
		certs = append(certs, saml.X509Certificate{Data: c})
	}
	return &saml.EntityDescriptor{EntityID: m.EntityID, IDPSSODescriptors: []saml.IDPSSODescriptor{{
		SSODescriptor: saml.SSODescriptor{RoleDescriptor: saml.RoleDescriptor{
			ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
			KeyDescriptors:             []saml.KeyDescriptor{{Use: "signing", KeyInfo: saml.KeyInfo{X509Data: saml.X509Data{X509Certificates: certs}}}},
		}},
		SingleSignOnServices: []saml.Endpoint{{Binding: saml.HTTPRedirectBinding, Location: m.SSOURL}},
	}}}
}

// spMetadataXML is the service provider metadata an identity provider registers: the entity id, the assertion consumer
// service and the signing certificate, then previousPEM while a rotation keeps it published. No encryption key is
// offered, since encrypted assertions are not taken.
func spMetadataXML(sp *saml.ServiceProvider, previousPEM string) ([]byte, error) {
	ed := sp.Metadata()
	for i := range ed.SPSSODescriptors {
		d := &ed.SPSSODescriptors[i]
		d.KeyDescriptors = slices.DeleteFunc(d.KeyDescriptors, func(k saml.KeyDescriptor) bool { return k.Use == "encryption" })
		if block, _ := pem.Decode([]byte(previousPEM)); block != nil {
			d.KeyDescriptors = append(d.KeyDescriptors, saml.KeyDescriptor{Use: "signing", KeyInfo: saml.KeyInfo{X509Data: saml.X509Data{
				X509Certificates: []saml.X509Certificate{{Data: base64.StdEncoding.EncodeToString(block.Bytes)}},
			}}})
		}
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(ed); err != nil {
		return nil, fmt.Errorf("encode the service provider metadata: %w", err)
	}
	return buf.Bytes(), nil
}
