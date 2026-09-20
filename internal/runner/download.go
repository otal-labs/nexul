package runner

import (
	"context"
	"fmt"
	"io"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/release"
	"github.com/otal-labs/nexul/internal/platform/version"
)

const runnerAssetPrefix = "nexul-runner-"

// Asset is a runner binary streamed from a GitHub release; callers must close Body.
type Asset struct {
	Name   string
	Size   int64
	Sha256 string
	Body   io.ReadCloser
}

var runnerTargets = map[string]bool{
	"linux-amd64":   true,
	"linux-arm64":   true,
	"darwin-amd64":  true,
	"darwin-arm64":  true,
	"windows-amd64": true,
}

// AssetName returns the release asset filename for target and whether target is a supported build.
func AssetName(target string) (string, bool) {
	if !runnerTargets[target] {
		return "", false
	}
	name := runnerAssetPrefix + target
	if target == "windows-amd64" {
		name += ".exe"
	}
	return name, true
}

// Download streams target's runner binary for tag, or for this server's own version/channel latest when tag is
// empty, alongside its sha256 from checksums.txt (empty when the release ships none).
func (s *Service) Download(ctx context.Context, target, tag string) (*Asset, error) {
	name, ok := AssetName(target)
	if !ok {
		return nil, fmt.Errorf("download target %q: %w", target, apperrs.ErrInvalid)
	}
	rel, err := s.resolveRelease(ctx, tag)
	if err != nil {
		return nil, err
	}
	sums, err := s.install.Release.Checksums(ctx, rel)
	if err != nil {
		return nil, err
	}
	body, size, err := s.install.Release.Download(ctx, rel, name)
	if err != nil {
		return nil, err
	}
	return &Asset{Name: name, Size: size, Sha256: sums[name], Body: body}, nil
}

// LatestVersion returns this server's own version when it is a release build, else the channel's latest tag.
func (s *Service) LatestVersion(ctx context.Context) (string, error) {
	if version.IsRelease() {
		return version.Version, nil
	}
	rel, err := s.install.Release.Latest(ctx, version.Channel())
	if err != nil {
		return "", err
	}
	return rel.Tag, nil
}

// resolveRelease looks tag up directly when given, else this server's own version (a release build) or the
// channel's latest (a dev build).
func (s *Service) resolveRelease(ctx context.Context, tag string) (*release.Release, error) {
	if tag != "" {
		return s.install.Release.ByTag(ctx, tag)
	}
	if version.IsRelease() {
		return s.install.Release.ByTag(ctx, version.Version)
	}
	return s.install.Release.Latest(ctx, version.Channel())
}
