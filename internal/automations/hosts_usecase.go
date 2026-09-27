package automations

import (
	"context"
	"crypto/hmac"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

const (
	hostEnrollmentTTL         = time.Hour
	instanceHostEnrollmentTTL = 24 * time.Hour
	hostCredentialPrefix      = "nxa_"
	// instanceHostEnrollFile is read by `nexul install` to enroll the bundled automations host.
	instanceHostEnrollFile = "automations-instance"
	// hostOnlineWindow is how recently a host must have polled to count as connected.
	hostOnlineWindow = time.Minute
	// HostRemovedRefusal is the body a removed host is refused with, so it uninstalls itself instead of retrying.
	HostRemovedRefusal = `{"error":"automations_host_removed"}`
)

var (
	errInvalidHostCode = apperrs.WithCode("invalid_code", fmt.Errorf("%w: the enrollment code is unknown, used or expired", apperrs.ErrUnauthorized))
	errHostRemoved     = apperrs.WithCode("automations_host_removed", fmt.Errorf("%w: this automations host was removed", apperrs.ErrUnauthorized))
	errInvalidToken    = fmt.Errorf("%w: invalid automation token", apperrs.ErrUnauthorized)
)

// HostsService enrolls, lists and removes automations hosts and tells each one which automations to run.
type HostsService struct {
	repo        HostRepo
	automations Repo
	// key signs host-scoped automation tokens; the server's own secret, never stored.
	key         []byte
	admin       identity.InstanceAdmin
	instanceURL InstanceURLReader
	enrollDir   string
	conns       ConnectionRegistry
	now         func() time.Time
}

// NewHostsService wires the automations host use-cases; key signs the tokens workers dial in with.
func NewHostsService(repo HostRepo, automations Repo, key []byte) *HostsService {
	return &HostsService{repo: repo, automations: automations, key: key, now: time.Now}
}

// WithAdminGate attaches the instance-admin fact enrolling and removing a host requires.
func (h *HostsService) WithAdminGate(admin identity.InstanceAdmin) *HostsService {
	h.admin = admin
	return h
}

// WithInstanceURL attaches the settings reader the install commands take their server address from.
func (h *HostsService) WithInstanceURL(r InstanceURLReader) *HostsService {
	h.instanceURL = r
	return h
}

// WithEnrollDir sets the directory the bundled host's enrollment code file lives in (<data dir>/enroll).
func (h *HostsService) WithEnrollDir(dir string) *HostsService {
	h.enrollDir = dir
	return h
}

// SetConnectionRegistry wires the dial-in registry, so a removal or a move drops the affected live connections.
func (h *HostsService) SetConnectionRegistry(r ConnectionRegistry) {
	h.conns = r
}

// CreateEnrollment mints a one-hour code for a host named name, optionally filed under machine, and renders the
// install one-liners that carry it. Only an instance admin may enroll a host.
func (h *HostsService) CreateEnrollment(ctx context.Context, name, machine string) (HostEnrollment, error) {
	if err := identity.RequireInstanceAdmin(ctx, h.admin); err != nil {
		return HostEnrollment{}, err
	}
	instanceURL := ""
	if h.instanceURL != nil {
		var err error
		if instanceURL, err = h.instanceURL.GetInstanceURL(ctx); err != nil {
			return HostEnrollment{}, fmt.Errorf("get instance url: %w", err)
		}
	}
	if instanceURL == "" {
		return HostEnrollment{}, fmt.Errorf("%w: set the instance URL in settings before adding an automations host", apperrs.ErrConflict)
	}
	code, expiresAt, err := h.mintEnrollment(ctx, name, machine, hostEnrollmentTTL)
	if err != nil {
		return HostEnrollment{}, err
	}
	unix, windows := hostcred.InstallCommands("automations", instanceURL, strings.TrimSpace(name), code)
	return HostEnrollment{Code: code, ExpiresAt: expiresAt, Commands: HostInstallCommands{Unix: unix, Windows: windows}}, nil
}

func (h *HostsService) mintEnrollment(ctx context.Context, name, machine string, ttl time.Duration) (string, time.Time, error) {
	name = strings.TrimSpace(name)
	if !hostcred.NamePattern.MatchString(name) {
		return "", time.Time{}, fmt.Errorf("%w: an automations host name is 1 to 32 lowercase letters, digits or dashes, starting with a letter or digit", apperrs.ErrInvalid)
	}
	_, err := h.repo.GetHostByName(ctx, name)
	if err == nil {
		return "", time.Time{}, fmt.Errorf("%w: an automations host named %s is already enrolled; remove it first", apperrs.ErrConflict, name)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return "", time.Time{}, err
	}
	raw, hash, err := hostcred.MintCode()
	if err != nil {
		return "", time.Time{}, err
	}
	now := h.now().UTC()
	code := &HostEnrollmentCode{CodeHash: hash, Name: name, Machine: strings.TrimSpace(machine), CreatedAt: now, ExpiresAt: now.Add(ttl)}
	if err := h.repo.CreateHostEnrollment(ctx, code); err != nil {
		return "", time.Time{}, fmt.Errorf("create enrollment for automations host %s: %w", name, err)
	}
	return raw, code.ExpiresAt, nil
}

// Enroll trades a code for the host's own credential, returned once. It is the public half of enrollment, so it
// takes no actor; a host with no machine on its code is filed under its own name.
func (h *HostsService) Enroll(ctx context.Context, req HostEnrollRequest) (HostEnrolled, error) {
	now := h.now().UTC()
	codeHash := hostcred.Hash(strings.TrimSpace(req.Code))
	code, err := h.repo.GetHostEnrollment(ctx, codeHash, now)
	if errors.Is(err, apperrs.ErrNotFound) {
		return HostEnrolled{}, errInvalidHostCode
	}
	if err != nil {
		return HostEnrolled{}, err
	}
	if strings.TrimSpace(req.Name) != code.Name {
		return HostEnrolled{}, apperrs.WithCode("name_mismatch", fmt.Errorf("%w: this code enrolls an automations host named %s", apperrs.ErrConflict, code.Name))
	}
	machine := code.Machine
	if machine == "" {
		machine = code.Name
	}
	credential, credentialHash, err := hostcred.MintCredential(hostCredentialPrefix)
	if err != nil {
		return HostEnrolled{}, err
	}
	host := &Host{ID: ids.New(), Name: code.Name, Machine: machine, OS: req.OS, Arch: req.Arch, Version: req.Version, LastSeen: now, CreatedAt: now}
	err = h.repo.EnrollHost(ctx, codeHash, host, credentialHash, now)
	if errors.Is(err, apperrs.ErrNotFound) {
		return HostEnrolled{}, errInvalidHostCode
	}
	if err != nil {
		return HostEnrolled{}, fmt.Errorf("enroll automations host %s: %w", code.Name, err)
	}
	if code.Name == InstanceHostName {
		h.removeInstanceEnrollFile()
	}
	return HostEnrolled{ID: host.ID, Name: host.Name, Machine: machine, Credential: credential}, nil
}

// List returns every automations host with whether it polled recently.
func (h *HostsService) List(ctx context.Context) ([]HostView, error) {
	hosts, err := h.repo.ListHosts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list automations hosts: %w", err)
	}
	now := h.now()
	out := make([]HostView, 0, len(hosts))
	for _, host := range hosts {
		out = append(out, HostView{
			ID: host.ID, Name: host.Name, Machine: host.Machine, OS: host.OS, Arch: host.Arch, Version: host.Version,
			Connected: now.Sub(host.LastSeen) < hostOnlineWindow, LastSeen: host.LastSeen,
		})
	}
	return out, nil
}

