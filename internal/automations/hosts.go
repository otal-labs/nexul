package automations

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
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
	// Version is what it reported at its latest poll, or at enrollment before the first one that reported any.
	Version string
	// LastSeen is its latest assignments poll, or its enrollment before the first one.
	LastSeen  time.Time
	CreatedAt time.Time
	// Managers is what it last reported about the secrets managers it holds tokens for.
	Managers []HostManager
}

// Manager states a host reports for a token: it opened a client, the manager refused the token, or could not be reached.
const (
	ManagerReady         = "ready"
	ManagerTokenRejected = "token rejected"
	ManagerUnreachable   = "unreachable"
)

// HostManager is one secrets manager a host holds a token for in one workspace, and whether that token works. The
// workspace is its id, since a rename can hand a slug to another workspace. Manager names the manager by its reference
// scheme ("op"), so a second manager adds a value and no field.
type HostManager struct {
	WorkspaceID string `json:"workspace_id"`
	Manager     string `json:"manager"`
	State       string `json:"state"`
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
	// Managers is never null, so a host that reported none lists an empty array.
	Managers []HostManager `json:"managers"`
}

// HostEnrollmentCode is a stored one-time code, bound to the host name it enrolls and optionally its machine.
type HostEnrollmentCode struct {
	CodeHash string
	Name     string
	Machine  string
	// EnrolledBy is the person who made the code, checked again when it is used; empty for the server's own.
	EnrolledBy string
	CreatedAt  time.Time
	ExpiresAt  time.Time
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
	Machine string `json:"machine"`
}

// HostEnrolled is the new host's record and its credential, returned exactly once.
type HostEnrolled struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Machine    string `json:"machine"`
	Credential string `json:"credential"`
}

// Assignment is one enabled automation placed on the polling host, with the token its worker dials in with and its
// workspace: the id names the secrets manager token the host resolves its pool with, the slug is for its logs.
type Assignment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Token       string `json:"token"`
	WorkspaceID string `json:"workspace_id"`
	Workspace   string `json:"workspace"`
}

// HostWorkspace is a workspace as a host's token command names it: its id, and its slug at the time of asking.
type HostWorkspace struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

// HostWorkspaceLookup is the GET /api/automation-hosts/self/workspace response: the workspace a slug names now, and each
// workspace with automations placed on the calling host, switched on or off.
type HostWorkspaceLookup struct {
	Workspace HostWorkspace   `json:"workspace"`
	Placed    []HostWorkspace `json:"placed"`
}

// HostManagersReport is the POST /api/automation-hosts/self/managers body: every manager the host holds a token for.
type HostManagersReport struct {
	Managers []HostManager `json:"managers"`
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
	// ListHostsOn lists the hosts filed under the machines with these ids, for a caller who sees only some.
	ListHostsOn(ctx context.Context, machineIDs []string) ([]Host, error)
	// TouchHost records a poll and the version the host reported with it.
	TouchHost(ctx context.Context, id string, at time.Time, version string) error
	// SetHostManagers replaces the managers the host reported; an unknown id is apperrs.ErrNotFound. events, when set,
	// is given the managers stored before and returns the outbox events to write in the same transaction.
	SetHostManagers(ctx context.Context, id string, managers []HostManager, events func(prev []HostManager) []eventbus.OutboxEvent) error
	// WorkspaceIDBySlug is the id of the workspace that bears slug now, or apperrs.ErrNotFound.
	WorkspaceIDBySlug(ctx context.Context, slug string) (string, error)
	// WorkspaceSlug is the slug the workspace with id bears now, or apperrs.ErrNotFound.
	WorkspaceSlug(ctx context.Context, id string) (string, error)
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
