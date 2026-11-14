package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/ee/branding"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

// BrandingRepo stores the instance's one branding record: its name, accent, imported themes and logo.
type BrandingRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// Load returns the saved branding, and false when the instance has none.
func (r *BrandingRepo) Load(ctx context.Context) (branding.Record, bool, error) {
	row, err := r.q.GetInstanceBranding(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return branding.Record{}, false, nil
	}
	if err != nil {
		return branding.Record{}, false, fmt.Errorf("get instance branding: %w", err)
	}
	rec := branding.Record{ProductName: row.ProductName, Accent: row.Accent, CSS: row.Css, UpdatedBy: row.UpdatedBy, UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC()}
	if rec.Light, err = decodeTheme(row.LightTheme); err != nil {
		return branding.Record{}, false, err
	}
	if rec.Dark, err = decodeTheme(row.DarkTheme); err != nil {
		return branding.Record{}, false, err
	}
	if row.LogoSize.Valid && row.LogoSize.Int64 > 0 {
		rec.Logo = &branding.Logo{ContentType: row.LogoType, Size: int(row.LogoSize.Int64), Version: row.LogoVersion}
	}
	return rec, true, nil
}

func decodeTheme(raw string) (*branding.Theme, error) {
	if raw == "" {
		return nil, nil
	}
	var t branding.Theme
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return nil, fmt.Errorf("decode branding theme: %w", err)
	}
	return &t, nil
}

func encodeTheme(t *branding.Theme) (string, error) {
	if t == nil {
		return "", nil
	}
	b, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("encode branding theme: %w", err)
	}
	return string(b), nil
}

// Save replaces the name, accent and themes, keeping the logo, with its events in the same transaction.
func (r *BrandingRepo) Save(ctx context.Context, rec branding.Record, evts ...eventbus.OutboxEvent) error {
	light, err := encodeTheme(rec.Light)
	if err != nil {
		return err
	}
	dark, err := encodeTheme(rec.Dark)
	if err != nil {
		return err
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SaveInstanceBranding(ctx, sqlcgen.SaveInstanceBrandingParams{
			ProductName: rec.ProductName, Accent: rec.Accent, LightTheme: light, DarkTheme: dark, Css: rec.CSS, UpdatedBy: rec.UpdatedBy, UpdatedAt: rec.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save instance branding: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

// Logo returns the stored image, or ErrNotFound when the instance holds none.
func (r *BrandingRepo) Logo(ctx context.Context) (string, []byte, error) {
	row, err := r.q.GetInstanceBrandingLogo(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, apperrs.ErrNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("get instance branding logo: %w", err)
	}
	return row.LogoType, row.Logo, nil
}

// SetLogo replaces the stored image, with its events in the same transaction.
func (r *BrandingRepo) SetLogo(ctx context.Context, contentType string, data []byte, version, by string, at time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SetInstanceBrandingLogo(ctx, sqlcgen.SetInstanceBrandingLogoParams{Logo: data, LogoType: contentType, LogoVersion: version, UpdatedBy: by, UpdatedAt: at.Unix()})
		if err != nil {
			return fmt.Errorf("set instance branding logo: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

// ClearLogo removes the stored image, with its events in the same transaction.
func (r *BrandingRepo) ClearLogo(ctx context.Context, by string, at time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).ClearInstanceBrandingLogo(ctx, sqlcgen.ClearInstanceBrandingLogoParams{UpdatedBy: by, UpdatedAt: at.Unix()}); err != nil {
			return fmt.Errorf("clear instance branding logo: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

// Delete removes everything saved, with its events in the same transaction.
func (r *BrandingRepo) Delete(ctx context.Context, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteInstanceBranding(ctx); err != nil {
			return fmt.Errorf("delete instance branding: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}
