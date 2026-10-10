package github

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	githubapi "github.com/google/go-github/v71/github"

	"github.com/otal-labs/nexul/internal/gitprovider"
	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/githubapp"
	"github.com/otal-labs/nexul/internal/platform/logging"
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
	owners map[string]appInstallation
}

type appToken struct {
	value   string
	expires time.Time
}

// appInstallation is the installation on one account: its id and the account's.
type appInstallation struct {
	id        int64
	accountID int64
}

// NewApp builds an App signing as clientID; now defaults to time.Now.
func NewApp(clientID string, key *rsa.PrivateKey, now func() time.Time, opts ...Option) *App {
	if now == nil {
		now = time.Now
	}
	return &App{clientID: clientID, key: key, opts: opts, now: now, tokens: map[int64]appToken{}, owners: map[string]appInstallation{}}
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
	tok, err := a.mint(ctx, id, nil)
	if err != nil {
		return "", err
	}
	minted := appToken{value: tok.GetToken(), expires: tok.GetExpiresAt().Time}
	a.mu.Lock()
	a.tokens[id] = minted
	a.mu.Unlock()
	return minted.value, nil
}

func (a *App) mint(ctx context.Context, id int64, opts *githubapi.InstallationTokenOptions) (*githubapi.InstallationToken, error) {
	gh, err := a.appClient()
	if err != nil {
		return nil, err
	}
	tok, _, err := gh.Apps.CreateInstallationToken(ctx, id, opts)
	if err != nil {
		return nil, fmt.Errorf("mint a token for installation %d: %w", id, mapErr(err))
	}
	if tok.GetToken() == "" {
		return nil, fmt.Errorf("%w: GitHub returned no installation token", apperrors.ErrUnauthorized)
	}
	return tok, nil
}

// CloneToken mints a token that reads only owner/name's contents, what a runner clones with; it is never cached, so
// no other repository or permission of the installation ever reaches a runner.
func (a *App) CloneToken(ctx context.Context, owner, name string) (string, error) {
	inst, err := a.installationFor(ctx, owner, name)
	if err != nil {
		return "", err
	}
	tok, err := a.mint(ctx, inst.id, &githubapi.InstallationTokenOptions{
		Repositories: []string{name},
		Permissions:  &githubapi.InstallationPermissions{Contents: githubapi.Ptr("read")},
	})
	if err != nil {
		a.forget(owner)
		return "", err
	}
	return tok.GetToken(), nil
}

// AccountOf is the id of the account whose installation covers owner/name.
func (a *App) AccountOf(ctx context.Context, owner, name string) (int64, error) {
	inst, err := a.installationFor(ctx, owner, name)
	if err != nil {
		return 0, err
	}
	return inst.accountID, nil
}

// ForRepo is a Client reading owner/name with its installation's token.
func (a *App) ForRepo(ctx context.Context, owner, name string) (*Client, error) {
	inst, err := a.installationFor(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	tok, err := a.InstallationToken(ctx, inst.id)
	if err != nil {
		a.forget(owner)
		return nil, err
	}
	return New(tok, a.opts...), nil
}

// forget drops owner's installation, since a reinstalled App gets a new installation id on the same account.
func (a *App) forget(owner string) {
	a.mu.Lock()
	delete(a.owners, strings.ToLower(owner))
	a.mu.Unlock()
}

// installationFor finds the installation covering owner/name; an App has one per account, so it is kept per owner.
func (a *App) installationFor(ctx context.Context, owner, name string) (appInstallation, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return appInstallation{}, err
	}
	key := strings.ToLower(owner)
	a.mu.Lock()
	cached, ok := a.owners[key]
	a.mu.Unlock()
	if ok {
		return cached, nil
	}
	gh, err := a.appClient()
	if err != nil {
		return appInstallation{}, err
	}
	inst, _, err := gh.Apps.FindRepositoryInstallation(ctx, owner, name)
	if err != nil {
		return appInstallation{}, fmt.Errorf("find the App's installation on %s/%s: %w", owner, name, mapErr(err))
	}
	found := appInstallation{id: inst.GetID(), accountID: inst.GetAccount().GetID()}
	a.mu.Lock()
	a.owners[key] = found
	a.mu.Unlock()
	return found, nil
}