// Remove revokes a host's credential and deletes it; its automations fall back to the instance host and their live
// connections drop. The host learns it was removed on its next poll. Only an instance admin may remove a host.
func (h *HostsService) Remove(ctx context.Context, id string) error {
	if err := identity.RequireInstanceAdmin(ctx, h.admin); err != nil {
		return err
	}
	return h.remove(ctx, id)
}

// RemoveSelf is removal asked for by the host itself, authenticated by its own credential; a host that was already
// removed has nothing left to remove.
func (h *HostsService) RemoveSelf(ctx context.Context, credential string) error {
	cred, err := h.authenticate(ctx, credential)
	if err != nil {
		return err
	}
	if cred.Revoked {
		return nil
	}
	return h.remove(ctx, cred.HostID)
}

func (h *HostsService) remove(ctx context.Context, id string) error {
	host, err := h.repo.GetHost(ctx, id)
	if err != nil {
		return fmt.Errorf("get automations host %s: %w", id, err)
	}
	placed, err := h.placedOn(ctx, host)
	if err != nil {
		return err
	}
	if err := h.repo.RemoveHost(ctx, id, h.now().UTC()); err != nil {
		return fmt.Errorf("remove automations host %s: %w", id, err)
	}
	for _, a := range placed {
		h.disconnect(a.ID, "automations host removed")
	}
	return nil
}

