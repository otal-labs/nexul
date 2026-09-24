package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// TestIntegration_TicketTesting runs Fail then Pass over real storage: both results land in the ticket's thread.
func TestIntegration_TicketTesting(t *testing.T) {
	ctx := context.Background()
	s := storage.New(mentionsTestDB(t), []byte("0123456789abcdef0123456789abcdef"))
	seedMentionsUser(t, s, "u-alice", false)
	aliceCtx := identity.WithActor(ctx, identity.Actor{ID: "u-alice"})
	now := time.Now().UTC()
	require.NoError(t, s.Statuses.Create(ctx, &workspace.Status{ID: "qa", ProjectID: "project-general", Name: "QA", Kind: workspace.StatusKindTesting, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Stacks.Create(ctx, &deploy.Stack{
		ID: "st-1", ProjectID: "project-general", Name: "web", Slug: "web", Machine: "m1", Strategy: deploy.StrategyRun, DockerNetwork: "prod_net",
		BuildSource:       &deploy.BuildSource{RepoOwner: "otal", RepoName: "app", Branch: "main", Dockerfile: "Dockerfile"},
		BranchDeployRules: []deploy.BranchDeployRule{{Pattern: "feature/*", DockerNetwork: "qa_net", HostnameTemplate: "{branch}.example.com", Port: 80}},
		CreatedAt:         now, UpdatedAt: now,
	}))

	chatSvc := chat.NewService(s.Chat)
	svc := tickets.NewService(s.Tickets, s.Statuses, workspaceUserStore{users: s.Users})
	svc.SetTesting(tickets.Testing{
		Stages:  ticketStages{statuses: s.Statuses},
		Threads: ticketThreads{chat: chatSvc, projects: s.Projects},
		Targets: ticketTestTargets{deploy: deploy.NewService(s.Deploys, s.Stacks, s.Services, deployProjectStore{projects: s.Projects})},
	})
	tk, err := svc.Create(ctx, "project-general", "Login page", "", "", "")
	require.NoError(t, err)
	require.NoError(t, svc.LinkBranch(ctx, tk.ID, "otal", "app", "feature/login"))
	_, err = svc.UpdateStatus(ctx, tk.ID, "qa")
	require.NoError(t, err)

	target, err := svc.TestTarget(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, tickets.TestTarget{URL: "https://login.example.com", Kind: "preview", Branch: "feature/login"}, target)

	failed, err := svc.TestFail(aliceCtx, tk.ID, tickets.TestReport{Steps: "Open /login", Actual: "Blank page"})
	require.NoError(t, err)
	assert.Equal(t, tickets.Status("in_progress"), failed.Status)
	thread, err := s.Chat.GetTicketThread(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "workspace-default", thread.WorkspaceID)
	msgs, err := s.Chat.ListMessages(ctx, thread.ID, 10)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "u-alice", msgs[0].AuthorID)
	assert.Equal(t, "Test failed\n\n## Steps to reproduce\nOpen /login\n\n## Actual result\nBlank page", msgs[0].Body)

	_, err = svc.UpdateStatus(ctx, tk.ID, "qa")
	require.NoError(t, err)
	passed, err := svc.TestPass(aliceCtx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, tickets.Status("done"), passed.Status)
	assert.Equal(t, "u-alice", passed.Tester)
	msgs, err = s.Chat.ListMessages(ctx, thread.ID, 10)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	bodies := []string{msgs[0].Body, msgs[1].Body}
	assert.Contains(t, bodies, "Passed by u-alice on https://login.example.com")

	_, err = svc.TestFail(aliceCtx, tk.ID, tickets.TestReport{Actual: "Still blank"})
	require.ErrorContains(t, err, "never reopened")
}
