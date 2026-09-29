package auth

import (
	"bufio"
	"context"
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// SetupUserID is the caller identity a setup pass carries; no user row ever has it.
const SetupUserID = "setup"

const (
	setupCodePrefix = "nxs_"
	setupCodeFile   = "setup"
	setupCodeTTL    = 24 * time.Hour
	setupPassPrefix = "nxsp_"
	setupPassTTL    = time.Hour
	// setupPassMACLabel namespaces the pass's MAC so nothing else signed with the secret ever verifies as one.
	setupPassMACLabel = "setup-pass."
	unlockMaxFailures = 10
	unlockWindow      = 10 * time.Minute
)

var (
	errInvalidSetupCode  = apperrs.WithCode("invalid_code", fmt.Errorf("%w: the setup code is wrong or expired", apperrs.ErrInvalid))
	errSetupDone         = apperrs.WithCode("setup_done", fmt.Errorf("%w: setup is finished, sign in instead", apperrs.ErrConflict))
	errSetupPassRequired = fmt.Errorf("%w: a setup pass is required", apperrs.ErrUnauthorized)
	errUnlockThrottled   = fmt.Errorf("%w: too many wrong setup codes, try again in a few minutes", apperrs.ErrRateLimited)
)

// setupPassRoutes is the only surface a setup pass reaches; anything else with a pass is a 401.
var setupPassRoutes = func() *http.ServeMux {
	mux := http.NewServeMux()
	for _, pattern := range []string{
		"/api/setup/",
		"/api/auth/bootstrap",
		"/api/auth/bootstrap/verify",
		"/api/connectors/cloudflare/manual",
		"/api/connectors/cloudflare/manual/verify",
		"GET /api/connectors",
		"/api/dns/",
		"GET /api/machines",
		"GET /api/services/{id}/deploys",
		"GET /api/deploys/{id}/log",
	} {
		mux.Handle(pattern, http.NotFoundHandler())
	}
	return mux
}()

func setupPassAllows(r *http.Request) bool {
	_, pattern := setupPassRoutes.Handler(r)
	return pattern != ""
}

// SetupOpen reports whether first run is still open: no user exists yet.
func (s *Service) SetupOpen(ctx context.Context) (bool, error) {
	n, err := s.cfg.Users.CountUsers(ctx)
	if err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}
	return n == 0, nil
}

// WriteSetupCode writes a fresh code to <enroll dir>/setup on every boot while no user exists, and clears it after.
func (s *Service) WriteSetupCode(ctx context.Context) error {
	open, err := s.SetupOpen(ctx)
	if err != nil {
		return err
	}
	if !open {
		return s.closeSetup(ctx)
	}
	raw, hash, err := hostcred.MintCredential(setupCodePrefix)
	if err != nil {
		return err
	}
	now := s.cfg.Now().UTC()
	if err := s.cfg.SetupCodes.ReplaceSetupCode(ctx, hash, now, now.Add(setupCodeTTL)); err != nil {
		return fmt.Errorf("store setup code: %w", err)
	}
	if err := os.MkdirAll(s.cfg.EnrollDir, 0o700); err != nil {
		return fmt.Errorf("create enroll dir: %w", err)
	}
	if err := hostcred.WriteCodeFile(filepath.Join(s.cfg.EnrollDir, setupCodeFile), raw); err != nil {
		return fmt.Errorf("write setup code: %w", err)
	}
	return nil
}

// closeSetup forgets every setup code and removes the installer's copy.
func (s *Service) closeSetup(ctx context.Context) error {
	if s.cfg.SetupCodes == nil {
		return nil
	}
	if err := s.cfg.SetupCodes.ClearSetupCodes(ctx); err != nil {
		return fmt.Errorf("clear setup codes: %w", err)
	}
	if s.cfg.EnrollDir == "" {
		return nil
	}
	if err := os.Remove(filepath.Join(s.cfg.EnrollDir, setupCodeFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove setup code file: %w", err)
	}
	return nil
}

// UnlockSetup trades the setup code for a pass, throttled per addr; the code survives for the domain handoff.
func (s *Service) UnlockSetup(ctx context.Context, addr, code string) (SetupPass, error) {
	open, err := s.SetupOpen(ctx)
	if err != nil {
		return SetupPass{}, err
	}
	if !open {
		return SetupPass{}, errSetupDone
	}
	now := s.cfg.Now()
	if s.unlocks.blocked(addr, now) {
		return SetupPass{}, errUnlockThrottled
	}
	valid, err := s.setupCodeValid(ctx, strings.TrimSpace(code), now)
	if err != nil {
		return SetupPass{}, err
	}
	if !valid {
		s.unlocks.fail(addr, now)
		return SetupPass{}, errInvalidSetupCode
	}
	return s.signSetupPass(now.Add(setupPassTTL))
}

func (s *Service) setupCodeValid(ctx context.Context, code string, now time.Time) (bool, error) {
	if code == "" || s.cfg.SetupCodes == nil {
		return false, nil
	}
	return s.cfg.SetupCodes.SetupCodeValid(ctx, hostcred.Hash(code), now)
}

type setupPassClaims struct {
	Exp int64 `json:"exp"`
}

func (s *Service) signSetupPass(exp time.Time) (SetupPass, error) {
	payload, err := json.Marshal(setupPassClaims{Exp: exp.Unix()})
	if err != nil {
		return SetupPass{}, fmt.Errorf("sign setup pass: %w", err)
	}
	enc := base64.RawURLEncoding.EncodeToString(payload)
	return SetupPass{Token: setupPassPrefix + enc + "." + s.mac(setupPassMACLabel+enc), ExpiresAt: exp.UTC()}, nil
}

func isSetupPass(token string) bool {
	return strings.HasPrefix(token, setupPassPrefix)
}

func (s *Service) verifySetupPass(token string) error {
	enc, sig, ok := strings.Cut(strings.TrimPrefix(token, setupPassPrefix), ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.mac(setupPassMACLabel+enc))) {
		return apperrs.ErrUnauthorized
	}
	payload, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return apperrs.ErrUnauthorized
	}
	var claims setupPassClaims
	if err := json.Unmarshal(payload, &claims); err != nil || s.cfg.Now().Unix() >= claims.Exp {
		return apperrs.ErrUnauthorized
	}
	return nil
}

