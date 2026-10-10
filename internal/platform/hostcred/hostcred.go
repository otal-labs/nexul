// Package hostcred mints and checks the one-time enrollment codes and long-lived credentials hosts (runners,
// automations hosts) authenticate with; only hashes are ever stored.
package hostcred

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	windowsPin := ""
	if version.IsRelease() {
		windowsPin = "$env:NEXUL_VERSION='" + version.Version + "'; "
	}
	windows = windowsPin + "& ([scriptblock]::Create((irm https://nexul.io/" + script + ".ps1))) " + args
	return unixCommand(script, args), windows
}

// ComputerCommand renders the one-liner that adds a computer through computer.sh, run with sudo from the person's own
// account; a site or release other than nexul.io's travels in the command, for an instance tested against its own
// build, as variables sudo sets for the script.
func ComputerCommand(site, release, token string) string {
	env := "sudo "
	if site != DefaultSite {
		env += "NEXUL_INSTALL_URL=" + site + "/install.sh "
	}
	if release != "" {
		env += "NEXUL_RELEASE_URL=" + release + " "
	}
	return unixCommandFrom(site, "computer", env, token)
}

func unixCommand(script, args string) string {
	return unixCommandFrom(DefaultSite, script, "", args)
}

// unixCommandFrom pipes site's script into sh with env set for it; a release build pins its own version.
func unixCommandFrom(site, script, env, args string) string {
	if version.IsRelease() {
		env += "NEXUL_VERSION=" + version.Version + " "
	}
	return "curl -fsSL " + site + "/" + script + ".sh | " + env + "sh -s -- " + args
}

// DefaultSite serves the install scripts.
const DefaultSite = "https://nexul.io"

// ErrBadToken is a token that is malformed, signed with another key, or past its expiry.
var ErrBadToken = errors.New("the token is not one this instance issued, or it expired")

var tokenHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

// SignToken renders claims as a compact HS256 JWT under key.
func SignToken(claims any, key []byte) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("encode token claims: %w", err)
	}
	unsigned := tokenHeader + "." + base64.RawURLEncoding.EncodeToString(payload)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(tokenMAC(unsigned, key)), nil
}

// ParseToken checks token's signature under key and decodes its claims; the caller checks their expiry.
func ParseToken(token string, key []byte, claims any) error {
	i := strings.LastIndex(token, ".")
	if strings.Count(token, ".") != 2 {
		return ErrBadToken
	}
	got, err := base64.RawURLEncoding.DecodeString(token[i+1:])
	if err != nil || !hmac.Equal(got, tokenMAC(token[:i], key)) {
		return ErrBadToken
	}
	return PeekToken(token, claims)
}

// PeekToken decodes token's claims without checking its signature, for an installer that only needs to know where
// to send it; the issuing instance is what checks it.
func PeekToken(token string, claims any) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ErrBadToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(payload, claims) != nil {
		return ErrBadToken
	}
	return nil
}

func tokenMAC(unsigned string, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(unsigned))
	return mac.Sum(nil)
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
