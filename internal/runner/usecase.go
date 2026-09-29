package runner

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/release"
	"github.com/otal-labs/nexul/internal/platform/version"
)

// Dispatch is the live, connection-scoped state of the WS layer the use-cases read (ADR 0019).
type Dispatch interface {
	// Runners returns the currently connected runners and their running job.
	Runners() []RunnerStatus
	// Queue returns the deploys waiting for a runner, in FIFO order.
	Queue() []QueuedJob
	// Discover asks one idle, connected runner on machine to scan its host, blocking up to timeout (spec §8).
	Discover(ctx context.Context, machine string, timeout time.Duration) (DiscoverReport, error)
	// Uninstall tells a connected runner to remove itself and drops its connection; a no-op when it is offline.
	Uninstall(ctx context.Context, runnerID string)
}

// RunnerStatus pairs a live connection with its running job.
type RunnerStatus struct {
	RunnerID   string
	Name       string
	RunningJob *RunningJob
}

// SettingsReader is the consumer-side view of the instance settings the install use-case needs (ADR 0017).
type SettingsReader interface {
	GetInstanceURL(ctx context.Context) (string, error)
}

// InstallConfig wires the instance URL the install commands carry and the release lookups downloads use.
type InstallConfig struct {
	// Settings resolves the instance URL; nil leaves it empty.
	Settings SettingsReader
	// Release wires the GitHub release lookups behind GET /api/runners/download/{target}
	// and the runner's own LatestVersion/Download; also shared with the /api/version handler for one cache.
	Release *release.Client
}

// Service is the runner use-case layer (ADR 0019): runner and queue visibility over the repo + live dispatch state.
type Service struct {
	repo     Repo
	live     Dispatch
	install  InstallConfig
	machines MachineRepo
	managed  ManagedLookup
	tunnels  TunnelDescriber
	upgrades UpgradeRepo
	bus      Publisher
	admin    identity.InstanceAdmin
	gate     Gate
	// automationsHosts lets machine_list show each machine's automations hosts; nil lists none.
	automationsHosts AutomationsHostLister
	// enrollDir holds the bundled runner's enrollment code file while it is not enrolled.
	enrollDir string
	// now is the clock UpgradeStatus/ResolvePendingUpgrade check the 15-minute timeout against; overridden only
	// in tests to exercise the expiry without sleeping.
	now func() time.Time
}

// NewService wires the runner use-cases over the given repo and dispatch view.
func NewService(repo Repo, live Dispatch) *Service {
	return &Service{repo: repo, live: live, now: time.Now}
}

// ReleaseClient returns the release client wired via InstallConfig, so other adapters (the /api/version handler)
// can share its cache instead of standing up a second one.
func (s *Service) ReleaseClient() *release.Client {
	return s.install.Release
}

// WithInstall attaches the install config and returns the same Service, for chaining onto NewService.
func (s *Service) WithInstall(cfg InstallConfig) *Service {
	s.install = cfg
	return s
}

// WithMachines attaches the machine repo and returns the same Service, for chaining onto NewService.
func (s *Service) WithMachines(machines MachineRepo) *Service {
	s.machines = machines
	return s
}

// AutomationsHost is an automations host as machine_list files it under its machine.
type AutomationsHost struct {
	ID        string
	Name      string
	Machine   string
	Version   string
	Connected bool
	LastSeen  time.Time
}

// AutomationsHostLister is the consumer-side slice of the automations hosts machine_list reads (ADR 0017).
type AutomationsHostLister interface {
	ListAutomationsHosts(ctx context.Context) ([]AutomationsHost, error)
}

// WithAutomationsHosts attaches the automations hosts machine_list shows beside each machine's runners.
func (s *Service) WithAutomationsHosts(l AutomationsHostLister) *Service {
	s.automationsHosts = l
	return s
}

