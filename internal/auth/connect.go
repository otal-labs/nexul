package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

const (
	// connectAlphabet is Crockford base32: no I, L, O or U, so a code read off a screen has no look-alikes.
	connectAlphabet     = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	connectCodeLen      = 12
	connectCodeTTL      = 2 * time.Minute
	exchangeMaxFailures = 5
	exchangeWindow      = 10 * time.Minute
	phonePlatform       = "Android"
)

var (
	errInvalidConnectCode = apperrs.WithCode("invalid_code", fmt.Errorf("%w: the connect code is invalid or expired", apperrs.ErrInvalid))
	errExchangeThrottled  = fmt.Errorf("%w: too many wrong connect codes, try again in a few minutes", apperrs.ErrRateLimited)
	errConnectUnavailable = fmt.Errorf("%w: connect codes are not configured", apperrs.ErrUnauthorized)
)

// IssueConnectCode mints the user's one live code, replacing the last; the raw code is returned once and only its hash kept.
func (s *Service) IssueConnectCode(ctx context.Context, userID, host string) (ConnectCode, error) {
	if s.cfg.ConnectCodes == nil {
		return ConnectCode{}, errConnectUnavailable
	}
	raw, err := newConnectCode()
	if err != nil {
		return ConnectCode{}, err
	}
	now := s.cfg.Now().UTC()
	expires := now.Add(connectCodeTTL)
	if err := s.cfg.ConnectCodes.ReplaceConnectCode(ctx, userID, hashToken(raw), now, expires); err != nil {
		return ConnectCode{}, fmt.Errorf("store connect code: %w", err)
	}
	return ConnectCode{Code: formatConnectCode(raw), Host: host, ExpiresAt: expires}, nil
}

// ExchangeConnectCode trades a code for a phone session, throttled per addr; every wrong answer is the same error.
func (s *Service) ExchangeConnectCode(ctx context.Context, addr, code string, dev ConnectDevice) (string, error) {
	if s.cfg.ConnectCodes == nil {
		return "", errConnectUnavailable
	}
	now := s.cfg.Now()
	if s.exchanges.blocked(addr, now) {
		return "", errExchangeThrottled
	}
	userID, err := s.cfg.ConnectCodes.ConsumeConnectCode(ctx, hashToken(normalizeConnectCode(code)), now)
	if errors.Is(err, apperrs.ErrNotFound) {
		s.exchanges.fail(addr, now)
		return "", errInvalidConnectCode
	}
	if err != nil {
		return "", fmt.Errorf("consume connect code: %w", err)
	}
	label := strings.TrimSpace(dev.Model)
	if label == "" {
		label = "Phone"
	}
	logging.FromCtx(ctx).Info("phone connected", "user_id", userID, "model", label, "os", dev.OS, "app_version", dev.AppVersion)
	return s.CreateSession(WithDevice(ctx, Device{Client: ClientPhone, Platform: phonePlatform, Label: label, IP: addr}), userID)
}

func newConnectCode() (string, error) {
	b := make([]byte, connectCodeLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate connect code: %w", err)
	}
	for i := range b {
		b[i] = connectAlphabet[int(b[i])%len(connectAlphabet)]
	}
	return string(b), nil
}

// formatConnectCode groups a raw code as XXXX-XXXX-XXXX for reading aloud.
func formatConnectCode(raw string) string {
	return raw[:4] + "-" + raw[4:8] + "-" + raw[8:]
}

// normalizeConnectCode accepts what a person types: any case, with or without dashes, and Crockford's look-alikes.
func normalizeConnectCode(code string) string {
	var out strings.Builder
	for _, r := range strings.ToUpper(code) {
		switch r {
		case '-', ' ':
			continue
		case 'O':
			r = '0'
		case 'I', 'L':
			r = '1'
		}
		out.WriteRune(r)
	}
	return out.String()
}
