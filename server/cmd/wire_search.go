package main

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/mentions"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

// searchPRScan caps the linked pull requests one search reads before ranking.
// ponytail: substring scan of the newest links; an FTS table over pr_title once a workspace links tens of thousands.
const searchPRScan = 200

// searchSources lists each kind the search box finds beside tickets and docs through its own domain's access checks.
func searchSources(deploySvc *deploy.Service, automationsSvc *automations.Service, chatSvc *chat.Service, memoriesSvc *memories.Service, store *storage.Store, gate projectGate) map[mentions.Kind]mentions.NamedSource {
	return map[mentions.Kind]mentions.NamedSource{
		mentions.KindPullRequest: searchPullRequests{repo: store.Tickets, gate: gate},
		mentions.KindStack:       searchStacks{svc: deploySvc},
		mentions.KindAutomation:  searchAutomations{svc: automationsSvc},
		mentions.KindChannel:     searchChannels{svc: chatSvc},
		mentions.KindMemory:      searchMemories{svc: memoriesSvc},
	}
}

// projectGate is the one access question pull request hits need: may the caller read the linked ticket's project.
type projectGate interface {
	RequireProject(ctx context.Context, projectID string, action permissions.Action) error
}

type searchPullRequests struct {
	repo *storage.TicketsRepo
	gate projectGate
}

// Find narrows by the first word in SQL and keeps a pull request once, under its most recently linked readable ticket.
func (a searchPullRequests) Find(ctx context.Context, workspaceID string, words []string) ([]mentions.Named, error) {
	links, err := a.repo.SearchPRLinks(ctx, workspaceID, words[0], searchPRScan)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := make([]mentions.Named, 0, len(links))
	for _, l := range links {
		key := fmt.Sprintf("%s/%s#%d", l.Owner, l.Repo, l.Number)
		if seen[key] || permissions.Refused(a.gate.RequireProject(ctx, l.ProjectID, permissions.TicketsRead)) {
			continue
		}
		seen[key] = true
		out = append(out, mentions.Named{
			ID: key, Title: l.Title, Hint: fmt.Sprintf("%s#%d", l.Repo, l.Number), ProjectID: l.ProjectID, TicketID: l.TicketID, UpdatedAt: l.LinkedAt,
		})
	}
	return out, nil
}

type searchStacks struct{ svc *deploy.Service }

func (a searchStacks) Find(ctx context.Context, workspaceID string, _ []string) ([]mentions.Named, error) {
	stacks, err := a.svc.ListScopedStacks(ctx, "", workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]mentions.Named, 0, len(stacks))
	for _, st := range stacks {
		out = append(out, mentions.Named{ID: st.ID, Title: st.Name, ProjectID: st.ProjectID, UpdatedAt: st.UpdatedAt})
	}
	return out, nil
}

type searchAutomations struct{ svc *automations.Service }

func (a searchAutomations) Find(ctx context.Context, workspaceID string, _ []string) ([]mentions.Named, error) {
	actor, _ := identity.ActorFromCtx(ctx)
	list, err := a.svc.List(ctx, actor.ID, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]mentions.Named, 0, len(list))
	for _, au := range list {
		out = append(out, mentions.Named{ID: au.ID, Title: au.Name, UpdatedAt: au.UpdatedAt})
	}
	return out, nil
}

type searchChannels struct{ svc *chat.Service }

func (a searchChannels) Find(ctx context.Context, workspaceID string, _ []string) ([]mentions.Named, error) {
	actor, _ := identity.ActorFromCtx(ctx)
	conversations, err := a.svc.ListConversations(ctx, workspaceID, actor.ID)
	if err != nil {
		return nil, err
	}
	out := make([]mentions.Named, 0, len(conversations))
	for _, c := range conversations {
		if c.Kind != chat.KindChannel {
			continue
		}
		out = append(out, mentions.Named{ID: c.ID, Title: c.Name, UpdatedAt: c.UpdatedAt})
	}
	return out, nil
}

type searchMemories struct{ svc *memories.Service }

func (a searchMemories) Find(ctx context.Context, workspaceID string, _ []string) ([]mentions.Named, error) {
	list, err := a.svc.List(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]mentions.Named, 0, len(list))
	for _, m := range list {
		out = append(out, mentions.Named{ID: m.ID, Title: m.Title, ProjectID: m.ProjectID, UpdatedAt: m.UpdatedAt})
	}
	return out, nil
}
