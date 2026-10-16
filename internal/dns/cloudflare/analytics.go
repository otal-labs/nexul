package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

var _ dns.AnalyticsProvider = (*Client)(nil)

// requestsQuery counts a zone's requests per minute, host and response status; limit is the API's page ceiling.
const requestsQuery = `query ($zone: string!, $from: Time!, $to: Time!, $hosts: [string!]) {
  viewer { zones(filter: {zoneTag: $zone}) {
    httpRequestsAdaptiveGroups(limit: 10000, filter: {datetime_geq: $from, datetime_lt: $to, clientRequestHTTPHost_in: $hosts}) {
      count
      dimensions { datetimeMinute clientRequestHTTPHost edgeResponseStatus }
    }
  } }
}`

type graphqlResponse struct {
	Data struct {
		Viewer struct {
			Zones []struct {
				Groups []struct {
					Count      int64 `json:"count"`
					Dimensions struct {
						Minute string `json:"datetimeMinute"`
						Host   string `json:"clientRequestHTTPHost"`
						Status int    `json:"edgeResponseStatus"`
					} `json:"dimensions"`
				} `json:"httpRequestsAdaptiveGroups"`
			} `json:"zones"`
		} `json:"viewer"`
	} `json:"data"`
	Errors []struct {
		Message    string `json:"message"`
		Extensions struct {
			Code string `json:"code"`
		} `json:"extensions"`
	} `json:"errors"`
}

// HostnameRequests implements dns.AnalyticsProvider: one query for the zone, counts summed per host and minute, with
// every 5xx status counted as an error.
func (c *Client) HostnameRequests(ctx context.Context, zoneID string, hostnames []string, from, to time.Time) (_ []dns.HostMinute, err error) {
	body, err := json.Marshal(map[string]any{
		"query": requestsQuery,
		"variables": map[string]any{
			"zone": zoneID, "from": from.UTC().Format(time.RFC3339), "to": to.UTC().Format(time.RFC3339), "hosts": hostnames,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal analytics query: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base.JoinPath("graphql").String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, apperrs.Retryable(fmt.Errorf("cloudflare analytics: %w", err))
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrs.Retryable(fmt.Errorf("read cloudflare analytics: %w", err))
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errAnalyticsDenied
	}
	if resp.StatusCode >= 300 {
		return nil, apperrs.Retryable(fmt.Errorf("cloudflare analytics: unexpected status %d", resp.StatusCode))
	}
	var out graphqlResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode cloudflare analytics: %w", err)
	}
	if len(out.Errors) > 0 {
		return nil, analyticsError(out)
	}
	return hostMinutes(out)
}

// errAnalyticsDenied is a token that cannot read analytics; ErrInvalid, not ErrUnauthorized, which signs the web client out.
var errAnalyticsDenied = fmt.Errorf("%w: the Cloudflare token can't read analytics; add Zone → Analytics: Read to it", apperrs.ErrInvalid)

func analyticsError(out graphqlResponse) error {
	first := out.Errors[0]
	if first.Extensions.Code == "authz" || strings.Contains(first.Message, "does not have access") {
		return errAnalyticsDenied
	}
	return apperrs.Retryable(fmt.Errorf("cloudflare analytics: %s", first.Message))
}

func hostMinutes(out graphqlResponse) ([]dns.HostMinute, error) {
	type key struct {
		host   string
		minute time.Time
	}
	sums := map[key]*dns.HostMinute{}
	for _, zone := range out.Data.Viewer.Zones {
		for _, g := range zone.Groups {
			minute, err := time.Parse(time.RFC3339, g.Dimensions.Minute)
			if err != nil {
				return nil, fmt.Errorf("decode cloudflare analytics minute %q: %w", g.Dimensions.Minute, err)
			}
			host := strings.ToLower(g.Dimensions.Host)
			k := key{host, minute.UTC()}
			if sums[k] == nil {
				sums[k] = &dns.HostMinute{Hostname: host, Minute: k.minute}
			}
			sums[k].Requests += g.Count
			if g.Dimensions.Status >= 500 && g.Dimensions.Status < 600 {
				sums[k].Errors += g.Count
			}
		}
	}
	counts := make([]dns.HostMinute, 0, len(sums))
	for _, v := range sums {
		counts = append(counts, *v)
	}
	slices.SortFunc(counts, func(a, b dns.HostMinute) int {
		if c := strings.Compare(a.Hostname, b.Hostname); c != 0 {
			return c
		}
		return a.Minute.Compare(b.Minute)
	})
	return counts, nil
}
