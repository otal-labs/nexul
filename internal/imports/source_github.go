package imports

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// SourceGitHub imports one attached repository's issues, read as the GitHub App, so it takes no credential.
const SourceGitHub = "github"

// GitHubIssue is an issue as the composition root reads it from GitHub (ADR 0017: imports never imports gitprovider).
type GitHubIssue struct {
	ID        int64
	Number    int
	Title     string
	Body      string
	State     string
	Labels    []string
	Assignees []string
	Author    string
	HTMLURL   string
	Comments  int
}

// GitHubComment is one comment on an issue.
type GitHubComment struct {
	ID              int64
	Body            string
	Author          string
	AuthorAvatarURL string
	CreatedAt       time.Time
}

// GitHubIssues reads a repository's issues, every state, a page at a time.
type GitHubIssues interface {
	// ListIssues returns one page of issues, pull requests left out, and the next page, 0 after the last.
	ListIssues(ctx context.Context, owner, name string, page int) ([]GitHubIssue, int, error)
	ListComments(ctx context.Context, owner, name string, number int) ([]GitHubComment, error)
}

// ProjectRepos lists the repositories attached to a project, as owner/name.
type ProjectRepos interface {
	Repos(ctx context.Context, projectID string) ([]string, error)
}

// IssueSyncs answers whether a repository's issues already sync to tickets, which an import would duplicate.
type IssueSyncs interface {
	SyncOn(ctx context.Context, owner, name string) (bool, error)
}

// GitHubSource is the GitHub issues source.
type GitHubSource struct {
	issues GitHubIssues
	repos  ProjectRepos
	syncs  IssueSyncs
}

var _ Source = (*GitHubSource)(nil)

// NewGitHubSource wires the GitHub issues source.
func NewGitHubSource(issues GitHubIssues, repos ProjectRepos, syncs IssueSyncs) *GitHubSource {
	return &GitHubSource{issues: issues, repos: repos, syncs: syncs}
}

// githubFileHosts are where GitHub serves the files people attach to issues and comments, and where it redirects them.
var githubFileHosts = []string{
	"github.com", "user-images.githubusercontent.com", "private-user-images.githubusercontent.com",
	"objects.githubusercontent.com", "github-production-user-asset-6210df.s3.amazonaws.com",
}

// Info implements Source.
func (g *GitHubSource) Info() Info { return Info{Title: "GitHub", FileHosts: githubFileHosts} }

// Scan implements Source: it pages through every issue to count them and their comments, assignees and states.
func (g *GitHubSource) Scan(ctx context.Context, job Job) (Scan, error) {
	owner, name, err := g.repository(ctx, job)
	if err != nil {
		return Scan{}, err
	}
	found := Scan{From: owner + "/" + name}
	var open, closed bool
	for page := 1; page != 0; {
		issues, next, err := g.issues.ListIssues(ctx, owner, name, page)
		if err != nil {
			return Scan{}, err
		}
		for _, i := range issues {
			found.Tickets++
			found.Comments += i.Comments
			if len(i.Assignees) > 0 {
				found.People = append(found.People, Person{Login: i.Assignees[0], Name: i.Assignees[0]})
			}
			open, closed = open || i.State != "closed", closed || i.State == "closed"
		}
		page = next
	}
	if open {
		found.Statuses = append(found.Statuses, githubStatus("open"))
	}
	if closed {
		found.Statuses = append(found.Statuses, githubStatus("closed"))
	}
	return found, nil
}

// Import implements Source: a page at a time from the checkpointed page, each issue with its comments.
func (g *GitHubSource) Import(ctx context.Context, job Job, sink Sink) error {
	owner, name, err := g.repository(ctx, job)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(job.Cursor) // an empty or unreadable cursor starts at the first page
	for page = max(page, 1); page != 0; {
		issues, next, err := g.issues.ListIssues(ctx, owner, name, page)
		if err != nil {
			return err
		}
		for _, i := range issues {
			var comments []GitHubComment
			if i.Comments > 0 {
				if comments, err = g.issues.ListComments(ctx, owner, name, i.Number); err != nil {
					return err
				}
			}
			if err := sink.Ticket(ctx, IssueTicket(owner+"/"+name, i, comments)); err != nil {
				return err
			}
		}
		if next != 0 {
			if err := sink.Checkpoint(ctx, strconv.Itoa(next)); err != nil {
				return err
			}
		}
		page = next
	}
	return nil
}

