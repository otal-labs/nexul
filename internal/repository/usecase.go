package repository

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Gate is the permission check the repository reads and the installation assignments pass (ADR 0087, ADR 0144).
type Gate interface {
	RequireAnywhere(ctx context.Context, action permissions.Action) error
	Require(ctx context.Context, workspaceID string, action permissions.Action) error
	// WorkspacesWith lists the workspaces in which the caller holds action.
	WorkspacesWith(ctx context.Context, action permissions.Action) ([]string, error)
}

// requireWizard is what a scan takes: projects:write, what making a project from a repository needs, anywhere.
func requireWizard(ctx context.Context, g Gate) error {
	if g == nil {
		return permissions.Ungated(ctx)
	}
	return g.RequireAnywhere(ctx, permissions.ProjectsWrite)
}

// Scan reads owner/name's tree at ref (the repo's default branch when ref is empty) and proposes deployable
// candidates: one per compose file, one per standalone Dockerfile, compose sorted before Dockerfile.
func Scan(ctx context.Context, g Gate, s Scanner, owner, name, ref string) (*ScanResult, error) {
	if err := requireWizard(ctx, g); err != nil {
		return nil, err
	}
	if owner == "" || name == "" {
		return nil, fmt.Errorf("%w: owner and name are required", apperrors.ErrInvalid)
	}
	resolvedRef, entries, err := s.GetTree(ctx, owner, name, ref)
	if err != nil {
		return nil, fmt.Errorf("scan %s/%s: %w", owner, name, err)
	}

	composePaths, dockerfilePaths, envPaths := classifyScanEntries(entries)

	candidates, err := scanCandidates(ctx, s, owner, name, resolvedRef, composePaths, dockerfilePaths)
	if err != nil {
		return nil, err
	}

	envKeys, err := scanEnvKeys(ctx, s, owner, name, resolvedRef, envPaths)
	if err != nil {
		return nil, fmt.Errorf("scan %s/%s: %w", owner, name, err)
	}

	return &ScanResult{DefaultBranch: resolvedRef, Candidates: candidates, EnvKeys: envKeys}, nil
}

// classifyScanEntries buckets a tree's blobs into compose files, standalone Dockerfiles, and .env.example
// files, each sorted so candidate order is deterministic.
func classifyScanEntries(entries []TreeEntry) (composePaths, dockerfilePaths, envPaths []string) {
	for _, e := range entries {
		if e.Type != "blob" || skipPath(e.Path) {
			continue
		}
		base := path.Base(e.Path)
		switch {
		case isComposeFile(base):
			composePaths = append(composePaths, e.Path)
		case isDockerfile(base):
			dockerfilePaths = append(dockerfilePaths, e.Path)
		case isEnvExample(base):
			envPaths = append(envPaths, e.Path)
		}
	}
	sort.Strings(composePaths)
	sort.Strings(dockerfilePaths)
	sort.Strings(envPaths)
	return composePaths, dockerfilePaths, envPaths
}

// scanCandidates builds one Candidate per compose file (compose sorted before Dockerfile) and one per
// standalone Dockerfile, skipping compose files that fail to parse.
func scanCandidates(ctx context.Context, s Scanner, owner, name, resolvedRef string, composePaths, dockerfilePaths []string) ([]Candidate, error) {
	candidates := make([]Candidate, 0, len(composePaths)+len(dockerfilePaths))
	for _, p := range composePaths {
		c, err := scanComposeCandidate(ctx, s, owner, name, resolvedRef, name, p)
		if errors.Is(err, errUnparsable) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("scan %s/%s: %w", owner, name, err)
		}
		candidates = append(candidates, c)
	}
	for _, p := range dockerfilePaths {
		c, err := scanDockerfileCandidate(ctx, s, owner, name, resolvedRef, name, p)
		if err != nil {
			return nil, fmt.Errorf("scan %s/%s: %w", owner, name, err)
		}
		candidates = append(candidates, c)
	}
	return candidates, nil
}

// MinSearchLength is the shortest q accepted; fewer characters match too much of an installation to be worth it.
const MinSearchLength = 3

// candidateName derives a candidate's display name from the repo name plus the file's directory, when not root.
func candidateName(repoName, filePath string) string {
	dir := path.Dir(filePath)
	if dir == "." {
		return repoName
	}
	return repoName + "/" + dir
}

// skipPath reports whether path lies under vendor/, node_modules/, or .git/.
func skipPath(p string) bool {
	for _, seg := range strings.Split(p, "/") {
		switch seg {
		case "vendor", "node_modules", ".git":
			return true
		}
	}
	return false
}

func isDockerfile(base string) bool {
	return base == "Dockerfile" || strings.HasPrefix(base, "Dockerfile.")
}

// isComposeFile matches docker-compose*.y(a)ml and compose*.y(a)ml, per spec §5's filename glob.
func isComposeFile(base string) bool {
	lower := strings.ToLower(base)
	if !strings.HasSuffix(lower, ".yml") && !strings.HasSuffix(lower, ".yaml") {
		return false
	}
	return strings.HasPrefix(lower, "docker-compose") || strings.HasPrefix(lower, "compose")
}

func isEnvExample(base string) bool {
	return base == ".env.example" || base == ".env.sample"
}

// firstReachable is the first service, in declared order, that publishes or exposes a port.
func firstReachable(services []DeclaredService) *Reachable {
	for _, svc := range services {
		if len(svc.Ports) > 0 {
			return &Reachable{Service: svc.Name, Port: svc.Ports[0]}
		}
		if len(svc.Expose) > 0 {
			return &Reachable{Service: svc.Name, Port: svc.Expose[0]}
		}
	}
	return nil
}

// parsePortToken parses one port token ("80", "80/tcp", "3000-3005") into its (first) integer port.
func parsePortToken(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "/"); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, "-"); i >= 0 {
		s = s[:i]
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
