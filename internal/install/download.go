package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// releaseAsset downloads name from the tag's release into dest, refusing it unless it matches the release's
// checksums.txt.
func (h *Host) releaseAsset(ctx context.Context, tag, name, dest string) error {
	base := h.ReleaseURL + "/" + tag + "/"
	sum, err := h.fetchChecksum(ctx, base+"checksums.txt", name)
	if err != nil {
		return err
	}
	return h.downloadVerified(ctx, base+name, dest, sum)
}

// assetName is a release binary's file name for this host, e.g. nexul-runner-linux-amd64.
func (h *Host) assetName(binary string) string {
	return fmt.Sprintf("%s-%s-%s", binary, h.GOOS, h.GOARCH)
}

// fetchChecksum reads a sha256sum-format file and returns the digest listed for name.
func (h *Host) fetchChecksum(ctx context.Context, url, name string) (string, error) {
	body, err := h.get(ctx, url)
	if err != nil {
		return "", err
	}
	defer func() { _ = body.Close() }() // read-only response body
	data, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.TrimPrefix(fields[len(fields)-1], "*") == name {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("%s lists no checksum for %s", url, name)
}

// downloadVerified writes url to dest (0755) only once its sha256 matches want; the old file stays until then.
func (h *Host) downloadVerified(ctx context.Context, url, dest, want string) error {
	tmp := dest + ".download"
	if err := h.download(ctx, url, tmp, 0o755); err != nil {
		return err
	}
	got, err := fileSHA256(tmp)
	if err != nil {
		return errors.Join(err, os.Remove(tmp))
	}
	if got != want {
		return errors.Join(fmt.Errorf("checksum mismatch for %s: got %s, want %s", url, got, want), os.Remove(tmp))
	}
	if err := os.Rename(tmp, dest); err != nil {
		return fmt.Errorf("install %s: %w", dest, err)
	}
	return nil
}

func (h *Host) download(ctx context.Context, url, dest string, mode os.FileMode) error {
	body, err := h.get(ctx, url)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }() // read-only response body
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(dest), err)
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	_, copyErr := io.Copy(f, body)
	if err := errors.Join(copyErr, f.Close()); err != nil {
		return errors.Join(fmt.Errorf("download %s: %w", url, err), os.Remove(dest))
	}
	return nil
}

func (h *Host) get(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", url, err)
	}
	resp, err := h.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close() // the error below is what matters
		return nil, fmt.Errorf("download %s: %s", url, resp.Status)
	}
	return resp.Body, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }() // read-only
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
