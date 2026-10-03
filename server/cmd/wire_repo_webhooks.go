package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/workspace"
)

// githubWebhookSecret derives the repository webhook secret, so the auth secret that signs sessions never leaves the instance.
func githubWebhookSecret(authSecret string) string {
	return hex.EncodeToString(crypto.DeriveKey("nexul github webhook secret:" + authSecret))
}

// repoWebhooks keeps this instance's /hooks/github webhook registered on the repositories attached to projects.
type repoWebhooks struct {
	git         gitprovider.GitProvider
	instanceURL func(ctx context.Context) (string, error)
	secret      string
}

func (h repoWebhooks) hookURL(ctx context.Context) (string, error) {
	base, err := h.instanceURL(ctx)
	if err != nil {
		return "", fmt.Errorf("read instance url: %w", err)
	}
	if base == "" {
		return "", errors.New("instance url is not set yet")
	}
	return strings.TrimRight(base, "/") + "/hooks/github", nil
}

// ensure registers the webhook unless one with this instance's URL already exists; another instance's hook is left alone.
func (h repoWebhooks) ensure(ctx context.Context, owner, name string) error {
	url, err := h.hookURL(ctx)
	if err != nil {
		return err
	}
	hooks, err := h.git.ListWebhooks(ctx, owner, name)
	if err != nil {
		return err
	}
	for _, hook := range hooks {
		if hook.URL == url {
			return nil
		}
	}
	_, err = h.git.CreateWebhook(ctx, owner, name, gitprovider.WebhookConfig{URL: url, Secret: h.secret})
	return err
}

// remove deletes only this instance's webhook from the repository.
func (h repoWebhooks) remove(ctx context.Context, owner, name string) error {
	url, err := h.hookURL(ctx)
	if err != nil {
		return err
	}
	hooks, err := h.git.ListWebhooks(ctx, owner, name)
	if err != nil {
		return err
	}
	for _, hook := range hooks {
		if hook.URL != url {
			continue
		}
		if err := h.git.DeleteWebhook(ctx, owner, name, hook.ID); err != nil {
			return err
		}
	}
	return nil
}

// ensureAll registers the webhook on every attached repository, covering repos attached before this existed.
func (h repoWebhooks) ensureAll(ctx context.Context, list func(context.Context) ([]workspace.RepoRef, error), logger *slog.Logger) {
	repos, err := list(ctx)
	if err != nil {
		logger.Warn("repo webhooks: list attached repos failed", "error", err)
		return
	}
	for _, r := range repos {
		if err := h.ensure(ctx, r.Owner, r.Name); err != nil {
			logger.Warn("repo webhooks: register failed", "repo", r.FullName, "error", err)
		}
	}
}

// hookedProjects wraps every repo attach and detach path; a webhook failure is logged, never fails the attach.
type hookedProjects struct {
	workspace.Repo
	hooks repoWebhooks
}

func (p hookedProjects) AddRepo(ctx context.Context, projectID string, r workspace.RepoRef) error {
	if err := p.Repo.AddRepo(ctx, projectID, r); err != nil {
		return err
	}
	if err := p.hooks.ensure(ctx, r.Owner, r.Name); err != nil {
		logging.FromCtx(ctx).Warn("repo webhooks: register failed", "repo", r.Owner+"/"+r.Name, "error", err)
	}
	return nil
}

// RemoveRepo deletes the webhook first, while the attachment still resolves which connector reaches the repo.
func (p hookedProjects) RemoveRepo(ctx context.Context, owner, name string) error {
	if err := p.hooks.remove(ctx, owner, name); err != nil {
		logging.FromCtx(ctx).Warn("repo webhooks: remove failed", "repo", owner+"/"+name, "error", err)
	}
	return p.Repo.RemoveRepo(ctx, owner, name)
}
