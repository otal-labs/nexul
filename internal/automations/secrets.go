package automations

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// secretNamePattern mirrors GitHub Actions' secret naming: an
// identifier every automation can reach as ctx.secrets.NAME.
var secretNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// SecretMeta is the safe, value-free view of a workspace secret — the only
// shape any API response ever returns ("write-only after save").
type SecretMeta struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SecretsRepo is the consumer-side persistence contract for the secrets pool, encrypted at rest (platform/crypto).
type SecretsRepo interface {
	Set(ctx context.Context, name, value string, now time.Time) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]SecretMeta, error)
	// All returns every secret decrypted as name->value, for the delivery layer only, never an HTTP response.
	All(ctx context.Context) (map[string]string, error)
}

// SecretsService is the secrets use-case layer (ADR 0047): a shared, GitHub-Actions-style pool, values never read back.
type SecretsService struct {
	repo SecretsRepo
	perm PermissionGate
	now  func() time.Time
}

// NewSecretsService wires the secrets use-cases over the given repo and the
// shared permission gate.
func NewSecretsService(repo SecretsRepo, perm PermissionGate) *SecretsService {
	return &SecretsService{repo: repo, perm: perm, now: time.Now}
}

// Set creates or replaces name's value. There is no partial update — saving
// always overwrites the whole value, matching GitHub Actions secrets.
func (s *SecretsService) Set(ctx context.Context, actorID, name, value string) error {
	if err := requirePermission(ctx, s.perm, actorID, permissions.AutomationsWrite); err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if !secretNamePattern.MatchString(name) {
		return fmt.Errorf("%w: secret name must match %s", apperrs.ErrInvalid, secretNamePattern.String())
	}
	if value == "" {
		return fmt.Errorf("%w: secret value is required", apperrs.ErrInvalid)
	}
	if err := s.repo.Set(ctx, name, value, s.now().UTC()); err != nil {
		return fmt.Errorf("set secret %s: %w", name, err)
	}
	return nil
}

// Delete removes name from the pool. Deleting a name that was never set is
// a no-op rather than an error, matching Automation deletion's shape.
func (s *SecretsService) Delete(ctx context.Context, actorID, name string) error {
	if err := requirePermission(ctx, s.perm, actorID, permissions.AutomationsWrite); err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%w: secret name is required", apperrs.ErrInvalid)
	}
	if err := s.repo.Delete(ctx, name); err != nil {
		return fmt.Errorf("delete secret %s: %w", name, err)
	}
	return nil
}

// List returns every secret's name and timestamps — never a value.
func (s *SecretsService) List(ctx context.Context, actorID string) ([]SecretMeta, error) {
	if err := requirePermission(ctx, s.perm, actorID, permissions.AutomationsRead); err != nil {
		return nil, err
	}
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	return list, nil
}

// Secrets returns every secret decrypted as name->value; ungated, an internal call the dial-in host makes.
func (s *SecretsService) Secrets(ctx context.Context) (map[string]string, error) {
	all, err := s.repo.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load secrets: %w", err)
	}
	return all, nil
}

// SecretsProvider is the narrow consumer seam the dial-in layer uses to
// build ctx.secrets for a connected automation (ADR 0046).
type SecretsProvider interface {
	// AllSecrets returns every workspace secret's plaintext name/value pair.
	AllSecrets(ctx context.Context) (map[string]string, error)
}

// AllSecrets satisfies SecretsProvider so the dial-in layer consumes the
// real pool directly.
func (s *SecretsService) AllSecrets(ctx context.Context) (map[string]string, error) {
	return s.Secrets(ctx)
}

// NoopSecretsProvider keeps dial-in usable with no secrets pool wired
// (tests, stripped-down composition roots).
type NoopSecretsProvider struct{}

// AllSecrets always returns an empty pool.
func (NoopSecretsProvider) AllSecrets(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
