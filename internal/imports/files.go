package imports

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"
)

// MaxFileSize is the largest file copied into an attachment, the attachments' own cap; a larger one stays a link.
const MaxFileSize = 10 << 20

// ErrTooLarge is a file over MaxFileSize.
var ErrTooLarge = errors.New("file is over 10 MB")

// errHost is a file, or a redirect, outside the source's hosts.
var errHost = errors.New("file host is not one the source allows")

// errPage is a web page served where a file was linked, such as the sign-in page GitHub answers a private asset with.
var errPage = errors.New("the link answered with a web page, not a file")

const maxRedirects = 5

// Files downloads one file a source links to.
type Files interface {
	// Fetch returns the file's name and bytes; ErrTooLarge past MaxFileSize, and an error for any host not in hosts.
	// header signs the request when the source's own API serves the file, nil otherwise.
	Fetch(ctx context.Context, rawURL string, hosts []string, header http.Header) (name string, data []byte, err error)
}

// Fetcher downloads over plain HTTPS: it carries the source's credential only where the source signs the request for
// its own API, Go drops that on a redirect to another domain, and every hop must be one of the source's hosts, so a
// link in imported text cannot reach the instance's own network.
type Fetcher struct {
	client *http.Client
}

// NewFetcher builds a Fetcher with a one-minute bound on each file.
func NewFetcher() *Fetcher { return &Fetcher{client: &http.Client{Timeout: time.Minute}} }

// Fetch implements Files.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string, hosts []string, header http.Header) (string, []byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil || !allowedHost(u, hosts) {
		return "", nil, errHost
	}
	client := *f.client
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("stopped after %d redirects", maxRedirects)
		}
		if !allowedHost(req.URL, hosts) {
			return errHost
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", nil, err
	}
	for k, v := range header {
		req.Header[k] = v
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = resp.Body.Close() }() // a read-only body: nothing to lose on close
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("download answered %s", resp.Status)
	}
	if resp.ContentLength > MaxFileSize {
		return "", nil, ErrTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxFileSize+1))
	if err != nil {
		return "", nil, err
	}
	if len(data) > MaxFileSize {
		return "", nil, ErrTooLarge
	}
	if isPage(resp.Header.Get("Content-Type")) || isPage(http.DetectContentType(data)) {
		return "", nil, errPage
	}
	return cmp.Or(strings.Trim(path.Base(u.Path), "/."), "file"), data, nil
}

func isPage(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "text/html")
}

func allowedHost(u *url.URL, hosts []string) bool {
	return u.Scheme == "https" && u.User == nil && slices.Contains(hosts, strings.ToLower(u.Hostname()))
}

// rewriteLinks points every copied file's URL in text at its attachment, longest URL first so none is cut short.
func rewriteLinks(text string, copied map[string]string) string {
	urls := make([]string, 0, len(copied))
	for u := range copied {
		urls = append(urls, u)
	}
	slices.SortFunc(urls, func(a, b string) int { return len(b) - len(a) })
	for _, u := range urls {
		text = strings.ReplaceAll(text, u, "/api/attachments/"+copied[u])
	}
	return text
}
