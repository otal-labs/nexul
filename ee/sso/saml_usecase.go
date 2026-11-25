package sso

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// SAMLRepo stores the instance's SAML provider; implemented in internal/platform/storage.
type SAMLRepo interface {
	// GetSAML returns the zero SAMLSettings when none are saved.
	GetSAML(ctx context.Context) (SAMLSettings, error)
	SetSAML(ctx context.Context, s SAMLSettings, evts ...eventbus.OutboxEvent) error
	// TurnOnSAMLOnly saves s with Only on and ends every session not started through single sign-on except keepUsers',
	// as TurnOnOnly does.
	TurnOnSAMLOnly(ctx context.Context, s SAMLSettings, keepUsers []string, evts ...eventbus.OutboxEvent) error
	DeleteSAML(ctx context.Context, evts ...eventbus.OutboxEvent) error
	// SaveSAMLRequest keeps the id of a sign-in request sent at at, and drops the ones sent before dropBefore.
	SaveSAMLRequest(ctx context.Context, id string, at, dropBefore time.Time) error
	// TakeSAMLRequest deletes the request and reports whether it was waiting and was sent at since or later.
	TakeSAMLRequest(ctx context.Context, id string, since time.Time) (bool, error)
}

const (
	pathSAMLMetadata = "/auth/saml/metadata"
	pathSAMLACS      = "/auth/saml/acs"
)

// spValues are the values an identity provider registers for this instance, built on its public address.
func (s *Service) spValues(ctx context.Context) (SPValues, error) {
	base, err := s.cfg.InstanceURL(ctx)
	if err != nil {
		return SPValues{}, fmt.Errorf("read instance url: %w", err)
	}
	base = strings.TrimSuffix(base, "/")
	return SPValues{EntityID: base + pathSAMLMetadata, ACSURL: base + pathSAMLACS, MetadataURL: base + pathSAMLMetadata}, nil
}

// SAMLView returns the SAML settings as they stand, whatever the license says; it needs instance:read.
func (s *Service) SAMLView(ctx context.Context) (SAMLView, error) {
	if err := s.require(ctx, permissions.InstanceRead); err != nil {
		return SAMLView{}, err
	}
	st, err := s.cfg.SAML.GetSAML(ctx)
	if err != nil {
		return SAMLView{}, err
	}
	return s.samlView(ctx, st)
}

func (s *Service) samlView(ctx context.Context, st SAMLSettings) (SAMLView, error) {
	sp, err := s.spValues(ctx)
	if err != nil {
		return SAMLView{}, err
	}
	sp.Certificate = st.SPCertificate
	unlocked := s.unlocked(ctx)
	return SAMLView{
		Unlocked: unlocked, Configured: st.Configured(), Active: unlocked && st.Configured(),
		MetadataURL: st.MetadataURL, MetadataXMLSet: st.MetadataXML != "", IdPEntityID: st.IdP.EntityID, IdPSSOURL: st.IdP.SSOURL,
		IdPCertificates: len(st.IdP.Certificates), EmailAttribute: st.EmailAttribute, NameAttribute: st.NameAttribute,
		DisplayName: st.DisplayName, JoinRoleID: st.JoinRoleID, Only: st.Only, SP: sp,
	}, nil
}

// UpdateSAML saves the SAML provider; it needs instance:write and a key that unlocks sso, and refuses it while an OpenID
// Connect provider is saved, metadata that does not name a provider, a role nobody may join as, or SSO-only with no
// Owner passkey to fall back on. The first save makes Nexul's key pair; a patch that changes nothing writes nothing.
func (s *Service) UpdateSAML(ctx context.Context, p SAMLPatch) (SAMLView, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return SAMLView{}, err
	}
	if !s.unlocked(ctx) {
		return SAMLView{}, errNeedsLicense
	}
	oidc, err := s.cfg.Repo.Get(ctx)
	if err != nil {
		return SAMLView{}, err
	}
	if oidc.IssuerURL != "" {
		return SAMLView{}, errOIDCSaved
	}
	cur, err := s.cfg.SAML.GetSAML(ctx)
	if err != nil {
		return SAMLView{}, err
	}
	next, err := s.applySAML(ctx, cur, p)
	if err != nil {
		return SAMLView{}, err
	}
	if reflect.DeepEqual(cur, next) {
		return s.samlView(ctx, cur)
	}
	if err := s.validateSAML(ctx, cur, next); err != nil {
		return SAMLView{}, err
	}
	next.UpdatedBy, next.UpdatedAt = actorID(ctx), s.cfg.Now()
	if next.SPKey == "" {
		if next.SPKey, next.SPCertificate, err = s.spKeyPair(ctx, next.UpdatedAt); err != nil {
			return SAMLView{}, err
		}
	}
	evt := s.samlChanged(SAMLChangedEvent{DisplayName: next.DisplayName, EntityID: next.IdP.EntityID, Only: next.Only, ChangedBy: next.UpdatedBy}, next.UpdatedAt)
	if err := s.saveSAML(ctx, cur, next, evt); err != nil {
		return SAMLView{}, err
	}
	return s.samlView(ctx, next)
}

