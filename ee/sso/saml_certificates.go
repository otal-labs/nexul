package sso

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

const (
	// samlRefreshAfter is how old fetched metadata gets before the loop reads its URL again.
	samlRefreshAfter = 24 * time.Hour
	// certificateWarning is how long before a certificate's expiry the card warns.
	certificateWarning = 30 * 24 * time.Hour
)

// The uses a certificate in the view has.
const (
	CertificateSP         = "sp"
	CertificateSPPrevious = "sp_previous"
	CertificateIdP        = "idp"
)

// CertificateView is one certificate's expiry as the card shows it.
type CertificateView struct {
	// Use is sp (Nexul's, signing its requests), sp_previous (Nexul's old one, published until retired) or idp.
	Use      string    `json:"use"`
	NotAfter time.Time `json:"not_after"`
	// ExpiresSoon is true within 30 days of NotAfter, and after it.
	ExpiresSoon bool `json:"expires_soon"`
}

// certificates lists the expiry of every certificate the settings hold; one that does not parse is left out.
func certificates(st SAMLSettings, now time.Time) []CertificateView {
	out := []CertificateView{}
	add := func(use string, der []byte) {
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return
		}
		out = append(out, CertificateView{Use: use, NotAfter: cert.NotAfter.UTC(), ExpiresSoon: !now.Add(certificateWarning).Before(cert.NotAfter)})
	}
	if block, _ := pem.Decode([]byte(st.SPCertificate)); block != nil {
		add(CertificateSP, block.Bytes)
	}
	if block, _ := pem.Decode([]byte(st.SPPreviousCertificate)); block != nil {
		add(CertificateSPPrevious, block.Bytes)
	}
	for _, c := range st.IdP.Certificates {
		der, _ := base64.StdEncoding.DecodeString(c)
		add(CertificateIdP, der)
	}
	return out
}

// RotateSAMLCertificate makes Nexul a new key pair, signs requests with it from now on and publishes the old certificate
// beside it until RetireSAMLCertificate; it needs instance:write, and refuses while an old one is still published.
func (s *Service) RotateSAMLCertificate(ctx context.Context) (SAMLView, error) {
	cur, err := s.certificateChange(ctx)
	if err != nil {
		return SAMLView{}, err
	}
	if cur.SPPreviousCertificate != "" {
		return SAMLView{}, fmt.Errorf("%w: an old certificate is still published; retire it once the provider uses the new one, then rotate again", apperrs.ErrConflict)
	}
	next := cur
	next.UpdatedBy, next.UpdatedAt = actorID(ctx), s.cfg.Now()
	if next.SPKey, next.SPCertificate, err = s.spKeyPair(ctx, next.UpdatedAt); err != nil {
		return SAMLView{}, err
	}
	next.SPPreviousCertificate = cur.SPCertificate
	return s.saveCertificates(ctx, next)
}

// RetireSAMLCertificate stops publishing the certificate a rotation replaced; with none to retire it changes nothing.
func (s *Service) RetireSAMLCertificate(ctx context.Context) (SAMLView, error) {
	cur, err := s.certificateChange(ctx)
	if err != nil {
		return SAMLView{}, err
	}
	if cur.SPPreviousCertificate == "" {
		return s.samlView(ctx, cur)
	}
	next := cur
	next.SPPreviousCertificate = ""
	next.UpdatedBy, next.UpdatedAt = actorID(ctx), s.cfg.Now()
	return s.saveCertificates(ctx, next)
}

// certificateChange is the saved provider a rotation or a retirement changes; it works without a key, so a lapsed
// license never leaves a certificate expiring.
func (s *Service) certificateChange(ctx context.Context) (SAMLSettings, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return SAMLSettings{}, err
	}
	cur, err := s.cfg.SAML.GetSAML(ctx)
	if err != nil {
		return SAMLSettings{}, err
	}
	if cur.SPKey == "" {
		return SAMLSettings{}, fmt.Errorf("%w: no SAML provider is saved", apperrs.ErrInvalid)
	}
	return cur, nil
}

func (s *Service) saveCertificates(ctx context.Context, next SAMLSettings) (SAMLView, error) {
	evt := s.samlChanged(SAMLChangedEvent{DisplayName: next.DisplayName, EntityID: next.IdP.EntityID, Only: next.Only, ChangedBy: next.UpdatedBy}, next.UpdatedAt)
	if err := s.cfg.SAML.SetSAML(ctx, next, evt); err != nil {
		return SAMLView{}, err
	}
	return s.samlView(ctx, next)
}

// refreshMetadata reads a metadata URL due for it again. What it reads replaces the certificates, so one the provider
// adds verifies at once and one it drops stops; a failure, or metadata naming another provider, keeps the last good
// ones and records why. The change is published only when the card would show something new.
func (s *Service) refreshMetadata(ctx context.Context) error {
	st, err := s.cfg.SAML.GetSAML(ctx)
	if err != nil || st.MetadataURL == "" {
		return err
	}
	now := s.cfg.Now()
	if now.Sub(st.RefreshedAt) < samlRefreshAfter {
		return nil
	}
	next := st
	next.RefreshedAt, next.RefreshError = now, ""
	idp, err := s.fetchIdP(ctx, st.MetadataURL)
	if err == nil && idp.EntityID != st.IdP.EntityID {
		err = errAnotherProvider(idp.EntityID)
	}
	if err != nil {
		next.RefreshError = err.Error()
	}
	if err == nil {
		next.IdP = idp
	}
	var evts []eventbus.OutboxEvent
	if !reflect.DeepEqual(next.IdP, st.IdP) || next.RefreshError != st.RefreshError {
		evts = append(evts, s.samlChanged(SAMLChangedEvent{DisplayName: st.DisplayName, EntityID: st.IdP.EntityID, Only: st.Only}, now))
	}
	return s.cfg.SAML.RefreshSAML(ctx, st.MetadataURL, next.IdP, next.RefreshedAt, next.RefreshError, evts...)
}

// RunSAMLRefreshLoop reads a due metadata URL on every tick until ctx is cancelled; the provider publishes by the
// clock, so this is a ticker, not the commit broadcast.
func (s *Service) RunSAMLRefreshLoop(ctx context.Context, tick time.Duration, log *slog.Logger) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.refreshMetadata(ctx); err != nil {
				log.Warn("saml metadata refresh failed", "error", err)
			}
		}
	}
}
