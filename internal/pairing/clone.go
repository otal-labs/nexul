package pairing

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

var githubRepository = regexp.MustCompile(`^[A-Za-z0-9-]+/[A-Za-z0-9._-]+$`)

// gitSignInFailure matches what git says when the computer holds no credentials for the repository.
var gitSignInFailure = regexp.MustCompile(`(?i)could not read username|authentication failed|terminal prompts disabled|repository not found`)

// cloneTimeout bounds a clone, through a computer's runner or T3 Code, which goes on when the person who asked stops waiting.
const cloneTimeout = 12 * time.Minute

// ErrCloneThroughT3 means the runner cannot clone the repository, so T3 Code clones it with the computer's own git sign-in.
var ErrCloneThroughT3 = errors.New("clone through T3 Code")

// RepositoryCloner clones a GitHub repository onto a computer through its runner, so the computer needs no git sign-in.
type RepositoryCloner interface {
	// CloneRepository clones repository ("owner/name") into path on computerID and returns the folder, or ErrCloneThroughT3.
	CloneRepository(ctx context.Context, computerID, repository, path string) (string, error)
}

// defaultCloneParent is where a clone or an empty project lands on a computer with no T3 projects to sit beside.
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
	path, _, err := s.newProjectPath(ctx, client, session, name)
	if err != nil {
		return harness.Project{}, err
	}
	if project, err := s.cloneThroughRunner(ctx, client, session, repository, name, path); !errors.Is(err, ErrCloneThroughT3) {
		return project, err
	}
	if !s.mayClone(ctx, client, session) {
		return harness.Project{}, harness.ErrPairAgainToClone
	}
	return s.cloneThroughT3(ctx, client, session, repository, name, path)
}

// cloneThroughT3 has T3 Code clone with the computer's own git sign-in and waits for the outcome; it finishes even when
// the person stops waiting, so a slow clone still becomes a project.
func (s *Service) cloneThroughT3(ctx context.Context, client harness.Client, session Computer, repository, name, path string) (harness.Project, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cloneTimeout)
	defer cancel()
	project, err := client.CloneProject(ctx, session.Session(), name, "https://github.com/"+repository+".git", path)
	var failed *harness.CloneFailedError
	if errors.As(err, &failed) && gitSignInFailure.MatchString(failed.Detail) {
		return harness.Project{}, fmt.Errorf("%w: %s has no GitHub sign-in, so it couldn't clone %s: sign in with `gh auth login` on it, or update the Nexul app on it so Nexul can clone with a token", apperrs.ErrInvalid, session.Name, repository)
	}
	if errors.As(err, &failed) {
		return harness.Project{}, fmt.Errorf("%w: couldn't clone %s on %s: %s", apperrs.ErrInvalid, repository, session.Name, failed.Detail)
	}
	if err != nil {
		return harness.Project{}, fmt.Errorf("clone %s on %s: %w", repository, session.Name, err)
	}
	return project, nil
}

// cloneThroughRunner clones with a token for the repository alone, then makes the checkout a T3 project; it finishes
// even when the person stops waiting, so a slow clone still becomes a project.
func (s *Service) cloneThroughRunner(ctx context.Context, client harness.Client, session Computer, repository, name, path string) (harness.Project, error) {
	if s.cloner == nil {
		return harness.Project{}, ErrCloneThroughT3
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cloneTimeout)
	defer cancel()
	dir, err := s.cloner.CloneRepository(ctx, session.ID, repository, path)
	if errors.Is(err, ErrCloneThroughT3) {
		return harness.Project{}, err
	}
	if err != nil {
		return harness.Project{}, fmt.Errorf("clone %s on %s: %w", repository, session.Name, err)
	}
	project, err := client.CreateProject(ctx, session.Session(), name, dir)
	if err != nil {
		return harness.Project{}, fmt.Errorf("add the clone of %s on %s to T3 Code: %w", repository, session.Name, err)
	}
	return project, nil
}

// AddProject is the one entry for a new T3 project on a computer: clone repository, or create an empty folder called name.
func (s *Service) AddProject(ctx context.Context, userID, computerID, repository, name string) (harness.Project, error) {
	repository, name = strings.TrimSpace(repository), strings.TrimSpace(name)
	if (repository == "") == (name == "") {
		return harness.Project{}, fmt.Errorf("%w: give either a repository to clone or a name for an empty project", apperrs.ErrInvalid)
	}
	add, what := s.CloneProject, repository
	if name != "" {
		add, what = s.CreateEmptyProject, name
	}
	project, err := add(ctx, userID, computerID, what)
	if err != nil {
		return harness.Project{}, err
	}
	// Placement reads the stored facts, so a project they don't list yet would leave its runs waiting with no_checkout.
	if err := s.recordFacts(ctx, userID, computerID, nil); err != nil {
		logging.FromCtx(ctx).Warn("computer facts not read after adding a project", "computer_id", computerID, "error", err)
	}
	return project, nil
}

// CreateEmptyProject makes a new empty folder called name as a T3 project on one of the caller's computers, beside the
// ones it already has, so a computer with no repository to clone still has a project to run in.
func (s *Service) CreateEmptyProject(ctx context.Context, userID, computerID, name string) (harness.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.Trim(name, ".") == "" || strings.ContainsAny(name, "/\\") || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return harness.Project{}, fmt.Errorf("%w: name must be a folder name, with no slashes, such as acme-notes", apperrs.ErrInvalid)
	}
	session, err := s.sessionComputer(ctx, userID, computerID)
	if err != nil {
		return harness.Project{}, err
	}
	client, err := s.client(session.Kind)
	if err != nil {
		return harness.Project{}, err
	}
	path, existing, err := s.newProjectPath(ctx, client, session, name)
	if err != nil {
		return harness.Project{}, err
	}
	if i := slices.IndexFunc(existing, func(p harness.Project) bool { return p.Path == path }); i >= 0 {
		return harness.Project{}, fmt.Errorf("%w: %s already has the T3 project %q at %s; use it instead", apperrs.ErrConflict, session.Name, existing[i].Title, path)
	}
	project, err := client.CreateProject(ctx, session.Session(), name, path)
	if err != nil {
		return harness.Project{}, fmt.Errorf("create %s on %s: %w", name, session.Name, err)
	}
	return project, nil
}

// newProjectPath is where a project called name lands on the computer, beside the projects it already has, which it returns too.
func (s *Service) newProjectPath(ctx context.Context, client harness.Client, session Computer, name string) (string, []harness.Project, error) {
	projects, err := client.ListProjects(ctx, session.Session())
	if err != nil {
		return "", nil, fmt.Errorf("list projects on %s: %w", session.Name, err)
	}
	parent, sep := cloneParent(projects)
	return parent + sep + name, projects, nil
}

// CanClone reports whether the computer's T3 session may start a clone, so the picker can say to pair it again before a click.
func (s *Service) CanClone(ctx context.Context, userID, computerID string) (bool, error) {
	session, err := s.sessionComputer(ctx, userID, computerID)
	if err != nil {
		return false, err
	}
	client, err := s.client(session.Kind)
	if err != nil {
		return false, err
	}
	return s.mayClone(ctx, client, session), nil
}

// mayClone fails open when the scopes cannot be read: T3 itself refuses a clone it will not take.
func (s *Service) mayClone(ctx context.Context, client harness.Client, session Computer) bool {
	allowed, err := client.CanClone(ctx, session.Session())
	if err != nil {
		logging.FromCtx(ctx).Warn("clone scope not read", "computer_id", session.ID, "error", err)
		return true
	}
	return allowed
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
