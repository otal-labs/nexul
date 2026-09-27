package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// enrollRetry is how often a first start looks again for the code file or a server still booting.
const enrollRetry = time.Second

// errEnrollRefused is a refusal retrying cannot fix: the server read the code and said no.
var errEnrollRefused = errors.New("enrollment refused")
var errEnrollCodeEmpty = errors.New("the enrollment code file is still empty")

// EnrollOnFirstStart trades the code in EnrollCodeFile for the runner's credential when it has none yet and writes
// it to CredentialFile (0600). It is how a dev stack's runner enrolls itself, where no `nexul install` runs; the
// server writes the code at boot, so a missing file or an unanswered request is retried until EnrollWait.
func (c *RunnerConfig) EnrollOnFirstStart(ctx context.Context, hc *http.Client, version string) error {
	if c.Credential != "" || c.EnrollCodeFile == "" || c.CredentialFile == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, c.EnrollWait)
	defer cancel()
	for {
		credential, err := c.enrollOnce(ctx, hc, version)
		if err == nil {
			return c.writeCredential(credential)
		}
		if errors.Is(err, errEnrollRefused) {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("enroll with the code in %s: %w", c.EnrollCodeFile, err)
		case <-time.After(enrollRetry):
		}
	}
}

func (c *RunnerConfig) enrollOnce(ctx context.Context, hc *http.Client, version string) (string, error) {
	data, err := os.ReadFile(c.EnrollCodeFile)
	if err != nil {
		return "", err
	}
	code := strings.TrimSpace(string(data))
	if code == "" {
		return "", errEnrollCodeEmpty
	}
	payload, err := json.Marshal(map[string]string{
		"code": code, "name": c.Name, "os": runtime.GOOS, "arch": runtime.GOARCH,
		"version": version, "stack_root": c.StackRoot,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.ServerURL, "/")+"/api/runners/enroll", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("%w: %w", errEnrollRefused, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }() // read-only response body
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("the server answered %s", resp.Status)
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("%w: the server answered %s: %s", errEnrollRefused, resp.Status, strings.TrimSpace(string(body)))
	}
	var got struct {
		Credential string `json:"credential"`
	}
	if err := json.Unmarshal(body, &got); err != nil || got.Credential == "" {
		return "", fmt.Errorf("%w: the server's answer holds no credential", errEnrollRefused)
	}
	return got.Credential, nil
}

func (c *RunnerConfig) writeCredential(credential string) error {
	if err := os.MkdirAll(filepath.Dir(c.CredentialFile), 0o700); err != nil {
		return fmt.Errorf("create the credential directory: %w", err)
	}
	if err := os.WriteFile(c.CredentialFile, []byte(credential+"\n"), 0o600); err != nil {
		return fmt.Errorf("write NEXUL_CREDENTIAL_FILE: %w", err)
	}
	c.Credential = credential
	return nil
}