// Assignments authenticates a host by its credential, records the poll, and returns the enabled automations
// placed on it, each with its worker's token. A removed host gets errHostRemoved.
func (h *HostsService) Assignments(ctx context.Context, credential string) (Assignments, error) {
	cred, err := h.authenticate(ctx, credential)
	if err != nil {
		return Assignments{}, err
	}
	if cred.Revoked {
		return Assignments{}, errHostRemoved
	}
	host, err := h.repo.GetHost(ctx, cred.HostID)
	if err != nil {
		return Assignments{}, fmt.Errorf("get automations host %s: %w", cred.HostID, err)
	}
	if err := h.repo.TouchHost(ctx, host.ID, h.now().UTC()); err != nil {
		return Assignments{}, fmt.Errorf("touch automations host %s: %w", host.ID, err)
	}
	placed, err := h.placedOn(ctx, host)
	if err != nil {
		return Assignments{}, err
	}
	out := Assignments{Automations: []Assignment{}}
	for _, a := range placed {
		if !a.Enabled {
			continue
		}
		out.Automations = append(out.Automations, Assignment{ID: a.ID, Name: a.Name, Token: hostToken(h.key, a.ID, cred.Hash, a.TokenHash)})
	}
	return out, nil
}

// placedOn lists the automations host runs: those placed on it, plus the unplaced ones when it is the instance host.
func (h *HostsService) placedOn(ctx context.Context, host *Host) ([]Automation, error) {
	all, err := h.automations.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list automations: %w", err)
	}
	var out []Automation
	for _, a := range all {
		if placedOn(&a, host) {
			out = append(out, a)
		}
	}
	return out, nil
}

func placedOn(a *Automation, host *Host) bool {
	if a.HostID == nil {
		return host.Name == InstanceHostName
	}
	return *a.HostID == host.ID
}

// authenticateToken checks a host-scoped automation token: it holds only while the automation is placed on the
// host whose live credential it was minted from, and while the automation's own token is neither rotated nor revoked.
func (h *HostsService) authenticateToken(ctx context.Context, raw string) (*Automation, error) {
	id, mac, ok := parseHostToken(raw)
	if !ok || len(h.key) == 0 {
		return nil, errInvalidToken
	}
	a, err := h.automations.Get(ctx, id)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, errInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("authenticate automation token: %w", err)
	}
	if a.TokenRevokedAt != nil {
		return nil, fmt.Errorf("%w: automation token revoked", apperrs.ErrUnauthorized)
	}
	host, err := h.placement(ctx, a)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, errInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("authenticate automation token: %w", err)
	}
	cred, err := h.repo.LiveHostCredential(ctx, host.ID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, errInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("authenticate automation token: %w", err)
	}
	if !hmac.Equal(mac, hostTokenMAC(h.key, a.ID, cred.Hash, a.TokenHash)) {
		return nil, errInvalidToken
	}
	return a, nil
}

// placement resolves the host an automation runs on; an unplaced one runs on the instance host.
func (h *HostsService) placement(ctx context.Context, a *Automation) (*Host, error) {
	if a.HostID != nil {
		return h.repo.GetHost(ctx, *a.HostID)
	}
	return h.repo.GetHostByName(ctx, InstanceHostName)
}

// authenticate resolves a raw host credential; an empty or unknown one is apperrs.ErrUnauthorized, while a revoked
// one is returned so the caller can say the host was removed.
func (h *HostsService) authenticate(ctx context.Context, raw string) (*HostCredential, error) {
	if raw == "" {
		return nil, apperrs.ErrUnauthorized
	}
	cred, err := h.repo.GetHostCredential(ctx, hostcred.Hash(raw))
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, apperrs.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	return cred, nil
}

func (h *HostsService) disconnect(automationID, reason string) {
	if h.conns != nil {
		h.conns.Disconnect(automationID, reason)
	}
}

// WriteInstanceEnrollment keeps <enroll dir>/automations-instance in step with the bundled host: a fresh 24-hour
// code while no host named instance is enrolled, no file once it is. Called at boot.
func (h *HostsService) WriteInstanceEnrollment(ctx context.Context) error {
	_, err := h.repo.GetHostByName(ctx, InstanceHostName)
	if err == nil {
		h.removeInstanceEnrollFile()
		return nil
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return err
	}
	code, _, err := h.mintEnrollment(ctx, InstanceHostName, "", instanceHostEnrollmentTTL)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(h.enrollDir, 0o700); err != nil {
		return fmt.Errorf("create enroll dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(h.enrollDir, instanceHostEnrollFile), []byte(code), 0o600); err != nil {
		return fmt.Errorf("write instance automations host enrollment code: %w", err)
	}
	return nil
}

// removeInstanceEnrollFile is best-effort: a leftover file only holds a code that no longer enrolls anything.
func (h *HostsService) removeInstanceEnrollFile() {
	if h.enrollDir == "" {
		return
	}
	_ = os.Remove(filepath.Join(h.enrollDir, instanceHostEnrollFile))
}