// ManagedLookup names the containers stacks on a machine already track, so discovery for the import door can
// leave them out (issue 08: "everything is ticked by default except containers a Nexul stack already
// manages"). Optional; nil means discovery reports everything.
type ManagedLookup interface {
	ListContainerNamesByMachine(ctx context.Context, machine string) ([]string, error)
}

// WithTunnelDescriber lets DiscoverForImport describe discovered cloudflared tunnels through the dns domain (ADR 0017).
func (s *Service) WithTunnelDescriber(d TunnelDescriber) *Service {
	s.tunnels = d
	return s
}

// WithUpgrades attaches the instance-upgrade repo and returns the same Service, for chaining onto NewService.
func (s *Service) WithUpgrades(upgrades UpgradeRepo) *Service {
	s.upgrades = upgrades
	return s
}

// WithBus attaches the event bus UpgradeStatus/RequestUpgrade announce instance.upgrade_changed on, for chaining
// onto NewService.
func (s *Service) WithBus(bus Publisher) *Service {
	s.bus = bus
	return s
}

func (s *Service) WithManaged(managed ManagedLookup) *Service {
	s.managed = managed
	return s
}

// Gate is the permission check runners and machines pass through; both are instance-wide, so a caller needs the
// action in any workspace they belong to (the access domain, ADR 0042).
type Gate interface {
	RequireAnywhere(ctx context.Context, action permissions.Action) error
}

// WithGate wires the permission check; unset, only the server's own calls pass.
func (s *Service) WithGate(g Gate) *Service {
	s.gate = g
	return s
}

func (s *Service) require(ctx context.Context, action permissions.Action) error {
	if s.gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.gate.RequireAnywhere(ctx, action)
}

// ListRunners returns every runner with its presence state and current job; unseen runners are absent.
func (s *Service) ListRunners(ctx context.Context) ([]RunnerView, error) {
	if err := s.require(ctx, permissions.RunnersRead); err != nil {
		return nil, err
	}
	rs, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list runners: %w", err)
	}
	running := make(map[string]*RunningJob, len(rs))
	for _, st := range s.live.Runners() {
		running[st.RunnerID] = st.RunningJob
	}
	machineNames := s.machineNames(ctx)
	out := make([]RunnerView, 0, len(rs))
	for _, r := range rs {
		out = append(out, RunnerView{
			ID:         r.ID,
			Name:       r.Name,
			Version:    r.Version,
			Connected:  r.Connected,
			LastSeen:   r.LastSeen,
			RunningJob: running[r.ID],
			Machine:    machineNames[r.MachineID],
		})
	}
	return out, nil
}

// machineNames resolves every machine id to its name in one pass; a lookup failure yields an empty map, so a
// runner list still renders without machine names rather than failing outright.
func (s *Service) machineNames(ctx context.Context) map[string]string {
	if s.machines == nil {
		return nil
	}
	ms, err := s.machines.List(ctx)
	if err != nil {
		return nil
	}
	out := make(map[string]string, len(ms))
	for _, m := range ms {
		out[m.ID] = m.Name
	}
	return out
}

// ListQueue returns the deploys waiting for a runner, FIFO order.
func (s *Service) ListQueue(ctx context.Context) ([]QueuedJob, error) {
	if err := s.require(ctx, permissions.RunnersRead); err != nil {
		return nil, err
	}
	return s.live.Queue(), nil
}

// ListMachines returns every machine.
func (s *Service) ListMachines(ctx context.Context) ([]*Machine, error) {
	if s.machines == nil {
		return nil, nil
	}
	return s.machines.List(ctx)
}

// GetMachine returns one machine by id.
func (s *Service) GetMachine(ctx context.Context, id string) (*Machine, error) {
	if s.machines == nil {
		return nil, fmt.Errorf("%w: machines are not configured", apperrs.ErrConflict)
	}
	return s.machines.Get(ctx, id)
}

