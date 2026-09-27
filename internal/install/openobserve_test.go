package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallOpenObserve(t *testing.T) {
	t.Run("a tarball's binary is extracted", func(t *testing.T) {
		th := newTestHost(t)
		dest := filepath.Join(th.root, "openobserve")
		require.NoError(t, th.installOpenObserve(t.Context(), dest))
		assert.Equal(t, "openobserve-v1.0.4", readFile(t, dest))
		assert.NoFileExists(t, dest+".archive")
	})
	t.Run("Windows gets the zip's exe", func(t *testing.T) {
		th := newTestHost(t)
		th.GOOS = "windows"
		dest := filepath.Join(th.root, "openobserve.exe")
		require.NoError(t, th.installOpenObserve(t.Context(), dest))
		assert.Equal(t, "openobserve-v1.0.4", readFile(t, dest))
	})
	t.Run("an archive that does not match the pin is refused and the old binary kept", func(t *testing.T) {
		th := newTestHost(t)
		th.OpenObserve.SHA256["linux-amd64"] = strings.Repeat("0", 64)
		dest := filepath.Join(th.root, "openobserve")
		require.NoError(t, os.WriteFile(dest, []byte("old"), 0o755))
		require.ErrorContains(t, th.installOpenObserve(t.Context(), dest), "checksum mismatch")
		assert.Equal(t, "old", readFile(t, dest))
		assert.NoFileExists(t, dest+".archive")
	})
	t.Run("a target without a build is named", func(t *testing.T) {
		th := newTestHost(t)
		th.GOARCH = "riscv64"
		require.ErrorContains(t, th.installOpenObserve(t.Context(), filepath.Join(th.root, "x")), "no build for linux-riscv64")
	})
	t.Run("a missing archive is an HTTP error", func(t *testing.T) {
		th := newTestHost(t)
		th.OpenObserve.Version = "v9.9.9"
		require.ErrorContains(t, th.installOpenObserve(t.Context(), filepath.Join(th.root, "x")), "404")
	})
}

func TestExtractBinary_Failures(t *testing.T) {
	dir := t.TempDir()
	tgz := filepath.Join(dir, "a.tar.gz")
	require.NoError(t, os.WriteFile(tgz, tarGz(t, "other", []byte("x")), 0o600))
	zp := filepath.Join(dir, "a.zip")
	require.NoError(t, os.WriteFile(zp, zipped(t, "other.exe", []byte("x")), 0o600))
	junk := filepath.Join(dir, "junk")
	require.NoError(t, os.WriteFile(junk, []byte("not an archive"), 0o600))

	require.ErrorContains(t, extractBinary(tgz, false, "openobserve", filepath.Join(dir, "out")), "holds no openobserve")
	require.ErrorContains(t, extractBinary(zp, true, "openobserve.exe", filepath.Join(dir, "out")), "holds no openobserve.exe")
	require.ErrorContains(t, extractBinary(junk, false, "openobserve", filepath.Join(dir, "out")), "read")
	require.ErrorContains(t, extractBinary(junk, true, "openobserve", filepath.Join(dir, "out")), "open")
	require.ErrorContains(t, extractBinary(filepath.Join(dir, "absent"), false, "openobserve", filepath.Join(dir, "out")), "open")
	require.ErrorContains(t, extractBinary(tgz, false, "other", filepath.Join(dir, "no", "such", "dir")), "create")
}

func TestPinnedOpenObserve_CoversEveryReleaseTarget(t *testing.T) {
	for _, target := range []string{"linux-amd64", "linux-arm64", "darwin-amd64", "darwin-arm64", "windows-amd64"} {
		assert.Len(t, pinnedOpenObserve.SHA256[target], 64, target)
	}
}
