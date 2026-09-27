// Package hostcred mints and checks the one-time enrollment codes and long-lived credentials hosts (runners,
// automations hosts) authenticate with; only hashes are ever stored.
package hostcred

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/version"
)

// CodePrefix marks an enrollment code.
const CodePrefix = "nxe_"

// NamePattern is what a host's name must match: it becomes the service and directory name on its machine.
var NamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

// MintCode returns a fresh enrollment code and its hash.
func MintCode() (raw, hash string, err error) {
	return MintCredential(CodePrefix)
}

// MintCredential returns prefix plus 32 random bytes base64url, and its hash.
func MintCredential(prefix string) (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate %s credential: %w", prefix, err)
	}
	raw = prefix + base64.RawURLEncoding.EncodeToString(buf)
	return raw, Hash(raw), nil
}

// Hash is the sha256 hex digest stored in place of a code or credential.
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Matches reports whether raw hashes to hash, in constant time.
func Matches(raw, hash string) bool {
	return subtle.ConstantTimeCompare([]byte(Hash(raw)), []byte(hash)) == 1
}

// InstallCommands renders the one-liners that install a host through script (runner or automations) on Linux or
// macOS and on Windows; a release build pins them to its own version.
func InstallCommands(script, instanceURL, name, code string) (unix, windows string) {
	args := fmt.Sprintf("--server %s --name %s --code %s", strings.TrimRight(instanceURL, "/"), name, code)
	unixPin, windowsPin := "", ""
	if version.IsRelease() {
		unixPin = "NEXUL_VERSION=" + version.Version + " "
		windowsPin = "$env:NEXUL_VERSION='" + version.Version + "'; "
	}
	unix = "curl -fsSL https://nexul.io/" + script + ".sh | " + unixPin + "sh -s -- " + args
	windows = windowsPin + "& ([scriptblock]::Create((irm https://nexul.io/" + script + ".ps1))) " + args
	return unix, windows
}

// WriteCodeFile writes a bundled host's code (0600) through a rename, so an installer polling the file never reads
// it half written.
func WriteCodeFile(path, code string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".code-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // a no-op once the rename has moved it
	if _, err := tmp.WriteString(code); err != nil {
		_ = tmp.Close() // the write error is the one worth returning
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close() // the chmod error is the one worth returning
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