// UpdateMachine renames a machine and/or changes its stack root; an empty field leaves that setting unchanged.
func (s *Service) UpdateMachine(ctx context.Context, id, name, stackRoot string) (*Machine, error) {
	if s.machines == nil {
		return nil, fmt.Errorf("%w: machines are not configured", apperrs.ErrConflict)
	}
	if err := s.require(ctx, permissions.MachinesWrite); err != nil {
		return nil, err
	}
	if name != "" {
		if err := s.machines.Rename(ctx, id, name); err != nil {
			return nil, fmt.Errorf("rename machine %s: %w", id, err)
		}
	}
	if stackRoot != "" {
		if err := s.machines.SetStackRoot(ctx, id, stackRoot); err != nil {
			return nil, fmt.Errorf("set machine %s stack root: %w", id, err)
		}
	}
	return s.machines.Get(ctx, id)
}

// Discover resolves machineID to its name and asks the live dispatch layer to run a discover job (spec §8).
func (s *Service) Discover(ctx context.Context, machineID string) (DiscoverReport, error) {
	_, report, err := s.discover(ctx, machineID)
	return report, err
}

// DiscoverUnmanaged is Discover minus the containers a stack on the machine already tracks: what the import
// door lists. Import itself uses Discover, since the containers it adopts are unmanaged until it writes them.
func (s *Service) DiscoverUnmanaged(ctx context.Context, machineID string) (DiscoverReport, error) {
	machine, report, err := s.discover(ctx, machineID)
	if err != nil || s.managed == nil {
		return report, err
	}
	names, err := s.managed.ListContainerNamesByMachine(ctx, machine)
	if err != nil {
		return DiscoverReport{}, fmt.Errorf("list managed containers on machine %s: %w", machine, err)
	}
	tracked := make(map[string]bool, len(names))
	for _, n := range names {
		tracked[n] = true
	}
	kept := report.Containers[:0]
	for _, c := range report.Containers {
		if !tracked[c.Name] {
			kept = append(kept, c)
		}
	}
	report.Containers = kept
	return report, nil
}

// DiscoverForImport is what the import door renders: unmanaged containers grouped, with every recognised
// cloudflared gateway described through the provider so hostnames already routed to this machine are visible
// before anything is adopted. A describe failure (no Cloudflare connector, unknown tunnel) is carried on the
// row as TunnelError rather than failing the whole scan.
func (s *Service) DiscoverForImport(ctx context.Context, machineID string) (GroupedDiscovery, error) {
	report, err := s.DiscoverUnmanaged(ctx, machineID)
	if err != nil {
		return GroupedDiscovery{}, err
	}
	grouped := GroupDiscovery(report)
	if s.tunnels == nil {
		return grouped, nil
	}
	for i := range grouped.Gateways {
		g := &grouped.Gateways[i]
		if g.TunnelID == "" {
			continue
		}
		info, err := s.tunnels.DescribeTunnel(ctx, g.TunnelID)
		if err != nil {
			g.TunnelError = err.Error()
			continue
		}
		g.Tunnel = info
	}
	return grouped, nil
}

func (s *Service) discover(ctx context.Context, machineID string) (string, DiscoverReport, error) {
	if s.machines == nil {
		return "", DiscoverReport{}, fmt.Errorf("%w: machines are not configured", apperrs.ErrConflict)
	}
	m, err := s.machines.Get(ctx, machineID)
	if err != nil {
		return "", DiscoverReport{}, fmt.Errorf("get machine %s: %w", machineID, err)
	}
	report, err := s.live.Discover(ctx, m.Name, discoverTimeout)
	if err != nil {
		return "", DiscoverReport{}, fmt.Errorf("discover on machine %s: %w", m.Name, err)
	}
	return m.Name, report, nil
}

// instanceURL reads the configured instance URL, or "" when no Settings reader is wired (local/dev composition).
func (s *Service) instanceURL(ctx context.Context) (string, error) {
	if s.install.Settings == nil {
		return "", nil
	}
	return s.install.Settings.GetInstanceURL(ctx)
}