// repository is the project's repository the import names, or its only one when it names none.
func (g *GitHubSource) repository(ctx context.Context, job Job) (owner, name string, err error) {
	if job.Credential != "" {
		return "", "", fmt.Errorf("%w: a GitHub import reads as Nexul's GitHub App, so it takes no credential", apperrs.ErrInvalid)
	}
	repos, err := g.repos.Repos(ctx, job.ProjectID)
	if err != nil {
		return "", "", fmt.Errorf("list the project's repositories: %w", err)
	}
	from := strings.TrimSpace(job.From)
	if from == "" && len(repos) == 1 {
		from = repos[0]
	}
	if from == "" && len(repos) == 0 {
		return "", "", fmt.Errorf("%w: the project has no repository; attach one with project_update first", apperrs.ErrInvalid)
	}
	if from == "" {
		return "", "", fmt.Errorf("%w: the project has %d repositories; name one in from (%s)", apperrs.ErrInvalid, len(repos), strings.Join(repos, ", "))
	}
	i := slices.IndexFunc(repos, func(r string) bool { return strings.EqualFold(r, from) })
	if i < 0 {
		return "", "", fmt.Errorf("%w: %s is not attached to the project; project_get lists its repositories", apperrs.ErrInvalid, from)
	}
	owner, name, _ = strings.Cut(repos[i], "/")
	synced, err := g.syncs.SyncOn(ctx, owner, name)
	if err != nil {
		return "", "", fmt.Errorf("check the issue sync of %s: %w", repos[i], err)
	}
	if synced {
		return "", "", fmt.Errorf("%w: issue sync is on for %s, so its issues are already tickets; its backfill brings the rest in", apperrs.ErrInvalid, repos[i])
	}
	return owner, name, nil
}

// IssueTicket is the one place a GitHub issue becomes a ticket: its title, its body with a line back to the issue,
// open as backlog and closed as done, its labels, its first assignee, its comments, and the files they link to.
func IssueTicket(repo string, i GitHubIssue, comments []GitHubComment) Ticket {
	body := withLine(i.Body, fmt.Sprintf("Imported from [%s#%d](%s), opened by %s.", repo, i.Number, i.HTMLURL, i.Author))
	t := Ticket{
		SourceID: strconv.FormatInt(i.ID, 10), Title: i.Title, Body: body, Status: githubStatus(i.State),
		Labels: i.Labels, Files: githubFiles(i.Body),
	}
	if len(i.Assignees) > 0 {
		t.Assignee = Person{Login: i.Assignees[0], Name: i.Assignees[0]}
	}
	for _, c := range comments {
		t.Comments = append(t.Comments, Comment{
			SourceID: strconv.FormatInt(c.ID, 10), Body: c.Body, Author: Person{Login: c.Author, Name: c.Author},
			AvatarURL: c.AuthorAvatarURL, At: c.CreatedAt,
		})
		t.Files = append(t.Files, githubFiles(c.Body)...)
	}
	return t
}

func githubStatus(state string) StatusRef {
	if state == "closed" {
		return StatusRef{Kind: StatusDone}
	}
	return StatusRef{Kind: StatusBacklog}
}

// githubFile matches the links GitHub writes when someone drops a file into an issue or comment.
var githubFile = regexp.MustCompile(`https://(?:github\.com/user-attachments/(?:assets|files)/|github\.com/[\w.-]+/[\w.-]+/(?:assets|files)/|(?:private-)?user-images\.githubusercontent\.com/)[^\s)"'<>\]]+`)

func githubFiles(text string) []string {
	return githubFile.FindAllString(text, -1)
}
