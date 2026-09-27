package install

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// OpenObserve is a log store build: where its archives are published, the version, and each archive's sha256 by
// <os>-<arch>.
type OpenObserve struct {
	URL     string
	Version string
	SHA256  map[string]string
}

// pinnedOpenObserve is the open-source build every install runs; bumping it is how `nexul upgrade` moves it.
var pinnedOpenObserve = OpenObserve{
	URL:     "https://downloads.openobserve.ai/releases/openobserve",
	Version: "v1.0.4",
	SHA256: map[string]string{
		"linux-amd64":   "5c1b18bc072658c045ff32ca7dd0d2e3f22fac1209755d7a0ef5fd6448896e61",
		"linux-arm64":   "b56fad8dd0acfd9386af639f62baf08f96b2439f5aaaf456e1e8a0c79fafd8d0",
		"darwin-amd64":  "af2535f0a83581b6ffd22665c831ce6aa74a2712f127ec625676d6c67dd18609",
		"darwin-arm64":  "eed4f77e44dcdf1cee7850b48792514642a4fafdc6faf0ef2c04cca7c66598c5",
		"windows-amd64": "3a4c24df87f09e6d3999a9f6289c4fde9fde50151712cd1992ad8d921dc0e949",
	},
}

// openObserveBase is the path OpenObserve serves under, so the server can proxy /openobserve/ to it unchanged.
const openObserveBase = "/openobserve"

// installOpenObserve downloads the pinned archive, checks it against the pinned sha256 and puts its binary at dest.
func (h *Host) installOpenObserve(ctx context.Context, dest string) error {
	oo := h.OpenObserve
	target := h.GOOS + "-" + h.GOARCH
	want, ok := oo.SHA256[target]
	if !ok {
		return fmt.Errorf("OpenObserve publishes no build for %s", target)
	}
	ext := ".tar.gz"
	if h.GOOS == "windows" {
		ext = ".zip"
	}
	url := fmt.Sprintf("%s/%s/openobserve-%s-%s%s", oo.URL, oo.Version, oo.Version, target, ext)
	archive := dest + ".archive"
	if err := h.download(ctx, url, archive, 0o600); err != nil {
		return err
	}
	defer func() { _ = os.Remove(archive) }() // only the extracted binary is kept
	got, err := fileSHA256(archive)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", url, got, want)
	}
	tmp := dest + ".download"
	if err := extractBinary(archive, h.GOOS == "windows", h.exeName("openobserve"), tmp); err != nil {
		return errors.Join(err, removeIfPresent(tmp))
	}
	return replaceFile(tmp, dest)
}

// extractBinary copies the file named name out of a .zip or .tar.gz archive into dest (0755).
func extractBinary(archive string, isZip bool, name, dest string) error {
	open := openInTarGz
	if isZip {
		open = openInZip
	}
	src, closeSrc, err := open(archive, name)
	if err != nil {
		return err
	}
	defer closeSrc()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	_, copyErr := io.Copy(out, src)
	if err := errors.Join(copyErr, out.Close()); err != nil {
		return fmt.Errorf("extract %s: %w", name, err)
	}
	return nil
}

func openInZip(archive, name string) (io.Reader, func(), error) {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", archive, err)
	}
	for _, f := range r.File {
		if filepath.Base(f.Name) != name || f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			_ = r.Close() // the open error is the one to report
			return nil, nil, fmt.Errorf("open %s in %s: %w", name, archive, err)
		}
		return rc, func() { _ = rc.Close(); _ = r.Close() }, nil // read-only
	}
	_ = r.Close() // read-only
	return nil, nil, fmt.Errorf("%s holds no %s", archive, name)
}

func openInTarGz(archive, name string) (io.Reader, func(), error) {
	f, err := os.Open(archive)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", archive, err)
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		_ = f.Close() // the gzip error is the one to report
		return nil, nil, fmt.Errorf("read %s: %w", archive, err)
	}
	closeAll := func() { _ = gz.Close(); _ = f.Close() } // read-only
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			closeAll()
			return nil, nil, fmt.Errorf("%s holds no %s", archive, name)
		}
		if err != nil {
			closeAll()
			return nil, nil, fmt.Errorf("read %s: %w", archive, err)
		}
		if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == name {
			return tr, closeAll, nil
		}
	}
}

func removeIfPresent(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// logsEnv is OpenObserve's env: both listeners on localhost only, served under /openobserve for the server's proxy,
// and memory caps inside the unit's MemoryMax, since it otherwise sizes its caches to half the host.
func logsEnv(dir string, settings map[string]string) map[string]string {
	return map[string]string{
		"ZO_ROOT_USER_EMAIL":       settings["NEXUL_LOGS_EMAIL"],
		"ZO_ROOT_USER_PASSWORD":    settings["NEXUL_LOGS_PASSWORD"],
		"ZO_ROOT_USER_TOKEN":       settings["NEXUL_LOGS_TOKEN"],
		"ZO_HTTP_ADDR":             "127.0.0.1",
		"ZO_HTTP_PORT":             settings["NEXUL_LOGS_PORT"],
		"ZO_GRPC_ADDR":             "127.0.0.1",
		"ZO_GRPC_PORT":             settings["NEXUL_LOGS_GRPC_PORT"],
		"ZO_BASE_URI":              openObserveBase,
		"ZO_DATA_DIR":              filepath.Join(dir, "logs"),
		"ZO_MEMORY_CACHE_MAX_SIZE": "256",
		"ZO_MEM_TABLE_MAX_SIZE":    "256",
		"ZO_TELEMETRY":             "false",
	}
}
