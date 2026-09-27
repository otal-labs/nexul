package automations

import (
	"context"
	"time"
)

// InstanceHostName names the automations host bundled with the instance; an automation with no host runs there.
const InstanceHostName = "instance"

// Host is an enrolled automations host: a process on some machine that runs the workers of the automations placed on it.
type Host struct {
	ID      string
	Name    string
	Machine string
	OS      string
	Arch    string
	Version string
	// LastSeen is its latest assignments poll, or its enrollment before the first one.
	LastSeen  time.Time
	CreatedAt time.Time
}

// HostView is an automations host as the list shows it; connected means it polled within hostOnlineWindow.
type HostView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Machine   string    `json:"machine"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	Version   string    `json:"version"`
	Connected bool      `json:"connected"`
	LastSeen  time.Time `json:"last_seen"`
}

// HostEnrollmentCode is a stored one-time code, bound to the host name it enrolls and optionally its machine.
type HostEnrollmentCode struct {
	CodeHash  string
	Name      string
	Machine   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// HostCredential is a host's stored credential hash; a removed host's stays behind revoked.
type HostCredential struct {
	Hash     string
	HostID   string
	HostName string
	Revoked  bool
}

// HostEnrollment is a freshly minted code with the one-line install commands that carry it.
type HostEnrollment struct {
	Code      string              `json:"code"`
	ExpiresAt time.Time           `json:"expires_at"`
	Commands  HostInstallCommands `json:"commands"`
}

// HostInstallCommands are the rendered installer one-liners, one per shell.
type HostInstallCommands struct {
	Unix    string `json:"unix"`
	Windows string `json:"windows"`
}

// HostEnrollRequest is what `nexul install automations` sends to trade a code for a credential.
type HostEnrollRequest struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Version string `json:"version"`
}

// HostEnrolled is the new host's record and its credential, returned exactly once.
type HostEnrolled struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Machine    string `json:"machine"`
	Credential string `json:"credential"`
}

// Assignment is one enabled automation placed on the polling host, with the token its worker dials in with.
type Assignment struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

// Assignments is the GET /api/automation-hosts/self/assignments response.
type Assignments struct {
	Automations []Assignment `json:"automations"`
}

// HostRepo persists automations hosts with their enrollment codes and credentials.
type HostRepo interface {
	HostStore
	HostCredentialStore
}

// HostStore persists the host rows.
type HostStore interface {
	GetHost(ctx context.Context, id string) (*Host, error)
	// GetHostByName returns the host enrolled under name, or apperrs.ErrNotFound.
	GetHostByName(ctx context.Context, name string) (*Host, error)
	ListHosts(ctx context.Context) ([]Host, error)
	TouchHost(ctx context.Context, id string, at time.Time) error
}

// HostCredentialStore persists enrollment codes and host credentials, both by hash only.
type HostCredentialStore interface {
	// CreateHostEnrollment stores a code and prunes every expired one.
	CreateHostEnrollment(ctx context.Context, e *HostEnrollmentCode) error
	// GetHostEnrollment returns the unexpired code with codeHash, or apperrs.ErrNotFound.
	GetHostEnrollment(ctx context.Context, codeHash string, now time.Time) (*HostEnrollmentCode, error)
	// EnrollHost consumes the code, creates the host and stores its credential in one transaction; a code already
	// used or expired is apperrs.ErrNotFound, a taken name apperrs.ErrConflict.
	EnrollHost(ctx context.Context, codeHash string, h *Host, credentialHash string, now time.Time) error
	// GetHostCredential returns the credential with hash, revoked ones included, or apperrs.ErrNotFound.
	GetHostCredential(ctx context.Context, hash string) (*HostCredential, error)
	// LiveHostCredential returns the host's unrevoked credential, or apperrs.ErrNotFound.
	LiveHostCredential(ctx context.Context, hostID string) (*HostCredential, error)
	// RemoveHost revokes the host's credentials and deletes it, moving its automations back to the instance host;
	// an unknown id is apperrs.ErrNotFound.
	RemoveHost(ctx context.Context, id string, at time.Time) error
}

// InstanceURLReader resolves the configured instance URL the install commands point at.
type InstanceURLReader interface {
	GetInstanceURL(ctx context.Context) (string, error)
}
