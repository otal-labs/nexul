package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// maxRateWaits is how many 429s in a row one request waits out before the import pauses and retries later.
const maxRateWaits = 5

// errRefused is a request the source refused as malformed, such as a paging token it no longer honours.
var errRefused = errors.New("refused the request")

// restAPI is how a source reaches its tool's REST API: its base URL, the HTTP client, and how long to sleep, so tests
// run against a stub server without waiting.
type restAPI struct {
	base   string
	client *http.Client
	sleep  func(ctx context.Context, d time.Duration) error
}

func newRestAPI(base string, client *http.Client) restAPI {
	if client == nil {
		client = &http.Client{Timeout: time.Minute}
	}
	return restAPI{base: base, client: client, sleep: sleepCtx}
}

// restClient reads one import's source as JSON, signed with its credential, one request at a time.
type restClient struct {
	api    restAPI
	title  string
	header http.Header
	// every is the least time between two requests.
	every time.Duration
	// pause is how long a response asks the next request to wait, from its rate limit headers.
	pause func(h http.Header) time.Duration
	next  time.Time
}

// get reads path with query into out, waiting out the source's rate limits; a refused credential is ErrUnauthorized,
// a missing record ErrNotFound, and a limit that keeps refusing ErrRetryable, so the import resumes later.
func (c *restClient) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.api.base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	for waits := 0; ; waits++ {
		if err := c.api.sleep(ctx, max(time.Until(c.next), 0)); err != nil {
			return err
		}
		resp, err := c.do(ctx, u)
		if err != nil {
			return fmt.Errorf("%w: read %s: %w", apperrs.ErrRetryable, c.title, err)
		}
		c.next = time.Now().Add(c.every)
		if c.pause != nil {
			c.next = c.next.Add(c.pause(resp.Header))
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			_ = resp.Body.Close() // nothing to read from a refusal
			if waits >= maxRateWaits {
				return fmt.Errorf("%w: %s kept refusing for its rate limit", apperrs.ErrRetryable, c.title)
			}
			if err := c.api.sleep(ctx, retryAfterOf(resp.Header.Get("Retry-After"))); err != nil {
				return err
			}
			continue
		}
		return c.decode(resp, path, out)
	}
}

func (c *restClient) do(ctx context.Context, u string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range c.header {
		req.Header[k] = v
	}
	req.Header.Set("Accept", "application/json")
	return c.api.client.Do(req)
}

func (c *restClient) decode(resp *http.Response, path string, out any) error {
	defer func() { _ = resp.Body.Close() }() // a read-only body: nothing to lose on close
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w: %s refused the credential; check it can read what the import names", apperrs.ErrUnauthorized, c.title)
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: %s has nothing at %s, or the credential cannot see it", apperrs.ErrNotFound, c.title, path)
	case resp.StatusCode == http.StatusBadRequest:
		return fmt.Errorf("%s %w for %s", c.title, errRefused, path)
	case resp.StatusCode >= 500:
		return fmt.Errorf("%w: %s answered %s", apperrs.ErrRetryable, c.title, resp.Status)
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("%s answered %s for %s", c.title, resp.Status, path)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<20)).Decode(out); err != nil {
		return fmt.Errorf("read %s's answer for %s: %w", c.title, path, err)
	}
	return nil
}
