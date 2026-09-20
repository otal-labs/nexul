package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ automations.SecretsRepo = (*AutomationSecretsRepo)(nil)

// AutomationSecretsRepo stores the workspace secrets pool, one AES-256-GCM encrypted value per name.
type AutomationSecretsRepo struct {
	db     *sql.DB
	w      *Serializer
	encKey []byte
	q      *sqlcgen.Queries
}

// Set upserts name's value, encrypting before it reaches SQL; created_at is preserved across a replace.
func (r *AutomationSecretsRepo) Set(ctx context.Context, name, value string, now time.Time) error {
	enc, err := crypto.Encrypt(r.encKey, []byte(value))
	if err != nil {
		return fmt.Errorf("encrypt secret %s: %w", name, err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SetAutomationSecret(ctx, sqlcgen.SetAutomationSecretParams{
			Name: name, Value: enc, CreatedAt: now.Unix(), UpdatedAt: now.Unix(),
		})
		if err != nil {
			return classifyWriteErr(err)
		}
		return nil
	})
}

// Delete removes name. Deleting a name that was never set is a no-op.
func (r *AutomationSecretsRepo) Delete(ctx context.Context, name string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteAutomationSecret(ctx, name); err != nil {
			return fmt.Errorf("delete secret %s: %w", name, err)
		}
		return nil
	})
}

// List returns every secret's name and timestamps — never a value.
func (r *AutomationSecretsRepo) List(ctx context.Context) ([]automations.SecretMeta, error) {
	rows, err := r.q.ListAutomationSecretMeta(ctx)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	var out []automations.SecretMeta
	for _, row := range rows {
		out = append(out, automations.SecretMeta{
			Name:      row.Name,
			CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
			UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
		})
	}
	return out, nil
}

// All decrypts every stored secret into a name->value map; never call this on a path that reaches an HTTP response.
func (r *AutomationSecretsRepo) All(ctx context.Context) (map[string]string, error) {
	rows, err := r.q.ListAutomationSecretValues(ctx)
	if err != nil {
		return nil, fmt.Errorf("load secrets: %w", err)
	}
	out := map[string]string{}
	for _, row := range rows {
		plain, err := crypto.Decrypt(r.encKey, row.Value)
		if err != nil {
			return nil, fmt.Errorf("decrypt secret %s: %w", row.Name, err)
		}
		out[row.Name] = string(plain)
	}
	return out, nil
}
