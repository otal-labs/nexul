package main

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/welcome"
)

// newWelcomeService reads each seam from the domain that owns its rows, one statement per seam for any number of
// people (the TestStatements_ guard holds it there).
func newWelcomeService(store *storage.Store, authSvc *auth.Service) *welcome.Service {
	return welcome.NewService(store.Welcome, welcome.Seams{
		TickedSteps: store.Roles.WelcomeStepsOfPeople,
		RoleSteps:   store.Roles.WelcomeStepsOfRoles,
		RoleHolders: store.Roles.Holders,
		WorkspaceMembers: func(ctx context.Context, workspaceID string) ([]string, error) {
			members, err := store.WorkspaceMembers.ListByWorkspace(ctx, workspaceID)
			if err != nil {
				return nil, fmt.Errorf("list members of workspace %s: %w", workspaceID, err)
			}
			ids := make([]string, 0, len(members))
			for _, m := range members {
				ids = append(ids, m.UserID)
			}
			return ids, nil
		},
		GitHub:       store.GitHubUserLinks.PeopleWithWorkingLink,
		Computer:     store.Pairing.PeopleWithReadyComputer,
		InAProject:   store.Access.PeopleInAProject,
		DeployServer: store.Runners.PeopleWithConnectedDeployRunner,
		FirstRunDone: store.Users.PeopleWithFirstRunDone,
		MarkFirstRun: authSvc.CompleteFirstLogin,
	})
}

// subscribeWelcome follows everything a person's progress reads, so welcome.progress_changed reaches the dot and Team.
func subscribeWelcome(ctx context.Context, bus *inprocess.Bus, svc *welcome.Service) {
	for _, topic := range []string{
		pairing.TopicComputerPaired, pairing.TopicSetupConfirmed, pairing.TopicSetupUnconfirmed, pairing.TopicComputerRemoved,
		auth.TopicGitHubLinkChanged,
	} {
		mustSubscribe(ctx, bus, "welcome.progress", topic, "", svc.HandlePersonChanged)
	}
	for _, topic := range []string{tenancy.TopicWorkspaceMemberAdded, tenancy.TopicWorkspaceMemberUpdated, tenancy.TopicWorkspaceMemberRemoved} {
		mustSubscribe(ctx, bus, "welcome.progress", topic, "", svc.HandleMemberChanged)
	}
	mustSubscribe(ctx, bus, "welcome.progress", runner.TopicRunnerConnected, "", svc.HandleRunnerChanged)
	mustSubscribe(ctx, bus, "welcome.progress", runner.TopicRunnerDisconnected, "", svc.HandleRunnerChanged)
	mustSubscribe(ctx, bus, "welcome.progress", auth.TopicSessionCreated, "", svc.HandleSessionCreated)
	mustSubscribe(ctx, bus, "welcome.progress", roles.TopicUpdated, "", svc.HandleRoleUpdated)
	mustSubscribe(ctx, bus, "welcome.progress", access.TopicGrantChanged, "", svc.HandleGrantChanged)
}

// welcomeProgress is auth's seam onto the welcome domain, for account_get.
type welcomeProgress struct{ svc *welcome.Service }

func (w welcomeProgress) Progress(ctx context.Context, userID string) (any, error) {
	return w.svc.Progress(ctx, userID)
}
