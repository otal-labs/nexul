package sso

import (
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// SAMLSettings is the instance's provider when it speaks SAML (ADR 0189): its metadata as the Owner gave it, what Nexul
// read from it, and Nexul's own key pair as the service provider.
type SAMLSettings struct {
	// MetadataURL is where the metadata is fetched from; empty when it was uploaded.
	MetadataURL string
	// MetadataXML is the uploaded metadata, kept as it was; empty when it comes from a URL.
	MetadataXML string
	IdP         IdPMetadata
	// EmailAttribute names the assertion attribute the person's email is read from.
	EmailAttribute string
	// NameAttribute names the attribute their display name is read from; empty reads none.
	NameAttribute string
	DisplayName   string
	JoinRoleID    string
	Only          bool
	// SPKey is Nexul's private key in PKCS #8 PEM, sealed at rest and never sent anywhere.
	SPKey string
	// SPCertificate is the PEM certificate of SPKey that the service provider metadata publishes.
	SPCertificate string
	// SPPreviousCertificate is the certificate a rotation replaced, published until it is retired; empty when none.
	SPPreviousCertificate string
	// RefreshedAt is when the metadata URL was last read, and RefreshError why that read failed; empty when it worked.
	RefreshedAt  time.Time
	RefreshError string
	UpdatedBy    string
	UpdatedAt    time.Time
}

// IdPMetadata is what Nexul reads from the identity provider's metadata.
type IdPMetadata struct {
	EntityID string
	// SSOURL is where the HTTP-Redirect binding sends a sign-in request.
	SSOURL string
	// Certificates are the base64 DER signing certificates; any of them verifies a response.
	Certificates []string
}

// Saved reports whether a SAML provider is saved, configured or not.
func (s SAMLSettings) Saved() bool { return s.IdP.EntityID != "" }

// Configured reports whether every field a sign-in needs is set.
func (s SAMLSettings) Configured() bool {
	return s.IdP.EntityID != "" && s.IdP.SSOURL != "" && len(s.IdP.Certificates) > 0 && s.EmailAttribute != "" &&
		s.DisplayName != "" && s.JoinRoleID != "" && s.SPKey != ""
}

// SAMLView is the SAML settings as the Owner and MCP clients read them; the private key never leaves the server.
type SAMLView struct {
	Unlocked   bool `json:"unlocked"`
	Configured bool `json:"configured"`
	// Active is Unlocked and Configured: the sign-in button shows.
	Active         bool   `json:"active"`
	MetadataURL    string `json:"metadata_url"`
	MetadataXMLSet bool   `json:"metadata_xml_set"`
	IdPEntityID    string `json:"idp_entity_id"`
	IdPSSOURL      string `json:"idp_sso_url"`
	// IdPCertificates counts the signing certificates read from the metadata.
	IdPCertificates int    `json:"idp_certificates"`
	EmailAttribute  string `json:"email_attribute"`
	NameAttribute   string `json:"name_attribute"`
	DisplayName     string `json:"display_name"`
	JoinRoleID      string `json:"join_role_id"`
	Only            bool   `json:"only"`
	// SP is what the identity provider registers for this instance.
	SP SPValues `json:"sp"`
	// Certificates are the expiries of Nexul's certificates and the provider's.
	Certificates []CertificateView `json:"certificates"`
	// MetadataRefreshedAt is when the metadata URL was last read; absent for uploaded metadata.
	MetadataRefreshedAt *time.Time `json:"metadata_refreshed_at,omitempty"`
	// MetadataRefreshError is why the last read failed, the certificates read before it still in use; empty when it worked.
	MetadataRefreshError string `json:"metadata_refresh_error"`
}

// SPValues are the service provider metadata's values: what an IdP needs to know about Nexul.
type SPValues struct {
	EntityID    string `json:"entity_id"`
	ACSURL      string `json:"acs_url"`
	MetadataURL string `json:"metadata_url"`
	// Certificate is the PEM certificate AuthnRequests are signed with; empty until the provider is first saved.
	Certificate string `json:"certificate"`
	// PreviousCertificate is the one a rotation replaced, still published until it is retired; empty when none.
	PreviousCertificate string `json:"previous_certificate"`
}

// SAMLPatch changes the SAML settings; an omitted field keeps its value. A metadata URL replaces uploaded metadata and
// the reverse.
type SAMLPatch struct {
	MetadataURL    *string `json:"metadata_url"`
	MetadataXML    *string `json:"metadata_xml"`
	EmailAttribute *string `json:"email_attribute"`
	NameAttribute  *string `json:"name_attribute"`
	DisplayName    *string `json:"display_name"`
	JoinRoleID     *string `json:"join_role_id"`
	Only           *bool   `json:"only"`
}

const (
	defaultEmailAttribute = "email"
	// maxMetadata caps fetched and uploaded metadata; an IdP's own is a few tens of kilobytes.
	maxMetadata  = 1 << 20
	maxAttribute = 256
)

var (
	errOIDCSaved = fmt.Errorf("%w: an OpenID Connect provider is saved and an instance holds one provider; remove it first", apperrs.ErrConflict)
	errSAMLSaved = fmt.Errorf("%w: a SAML provider is saved and an instance holds one provider; remove it first", apperrs.ErrConflict)
)
