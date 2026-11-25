package sso

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/crewjam/saml"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const (
	// samlRequestLife is how long a sign-in request Nexul sent waits for its answer.
	samlRequestLife = 10 * time.Minute
	// samlSkew is the clock difference taken from the provider; the library allows 180 seconds, ADR 0189 takes 60.
	samlSkew = 60 * time.Second
	// maxSAMLResponse caps a posted response before it is decoded; a signed one is a few kilobytes.
	maxSAMLResponse = 1 << 20
)

var (
	errSAMLRefused   = fmt.Errorf("%w: the provider's response was refused", apperrs.ErrUnauthorized)
	errSAMLTransient = fmt.Errorf("%w: the provider sent a transient NameID, which changes every sign-in; set it to send a persistent one", apperrs.ErrUnauthorized)
)

// activeSAML returns the SAML settings a sign-in runs on, or why nobody can sign in this way.
func (s *Service) activeSAML(ctx context.Context) (SAMLSettings, SPValues, error) {
	st, err := s.cfg.SAML.GetSAML(ctx)
	if err != nil {
		return SAMLSettings{}, SPValues{}, err
	}
	if !st.Configured() {
		return SAMLSettings{}, SPValues{}, fmt.Errorf("%w: SAML single sign-on is not on for this instance", apperrs.ErrInvalid)
	}
	values, err := s.spValues(ctx)
	if err != nil {
		return SAMLSettings{}, SPValues{}, err
	}
	return st, values, nil
}

// StartSAML makes a signed AuthnRequest, keeps its id for samlRequestLife and returns the provider's URL that carries
// it, with a random RelayState.
func (s *Service) StartSAML(ctx context.Context) (string, error) {
	st, values, err := s.activeSAML(ctx)
	if err != nil {
		return "", err
	}
	sp, err := serviceProvider(st, values)
	if err != nil {
		return "", err
	}
	now := s.cfg.Now()
	req, err := sp.MakeAuthenticationRequest(st.IdP.SSOURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if err != nil {
		return "", fmt.Errorf("make the sign-in request: %w", err)
	}
	if err := s.cfg.SAML.SaveSAMLRequest(ctx, req.ID, now, now.Add(-samlRequestLife)); err != nil {
		return "", err
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate relay state: %w", err)
	}
	u, err := req.Redirect(hex.EncodeToString(b), sp)
	if err != nil {
		return "", fmt.Errorf("encode the sign-in request: %w", err)
	}
	return u.String(), nil
}

// responseHead is what is read from a response before its signature is checked: which request it answers, and
// whether it hides its assertion. Both only narrow what is accepted; the library checks the rest.
type responseHead struct {
	InResponseTo string     `xml:"InResponseTo,attr"`
	Encrypted    []struct{} `xml:"EncryptedAssertion"`
}

// CompleteSAML checks the provider's posted response and returns the account to start a session for, found or made as
// Complete does. The pending request it answers is deleted first, so a response is taken once.
func (s *Service) CompleteSAML(ctx context.Context, posted string) (string, error) {
	st, values, err := s.activeSAML(ctx)
	if err != nil {
		return "", err
	}
	if len(posted) > maxSAMLResponse {
		return "", fmt.Errorf("%w: the response is larger than %d KB", errSAMLRefused, maxSAMLResponse>>10)
	}
	raw, err := base64.StdEncoding.DecodeString(posted)
	if err != nil {
		return "", fmt.Errorf("%w: the response is not base64", errSAMLRefused)
	}
	var head responseHead
	if err := xml.Unmarshal(raw, &head); err != nil {
		return "", fmt.Errorf("%w: the response is not XML", errSAMLRefused)
	}
	if len(head.Encrypted) > 0 {
		return "", fmt.Errorf("%w: encrypted assertions are not supported; turn assertion encryption off at the provider", errSAMLRefused)
	}
	if head.InResponseTo == "" {
		return "", fmt.Errorf("%w: it answers no request; sign-in started at the provider is not supported", errSAMLRefused)
	}
	found, err := s.cfg.SAML.TakeSAMLRequest(ctx, head.InResponseTo, s.cfg.Now().Add(-samlRequestLife))
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("%w: it answers no sign-in request waiting here, or one already used or expired", errSAMLRefused)
	}
	a, err := parseAssertion(st, values, raw, head.InResponseTo)
	if err != nil {
		return "", err
	}
	if err := checkSkew(a, s.cfg.Now()); err != nil {
		return "", err
	}
	id, err := samlIdentity(st, a)
	if err != nil {
		return "", err
	}
	return s.account(ctx, st.JoinRoleID, id)
}

