package runner

import (
	"context"
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
	enrollmentTTL         = time.Hour
	instanceEnrollmentTTL = 24 * time.Hour
	credentialPrefix      = "nxr_"
	// instanceEnrollFile is read by `nexul install` to enroll the bundled runner.
	instanceEnrollFile = "runner-instance"
	// RemovedRefusal is the body a removed runner is refused with, so it uninstalls itself instead of retrying.
	RemovedRefusal = `{"error":"runner_removed"}`
)

var (
	errInvalidCode   = apperrs.WithCode("invalid_code", fmt.Errorf("%w: the enrollment code is unknown, used or expired", apperrs.ErrUnauthorized))
	errRunnerRemoved = apperrs.WithCode("runner_removed", fmt.Errorf("%w: this runner was removed", apperrs.ErrUnauthorized))
)

// WithEnrollDir sets the directory the bundled runner's enrollment code file lives in (<data dir>/enroll).
func (s *Service) WithEnrollDir(dir string) *Service {
	s.enrollDir = dir
	return s
}

// CreateEnrollment mints a one-hour code for a runner named name, optionally bound to machine, and renders the
// install one-liners that carry it. Only an instance admin may enroll a runner.
func (s *Service) CreateEnrollment(ctx context.Context, name, machine string) (Enrollment, error) {
	if err := identity.RequireInstanceAdmin(ctx, s.admin); err != nil {
		return Enrollment{}, err
	}
	instanceURL, err := s.instanceURL(ctx)
	if err != nil {
		return Enrollment{}, fmt.Errorf("get instance url: %w", err)
	}
	if instanceURL == "" {
		return Enrollment{}, fmt.Errorf("%w: set the instance URL in settings before adding a runner", apperrs.ErrConflict)
	}
	code, expiresAt, err := s.mintEnrollment(ctx, name, machine, enrollmentTTL)
	if err != nil {
		return Enrollment{}, err
	}
	return Enrollment{Code: code, ExpiresAt: expiresAt, Commands: installCommands(instanceURL, strings.TrimSpace(name), code)}, nil
}

func (s *Service) mintEnrollment(ctx context.Context, name, machine string, ttl time.Duration) (string, time.Time, error) {
	name = strings.TrimSpace(name)
	if !hostcred.NamePattern.MatchString(name) {
		return "", time.Time{}, fmt.Errorf("%w: a runner name is 1 to 32 lowercase letters, digits or dashes, starting with a letter or digit", apperrs.ErrInvalid)
	}
	_, err := s.repo.GetByName(ctx, name)
	if err == nil {
		return "", time.Time{}, fmt.Errorf("%w: a runner named %s is already enrolled; remove it first", apperrs.ErrConflict, name)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return "", time.Time{}, err
	}
	raw, hash, err := hostcred.MintCode()
	if err != nil {
		return "", time.Time{}, err
	}
	now := s.now().UTC()
	code := &EnrollmentCode{CodeHash: hash, Name: name, Machine: strings.TrimSpace(machine), CreatedAt: now, ExpiresAt: now.Add(ttl)}
	if err := s.repo.CreateEnrollment(ctx, code); err != nil {
		return "", time.Time{}, fmt.Errorf("create enrollment for %s: %w", name, err)
	}
	return raw, code.ExpiresAt, nil
}

// installCommands renders the runner installer one-liners.
func installCommands(instanceURL, name, code string) InstallCommands {
	unix, windows := hostcred.InstallCommands("runner", instanceURL, name, code)
	return InstallCommands{Unix: unix, Windows: windows}
}

// Enroll trades a code for the runner's own credential: the runner record is created on its machine and the
// credential is returned once. It is the public half of enrollment, so it takes no actor.
func (s *Service) Enroll(ctx context.Context, req EnrollRequest) (Enrolled, error) {
	now := s.now().UTC()
	codeHash := hostcred.Hash(strings.TrimSpace(req.Code))
	code, err := s.repo.GetEnrollment(ctx, codeHash, now)
	if errors.Is(err, apperrs.ErrNotFound) {
		return Enrolled{}, errInvalidCode
	}
	if err != nil {
		return Enrolled{}, err
	}
	if strings.TrimSpace(req.Name) != code.Name {
		return Enrolled{}, apperrs.WithCode("name_mismatch", fmt.Errorf("%w: this code enrolls a runner named %s", apperrs.ErrConflict, code.Name))
	}
	machine := enrolledMachine(code.Machine, req.Machine, code.Name)
	machineID, err := s.machineFor(ctx, machine, strings.TrimSpace(req.StackRoot), now)
	if err != nil {
		return Enrolled{}, err
	}
	credential, credentialHash, err := hostcred.MintCredential(credentialPrefix)
	if err != nil {
		return Enrolled{}, err
	}
	r := &Runner{ID: ids.New(), Name: code.Name, Version: req.Version, LastSeen: now, CreatedAt: now, MachineID: machineID}
	err = s.repo.Enroll(ctx, codeHash, r, credentialHash, now)
	if errors.Is(err, apperrs.ErrNotFound) {
		return Enrolled{}, errInvalidCode
	}
	if err != nil {
		return Enrolled{}, fmt.Errorf("enroll runner %s: %w", code.Name, err)
	}
	if code.Name == instanceRunnerName {
		s.removeInstanceEnrollFile()
	}
	return Enrolled{ID: r.ID, Name: r.Name, Machine: machine, Credential: credential}, nil
}

