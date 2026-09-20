// Package oauthx is the shared form-encoded OAuth token-exchange POST used by every provider client.
package oauthx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// PostForm POSTs form-encoded data to tokenURL and decodes the JSON response into out.
func PostForm(ctx context.Context, client *http.Client, tokenURL string, form url.Values, basicAuthUser, basicAuthPass, accept string, out any) (status int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if basicAuthUser != "" {
		req.SetBasicAuth(basicAuthUser, basicAuthPass)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return resp.StatusCode, err
	}
	return resp.StatusCode, nil
}