func (s *Service) spKeyPair(ctx context.Context, now time.Time) (string, string, error) {
	sp, err := s.spValues(ctx)
	if err != nil {
		return "", "", err
	}
	u, _ := url.Parse(sp.EntityID)
	return newSPKeyPair(u.Hostname(), now)
}

// applySAML is cur with the patch on it; metadata that changed is fetched or parsed now, so a bad one is refused before
// anything is stored.
func (s *Service) applySAML(ctx context.Context, cur SAMLSettings, p SAMLPatch) (SAMLSettings, error) {
	next := cur
	next.EmailAttribute = trim(p.EmailAttribute, cur.EmailAttribute)
	if next.EmailAttribute == "" {
		next.EmailAttribute = defaultEmailAttribute
	}
	next.NameAttribute = trim(p.NameAttribute, cur.NameAttribute)
	next.DisplayName = trim(p.DisplayName, cur.DisplayName)
	next.JoinRoleID = trim(p.JoinRoleID, cur.JoinRoleID)
	if p.Only != nil {
		next.Only = *p.Only
	}
	rawURL, xmlText := trim(p.MetadataURL, ""), ""
	if p.MetadataXML != nil {
		xmlText = *p.MetadataXML
	}
	if rawURL != "" && strings.TrimSpace(xmlText) != "" {
		return SAMLSettings{}, fmt.Errorf("%w: give metadata_url or metadata_xml, not both", apperrs.ErrInvalid)
	}
	if rawURL != "" && rawURL != cur.MetadataURL {
		idp, err := s.fetchIdP(ctx, rawURL)
		if err != nil {
			return SAMLSettings{}, err
		}
		next.MetadataURL, next.MetadataXML, next.IdP = rawURL, "", idp
	}
	if strings.TrimSpace(xmlText) != "" && xmlText != cur.MetadataXML {
		idp, err := parseIdPMetadata([]byte(xmlText))
		if err != nil {
			return SAMLSettings{}, err
		}
		next.MetadataURL, next.MetadataXML, next.IdP = "", xmlText, idp
	}
	return next, nil
}

func (s *Service) fetchIdP(ctx context.Context, rawURL string) (IdPMetadata, error) {
	if err := checkHTTPS("metadata URL", "https://login.example.com/saml/metadata", rawURL); err != nil {
		return IdPMetadata{}, err
	}
	data, err := s.cfg.FetchMetadata(ctx, rawURL)
	if err != nil {
		return IdPMetadata{}, err
	}
	return parseIdPMetadata(data)
}

func (s *Service) validateSAML(ctx context.Context, cur, next SAMLSettings) error {
	if !next.Saved() || next.DisplayName == "" || next.JoinRoleID == "" {
		return fmt.Errorf("%w: metadata_url or metadata_xml, display_name and join_role_id are all needed", apperrs.ErrInvalid)
	}
	if utf8.RuneCountInString(next.DisplayName) > maxDisplayName {
		return fmt.Errorf("%w: display_name is at most %d characters", apperrs.ErrInvalid, maxDisplayName)
	}
	if len(next.EmailAttribute) > maxAttribute || len(next.NameAttribute) > maxAttribute {
		return fmt.Errorf("%w: an attribute name is at most %d characters", apperrs.ErrInvalid, maxAttribute)
	}
	if next.JoinRoleID != cur.JoinRoleID {
		if err := s.checkJoinRole(ctx, next.JoinRoleID); err != nil {
			return err
		}
	}
	if next.Only && !cur.Only {
		return s.requireOwnerPasskey(ctx)
	}
	return nil
}

