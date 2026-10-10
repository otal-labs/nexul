package github

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	githubapi "github.com/google/go-github/v71/github"

	"github.com/otal-labs/nexul/internal/gitprovider"
	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/githubapp"
)

// tokenRefreshMargin renews a token this long before it expires, so a clone never starts on one about to lapse.
const tokenRefreshMargin = 5 * time.Minute

// App reads GitHub as the GitHub App: a JWT signed with its private key mints a token per installation (ADR 0144).
type App struct {
	clientID string
	key      *rsa.PrivateKey
	opts     []Option
	now      func() time.Time

	mu     sync.Mutex
	tokens map[int64]appToken
	owners map[string]int64
}

type appToken struct {
	value   string
	expires time.Time
}

// NewApp builds an App signing as clientID; now defaults to time.Now.
func NewApp(clientID string, key *rsa.PrivateKey, now func() time.Time, opts ...Option) *App {
	if now == nil {
		now = time.Now
	}
	return &App{clientID: clientID, key: key, opts: opts, now: now, tokens: map[int64]appToken{}, owners: map[string]int64{}}
}

func (a *App) appClient() (*githubapi.Client, error) {
	jwt, err := githubapp.SignJWT(a.clientID, a.key, a.now())
	if err != nil {
		return nil, err
	}
	return New(jwt, a.opts...).gh, nil
}

// InstallationToken returns a token for installation id, minted once and reused until shortly before it expires.
func (a *App) InstallationToken(ctx context.Context, id int64) (string, error) {
	a.mu.Lock()
	cached, ok := a.tokens[id]
	a.mu.Unlock()
	if ok && a.now().Before(cached.expires.Add(-tokenRefreshMargin)) {
		return cached.value, nil
	}
	gh, err := a.appClient()
	if err != nil {
		return "", err
	}
	tok, _, err := gh.Apps.CreateInstallationToken(ctx, id, nil)
	if err != nil {
		return "", fmt.Errorf("mint a token for installation %d: %w", id, mapErr(err))
	}
	if tok.GetToken() == "" {
		return "", fmt.Errorf("%w: GitHub returned no installation token", apperrors.ErrUnauthorized)
	}
	minted := appToken{value: tok.GetToken(), expires: tok.GetExpiresAt().Time}
	a.mu.Lock()
	a.tokens[id] = minted
	a.mu.Unlock()
	return minted.value, nil
}

// RepoToken returns a token of the installation on owner/name's account, what a runner clones with.
func (a *App) RepoToken(ctx context.Context, owner, name string) (string, error) {
	id, err := a.installationFor(ctx, owner, name)
	if err != nil {
		return "", err
	}
	tok, err := a.InstallationToken(ctx, id)
	if err != nil {
		// A reinstalled App gets a new installation id on the same account; forget the old one for the next call.
		a.mu.Lock()
		delete(a.owners, strings.ToLower(owner))
		a.mu.Unlock()
		return "", err
	}
	return tok, nil
}

// ForRepo is a Client reading owner/name with its installation's token.
func (a *App) ForRepo(ctx context.Context, owner, name string) (*Client, error) {
	tok, err := a.RepoToken(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	return New(tok, a.opts...), nil
}

// installationFor finds the installation covering owner/name; an App has one per account, so it is kept per owner.
func (a *App) installationFor(ctx context.Context, owner, name string) (int64, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return 0, err
	}
	key := strings.ToLower(owner)
	a.mu.Lock()
	id, ok := a.owners[key]
	a.mu.Unlock()
	if ok {
		return id, nil
	}
	gh, err := a.appClient()
	if err != nil {
		return 0, err
	}
	inst, _, err := gh.Apps.FindRepositoryInstallation(ctx, owner, name)
	if err != nil {
		return 0, fmt.Errorf("find the App's installation on %s/%s: %w", owner, name, mapErr(err))
	}
	a.mu.Lock()
	a.owners[key] = inst.GetID()
	a.mu.Unlock()
	return inst.GetID(), nil
}

// eachInstallation walks every installation of the App, a page of 100 at a time.
func (a *App) eachInstallation(ctx context.Context, fn func(*githubapi.Installation) error) error {
	gh, err := a.appClient()
	if err != nil {
		return err
	}
	opts := &githubapi.ListOptions{PerPage: 100}
	for {
		installs, resp, err := gh.Apps.ListInstallations(ctx, opts)
		if err != nil {
			return fmt.Errorf("list the App's installations: %w", mapErr(err))
		}
		for _, inst := range installs {
			if err := fn(inst); err != nil {
				return err
			}
		}
		if resp == nil || resp.NextPage == 0 {
			return nil
		}
		opts.Page = resp.NextPage
	}
}

// ListInstallations implements gitprovider.GitProvider's installation read for every account the App is on.
func (a *App) ListInstallations(ctx context.Context) ([]*gitprovider.Installation, error) {
	var out []*gitprovider.Installation
	err := a.eachInstallation(ctx, func(inst *githubapi.Installation) error {
		i := toInstallation(inst)
		if i.RepositorySelection == "selected" {
			gh, err := a.installationClient(ctx, inst.GetID())
			if err != nil {
				return err
			}
			page, _, err := gh.Apps.ListRepos(ctx, &githubapi.ListOptions{PerPage: 1})
			if err != nil {
				return fmt.Errorf("count repos for installation %d: %w", inst.GetID(), mapErr(err))
			}
			n := page.GetTotalCount()
			i.RepositoryCount = &n
		}
		out = append(out, i)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListInstallationRepos implements gitprovider.GitProvider's repository read across every installation of the App.
func (a *App) ListInstallationRepos(ctx context.Context) ([]*gitprovider.Repo, error) {
	var out []*gitprovider.Repo
	err := a.eachInstallation(ctx, func(inst *githubapi.Installation) error {
		gh, err := a.installationClient(ctx, inst.GetID())
		if err != nil {
			return err
		}
		opts := &githubapi.ListOptions{PerPage: 100}
		for {
			page, resp, err := gh.Apps.ListRepos(ctx, opts)
			if err != nil {
				return fmt.Errorf("list repos for installation %d: %w", inst.GetID(), mapErr(err))
			}
			for _, r := range page.Repositories {
				out = append(out, toRepo(r))
			}
			if resp == nil || resp.NextPage == 0 {
				return nil
			}
			opts.Page = resp.NextPage
		}
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (a *App) installationClient(ctx context.Context, id int64) (*githubapi.Client, error) {
	tok, err := a.InstallationToken(ctx, id)
	if err != nil {
		return nil, err
	}
	return New(tok, a.opts...).gh, nil
}

// AppCache keeps one App per key, client ID and server, so its tokens outlive the request that minted them.
type AppCache struct {
	mu  sync.Mutex
	fp  [sha256.Size]byte
	app *App
}

// Get returns the App for these settings, building it on first use or after any of them changed.
func (c *AppCache) Get(clientID, privateKey, baseURL string) (*App, error) {
	fp := sha256.Sum256([]byte(clientID + "\x00" + privateKey + "\x00" + baseURL))
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.app != nil && c.fp == fp {
		return c.app, nil
	}
	key, err := githubapp.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	var opts []Option
	if baseURL != "" {
		u, err := url.Parse(baseURL)
		if err != nil {
			return nil, fmt.Errorf("%w: parse base url: %v", apperrors.ErrInvalid, err)
		}
		opts = append(opts, WithBaseURL(u))
	}
	c.app, c.fp = NewApp(clientID, key, nil, opts...), fp
	return c.app, nil
}