// authenticateSetupPass admits a pass only on the allowlisted routes and only while no user exists.
func (s *Service) authenticateSetupPass(r *http.Request, token string) (*User, error) {
	if err := s.verifySetupPass(token); err != nil {
		return nil, err
	}
	if !setupPassAllows(r) {
		return nil, apperrs.ErrUnauthorized
	}
	open, err := s.SetupOpen(r.Context())
	if err != nil {
		return nil, err
	}
	if !open {
		return nil, apperrs.ErrUnauthorized
	}
	return &User{ID: SetupUserID, Login: SetupUserID, AccountStatus: AccountActive}, nil
}

func requireSetupPass(callerID string) error {
	if callerID != SetupUserID {
		return errSetupPassRequired
	}
	return nil
}

// SetSetupInstanceURL stores the instance URL first run found answering as this server; setup pass only.
func (s *Service) SetSetupInstanceURL(ctx context.Context, callerID, raw string) (string, error) {
	if err := requireSetupPass(callerID); err != nil {
		return "", err
	}
	open, err := s.SetupOpen(ctx)
	if err != nil {
		return "", err
	}
	if !open {
		return "", errSetupDone
	}
	instanceURL := strings.TrimSuffix(strings.TrimSpace(raw), "/")
	if strings.HasPrefix(strings.ToLower(instanceURL), "http://") && !s.cfg.Local {
		return "", fmt.Errorf("%w: the instance URL must be https", apperrs.ErrInvalid)
	}
	if err := s.VerifyInstanceURL(ctx, instanceURL); err != nil {
		return "", err
	}
	st, err := s.cfg.Settings.Set(ctx, instanceURL)
	if err != nil {
		return "", fmt.Errorf("set instance url: %w", err)
	}
	return st.InstanceURL, nil
}

// PublicAddress reports the server's public addresses to a setup pass or a holder of instance:read.
func (s *Service) PublicAddress(ctx context.Context, callerID string) (PublicAddress, error) {
	if callerID == "" {
		return PublicAddress{}, apperrs.ErrUnauthorized
	}
	if callerID != SetupUserID {
		if err := s.requireAnywhere(ctx, callerID, permissions.InstanceRead); err != nil {
			return PublicAddress{}, err
		}
	}
	return s.cfg.PublicAddress.PublicAddress(ctx), nil
}

// PublicAddressLookup asks the outside world which address this server reaches it from.
type PublicAddressLookup interface {
	PublicAddress(ctx context.Context) PublicAddress
}

const (
	traceURLv4     = "https://1.1.1.1/cdn-cgi/trace"
	traceURLv6     = "https://[2606:4700:4700::1111]/cdn-cgi/trace"
	traceTimeout   = 5 * time.Second
	traceBodyLimit = 4096
)

// cloudflareTrace reads the ip= line Cloudflare's trace endpoint echoes back, once per IP family.
type cloudflareTrace struct {
	client       *http.Client
	v4URL, v6URL string
}

func newCloudflareTrace() cloudflareTrace {
	return cloudflareTrace{client: &http.Client{Timeout: traceTimeout}, v4URL: traceURLv4, v6URL: traceURLv6}
}

func (c cloudflareTrace) PublicAddress(ctx context.Context) PublicAddress {
	var out PublicAddress
	var wg sync.WaitGroup
	wg.Go(func() { out.IPv4 = c.fetch(ctx, c.v4URL, netip.Addr.Is4) })
	wg.Go(func() { out.IPv6 = c.fetch(ctx, c.v6URL, netip.Addr.Is6) })
	wg.Wait()
	return out
}

// fetch returns "" for any failure: an unreachable family is an answer, not an error.
func (c cloudflareTrace) fetch(ctx context.Context, url string, family func(netip.Addr) bool) string {
	ctx, cancel := context.WithTimeout(ctx, traceTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }() // read-only body; a close error changes nothing
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, traceBodyLimit))
	for scanner.Scan() {
		raw, ok := strings.CutPrefix(scanner.Text(), "ip=")
		if !ok {
			continue
		}
		addr, err := netip.ParseAddr(strings.TrimSpace(raw))
		if err != nil || !family(addr) {
			return ""
		}
		return addr.String()
	}
	return ""
}

// ponytail: failures per address, in memory; a restart forgets them, and rotates the setup code anyway.
type failureLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
	max      int
	window   time.Duration
}

func newFailureLimiter(maxFailures int, window time.Duration) *failureLimiter {
	return &failureLimiter{failures: map[string][]time.Time{}, max: maxFailures, window: window}
}

func (l *failureLimiter) blocked(addr string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recent(l.failures[addr], now)) >= l.max
}

func (l *failureLimiter) fail(addr string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for key, times := range l.failures {
		if kept := l.recent(times, now); len(kept) > 0 {
			l.failures[key] = kept
			continue
		}
		delete(l.failures, key)
	}
	l.failures[addr] = append(l.failures[addr], now)
}

func (l *failureLimiter) recent(times []time.Time, now time.Time) []time.Time {
	var kept []time.Time
	for _, t := range times {
		if now.Sub(t) < l.window {
			kept = append(kept, t)
		}
	}
	return kept
}