// InstallDownloadURL: the instance URL's scheme+host if set (same origin, unlike the WS port), else the request's.
func InstallDownloadURL(instanceURL, requestHost string, requestTLS bool) string {
	if instanceURL != "" {
		if u, err := url.Parse(instanceURL); err == nil && u.Host != "" {
			return u.Scheme + "://" + u.Host + "/api/runners/download"
		}
	}
	scheme := "http"
	if requestTLS {
		scheme = "https"
	}
	return scheme + "://" + requestHost + "/api/runners/download"
}

// upgradeResolveWindow bounds how long a pending/started record may sit unresolved before it fails; nothing confirms
// the services survived their own restart, so the booted version is the only signal, checked lazily and at boot.
const upgradeResolveWindow = 15 * time.Minute

// WithAdminGate attaches the instance-admin fact the upgrade use-cases require of their caller.
func (s *Service) WithAdminGate(admin identity.InstanceAdmin) *Service {
	s.admin = admin
	return s
}

// UpgradeStatus reports the running version, the channel's newest release, and whether an upgrade can start now.
// Only an instance admin may read it.
// RefreshReleases makes the next release lookup ask GitHub instead of the cache, for a release published moments ago.
func (s *Service) RefreshReleases() {
	if s.install.Release != nil {
		s.install.Release.Refresh()
	}
}

func (s *Service) UpgradeStatus(ctx context.Context) (UpgradeStatus, error) {
	if err := identity.RequireInstanceAdmin(ctx, s.admin); err != nil {
		return UpgradeStatus{}, err
	}
	return s.upgradeStatus(ctx)
}

// upgradeStatus also lazily resolves the latest upgrade record, so a stuck upgrade fails without waiting for a reboot.
func (s *Service) upgradeStatus(ctx context.Context) (UpgradeStatus, error) {
	latest, err := s.latestUpgrade(ctx)
	if err != nil {
		return UpgradeStatus{}, err
	}
	status := UpgradeStatus{Version: version.Version, Channel: version.Channel(), Upgrade: latest}
	rel, relErr := s.install.Release.Latest(ctx, version.Channel())
	if rel != nil {
		status.Latest = &LatestRelease{Version: rel.Tag, URL: rel.URL}
		status.UpdateAvailable = rel.Tag != version.Version
	}
	status.CanUpgrade, status.Reason = s.canUpgrade(rel, relErr, latest)
	return status, nil
}

// canUpgrade is the spec's reason precedence, checked in order; the first blocking condition wins.
func (s *Service) canUpgrade(rel *release.Release, relErr error, latest *Upgrade) (bool, string) {
	if !version.IsRelease() {
		return false, "dev build"
	}
	if relErr != nil || rel == nil {
		return false, "release lookup failed"
	}
	if rel.Tag == version.Version {
		return false, "already on the newest release"
	}
	if latest.unresolved() {
		return false, "an upgrade is already in progress"
	}
	c, ok := s.instanceRunner()
	if !ok {
		return false, "instance runner is not connected"
	}
	if c.RunningJob != nil {
		return false, "instance runner is busy"
	}
	return true, ""
}

// instanceRunner returns the live status of the connected runner named instance, if there is one.
func (s *Service) instanceRunner() (RunnerStatus, bool) {
	for _, r := range s.live.Runners() {
		if r.Name == instanceRunnerName {
			return r, true
		}
	}
	return RunnerStatus{}, false
}

// latestUpgrade fetches the most recent record and applies the resolution rule; nil, nil when none exists yet.
func (s *Service) latestUpgrade(ctx context.Context) (*Upgrade, error) {
	if s.upgrades == nil {
		return nil, nil
	}
	u, err := s.upgrades.Latest(ctx)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest instance upgrade: %w", err)
	}
	return s.resolveOne(ctx, u)
}

