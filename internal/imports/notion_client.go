package imports

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// NotionVersion is the Notion-Version every request sends, the latest the API accepts.
const NotionVersion = "2026-03-11"

// notionBaseURL is Notion's public API.
const notionBaseURL = "https://api.notion.com"

// notionRetries is how many times one request waits out a 429 before the import pauses and reads again later.
const notionRetries = 3

// NotionClient reads Notion's API with an internal integration token, at Notion's average of 3 requests a second.
type NotionClient struct {
	base    string
	http    *http.Client
	limiter *rate.Limiter
	// sleep waits out a Retry-After; tests replace it.
	sleep func(ctx context.Context, d time.Duration) error
}

// NewNotionClient reads base, Notion's API when empty, with requests a second on average (0 is unlimited).
func NewNotionClient(base string, client *http.Client, perSecond float64) *NotionClient {
	if base == "" {
		base = notionBaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: time.Minute}
	}
	limiter := rate.NewLimiter(rate.Inf, 1)
	if perSecond > 0 {
		limiter = rate.NewLimiter(rate.Limit(perSecond), int(max(perSecond, 1)))
	}
	return &NotionClient{base: strings.TrimRight(base, "/"), http: client, limiter: limiter, sleep: sleepCtx}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// call sends one request with token and decodes the answer into out; the token never reaches an error or a log.
func (c *NotionClient) call(ctx context.Context, token, method, path string, body any, out any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	for attempt := 0; ; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Notion-Version", NotionVersion)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("%w: reach Notion: %v", apperrs.ErrRetryable, err)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		_ = resp.Body.Close() // a read-only body: nothing to lose on close
		if err != nil {
			return fmt.Errorf("%w: read Notion's answer: %v", apperrs.ErrRetryable, err)
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < notionRetries {
			if err := c.sleep(ctx, retryAfterOf(resp.Header.Get("Retry-After"))); err != nil {
				return err
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return notionError(resp.StatusCode, data)
		}
		return json.Unmarshal(data, out)
	}
}

// retryAfterOf reads Retry-After in seconds, waiting a second when it is missing or unreadable.
func retryAfterOf(v string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || secs < 1 {
		return time.Second
	}
	return time.Duration(secs) * time.Second
}

// notionError maps an answer onto the error that says what to fix; a 404 is a page not shared with the integration.
func notionError(status int, data []byte) error {
	var e struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(data, &e) // a body that is not Notion's error shape leaves the status to speak
	switch {
	case status == http.StatusUnauthorized:
		return fmt.Errorf("%w: Notion refused the token; copy the internal integration secret again", apperrs.ErrUnauthorized)
	case status == http.StatusNotFound || status == http.StatusForbidden:
		return fmt.Errorf("%w: Notion has nothing shared with the integration there (%s)", apperrs.ErrNotFound, e.Code)
	case status == http.StatusTooManyRequests || status >= 500:
		return fmt.Errorf("%w: Notion answered %d", apperrs.ErrRetryable, status)
	}
	return fmt.Errorf("%w: Notion answered %d: %s", apperrs.ErrInvalid, status, e.Message)
}

type notionList[T any] struct {
	Results    []T    `json:"results"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor"`
}

// notionParent is where a page, database or block sits.
type notionParent struct {
	Type         string `json:"type"`
	PageID       string `json:"page_id"`
	DatabaseID   string `json:"database_id"`
	DataSourceID string `json:"data_source_id"`
	BlockID      string `json:"block_id"`
}

// notionPage is a page or a database row, as search and the page endpoints return it.
type notionPage struct {
	ID          string                    `json:"id"`
	CreatedTime time.Time                 `json:"created_time"`
	InTrash     bool                      `json:"in_trash"`
	Parent      notionParent              `json:"parent"`
	Properties  map[string]notionProperty `json:"properties"`
}

// notionDatabase is a database: a title, a parent, and the data sources whose rows are its pages.
type notionDatabase struct {
	ID     string           `json:"id"`
	Title  []notionRichText `json:"title"`
	Parent notionParent     `json:"parent"`
}

// searchPages pages through every page shared with the integration, database rows included.
func (c *NotionClient) searchPages(ctx context.Context, token string) ([]notionPage, error) {
	var out []notionPage
	cursor := ""
	for {
		body := map[string]any{"page_size": 100, "filter": map[string]string{"property": "object", "value": "page"}}
		if cursor != "" {
			body["start_cursor"] = cursor
		}
		var page notionList[notionPage]
		if err := c.call(ctx, token, http.MethodPost, "/v1/search", body, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Results...)
		if !page.HasMore || page.NextCursor == "" {
			return out, nil
		}
		cursor = page.NextCursor
	}
}

func (c *NotionClient) database(ctx context.Context, token, id string) (notionDatabase, error) {
	var db notionDatabase
	err := c.call(ctx, token, http.MethodGet, "/v1/databases/"+url.PathEscape(id), nil, &db)
	return db, err
}

func (c *NotionClient) block(ctx context.Context, token, id string) (notionBlock, error) {
	var b notionBlock
	err := c.call(ctx, token, http.MethodGet, "/v1/blocks/"+url.PathEscape(id), nil, &b)
	return b, err
}

// children reads a block's children, 100 at a time.
func (c *NotionClient) children(ctx context.Context, token, id string) ([]notionBlock, error) {
	var out []notionBlock
	cursor := ""
	for {
		q := url.Values{"page_size": {"100"}}
		if cursor != "" {
			q.Set("start_cursor", cursor)
		}
		var page notionList[notionBlock]
		if err := c.call(ctx, token, http.MethodGet, "/v1/blocks/"+url.PathEscape(id)+"/children?"+q.Encode(), nil, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Results...)
		if !page.HasMore || page.NextCursor == "" {
			return out, nil
		}
		cursor = page.NextCursor
	}
}
