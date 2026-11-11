package runner

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const (
	fingerprintDir = "secret-check"
	fingerprintKey = "key"
	keyBytes       = 32
)

// slugName is the shape of a stack slug, the only thing a file name here is made from.
var slugName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// trackedVar is one resolved variable: the reference it came from and the keyed hash of the value it resolved to.
type trackedVar struct {
	Ref   string `json:"ref"`
	Print string `json:"print"`
}

// stackFingerprints is what a stack's last successful deploy resolved, by variable name; DeployID says which deploy.
type stackFingerprints struct {
	DeployID string                `json:"deploy_id"`
	Vars     map[string]trackedVar `json:"vars"`
}

// Fingerprints keeps, in the runner's own directory, a keyed hash of every secret each stack's last deploy resolved,
// so a later check can tell a changed value without keeping the value. The key is made once on this machine and never
// leaves it; the files are 0600. Each stack's file also holds its references, which the check needs to read again.
type Fingerprints struct {
	dir string

	mu  sync.Mutex
	key []byte
}

// NewFingerprints keeps its files under stateDir; an empty one keeps nothing and returns nil.
func NewFingerprints(stateDir string) *Fingerprints {
	if stateDir == "" {
		return nil
	}
	return &Fingerprints{dir: filepath.Join(stateDir, fingerprintDir)}
}

// print is the keyed hash of a value.
func (f *Fingerprints) print(value string) (string, error) {
	key, err := f.loadKey()
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// loadKey reads the machine's key, making it on first use. Two runners of one directory race on creation, and the
// loser reads what the winner wrote.
func (f *Fingerprints) loadKey() ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.key != nil {
		return f.key, nil
	}
	if err := os.MkdirAll(f.dir, 0o700); err != nil {
		return nil, fmt.Errorf("create secret check directory: %w", err)
	}
	path := filepath.Join(f.dir, fingerprintKey)
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		raw, err = f.makeKey(path)
	}
	if err != nil {
		return nil, fmt.Errorf("secret check key: %w", err)
	}
	key, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(key) != keyBytes {
		return nil, errors.New("secret check key file is not a key")
	}
	f.key = key
	return key, nil
}

func (f *Fingerprints) makeKey(path string) ([]byte, error) {
	key := make([]byte, keyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	text := []byte(hex.EncodeToString(key))
	if _, err := file.Write(text); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return text, file.Close()
}

func (f *Fingerprints) stackPath(slug string) (string, error) {
	if !slugName.MatchString(slug) {
		return "", fmt.Errorf("not a stack slug: %q", slug)
	}
	return filepath.Join(f.dir, slug+".json"), nil
}

// save replaces a stack's record, through a temporary file so a crash never leaves half of one.
func (f *Fingerprints) save(slug string, rec stackFingerprints) error {
	path, err := f.stackPath(slug)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("encode fingerprints: %w", err)
	}
	if err := os.MkdirAll(f.dir, 0o700); err != nil {
		return fmt.Errorf("create secret check directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("write fingerprints: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return errors.Join(fmt.Errorf("replace fingerprints: %w", err), os.Remove(tmp))
	}
	return nil
}

// load reads a stack's record; a stack with none yields fs.ErrNotExist.
func (f *Fingerprints) load(slug string) (stackFingerprints, error) {
	path, err := f.stackPath(slug)
	if err != nil {
		return stackFingerprints{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return stackFingerprints{}, err
	}
	var rec stackFingerprints
	if err := json.Unmarshal(raw, &rec); err != nil {
		return stackFingerprints{}, fmt.Errorf("decode fingerprints of %s: %w", slug, err)
	}
	return rec, nil
}

// remove drops a stack's record; a stack with none is already clean.
func (f *Fingerprints) remove(slug string) error {
	path, err := f.stackPath(slug)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove fingerprints: %w", err)
	}
	return nil
}

// slugs lists the stacks that hold a record.
func (f *Fingerprints) slugs() ([]string, error) {
	entries, err := os.ReadDir(f.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list fingerprints: %w", err)
	}
	var out []string
	for _, e := range entries {
		slug, ok := strings.CutSuffix(e.Name(), ".json")
		if ok && slugName.MatchString(slug) {
			out = append(out, slug)
		}
	}
	return out, nil
}
