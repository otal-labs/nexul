package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/coder/websocket"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/version"
)

const (
	// installTimeout bounds one install; winget may wait on the person allowing it on the computer.
	installTimeout = 10 * time.Minute
	// gitIdentityTimeout bounds setting git's identity, two git config writes.
	gitIdentityTimeout = 30 * time.Second
	// lastWithoutInstall is the newest release whose personal runner does not answer install_tool or git_identity.
	lastWithoutInstall = "v0.3.77-beta"
	// lastWithoutSudoPassword is the newest release whose personal runner ignores an install_tool's sudo password.
	lastWithoutSudoPassword = "v0.3.78-beta"
)

// ErrCannotInstall means the computer's runner is too old to install a tool or set git's identity when Nexul asks.
var ErrCannotInstall = errors.New("the runner cannot install tools on request")

// ErrCannotTakeSudoPassword means the computer's runner is too old to install with the sudo password Nexul passes on.
var ErrCannotTakeSudoPassword = errors.New("the runner cannot install with a sudo password")

// SudoPasswordError is an install sudo refused for want of the person's password: none was sent, or it was wrong.
type SudoPasswordError struct {
	Detail string
}

func (e *SudoPasswordError) Error() string { return e.Detail }

// sudoWrongPassword is what sudo -S says when the password it read was wrong.
var sudoWrongPassword = regexp.MustCompile(`(?i)incorrect password|sorry, try again`)

// MissingToolError is a command that failed because its tool is not installed on the computer.
type MissingToolError struct {
	Tool string
}

func (e *MissingToolError) Error() string {
	return e.Tool + " isn't installed on this computer"
}

// missingTool is the tool err says is not installed, "" for any other failure.
func missingTool(err error) string {
	var missing *MissingToolError
	if errors.As(err, &missing) {
		return missing.Tool
	}
	return ""
}

// packageManagers are the package managers Nexul installs a tool with on each OS, in the order it looks for them.
var packageManagers = map[string][]string{"windows": {"winget"}, "darwin": {"brew"}, "linux": {"apt-get", "dnf", "pacman"}}

// toolPackages names each tool's package per package manager.
var toolPackages = map[string]map[string]string{
	"winget":  {"git": "Git.Git", "gh": "GitHub.cli"},
	"brew":    {"git": "git", "gh": "gh"},
	"apt-get": {"git": "git", "gh": "gh"},
	"dnf":     {"git": "git", "gh": "gh"},
	"pacman":  {"git": "git", "gh": "github-cli"},
}

// sudoNeedsPassword is what sudo -n says when it would have asked for a password.
var sudoNeedsPassword = regexp.MustCompile(`(?i)a password is required`)

// installCommand is the command that installs tool on this computer with the first package manager it has, nil when it
// has none Nexul knows; the website shows it before the person agrees to run it.
func installCommand(goos, tool string, root bool) []string {
	for _, manager := range packageManagers[goos] {
		if _, err := lookTool(manager); err == nil {
			return packageCommand(manager, toolPackages[manager][tool], root)
		}
	}
	return nil
}