// parseAssertion has the library verify the response: a signature by a certificate in the metadata, this request,
// this issuer, this audience and its time conditions.
func parseAssertion(st SAMLSettings, values SPValues, raw []byte, requestID string) (*saml.Assertion, error) {
	sp, err := serviceProvider(st, values)
	if err != nil {
		return nil, err
	}
	acs, err := url.Parse(values.ACSURL)
	if err != nil {
		return nil, fmt.Errorf("parse the assertion consumer service URL: %w", err)
	}
	a, err := sp.ParseXMLResponse(raw, []string{requestID}, *acs)
	var invalid *saml.InvalidResponseError
	if errors.As(err, &invalid) {
		return nil, fmt.Errorf("%w: %w", errSAMLRefused, invalid.PrivateErr)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errSAMLRefused, err)
	}
	return a, nil
}

// checkSkew holds the assertion's times to samlSkew, tighter than the library's own allowance.
func checkSkew(a *saml.Assertion, now time.Time) error {
	early := a.IssueInstant.After(now.Add(samlSkew)) || (a.Conditions != nil && a.Conditions.NotBefore.After(now.Add(samlSkew)))
	if early {
		return fmt.Errorf("%w: the assertion is not valid yet", errSAMLRefused)
	}
	deadlines := []time.Time{}
	if a.Conditions != nil {
		deadlines = append(deadlines, a.Conditions.NotOnOrAfter)
	}
	if a.Subject != nil {
		for _, c := range a.Subject.SubjectConfirmations {
			if c.SubjectConfirmationData != nil {
				deadlines = append(deadlines, c.SubjectConfirmationData.NotOnOrAfter)
			}
		}
	}
	for _, d := range deadlines {
		if !d.IsZero() && !now.Before(d.Add(samlSkew)) {
			return fmt.Errorf("%w: the assertion has expired", errSAMLRefused)
		}
	}
	return nil
}

// samlIdentity is who the assertion names: the provider's entity id with its persistent NameID, and the email and
// display name from the attributes the Owner named.
func samlIdentity(st SAMLSettings, a *saml.Assertion) (Identity, error) {
	if a.Subject == nil || a.Subject.NameID == nil || strings.TrimSpace(a.Subject.NameID.Value) == "" {
		return Identity{}, fmt.Errorf("%w: the assertion names nobody", errSAMLRefused)
	}
	if a.Subject.NameID.Format == string(saml.TransientNameIDFormat) {
		return Identity{}, errSAMLTransient
	}
	email := attribute(a, st.EmailAttribute)
	if email == "" {
		return Identity{}, fmt.Errorf("%w: no %s attribute", errUnverified, st.EmailAttribute)
	}
	return Identity{
		Provider: ProtocolSAML, Key: st.IdP.EntityID + "#" + strings.TrimSpace(a.Subject.NameID.Value), Email: email,
		Name: attribute(a, st.NameAttribute),
	}, nil
}

// attribute is the first non-empty value of the attribute with that name or friendly name; empty for an empty name.
func attribute(a *saml.Assertion, name string) string {
	if name == "" {
		return ""
	}
	for _, stmt := range a.AttributeStatements {
		for _, attr := range stmt.Attributes {
			if attr.Name != name && attr.FriendlyName != name {
				continue
			}
			for _, v := range attr.Values {
				if value := strings.TrimSpace(v.Value); value != "" {
					return value
				}
			}
		}
	}
	return ""
}