// eachInstallation walks every installation of the App, a page of 100 at a time.
func (a *App) eachInstallation(ctx context.Context, fn func(*githubapi.Installation)) error {
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
			fn(inst)
		}
		if resp == nil || resp.NextPage == 0 {
			return nil
		}
		opts.Page = resp.NextPage
	}
}

// suspendedProblem is what an installation its account suspended reports; GitHub refuses it every token.
const suspendedProblem = "suspended on GitHub: its repositories cannot be read until the account unsuspends the App"

// InstallationAccounts lists every installation of the App with its account, one request per hundred.
func (a *App) InstallationAccounts(ctx context.Context) ([]*gitprovider.Installation, error) {
	var out []*gitprovider.Installation
	err := a.eachInstallation(ctx, func(inst *githubapi.Installation) {
		out = append(out, toInstallation(inst))
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListInstallations implements gitprovider.GitProvider's installation read for every account the App is on. An
// installation GitHub refuses, a suspended one included, carries its problem and the others still list.
func (a *App) ListInstallations(ctx context.Context) ([]*gitprovider.Installation, error) {
	var out []*gitprovider.Installation
	err := a.eachInstallation(ctx, func(inst *githubapi.Installation) {
		i := toInstallation(inst)
		out = append(out, i)
		if inst.SuspendedAt != nil {
			i.Problem = suspendedProblem
			return
		}
		if i.RepositorySelection != "selected" {
			return
		}
		n, err := a.countRepos(ctx, inst.GetID())
		if err != nil {
			i.Problem = unreadable(ctx, inst, err)
			return
		}
		i.RepositoryCount = &n
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (a *App) countRepos(ctx context.Context, id int64) (int, error) {
	gh, err := a.installationClient(ctx, id)
	if err != nil {
		return 0, err
	}
	page, _, err := gh.Apps.ListRepos(ctx, &githubapi.ListOptions{PerPage: 1})
	if err != nil {
		return 0, fmt.Errorf("count repos for installation %d: %w", id, mapErr(err))
	}
	return page.GetTotalCount(), nil
}

// unreadable logs an installation GitHub refused, without its token, and returns the problem the owner sees.
func unreadable(ctx context.Context, inst *githubapi.Installation, err error) string {
	logging.FromCtx(ctx).Warn("github installation unreadable, skipped",
		"installation_id", inst.GetID(), "account", inst.GetAccount().GetLogin(), "error", err)
	if errors.Is(err, apperrors.ErrUnauthorized) || errors.Is(err, apperrors.ErrForbidden) {
		return "GitHub refused the App access to it; check the installation on GitHub"
	}
	return "GitHub could not read it just now; it is retried on the next refresh"
}

// ListInstallationRepos implements gitprovider.GitProvider's repository read across every installation of the App;
// a suspended or refused installation is skipped and logged, so the others still list.
func (a *App) ListInstallationRepos(ctx context.Context) ([]*gitprovider.Repo, error) {
	var out []*gitprovider.Repo
	err := a.eachInstallation(ctx, func(inst *githubapi.Installation) {
		if inst.SuspendedAt != nil {
			return
		}
		repos, err := a.installationRepos(ctx, inst)
		if err != nil {
			unreadable(ctx, inst, err)
			return
		}
		out = append(out, repos...)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (a *App) installationRepos(ctx context.Context, inst *githubapi.Installation) ([]*gitprovider.Repo, error) {
	gh, err := a.installationClient(ctx, inst.GetID())
	if err != nil {
		return nil, err
	}
	var out []*gitprovider.Repo
	opts := &githubapi.ListOptions{PerPage: 100}
	for {
		page, resp, err := gh.Apps.ListRepos(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("list repos for installation %d: %w", inst.GetID(), mapErr(err))
		}
		for _, r := range page.Repositories {
			repo := toRepo(r)
			repo.AccountID = inst.GetAccount().GetID()
			out = append(out, repo)
		}
		if resp == nil || resp.NextPage == 0 {
			return out, nil
		}
		opts.Page = resp.NextPage
	}
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
