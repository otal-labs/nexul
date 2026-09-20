package runner

import (
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type Runner struct {
	ID        string
	Name      string
	Version   string
	LastSeen  time.Time
	Connected bool
	CreatedAt time.Time
	// MachineID is the machine this runner belongs to (issue 05); empty until its first connect resolves one.
	MachineID string
}

// RunnerView is the product-facing runner record; running_job is always present, null when idle for a stable contract.
type RunnerView struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Version    string      `json:"version"`
	Connected  bool        `json:"connected"`
	LastSeen   time.Time   `json:"last_seen"`
	RunningJob *RunningJob `json:"running_job"`
	// Machine is the name of the machine this runner belongs to (issue 05); empty if unresolved.
	Machine string `json:"machine,omitempty"`
}

// RunningJob is the job a runner is executing right now, or nil when idle.
type RunningJob struct {
	ID      string      `json:"id"`
	Kind    RequestKind `json:"kind"`
	Service string      `json:"service,omitempty"`
}

// InstallInfo is what the "Add runner" flow renders as a copyable install command: WS URL, secret, download URL.
type InstallInfo struct {
	WSURL       string `json:"ws_url"`
	Secret      string `json:"secret"`
	DownloadURL string `json:"download_url"`
}

// QueuedJob is a deploy waiting for a runner; it mirrors deploy.requested's identity for correlation.
type QueuedJob struct {
	ID      string      `json:"id"`
	Kind    RequestKind `json:"kind"`
	Service string      `json:"service,omitempty"`
	Target  string      `json:"target,omitempty"`
}

// instanceRunnerID is the fixed id the bundled instance runner connects with; RequestUpgrade and dispatch
// both target this single runner, never a pool (instance-upgrade spec).
const instanceRunnerID = "instance"

// Upgrade statuses beyond UpgradeStatusStarted/UpgradeStatusFailed (protocol.go, shared with the result frame).
const (
	UpgradeStatusPending   = "pending"
	UpgradeStatusCompleted = "completed"
)

// Upgrade is one instance-upgrade record (instance-upgrade spec): written pending before dispatch, resolved to
// completed/failed by ResolvePendingUpgrade or UpgradeStatus's lazy check once the instance is back up.
type Upgrade struct {
	ID          string `json:"id"`
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
	Status      string `json:"status"`
	Error       string `json:"error"`
	RequestedBy string `json:"requested_by"`
	// RunnerID is always instanceRunnerID today; kept on the record rather than assumed, since dispatch already
	// resolves it once per request.
	RunnerID  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// unresolved reports whether u is still pending or started — "an upgrade is already in progress".
func (u *Upgrade) unresolved() bool {
	return u != nil && (u.Status == UpgradeStatusPending || u.Status == UpgradeStatusStarted)
}

// LatestRelease is the channel's newest release, as GET /api/instance/upgrade and GET /api/version both report it.
type LatestRelease struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// UpgradeStatus is the GET /api/instance/upgrade response: the facts row plus whether an upgrade can start now.
type UpgradeStatus struct {
	Version         string         `json:"version"`
	Channel         string         `json:"channel"`
	Latest          *LatestRelease `json:"latest"`
	UpdateAvailable bool           `json:"update_available"`
	CanUpgrade      bool           `json:"can_upgrade"`
	// Reason is non-empty exactly when CanUpgrade is false.
	Reason  string   `json:"reason"`
	Upgrade *Upgrade `json:"upgrade"`
}

// UpgradeBlockedError is RequestUpgrade's can_upgrade-is-false outcome; it wraps apperrs.ErrConflict so the
// HTTP adapter's generic error mapping still lands on 409, while Reason carries the body the spec wants.
type UpgradeBlockedError struct {
	Reason string
}

func (e *UpgradeBlockedError) Error() string { return e.Reason }
func (e *UpgradeBlockedError) Unwrap() error { return apperrs.ErrConflict }