// packageCommand installs pkg with manager; a system package manager runs through sudo -n unless the runner is root.
func packageCommand(manager, pkg string, root bool) []string {
	if pkg == "" {
		return nil
	}
	var cmd []string
	switch manager {
	case "winget":
		return []string{"winget", "install", "--id", pkg, "--exact", "--source", "winget", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
	case "brew":
		return []string{"brew", "install", pkg}
	case "apt-get":
		cmd = []string{"apt-get", "install", "-y", pkg}
	case "dnf":
		cmd = []string{"dnf", "install", "-y", pkg}
	case "pacman":
		cmd = []string{"pacman", "-S", "--noconfirm", pkg}
	default:
		return nil
	}
	if root {
		return cmd
	}
	return append([]string{"sudo", "-n"}, cmd...)
}

// isRoot reports a runner running as root, which needs no sudo; Windows has no root and says -1.
func isRoot() bool {
	return os.Geteuid() == 0
}

// ---- server side ----

// InstallTool has computerID's runner install tool with the command its facts named, passing sudoPassword to sudo
// when given, and returns once it is installed.
func (h *Handler) InstallTool(ctx context.Context, computerID, tool, sudoPassword string) error {
	request := Frame{Type: FrameInstallTool, Tool: tool, SudoPassword: sudoPassword}
	f, err := h.askForTools(ctx, computerID, request, installTimeout+time.Minute)
	if err != nil {
		return fmt.Errorf("ask the runner of computer %s to install %s: %w", computerID, tool, err)
	}
	if f.SudoPasswordNeeded {
		return &SudoPasswordError{Detail: f.Error}
	}
	if f.Error != "" {
		return fmt.Errorf("%w: %s", apperrs.ErrConflict, f.Error)
	}
	return nil
}

// SetGitIdentity has computerID's runner set git's global user.name and user.email, each only where given.
func (h *Handler) SetGitIdentity(ctx context.Context, computerID, name, email string) error {
	f, err := h.askForTools(ctx, computerID, Frame{Type: FrameGitIdentity, GitName: name, GitEmail: email}, gitIdentityTimeout+15*time.Second)
	if err != nil {
		return fmt.Errorf("ask the runner of computer %s to set git's identity: %w", computerID, err)
	}
	if f.Error != "" {
		return fmt.Errorf("%w: %s", apperrs.ErrConflict, f.Error)
	}
	return nil
}

func (h *Handler) askForTools(ctx context.Context, computerID string, request Frame, timeout time.Duration) (Frame, error) {
	c, err := h.computerConn(ctx, computerID)
	if err != nil {
		return Frame{}, err
	}
	if c.version == "" || !version.Newer(c.version, lastWithoutInstall) {
		return Frame{}, ErrCannotInstall
	}
	if request.SudoPassword != "" && !version.Newer(c.version, lastWithoutSudoPassword) {
		return Frame{}, ErrCannotTakeSudoPassword
	}
	return h.ask(ctx, c, request, timeout)
}

// ---- runner side ----

// answerInstallTool installs a tool for the server, then reports the computer's facts again so its row shows it.
func (c *Client) answerInstallTool(ctx context.Context, conn *websocket.Conn, f Frame) {
	c.streams.Go(func() {
		defer c.recoverStream(f.ID)
		answer := Frame{Type: FrameInstallToolResult, ID: f.ID}
		if err := c.installTool(ctx, f.Tool, f.SudoPassword); err != nil {
			c.log.Warn("install failed", "runner", c.cfg.Name, "tool", f.Tool, "error", err)
			var sudo *SudoPasswordError
			answer.Error, answer.SudoPasswordNeeded = err.Error(), errors.As(err, &sudo)
		}
		c.sendFrame(ctx, conn, answer)
		c.reportFactsNow()
	})
}

// installTool runs the install command the facts showed, and checks the tool is then found on the PATH saved now.
func (c *Client) installTool(ctx context.Context, tool, sudoPassword string) error {
	if !c.cfg.Personal {
		return errors.New("this runner is not a personal runner")
	}
	if !slices.Contains(requiredTools, tool) {
		return fmt.Errorf("only %s can be installed from Nexul", strings.Join(requiredTools, " and "))
	}
	if _, err := lookTool(tool); err == nil {
		return nil
	}
	cmd := installCommand(runtime.GOOS, tool, isRoot())
	if cmd == nil {
		return fmt.Errorf("this computer has no package manager Nexul installs %s with; install it yourself", tool)
	}
	out, err := runInstall(ctx, cmd, sudoPassword)
	line := strings.Join(cmd, " ")
	if refused := sudoRefusal(tool, cmd, out, err, sudoPassword != ""); refused != nil {
		return refused
	}
	if err != nil {
		return fmt.Errorf("%s failed (%w): %s", line, err, lastLine(out))
	}
	if _, err := lookTool(tool); err != nil {
		return fmt.Errorf("%s finished, but %s still isn't on the PATH; restart the Nexul app", line, tool)
	}
	return nil
}

// runInstall runs cmd; with a sudo password, sudo reads it from stdin instead of refusing, and asks for it even when it
// remembers an earlier one, so the password never reaches the package manager's stdin. sudo speaks English for sudoRefusal.
func runInstall(ctx context.Context, cmd []string, sudoPassword string) ([]byte, error) {
	var stdin io.Reader
	if sudoPassword != "" && cmd[0] == "sudo" {
		cmd = append([]string{"sudo", "-k", "-S", "-p", ""}, cmd[2:]...)
		stdin = strings.NewReader(sudoPassword + "\n")
	}
	path, err := lookTool(cmd[0])
	if err != nil {
		return nil, fmt.Errorf("find %s: %w", cmd[0], err)
	}
	ctx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()
	run := exec.CommandContext(ctx, path, cmd[1:]...)
	run.Stdin = stdin
	run.Env = append(os.Environ(), "LC_ALL=C")
	return run.CombinedOutput()
}

// sudoRefusal is the SudoPasswordError for an install sudo refused for want of a password, nil for any other outcome.
func sudoRefusal(tool string, cmd []string, out []byte, err error, sentPassword bool) error {
	if err == nil || cmd[0] != "sudo" {
		return nil
	}
	if sentPassword && sudoWrongPassword.Match(out) {
		return &SudoPasswordError{Detail: "that sudo password was wrong"}
	}
	if !sentPassword && sudoNeedsPassword.Match(out) {
		return &SudoPasswordError{Detail: fmt.Sprintf("installing %s needs your sudo password: enter it from Install %s in Settings → Computers, or run sudo %s on the computer", tool, tool, strings.Join(cmd[2:], " "))}
	}
	return nil
}

// answerGitIdentity sets git's identity for the server, then reports the computer's facts again.
func (c *Client) answerGitIdentity(ctx context.Context, conn *websocket.Conn, f Frame) {
	c.streams.Go(func() {
		defer c.recoverStream(f.ID)
		answer := Frame{Type: FrameGitIdentityResult, ID: f.ID}
		if err := c.setGitIdentity(ctx, f.GitName, f.GitEmail); err != nil {
			c.log.Warn("git identity not set", "runner", c.cfg.Name, "error", err)
			answer.Error = err.Error()
		}
		c.sendFrame(ctx, conn, answer)
		c.reportFactsNow()
	})
}

func (c *Client) setGitIdentity(ctx context.Context, name, email string) error {
	if !c.cfg.Personal {
		return errors.New("this runner is not a personal runner")
	}
	git, err := lookTool("git")
	if err != nil {
		return &MissingToolError{Tool: "git"}
	}
	ctx, cancel := context.WithTimeout(ctx, gitIdentityTimeout)
	defer cancel()
	for _, setting := range [][2]string{{"user.name", name}, {"user.email", email}} {
		if setting[1] == "" {
			continue
		}
		if out, err := exec.CommandContext(ctx, git, "config", "--global", setting[0], setting[1]).CombinedOutput(); err != nil {
			return fmt.Errorf("git config --global %s failed (%w): %s", setting[0], err, lastLine(out))
		}
	}
	return nil
}

// reportFactsNow wakes the connection's facts loop to report at once, as a change to T3 Code does.
func (c *Client) reportFactsNow() {
	select {
	case c.t3.changed <- struct{}{}:
	default:
	}
}
