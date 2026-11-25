package sso

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/russellhaering/goxmldsig/etreeutils"
	"github.com/stretchr/testify/require"
)

// testIdP is an identity provider that signs responses with its own key pair, for the sign-in tests.
type testIdP struct {
	entityID string
	cert     string
	signer   *dsig.SigningContext
}

func newTestIdP(t *testing.T, entityID string) *testIdP {
	t.Helper()
	keyPEM, certPEM, err := newSPKeyPair("idp.example.com", time.Now())
	require.NoError(t, err)
	kb, _ := pem.Decode([]byte(keyPEM))
	cb, _ := pem.Decode([]byte(certPEM))
	key, err := x509.ParsePKCS8PrivateKey(kb.Bytes)
	require.NoError(t, err)
	ctx := dsig.NewDefaultSigningContext(dsig.TLSCertKeyStore(tls.Certificate{Certificate: [][]byte{cb.Bytes}, PrivateKey: key.(*rsa.PrivateKey)}))
	ctx.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")
	require.NoError(t, ctx.SetSignatureMethod(dsig.RSASHA256SignatureMethod))
	return &testIdP{entityID: entityID, cert: base64.StdEncoding.EncodeToString(cb.Bytes), signer: ctx}
}

func (p *testIdP) metadata() IdPMetadata {
	return IdPMetadata{EntityID: p.entityID, SSOURL: "https://idp.example.com/sso", Certificates: []string{p.cert}}
}

// assertionOpts shapes one assertion; the zero value of each field takes the happy path's.
type assertionOpts struct {
	requestID, audience, issuer, nameID, nameIDFormat, email string
	issued, notBefore, notOnOrAfter                          time.Time
	// edit changes the assertion element before it is signed.
	edit func(*etree.Element)
	// unsigned leaves the assertion without a signature.
	unsigned bool
}

const testACS = "https://nexul.example.com/auth/saml/acs"

// response is a base64 Response carrying one assertion, which the IdP signs unless o.unsigned.
func (p *testIdP) response(t *testing.T, o assertionOpts) (string, *etree.Element) {
	t.Helper()
	now := time.Now().UTC()
	o = withDefaults(o, p.entityID, now)
	a := saml.Assertion{
		ID: fmt.Sprintf("id-%d", now.UnixNano()), IssueInstant: o.issued, Version: "2.0", Issuer: saml.Issuer{Value: o.issuer},
		Subject: &saml.Subject{
			NameID: &saml.NameID{Format: o.nameIDFormat, Value: o.nameID},
			SubjectConfirmations: []saml.SubjectConfirmation{{Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer", SubjectConfirmationData: &saml.SubjectConfirmationData{
				InResponseTo: o.requestID, NotOnOrAfter: o.notOnOrAfter, Recipient: testACS,
			}}},
		},
		Conditions: &saml.Conditions{
			NotBefore: o.notBefore, NotOnOrAfter: o.notOnOrAfter,
			AudienceRestrictions: []saml.AudienceRestriction{{Audience: saml.Audience{Value: o.audience}}},
		},
		AuthnStatements: []saml.AuthnStatement{{AuthnInstant: o.issued, SessionIndex: "s-1", AuthnContext: saml.AuthnContext{
			AuthnContextClassRef: &saml.AuthnContextClassRef{Value: "urn:oasis:names:tc:SAML:2.0:ac:classes:Password"},
		}}},
	}
	if o.email != "-" {
		a.AttributeStatements = []saml.AttributeStatement{{Attributes: []saml.Attribute{
			{Name: "email", Values: []saml.AttributeValue{{Type: "xs:string", Value: o.email}}},
			{Name: "name", Values: []saml.AttributeValue{{Type: "xs:string", Value: "Alice"}}},
		}}}
	}
	el := a.Element()
	if o.edit != nil {
		o.edit(el)
	}
	if !o.unsigned {
		el = p.sign(t, el)
	}
	res := saml.Response{
		ID: fmt.Sprintf("res-%d", now.UnixNano()), InResponseTo: o.requestID, IssueInstant: o.issued, Version: "2.0",
		Destination: testACS, Issuer: &saml.Issuer{Value: o.issuer}, Status: saml.Status{StatusCode: saml.StatusCode{Value: saml.StatusSuccess}},
	}
	resEl := res.Element()
	resEl.AddChild(el)
	return encodeResponse(t, resEl), resEl
}

func encodeResponse(t *testing.T, el *etree.Element) string {
	t.Helper()
	doc := etree.NewDocument()
	doc.SetRoot(el.Copy())
	b, err := doc.WriteToBytes()
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(b)
}

func withDefaults(o assertionOpts, entityID string, now time.Time) assertionOpts {
	if o.audience == "" {
		o.audience = "https://nexul.example.com/auth/saml/metadata"
	}
	if o.issuer == "" {
		o.issuer = entityID
	}
	if o.nameID == "" {
		o.nameID = "alice-1"
	}
	if o.nameIDFormat == "" {
		o.nameIDFormat = string(saml.PersistentNameIDFormat)
	}
	if o.email == "" {
		o.email = "alice@example.com"
	}
	if o.issued.IsZero() {
		o.issued = now
	}
	if o.notBefore.IsZero() {
		o.notBefore = o.issued.Add(-time.Minute)
	}
	if o.notOnOrAfter.IsZero() {
		o.notOnOrAfter = o.issued.Add(5 * time.Minute)
	}
	return o
}

// sign puts an enveloped signature on el right after its Issuer, where the schema wants it.
func (p *testIdP) sign(t *testing.T, el *etree.Element) *etree.Element {
	t.Helper()
	signed, err := p.signer.SignEnveloped(el)
	require.NoError(t, err)
	// SignEnveloped appends the signature without parenting it, so it is taken off the end by index.
	sig := signed.Child[len(signed.Child)-1].(*etree.Element)
	signed.Child = signed.Child[:len(signed.Child)-1]
	signed.InsertChildAt(1, sig)
	return signed
}

// resign signs the SignedInfo of the signature in el again, so a test can change the references and keep a valid
// signature value over them.
func (p *testIdP) resign(t *testing.T, el *etree.Element) {
	t.Helper()
	info := el.FindElement("./Signature/SignedInfo")
	require.NotNil(t, info)
	nsctx, err := etreeutils.NSBuildParentContext(info)
	require.NoError(t, err)
	detached, err := etreeutils.NSDetatch(nsctx, info)
	require.NoError(t, err)
	canonical, err := p.signer.Canonicalizer.Canonicalize(detached)
	require.NoError(t, err)
	value, err := p.signer.SignString(string(canonical))
	require.NoError(t, err)
	el.FindElement("./Signature/SignatureValue").SetText(base64.StdEncoding.EncodeToString(value))
}
