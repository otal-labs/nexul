package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder/websocket"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/version"
)

const (
	// gitCloneTimeout bounds one clone on a computer; the server waits a little longer for the runner's answer.
	gitCloneTimeout = 10 * time.Minute
	// lastWithoutGitClone is the newest release whose personal runner does not answer git_clone.
	lastWithoutGitClone = "v0.3.55-beta"
)

// ErrCannotClone means the computer's runner is too old to clone for Nexul.
var ErrCannotClone = errors.New("the runner cannot clone")

// ---- server side ----

// CloneRepository has computerID's runner clone repo ("owner/name") into path with token, so a computer with no GitHub
// sign-in still clones a private repository; it returns the folder the clone landed in.
func (h *Handler) CloneRepository(ctx context.Context, computerID, repo, path, token string) (string, error) {
	c, err := h.computerConn(ctx, computerID)
	if err != nil {
		return "", err
	}
	if c.version == "" || !version.Newer(c.version, lastWithoutGitClone) {
		return "", ErrCannotClone
	}
	f, err := h.ask(ctx, c, Frame{Type: FrameGitClone, Repo: repo, Path: path, GitToken: token}, gitCloneTimeout+time.Minute)
	if err != nil {
		return "", fmt.Errorf("ask the runner of computer %s to clone %s: %w", computerID, repo, err)
	}
	if f.Error != "" {
		return "", fmt.Errorf("%w: %s", apperrs.ErrInvalid, f.Error)
	}
	return f.Path, nil
}

// ---- runner side ----

// answerForComputer answers what the server asks a personal runner to do on its computer.
func (c *Client) answerForComputer(ctx context.Context, conn *websocket.Conn, f Frame) {
	if f.Type == FrameGitClone {
		c.answerGitClone(ctx, conn, f)
		return
	}
	c.answerPairToken(ctx, conn, f.ID)
}

// answerGitClone clones a repository for the server and answers with the folder or why git failed.
func (c *Client) answerGitClone(ctx context.Context, conn *websocket.Conn, f Frame) {
	c.streams.Go(func() {
		defer c.recoverStream(f.ID)
		path, err := c.gitClone(ctx, f)
		if err != nil {
			c.log.Warn("clone failed", "runner", c.cfg.Name, "repo", f.Repo, "error", err)
			c.sendFrame(ctx, conn, Frame{Type: FrameGitCloneResult, ID: f.ID, Error: err.Error()})
			return
		}
		c.sendFrame(ctx, conn, Frame{Type: FrameGitCloneResult, ID: f.ID, Path: path})
	})
}

// gitClone passes the token through git's environment, so it is in neither the process list nor the clone's config.
func (c *Client) gitClone(ctx context.Context, f Frame) (string, error) {
	if !c.cfg.Personal {
		return "", errors.New("this runner is not a personal runner")
	}
	path, err := expandHome(f.Path)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, gitCloneTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "clone", "--quiet", "--", "https://github.com/"+f.Repo+".git", path)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=http.extraheader", "GIT_CONFIG_VALUE_0="+basicAuthHeader(f.GitToken))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone %s failed (%w): %s", f.Repo, err, lastLine(out))
	}
	return path, nil
}

// expandHome turns a leading ~ into the home folder of the person the runner runs as.
func expandHome(path string) (string, error) {
	rest, ok := strings.CutPrefix(path, "~")
	if !ok || (rest != "" && rest[0] != '/' && rest[0] != '\\') {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find the home folder: %w", err)
	}
	return filepath.Join(home, rest), nil
}
