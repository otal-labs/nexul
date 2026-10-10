package runner

import (
	"cmp"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
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
	errComputerCode  = apperrs.WithCode("computer_code", fmt.Errorf("%w: this code adds a computer; run it with nexul install computer, as Add a computer shows it", apperrs.ErrConflict))
	errRunnerCode    = apperrs.WithCode("runner_code", fmt.Errorf("%w: this code enrolls a runner, not a computer; make a new code with Add a computer", apperrs.ErrConflict))
)

// ComputerClaims are what a computer's enrollment token carries: the instance to enroll with, the one-time code, the
// computer it adds and the code's expiry. The installer reads the server without the key; only this instance can
// check the rest.
type ComputerClaims struct {
	Server   string `json:"server"`
	Code     string `json:"code"`
	Computer string `json:"computer"`
	Exp      int64  `json:"exp"`
}

// WithEnrollDir sets the directory the bundled runner's enrollment code file lives in (<data dir>/enroll).
func (s *Service) WithEnrollDir(dir string) *Service {
	s.enrollDir = dir
	return s
}

// CreateEnrollment mints a one-hour code for a runner named name, optionally bound to machine, and renders the
// install one-liners that carry it. It needs runners:write.
func (s *Service) CreateEnrollment(ctx context.Context, name, machine string) (Enrollment, error) {
	if err := s.require(ctx, permissions.RunnersWrite); err != nil {
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

// CreatePersonalEnrollment mints a one-hour code that enrolls a personal runner for userID's computerID and renders
// the command that installs it. It takes no permission: the pairing domain mints it only for the caller's own
// computer, and a computer already reached by a runner gets no second one.
func (s *Service) CreatePersonalEnrollment(ctx context.Context, userID, computerID string) (Enrollment, error) {
	if userID == "" || computerID == "" {
		return Enrollment{}, fmt.Errorf("%w: a personal runner needs its person and computer", apperrs.ErrInvalid)
	}
	_, err := s.repo.GetByComputer(ctx, computerID)
	if err == nil {
		return Enrollment{}, fmt.Errorf("%w: this computer already has its runner; remove the computer to add it again", apperrs.ErrConflict)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return Enrollment{}, err
	}
	instanceURL, err := s.instanceURL(ctx)
	if err != nil {
		return Enrollment{}, fmt.Errorf("get instance url: %w", err)
	}
	if instanceURL == "" {
		return Enrollment{}, fmt.Errorf("%w: set the instance URL in settings before adding a computer", apperrs.ErrConflict)
	}
	if len(s.install.TokenKey) == 0 {
		return Enrollment{}, apperrs.Fatal(fmt.Errorf("%w: the computer token key is not wired", apperrs.ErrFatal))
	}
	code := &EnrollmentCode{Name: "computer-" + strings.ToLower(rand.Text()[:8]), OwnerUserID: userID, ComputerID: computerID}
	raw, err := s.storeCode(ctx, code, enrollmentTTL)
	if err != nil {
		return Enrollment{}, err
	}
	claims := ComputerClaims{Server: strings.TrimRight(instanceURL, "/"), Code: raw, Computer: computerID, Exp: code.ExpiresAt.Unix()}
	token, err := hostcred.SignToken(claims, s.install.TokenKey)
	if err != nil {
		return Enrollment{}, err
	}
	site := cmp.Or(strings.TrimRight(s.install.SiteURL, "/"), hostcred.DefaultSite)
	unix := hostcred.ComputerCommand(site, strings.TrimRight(s.install.ReleaseURL, "/"), token)
	return Enrollment{Token: token, ExpiresAt: code.ExpiresAt, Commands: InstallCommands{Unix: unix}}, nil
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
	code := &EnrollmentCode{Name: name, Machine: strings.TrimSpace(machine)}
	raw, err := s.storeCode(ctx, code, ttl)
	if err != nil {
		return "", time.Time{}, err
	}
	return raw, code.ExpiresAt, nil
}

// storeCode mints a raw code for code and stores its hash, valid for ttl.
func (s *Service) storeCode(ctx context.Context, code *EnrollmentCode, ttl time.Duration) (string, error) {
	raw, hash, err := hostcred.MintCode()
	if err != nil {
		return "", err
	}
	now := s.now().UTC()
	code.CodeHash, code.CreatedAt, code.ExpiresAt = hash, now, now.Add(ttl)
	if err := s.repo.CreateEnrollment(ctx, code); err != nil {
		return "", fmt.Errorf("create enrollment for %s: %w", code.Name, err)
	}
	return raw, nil
}

// ComputerRunner returns the personal runner that reaches computerID, or apperrs.ErrNotFound. The server's own seam
// for the pairing domain, which checks the computer is the caller's own, so it takes no permission.
func (s *Service) ComputerRunner(ctx context.Context, computerID string) (*Runner, error) {
	return s.repo.GetByComputer(ctx, computerID)
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
	computerID, err := s.tokenCode(&req, now)
	if err != nil {
		return Enrolled{}, err
	}
	codeHash := hostcred.Hash(strings.TrimSpace(req.Code))
	code, err := s.repo.GetEnrollment(ctx, codeHash, now)
	if errors.Is(err, apperrs.ErrNotFound) {
		return Enrolled{}, errInvalidCode
	}
	if err != nil {
		return Enrolled{}, err
	}
	if code.Personal() {
		return s.enrollPersonal(ctx, codeHash, code, req, computerID, now)
	}
	if req.Token != "" {
		return Enrolled{}, errRunnerCode
	}
	if strings.TrimSpace(req.Name) != code.Name {
		return Enrolled{}, apperrs.WithCode("name_mismatch", fmt.Errorf("%w: this code enrolls a runner named %s", apperrs.ErrConflict, code.Name))
	}
	machine := enrolledMachine(code.Machine, req.Machine, code.Name)
	machineID, err := s.machineFor(ctx, machine, strings.TrimSpace(req.StackRoot), now)
	if err != nil {
		return Enrolled{}, err
	}
	r := &Runner{ID: ids.New(), Name: code.Name, Version: req.Version, LastSeen: now, CreatedAt: now, MachineID: machineID}
	credential, err := s.enroll(ctx, codeHash, r, now)
	if err != nil {
		return Enrolled{}, err
	}
	if code.Name == instanceRunnerName {
		s.removeInstanceEnrollFile()
	}
	return Enrolled{ID: r.ID, Name: r.Name, Machine: machine, Credential: credential}, nil
}

// tokenCode checks a computer's token, signature and expiry, and moves its code into req; it returns the computer
// the token names. A request without a token is a runner's and passes through.
func (s *Service) tokenCode(req *EnrollRequest, now time.Time) (string, error) {
	if req.Token == "" {
		return "", nil
	}
	var claims ComputerClaims
	if err := hostcred.ParseToken(strings.TrimSpace(req.Token), s.install.TokenKey, &claims); err != nil || len(s.install.TokenKey) == 0 {
		return "", errInvalidCode
	}
	if claims.Exp <= now.Unix() {
		return "", errInvalidCode
	}
	req.Code = claims.Code
	return claims.Computer, nil
}

// enrollPersonal makes the person's computer runner the code is bound to, on no machine; the code names it, so the
// installer sends no name.
func (s *Service) enrollPersonal(ctx context.Context, codeHash string, code *EnrollmentCode, req EnrollRequest, computerID string, now time.Time) (Enrolled, error) {
	if req.Token == "" {
		return Enrolled{}, errComputerCode
	}
	if computerID != code.ComputerID {
		return Enrolled{}, errInvalidCode
	}
	r := &Runner{ID: ids.New(), Name: code.Name, Version: req.Version, LastSeen: now, CreatedAt: now, OwnerUserID: code.OwnerUserID, ComputerID: code.ComputerID}
	credential, err := s.enroll(ctx, codeHash, r, now)
	if errors.Is(err, apperrs.ErrConflict) {
		return Enrolled{}, fmt.Errorf("%w: this computer already has its runner; remove the computer to add it again", apperrs.ErrConflict)
	}
	if err != nil {
		return Enrolled{}, err
	}
	s.publish(ctx, TopicPersonalChanged, personalChanged(r, PersonalEnrolled, strings.TrimSpace(req.Machine)))
	return Enrolled{ID: r.ID, Name: r.Name, Credential: credential}, nil
}

// enroll consumes the code, stores r and returns its new credential, once.
func (s *Service) enroll(ctx context.Context, codeHash string, r *Runner, now time.Time) (string, error) {
	credential, credentialHash, err := hostcred.MintCredential(credentialPrefix)
	if err != nil {
		return "", err
	}
	err = s.repo.Enroll(ctx, codeHash, r, credentialHash, now)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", errInvalidCode
	}
	if err != nil {
		return "", fmt.Errorf("enroll runner %s: %w", r.Name, err)
	}
	return credential, nil
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
// uninstall itself. It needs runners:delete.
func (s *Service) RemoveRunner(ctx context.Context, id string) error {
	if err := s.require(ctx, permissions.RunnersDelete); err != nil {
		return err
	}
	r, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get runner %s: %w", id, err)
	}
	if r.Personal() {
		return fmt.Errorf("get runner %s: %w", id, apperrs.ErrNotFound)
	}
	return s.remove(ctx, r)
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
	r, err := s.repo.GetByID(ctx, cred.RunnerID)
	if err != nil {
		return fmt.Errorf("get runner %s: %w", cred.RunnerID, err)
	}
	return s.remove(ctx, r)
}

func (s *Service) remove(ctx context.Context, r *Runner) error {
	if err := s.repo.Remove(ctx, r.ID, s.now().UTC()); err != nil {
		return fmt.Errorf("remove runner %s: %w", r.ID, err)
	}
	s.live.Uninstall(ctx, r.ID)
	if r.Personal() {
		s.publish(ctx, TopicPersonalChanged, personalChanged(r, PersonalRemoved, ""))
		return nil
	}
	s.publish(ctx, TopicRunnerDisconnected, RunnerDisconnectedEvent{RunnerID: r.ID, Reason: "removed"})
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
