package pairing

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

var githubRepository = regexp.MustCompile(`^[A-Za-z0-9-]+/[A-Za-z0-9._-]+$`)

// defaultCloneParent is where a clone lands on a computer with no T3 projects to sit beside.
const defaultCloneParent = "~/Code"

// CloneProject clones a GitHub repository ("owner/name") onto one of the caller's computers as a new T3 project,
// in the folder most of its projects already share.
func (s *Service) CloneProject(ctx context.Context, userID, computerID, repository string) (harness.Project, error) {
	repository = strings.TrimSpace(repository)
	_, name, _ := strings.Cut(repository, "/")
	if !githubRepository.MatchString(repository) || strings.Trim(name, ".") == "" {
		return harness.Project{}, fmt.Errorf("%w: repository must be a GitHub owner/name, such as acme/api", apperrs.ErrInvalid)
	}
	session, err := s.sessionComputer(ctx, userID, computerID)
	if err != nil {
		return harness.Project{}, err
	}
	client, err := s.client(session.Kind)
	if err != nil {
		return harness.Project{}, err
	}
	projects, err := client.ListProjects(ctx, session.Session())
	if err != nil {
		return harness.Project{}, fmt.Errorf("list projects on %s: %w", session.Name, err)
	}
	parent, sep := cloneParent(projects)
	project, err := client.CloneProject(ctx, session.Session(), name, "https://github.com/"+repository+".git", parent+sep+name)
	if err != nil {
		return harness.Project{}, fmt.Errorf("clone %s on %s: %w", repository, session.Name, err)
	}
	return project, nil
}

// cloneParent is the folder holding the most of projects, with the separator its paths use; ties go to the first seen.
func cloneParent(projects []harness.Project) (parent, sep string) {
	counts := map[string]int{}
	parent, sep, best := defaultCloneParent, "/", 0
	for _, p := range projects {
		i := strings.LastIndexAny(p.Path, `/\`)
		if i <= 0 {
			continue
		}
		dir := p.Path[:i]
		counts[dir]++
		if counts[dir] > best {
			parent, sep, best = dir, p.Path[i:i+1], counts[dir]
		}
	}
	return parent, sep
}