// machineFor finds or creates the machine named name; a new machine takes stackRoot, else the default.
func (s *Service) machineFor(ctx context.Context, name, stackRoot string, now time.Time) (string, error) {
	if s.machines == nil {
		return "", nil
	}
	m, err := s.machines.GetByName(ctx, name)
	if err == nil {
		return m.ID, nil
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return "", fmt.Errorf("get machine %s: %w", name, err)
	}
	if stackRoot == "" {
		stackRoot = defaultStackRoot
	}
	m = &Machine{ID: ids.New(), Name: name, StackRoot: stackRoot, ReportedHostname: name, FirstSeen: now, LastSeen: now}
	if err := s.machines.Create(ctx, m); err != nil {
		return "", fmt.Errorf("create machine %s: %w", name, err)
	}
	return m.ID, nil
}

// RemoveRunner revokes a runner's credential, deletes its record and, when it is connected, tells it to
// uninstall itself. Only an instance admin may remove a runner.
func (s *Service) RemoveRunner(ctx context.Context, id string) error {
	if err := identity.RequireInstanceAdmin(ctx, s.admin); err != nil {
		return err
	}
	return s.remove(ctx, id)
}

// RemoveSelf is removal asked for by the runner itself, authenticated by its own credential; a runner that was
// already removed has nothing left to remove.
func (s *Service) RemoveSelf(ctx context.Context, credential string) error {
	cred, err := authenticate(ctx, s.repo, credential)
	if err != nil {
		return err
	}
	if cred.Revoked {
		return nil
	}
	return s.remove(ctx, cred.RunnerID)
}

func (s *Service) remove(ctx context.Context, id string) error {
	if err := s.repo.Remove(ctx, id, s.now().UTC()); err != nil {
		return fmt.Errorf("remove runner %s: %w", id, err)
	}
	s.live.Uninstall(ctx, id)
	s.publish(ctx, TopicRunnerDisconnected, RunnerDisconnectedEvent{RunnerID: id, Reason: "removed"})
	return nil
}

// authenticate resolves a raw runner credential; an empty or unknown one is apperrs.ErrUnauthorized, while a
// revoked one is returned so the caller can say the runner was removed.
func authenticate(ctx context.Context, creds CredentialStore, raw string) (*Credential, error) {
	if raw == "" {
		return nil, apperrs.ErrUnauthorized
	}
	cred, err := creds.GetCredential(ctx, hostcred.Hash(raw))
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, apperrs.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// WriteInstanceEnrollment keeps <enroll dir>/runner-instance in step with the bundled runner: a fresh 24-hour
// code while no runner named instance is enrolled, no file once it is. Called at boot.
func (s *Service) WriteInstanceEnrollment(ctx context.Context) error {
	_, err := s.repo.GetByName(ctx, instanceRunnerName)
	if err == nil {
		s.removeInstanceEnrollFile()
		return nil
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return err
	}
	code, _, err := s.mintEnrollment(ctx, instanceRunnerName, "", instanceEnrollmentTTL)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.enrollDir, 0o700); err != nil {
		return fmt.Errorf("create enroll dir: %w", err)
	}
	if err := hostcred.WriteCodeFile(filepath.Join(s.enrollDir, instanceEnrollFile), code); err != nil {
		return fmt.Errorf("write instance runner enrollment code: %w", err)
	}
	return nil
}

// removeInstanceEnrollFile is best-effort: a leftover file only holds a code that no longer enrolls anything.
func (s *Service) removeInstanceEnrollFile() {
	if s.enrollDir == "" {
		return
	}
	_ = os.Remove(filepath.Join(s.enrollDir, instanceEnrollFile))
}

// enrolledMachine picks the code's machine, then the hostname the installer reported, then the runner's own name, so
// runners installed on one host share its machine unless the code says otherwise.
func enrolledMachine(onCode, reported, name string) string {
	if m := strings.TrimSpace(onCode); m != "" {
		return m
	}
	if m := strings.TrimSpace(reported); m != "" {
		return m
	}
	return name
}
