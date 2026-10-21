package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// DeleteAccessApp implements dns.AccessAppProvider.
func (c *Client) DeleteAccessApp(ctx context.Context, appID string) error {
	var out apiResponse
	return ignoreNotFound(c.accessDo(ctx, http.MethodDelete, "access/apps/"+url.PathEscape(appID), nil, &out))
}

// DeleteServiceToken implements dns.ServiceTokenProvider.
func (c *Client) DeleteServiceToken(ctx context.Context, tokenID string) error {
	var out apiResponse
	return ignoreNotFound(c.accessDo(ctx, http.MethodDelete, "access/service_tokens/"+url.PathEscape(tokenID), nil, &out))
}

// accessDo runs an account-scoped Access call, turning a missing Zero Trust organization into its own error.
func (c *Client) accessDo(ctx context.Context, method, path string, body any, out *apiResponse) error {
	acct, err := c.accountID(ctx)
	if err != nil {
		return err
	}
	err = c.do(ctx, method, "accounts/"+url.PathEscape(acct)+"/"+path, body, out)
	if err == nil || !zeroTrustMissing(err.Error()) {
		return err
	}
	return fmt.Errorf("%w: %w: enable Zero Trust once in the Cloudflare dashboard (pick a team name and the Free plan), then try again", apperrs.ErrInvalid, dns.ErrZeroTrustDisabled)
}

// zeroTrustMissing matches Cloudflare's "no Access organization" errors by phrase; the exact wording is unverified live.
func zeroTrustMissing(msg string) bool {
	msg = strings.ToLower(msg)
	for _, phrase := range []string{"access organization", "organization not found", "organization_not_found", "not enabled"} {
		if strings.Contains(msg, phrase) {
			return true
		}
	}
	return false
}

func ignoreNotFound(err error) error {
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	return err
}
