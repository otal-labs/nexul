package pairing

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// ComputerTools installs a tool on a computer and sets its git identity, through its runner.
type ComputerTools interface {
	// InstallTool has computerID's runner install tool with the command its facts named; it returns once installed.
	InstallTool(ctx context.Context, computerID, tool string) error
	// SetGitIdentity sets git's global user.name and user.email on computerID, each only where given.
	SetGitIdentity(ctx context.Context, computerID, name, email string) error
}

// GitIdentities reads the git identity a person's GitHub sign-in gives them.
type GitIdentities interface {
	// GitHubIdentity is userID's GitHub name and no-reply email, ErrNotFound when they have no GitHub sign-in.
	GitHubIdentity(ctx context.Context, userID string) (GitIdentity, error)
}

// GitIdentity is the user.name and user.email git commits on a computer carry.
type GitIdentity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// installableTools are the tools Nexul installs on a computer: the ones it runs there itself.
var installableTools = []string{"git", "gh"}

// gitMissing matches what T3 Code says when the computer has no git to clone with.
var gitMissing = regexp.MustCompile(`(?i)spawn git ENOENT|git: (command )?not found|'git' is not recognized|executable file not found`)

// MissingToolError is a step on a computer that failed because a tool Nexul needs is not installed there.
type MissingToolError struct {
	Tool       string
	ComputerID string
	Computer   string
}

func (e *MissingToolError) Error() string {
	if e.Computer == "" {
		return e.Tool + " isn't installed on the computer"
	}
	return fmt.Sprintf("%s isn't installed on %s; install it from the computer's row in Settings → Computers", e.Tool, e.Computer)
}

func (e *MissingToolError) Unwrap() error { return apperrs.ErrConflict }

// MissingToolDetails is a MissingToolError's machine-readable side, so the web can offer to install the tool.
type MissingToolDetails struct {
	MissingTool string `json:"missing_tool"`
	ComputerID  string `json:"computer_id"`
}

// ErrorDetails satisfies httpx.DetailedError.
func (e *MissingToolError) ErrorDetails() any {
	return MissingToolDetails{MissingTool: e.Tool, ComputerID: e.ComputerID}
}

// onComputer names c on a missing tool err carries, so the person reads which computer lacks it; any other err is as is.
func onComputer(err error, c Computer) error {
	var missing *MissingToolError
	if !errors.As(err, &missing) {
		return err
	}
	return &MissingToolError{Tool: missing.Tool, ComputerID: c.ID, Computer: c.Name}
}

// InstallTool installs git or gh on one of the caller's own computers with the command its facts showed them.
func (s *Service) InstallTool(ctx context.Context, userID, computerID, tool string) error {
	if !slices.Contains(installableTools, tool) {
		return fmt.Errorf("%w: tool must be git or gh", apperrs.ErrInvalid)
	}
	c, err := s.toolsComputer(ctx, userID, computerID)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(c.Facts.Tools, func(t ToolFacts) bool { return t.Name == tool })
	if i < 0 {
		return fmt.Errorf("%w: %s's Nexul app hasn't reported its tools yet", apperrs.ErrConflict, c.Name)
	}
	if c.Facts.Tools[i].Path != "" {
		return fmt.Errorf("%w: %s is already installed on %s", apperrs.ErrConflict, tool, c.Name)
	}
	if len(c.Facts.Tools[i].Install) == 0 {
		return fmt.Errorf("%w: %s has no package manager Nexul installs %s with; install it yourself", apperrs.ErrConflict, c.Name, tool)
	}
	if err := s.tools.InstallTool(ctx, c.ID, tool); err != nil {
		return fmt.Errorf("install %s on %s: %w", tool, c.Name, err)
	}
	return nil
}

// GitHubGitIdentity is the git identity the caller's GitHub sign-in gives them, which a computer without one can take.
func (s *Service) GitHubGitIdentity(ctx context.Context, userID string) (GitIdentity, error) {
	if s.identities == nil {
		return GitIdentity{}, apperrs.Fatal(fmt.Errorf("%w: git identities are not wired", apperrs.ErrFatal))
	}
	id, err := s.identities.GitHubIdentity(ctx, userID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return GitIdentity{}, fmt.Errorf("%w: you have no GitHub sign-in to take a git identity from", apperrs.ErrConflict)
	}
	if err != nil {
		return GitIdentity{}, fmt.Errorf("read your GitHub identity: %w", err)
	}
	return id, nil
}

// SetGitIdentity gives one of the caller's own computers the git name and email of their GitHub sign-in, where git
// has none; it never replaces one already set, and returns what it set.
func (s *Service) SetGitIdentity(ctx context.Context, userID, computerID string) (GitIdentity, error) {
	c, err := s.toolsComputer(ctx, userID, computerID)
	if err != nil {
		return GitIdentity{}, err
	}
	if i := slices.IndexFunc(c.Facts.Tools, func(t ToolFacts) bool { return t.Name == "git" }); i >= 0 && c.Facts.Tools[i].Path == "" {
		return GitIdentity{}, &MissingToolError{Tool: "git", ComputerID: c.ID, Computer: c.Name}
	}
	if c.Facts.GitName != "" && c.Facts.GitEmail != "" {
		return GitIdentity{}, fmt.Errorf("%w: git on %s already has a name and email", apperrs.ErrConflict, c.Name)
	}
	id, err := s.GitHubGitIdentity(ctx, userID)
	if err != nil {
		return GitIdentity{}, err
	}
	set := GitIdentity{}
	if c.Facts.GitName == "" {
		set.Name = id.Name
	}
	if c.Facts.GitEmail == "" {
		set.Email = id.Email
	}
	if err := s.tools.SetGitIdentity(ctx, c.ID, set.Name, set.Email); err != nil {
		return GitIdentity{}, onComputer(fmt.Errorf("set git's identity on %s: %w", c.Name, err), *c)
	}
	return set, nil
}

// toolsComputer is a computer of the caller's own whose runner is connected and has reported its facts.
func (s *Service) toolsComputer(ctx context.Context, userID, computerID string) (*Computer, error) {
	c, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if s.tools == nil || s.runners == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: computer tools are not wired", apperrs.ErrFatal))
	}
	runner, err := s.runners.ComputerRunner(ctx, c.ID)
	if errors.Is(err, apperrs.ErrNotFound) || (err == nil && !runner.Connected) {
		return nil, fmt.Errorf("%w: %s is offline, so Nexul can't reach it", apperrs.ErrConflict, c.Name)
	}
	if err != nil {
		return nil, fmt.Errorf("read the runner of computer %s: %w", c.ID, err)
	}
	if c.Facts == nil {
		return nil, fmt.Errorf("%w: %s's Nexul app hasn't reported on it yet", apperrs.ErrConflict, c.Name)
	}
	return c, nil
}
