package connectors

import (
	"context"
	"errors"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/secretref"
)

// SecretValues turns a stored secret into the value to use: a literal as it is, a reference through its manager.
type SecretValues interface {
	// Value answers from memory, reading a reference at most when its backoff allows.
	Value(ctx context.Context, stored string) (string, error)
	// Read always asks the manager, for a save to verify what it is about to store.
	Read(ctx context.Context, stored string) (string, error)
}

// SecretProblem is a stored reference that cannot be resolved now; no value or reference is in it.
type SecretProblem struct {
	// Manager is the scheme of the reference, such as op.
	Manager string `json:"manager"`
	// State is set when the manager is the cause: "unreachable" or "token rejected".
	State string `json:"state,omitempty"`
	// Error is a fixed reason when the manager is not the cause, such as "not found"; never the manager's own text.
	Error string `json:"error,omitempty"`
}

// ResolvedAppConfigs reads app configs with their references resolved, for every consumer that uses the secret.
// The Service keeps the raw store, so a resolved value is never written back.
type ResolvedAppConfigs struct {
	AppConfigStore
	Secrets SecretValues
}

// GetAppConfig returns the app config with its client secret and private key resolved.
func (r ResolvedAppConfigs) GetAppConfig(ctx context.Context, connectorID string) (AppConfig, error) {
	cfg, err := r.AppConfigStore.GetAppConfig(ctx, connectorID)
	if err != nil {
		return AppConfig{}, err
	}
	if cfg, err = resolveApp(ctx, r.Secrets.Value, cfg); err != nil {
		return AppConfig{}, fmt.Errorf("resolve %s app config: %w", connectorID, err)
	}
	return cfg, nil
}

func resolveApp(ctx context.Context, resolve func(context.Context, string) (string, error), cfg AppConfig) (AppConfig, error) {
	var err error
	if cfg.ClientSecret, err = resolve(ctx, cfg.ClientSecret); err != nil {
		return AppConfig{}, err
	}
	if cfg.PrivateKey, err = resolve(ctx, cfg.PrivateKey); err != nil {
		return AppConfig{}, err
	}
	return cfg, nil
}

// managerError tags a failed secret with its manager's name, so a refusal can say whose it was.
type managerError struct {
	manager string
	err     error
}

func (e *managerError) Error() string { return e.err.Error() }
func (e *managerError) Unwrap() error { return e.err }

func tagged(stored string, err error) error {
	if err == nil {
		return nil
	}
	if kind, ref := secretref.Parse(stored); kind == secretref.Reference {
		return &managerError{manager: secretref.ManagerName(ref.Scheme), err: err}
	}
	return err
}

func (s *Service) value(ctx context.Context, stored string) (string, error) {
	if s.secrets == nil {
		return stored, nil
	}
	v, err := s.secrets.Value(ctx, stored)
	return v, tagged(stored, err)
}

func (s *Service) read(ctx context.Context, stored string) (string, error) {
	if s.secrets == nil {
		return stored, nil
	}
	v, err := s.secrets.Read(ctx, stored)
	return v, tagged(stored, err)
}

// problem describes why stored cannot be used now, or nil when it is a literal or resolves.
func (s *Service) problem(ctx context.Context, stored string) *SecretProblem {
	kind, ref := secretref.Parse(stored)
	if kind != secretref.Reference {
		return nil
	}
	_, err := s.value(ctx, stored)
	if err == nil {
		return nil
	}
	if errors.Is(err, secretref.ErrUnreachable) {
		return &SecretProblem{Manager: ref.Scheme, State: secretref.ManagerUnreachable}
	}
	if errors.Is(err, secretref.ErrTokenRejected) {
		return &SecretProblem{Manager: ref.Scheme, State: secretref.ManagerTokenRejected}
	}
	return &SecretProblem{Manager: ref.Scheme, Error: secretref.Reason(secretref.ManagerName(ref.Scheme), err)}
}

// firstProblem is the first stored value that cannot be resolved.
func (s *Service) firstProblem(ctx context.Context, stored ...string) *SecretProblem {
	for _, v := range stored {
		if p := s.problem(ctx, v); p != nil {
			return p
		}
	}
	return nil
}

// refuse is a save's answer for a secret that did not resolve, naming the field and why.
func refuse(label string, err error) error {
	manager := "secrets manager"
	var tag *managerError
	if errors.As(err, &tag) {
		manager = tag.manager
	}
	return fmt.Errorf("%w: %s: %s", apperrs.ErrInvalid, label, secretref.Reason(manager, err))
}

// StoredSecrets is every secret the connectors hold, so one warm read can cover them with the other consumers'.
func (s *Service) StoredSecrets(ctx context.Context) []string {
	var stored []string
	for id, c := range s.registry {
		cred, err := s.store.GetCredentials(ctx, id)
		if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
			continue
		}
		stored = append(stored, s.storedSecrets(ctx, c, cred)...)
	}
	return stored
}

// Warm returns what could not be resolved by connector; a warm read of StoredSecrets before it puts the values in memory.
func (s *Service) Warm(ctx context.Context) map[string]*SecretProblem {
	problems := map[string]*SecretProblem{}
	for id, c := range s.registry {
		cred, err := s.store.GetCredentials(ctx, id)
		if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
			continue
		}
		if p := s.connectorProblem(ctx, c, cred); p != nil {
			problems[id] = p
		}
	}
	return problems
}