// resolveOne applies the boot/lazy resolution rule to one record: a match on the running version completes it;
// anything else unresolved past upgradeResolveWindow fails it with the journal hint. An already-resolved
// record, or one still within the window, passes through unchanged.
func (s *Service) resolveOne(ctx context.Context, u *Upgrade) (*Upgrade, error) {
	if !u.unresolved() {
		return u, nil
	}
	if u.ToVersion == version.Version {
		return s.transitionUpgrade(ctx, u, UpgradeStatusCompleted, "")
	}
	if s.now().Sub(u.UpdatedAt) > upgradeResolveWindow {
		msg := fmt.Sprintf("instance is still on %s; run nexul upgrade on the host to see why", version.Version)
		return s.transitionUpgrade(ctx, u, UpgradeStatusFailed, msg)
	}
	return u, nil
}

// transitionUpgrade persists a status change and announces it; the returned record reflects the write.
func (s *Service) transitionUpgrade(ctx context.Context, u *Upgrade, status, errMsg string) (*Upgrade, error) {
	if err := s.upgrades.SetStatus(ctx, u.ID, status, errMsg); err != nil {
		return nil, fmt.Errorf("set instance upgrade %s status: %w", u.ID, err)
	}
	updated := *u
	updated.Status = status
	updated.Error = errMsg
	updated.UpdatedAt = s.now().UTC()
	s.publish(ctx, TopicInstanceUpgradeChanged, &updated)
	return &updated, nil
}

// RequestUpgrade starts an upgrade to the channel's newest release: upgradeStatus is the single source of the
// can_upgrade decision, so RequestUpgrade re-runs it rather than duplicating the reason precedence. A pending
// record is written, then instance.upgrade_requested asks Handler.Run's subscription to dispatch assign_upgrade
// — the same bus-topic handoff deploy.requested already uses, so dispatch has one entry point, not two.
func (s *Service) RequestUpgrade(ctx context.Context, actor string) (Upgrade, error) {
	if err := identity.RequireInstanceAdmin(ctx, s.admin); err != nil {
		return Upgrade{}, err
	}
	status, err := s.upgradeStatus(ctx)
	if err != nil {
		return Upgrade{}, err
	}
	if !status.CanUpgrade {
		return Upgrade{}, &UpgradeBlockedError{Reason: status.Reason}
	}
	instance, _ := s.instanceRunner()
	now := s.now().UTC()
	u := &Upgrade{
		ID:          ids.New(),
		FromVersion: version.Version,
		ToVersion:   status.Latest.Version,
		Status:      UpgradeStatusPending,
		RequestedBy: actor,
		RunnerID:    instance.RunnerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.upgrades.Create(ctx, u); err != nil {
		return Upgrade{}, fmt.Errorf("create instance upgrade: %w", err)
	}
	s.publish(ctx, TopicInstanceUpgradeChanged, u)
	s.publish(ctx, TopicInstanceUpgradeRequested, InstanceUpgradeRequestedEvent{ID: u.ID, Version: u.ToVersion})
	return *u, nil
}

// ResolvePendingUpgrade applies the boot-time resolution to every unresolved record (server/cmd/bootstrap.go,
// called after migrations): the instance may have restarted on the new version, or the update may have failed
// silently, and the booted version is the only signal either way.
func (s *Service) ResolvePendingUpgrade(ctx context.Context) error {
	if s.upgrades == nil {
		return nil
	}
	pending, err := s.upgrades.ListUnresolved(ctx)
	if err != nil {
		return fmt.Errorf("list unresolved instance upgrades: %w", err)
	}
	for _, u := range pending {
		if _, err := s.resolveOne(ctx, u); err != nil {
			return err
		}
	}
	return nil
}

// publish is best-effort: a live-push miss never fails the use-case that triggered it, since the HTTP/MCP
// response already carries the fresh state the event would have announced.
func (s *Service) publish(ctx context.Context, topic string, payload any) {
	if s.bus == nil {
		return
	}
	_ = s.bus.Publish(ctx, topic, payload)
}