func (s *Service) checkJoinRole(ctx context.Context, roleID string) error {
	ok, err := s.cfg.Accounts.CanJoinAs(ctx, roleID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: join_role_id must be a role of the default workspace other than Owner; workspace_list with the default workspace id lists its roles", apperrs.ErrInvalid)
	}
	return nil
}

// saveSAML stores next; turning SSO-only on signs out everyone the switch now refuses, as save does for OIDC.
func (s *Service) saveSAML(ctx context.Context, cur, next SAMLSettings, evt eventbus.OutboxEvent) error {
	if !next.Only || cur.Only {
		return s.cfg.SAML.SetSAML(ctx, next, evt)
	}
	owners, err := s.cfg.InstanceOwners(ctx)
	if err != nil {
		return fmt.Errorf("list instance owners: %w", err)
	}
	return s.cfg.SAML.TurnOnSAMLOnly(ctx, next, owners, evt)
}

// CheckSAML runs one named check of what is about to be saved, "metadata" (it parses and names a signing certificate)
// or "email" (the email attribute is not empty), and stores nothing; it needs what UpdateSAML does.
func (s *Service) CheckSAML(ctx context.Context, check string, p SAMLPatch) (string, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return "", err
	}
	if !s.unlocked(ctx) {
		return "", errNeedsLicense
	}
	cur, err := s.cfg.SAML.GetSAML(ctx)
	if err != nil {
		return "", err
	}
	switch check {
	case "metadata":
		return s.checkMetadata(ctx, cur, p)
	case "email":
		if trim(p.EmailAttribute, cur.EmailAttribute) == "" {
			return "", fmt.Errorf("%w: email_attribute names the attribute the email is read from, such as email", apperrs.ErrInvalid)
		}
		return "", nil
	}
	return "", fmt.Errorf("%w: check is metadata or email", apperrs.ErrInvalid)
}

func (s *Service) checkMetadata(ctx context.Context, cur SAMLSettings, p SAMLPatch) (string, error) {
	if trim(p.MetadataURL, "") == "" && (p.MetadataXML == nil || strings.TrimSpace(*p.MetadataXML) == "") && !cur.Saved() {
		return "", fmt.Errorf("%w: metadata_url or metadata_xml is needed", apperrs.ErrInvalid)
	}
	next, err := s.applySAML(ctx, SAMLSettings{IdP: cur.IdP}, p)
	if err != nil {
		return "", err
	}
	n := len(next.IdP.Certificates)
	noun := "signing certificate"
	if n != 1 {
		noun += "s"
	}
	return fmt.Sprintf("Read %s with %d %s", next.IdP.EntityID, n, noun), nil
}

// RemoveSAML deletes the SAML provider: the button goes, the other sign-in methods come back, and nobody's account
// changes. It needs instance:write, and works without a key so a lapsed license never traps a provider in place.
func (s *Service) RemoveSAML(ctx context.Context) (SAMLView, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return SAMLView{}, err
	}
	cur, err := s.cfg.SAML.GetSAML(ctx)
	if err != nil {
		return SAMLView{}, err
	}
	if !cur.Saved() {
		return s.samlView(ctx, SAMLSettings{})
	}
	evt := s.samlChanged(SAMLChangedEvent{Removed: true, ChangedBy: actorID(ctx)}, s.cfg.Now())
	if err := s.cfg.SAML.DeleteSAML(ctx, evt); err != nil {
		return SAMLView{}, err
	}
	return s.samlView(ctx, SAMLSettings{})
}

// SPMetadata is the service provider metadata XML an identity provider registers; ErrNotFound until a SAML provider is
// saved, since the key pair is made then.
func (s *Service) SPMetadata(ctx context.Context) ([]byte, error) {
	st, err := s.cfg.SAML.GetSAML(ctx)
	if err != nil {
		return nil, err
	}
	if st.SPKey == "" {
		return nil, fmt.Errorf("%w: no SAML provider is saved", apperrs.ErrNotFound)
	}
	values, err := s.spValues(ctx)
	if err != nil {
		return nil, err
	}
	sp, err := serviceProvider(st, values)
	if err != nil {
		return nil, err
	}
	return spMetadataXML(sp)
}

func (s *Service) samlChanged(e SAMLChangedEvent, at time.Time) eventbus.OutboxEvent {
	e.ChangedAt = at.UTC()
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicSAMLChanged, Payload: e}
}
