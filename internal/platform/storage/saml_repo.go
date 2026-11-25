package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/ee/sso"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ sso.SAMLRepo = (*SSORepo)(nil)

// GetSAML returns the zero SAMLSettings when the instance has none.
func (r *SSORepo) GetSAML(ctx context.Context) (sso.SAMLSettings, error) {
	row, err := r.q.GetSAMLSettings(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return sso.SAMLSettings{}, nil
	}
	if err != nil {
		return sso.SAMLSettings{}, fmt.Errorf("get saml settings: %w", err)
	}
	key, err := crypto.Decrypt(r.encKey, row.SpKey)
	if err != nil {
		return sso.SAMLSettings{}, fmt.Errorf("decrypt saml service provider key: %w", err)
	}
	var certs []string
	if err := json.Unmarshal([]byte(row.IdpCertificates), &certs); err != nil {
		return sso.SAMLSettings{}, fmt.Errorf("read saml idp certificates: %w", err)
	}
	return sso.SAMLSettings{
		MetadataURL: row.MetadataUrl, MetadataXML: row.MetadataXml,
		IdP:            sso.IdPMetadata{EntityID: row.IdpEntityID, SSOURL: row.IdpSsoUrl, Certificates: certs},
		EmailAttribute: row.EmailAttribute, NameAttribute: row.NameAttribute, DisplayName: row.DisplayName, JoinRoleID: row.JoinRoleID,
		Only: row.SamlOnly != 0, SPKey: string(key), SPCertificate: row.SpCertificate,
		UpdatedBy: row.UpdatedBy, UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}, nil
}

// SetSAML replaces the SAML settings, with its events in the same transaction.
func (r *SSORepo) SetSAML(ctx context.Context, s sso.SAMLSettings, evts ...eventbus.OutboxEvent) error {
	params, err := r.samlParams(s)
	if err != nil {
		return err
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).SetSAMLSettings(ctx, params); err != nil {
			return fmt.Errorf("set saml settings: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

// TurnOnSAMLOnly saves the settings and ends every session not started through single sign-on outside keepUsers', with
// a session.revoked event per session, in one transaction.
func (r *SSORepo) TurnOnSAMLOnly(ctx context.Context, s sso.SAMLSettings, keepUsers []string, evts ...eventbus.OutboxEvent) error {
	params, err := r.samlParams(s)
	if err != nil {
		return err
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.SetSAMLSettings(ctx, params); err != nil {
			return fmt.Errorf("set saml settings: %w", err)
		}
		revoked, err := endSessionsOutsideSSO(ctx, q, keepUsers)
		if err != nil {
			return err
		}
		return enqueueWorkspaceOutbox(ctx, tx, append(evts, revoked...))
	})
}

// DeleteSAML removes the SAML settings, with its events in the same transaction.
func (r *SSORepo) DeleteSAML(ctx context.Context, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteSAMLSettings(ctx); err != nil {
			return fmt.Errorf("delete saml settings: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

func (r *SSORepo) samlParams(s sso.SAMLSettings) (sqlcgen.SetSAMLSettingsParams, error) {
	sealed, err := crypto.Encrypt(r.encKey, []byte(s.SPKey))
	if err != nil {
		return sqlcgen.SetSAMLSettingsParams{}, fmt.Errorf("encrypt saml service provider key: %w", err)
	}
	certs, err := json.Marshal(append([]string{}, s.IdP.Certificates...))
	if err != nil {
		return sqlcgen.SetSAMLSettingsParams{}, fmt.Errorf("encode saml idp certificates: %w", err)
	}
	return sqlcgen.SetSAMLSettingsParams{
		MetadataUrl: s.MetadataURL, MetadataXml: s.MetadataXML, IdpEntityID: s.IdP.EntityID, IdpSsoUrl: s.IdP.SSOURL,
		IdpCertificates: string(certs), EmailAttribute: s.EmailAttribute, NameAttribute: s.NameAttribute,
		DisplayName: s.DisplayName, JoinRoleID: s.JoinRoleID, SamlOnly: boolToInt(s.Only), SpKey: sealed,
		SpCertificate: s.SPCertificate, UpdatedBy: s.UpdatedBy, UpdatedAt: s.UpdatedAt.Unix(),
	}, nil
}
