package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/welcome"
)

const uLena = "u-lena"

// withNewcomer adds lena to the default workspace on a role giving every welcome step, after her first run.
func withNewcomer(t *testing.T, f permFixture) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: uLena, Provider: auth.ProviderGitHub, ProviderUserID: uLena, Login: "lena"})
	require.NoError(t, err)
	require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-newcomer", WorkspaceID: wsDefault, Name: "Newcomer",
		WelcomeSteps: []string{"github", "computer", "projects", "deploy_server", "phone"}, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: uLena, WorkspaceID: wsDefault, RoleID: "role-newcomer", CreatedAt: now}))
	require.NoError(t, f.store.Users.MarkFirstLoginDone(ctx, uLena))
}

func stepStates(t *testing.T, f permFixture, userID string) map[string]welcome.State {
	t.Helper()
	p, err := f.svc.welcomeSvc.Progress(context.Background(), userID)
	require.NoError(t, err)
	out := map[string]welcome.State{}
	for _, s := range p.Steps {
		out[s.Key] = s.State
	}
	return out
}

func saveComputer(t *testing.T, f permFixture, id string, tunnel bool, confirmed bool) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	c := pairing.Computer{ID: id, UserID: uLena, Kind: "t3code-v2", Name: id, ServerURL: "http://" + id + ".nexul-computer.invalid",
		BearerToken: "sealed", TokenExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now}
	if tunnel {
		c.Tunnel = &pairing.ComputerTunnel{TunnelID: "tun-" + id, Hostname: id + ".example.com"}
	}
	require.NoError(t, f.store.Pairing.SaveComputer(ctx, c))
	if confirmed {
		require.NoError(t, f.store.Pairing.SetSetupConfirmedAt(ctx, uLena, id, &now, eventbus.OutboxEvent{ID: "evt-" + id, Topic: pairing.TopicSetupConfirmed, Payload: map[string]string{}}))
	}
}

// TestWelcome_EachStepIsDoneWhileWhatItSetsUpExists reads each step's done rule from the real tables the seams read.
func TestWelcome_EachStepIsDoneWhileWhatItSetsUpExists(t *testing.T) {
	t.Parallel()
	now := time.Now()
	tests := []struct {
		name  string
		setup func(t *testing.T, f permFixture)
		step  string
		want  welcome.State
	}{
		{"nothing set up leaves each step left", func(*testing.T, permFixture) {}, "github", welcome.Left},
		{"a working GitHub link", func(t *testing.T, f permFixture) {
			require.NoError(t, f.store.GitHubUserLinks.SaveGitHubLink(context.Background(), auth.GitHubLink{UserID: uLena, AccessToken: "t", ConnectedAt: now}))
		}, "github", welcome.Done},
		{"a GitHub link GitHub revoked", func(t *testing.T, f permFixture) {
			require.NoError(t, f.store.GitHubUserLinks.SaveGitHubLink(context.Background(), auth.GitHubLink{UserID: uLena, NeedsReconnect: true, ConnectedAt: now}))
		}, "github", welcome.Left},
		{"a computer paired by URL and confirmed", func(t *testing.T, f permFixture) { saveComputer(t, f, "c-laptop", false, true) }, "computer", welcome.Done},
		{"a tunnel-paired computer once confirmed", func(t *testing.T, f permFixture) { saveComputer(t, f, "c-tunnel", true, true) }, "computer", welcome.Done},
		{"a paired computer not confirmed yet", func(t *testing.T, f permFixture) { saveComputer(t, f, "c-laptop", false, false) }, "computer", welcome.Left},
		{"the last ready computer removed", func(t *testing.T, f permFixture) {
			saveComputer(t, f, "c-laptop", false, true)
			require.NoError(t, f.store.Pairing.DeleteComputer(context.Background(), uLena, "c-laptop"))
		}, "computer", welcome.Left},
		{"in a project by name", func(t *testing.T, f permFixture) {
			require.NoError(t, f.store.Access.Set(context.Background(), "project", pGeneral, uLena, grant("projects:open"), nil))
		}, "projects", welcome.Done},
		{"a connected deploy runner they enrolled", func(t *testing.T, f permFixture) {
			require.NoError(t, f.store.Runners.Create(context.Background(), &runner.Runner{ID: "r-1", Name: "builder", Connected: true, EnrolledBy: uLena, LastSeen: now, CreatedAt: now}))
		}, "deploy_server", welcome.Done},
		{"a deploy runner they enrolled, gone offline", func(t *testing.T, f permFixture) {
			require.NoError(t, f.store.Runners.Create(context.Background(), &runner.Runner{ID: "r-1", Name: "builder", EnrolledBy: uLena, LastSeen: now, CreatedAt: now}))
		}, "deploy_server", welcome.Left},
		{"someone else's connected runner", func(t *testing.T, f permFixture) {
			require.NoError(t, f.store.Runners.Create(context.Background(), &runner.Runner{ID: "r-1", Name: "builder", Connected: true, EnrolledBy: uOwner, LastSeen: now, CreatedAt: now}))
		}, "deploy_server", welcome.Left},
		{"a phone that signed out after connecting", func(t *testing.T, f permFixture) {
			signIn, err := json.Marshal(auth.SessionChangedEvent{SessionID: "s-1", UserID: uLena, Client: "phone"})
			require.NoError(t, err)
			require.NoError(t, f.svc.welcomeSvc.HandleSessionCreated(context.Background(), eventbus.Event{ID: "evt-1", Topic: auth.TopicSessionCreated, Payload: signIn}))
			_, err = f.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, uLena)
			require.NoError(t, err)
		}, "phone", welcome.Done},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPermFixture(t)
			withNewcomer(t, f)
			tt.setup(t, f)
			assert.Equal(t, tt.want, stepStates(t, f, uLena)[tt.step])
		})
	}
}

// TestWelcome_InAProjectAnswersAsTheProjectCheckDoes holds the one-statement seam to RequireProject's own answer for
// every kind of member: an Owner, roles a project lists, a role none lists, no role, and a person denied by name.
func TestWelcome_InAProjectAnswersAsTheProjectCheckDoes(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	listedClient(t, f)
	unlistedMembers(t, f)
	require.NoError(t, f.svc.accessSvc.SetRules(context.Background(), uOwner, "project", []string{pGeneral, pClient}, access.Targets{UserIDs: []string{uReader}},
		[]permissions.Action{permissions.ProjectsOpen}, access.StateDeny))
	everyone := append([]string{uClient, uUnlisted, uNoRole}, people...)

	in, err := f.store.Access.PeopleInAProject(context.Background(), everyone)
	require.NoError(t, err)
	for _, id := range everyone {
		_, projects, err := f.svc.accessSvc.ProjectsAnywhere(context.Background(), id, permissions.Member)
		require.NoError(t, err)
		assert.Equal(t, len(projects) > 0, in[id], id)
	}
	assert.False(t, in[uReader], "a rule naming them keeps them out whatever their roles")
	assert.True(t, in[uClient])
	assert.False(t, in[uUnlisted])
}

// lastOutboxPayload is the newest outbox row's payload on topic, as the relay would hand it to a consumer.
func lastOutboxPayload(t *testing.T, f permFixture, topic string) eventbus.Event {
	t.Helper()
	var id string
	var payload []byte
	require.NoError(t, f.db.QueryRow(`SELECT id, payload FROM outbox WHERE topic = ? ORDER BY created_at DESC, rowid DESC LIMIT 1`, topic).Scan(&id, &payload))
	return eventbus.Event{ID: id, Topic: topic, Payload: payload}
}

// TestWelcome_AGivenStepComesBackAfterASkip: a skip lasts until the step is given again, through a tick a role they
// hold gains or a role given to them, each read from the event the real save writes.
func TestWelcome_AGivenStepComesBackAfterASkip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	t.Run("a role they hold gains the tick", func(t *testing.T) {
		f := newPermFixture(t)
		withNewcomer(t, f)
		_, err := f.svc.rolesSvc.Update(as(uOwner), wsDefault, "role-newcomer", uOwner, "Newcomer", nil, nil, []string{"github"}, nil)
		require.NoError(t, err)
		_, err = f.svc.welcomeSvc.Skip(ctx, uLena, "github")
		require.NoError(t, err)
		require.Equal(t, welcome.Skipped, stepStates(t, f, uLena)["github"])

		_, err = f.svc.rolesSvc.Update(as(uOwner), wsDefault, "role-newcomer", uOwner, "Newcomer", nil, nil, []string{"github", "projects"}, nil)
		require.NoError(t, err)
		require.NoError(t, f.svc.welcomeSvc.HandleRoleUpdated(ctx, lastOutboxPayload(t, f, roles.TopicUpdated)))
		assert.Equal(t, welcome.Skipped, stepStates(t, f, uLena)["github"], "a tick the role already had gives nothing again")

		_, err = f.svc.rolesSvc.Update(as(uOwner), wsDefault, "role-newcomer", uOwner, "Newcomer", nil, nil, []string{"projects"}, nil)
		require.NoError(t, err)
		_, err = f.svc.rolesSvc.Update(as(uOwner), wsDefault, "role-newcomer", uOwner, "Newcomer", nil, nil, []string{"github", "projects"}, nil)
		require.NoError(t, err)
		require.NoError(t, f.svc.welcomeSvc.HandleRoleUpdated(ctx, lastOutboxPayload(t, f, roles.TopicUpdated)))
		assert.Equal(t, welcome.Left, stepStates(t, f, uLena)["github"])
	})
	t.Run("they are given a role that ticks it", func(t *testing.T) {
		f := newPermFixture(t)
		withNewcomer(t, f)
		now := time.Now()
		require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-phone", WorkspaceID: wsDefault, Name: "Field", WelcomeSteps: []string{"phone"}, CreatedAt: now, UpdatedAt: now}))
		_, err := f.svc.welcomeSvc.Skip(ctx, uLena, "phone")
		require.NoError(t, err)

		require.NoError(t, f.svc.tenancySvc.UpdateMemberRoles(as(uOwner), uOwner, wsDefault, uLena, []string{"role-phone"}, nil))
		require.NoError(t, f.svc.welcomeSvc.HandleMemberChanged(ctx, lastOutboxPayload(t, f, tenancy.TopicWorkspaceMemberUpdated)))
		assert.Equal(t, welcome.Left, stepStates(t, f, uLena)["phone"])
	})
}

// TestWelcome_ProjectsLightsTheDotOnlyOnceGiven: a person in no project isn't nagged for what they can't do alone,
// until a role they hold is given the step.
func TestWelcome_ProjectsLightsTheDotOnlyOnceGiven(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := newPermFixture(t)
	withNewcomer(t, f)
	_, err := f.svc.rolesSvc.Update(as(uOwner), wsDefault, "role-newcomer", uOwner, "Newcomer", nil, nil, []string{"projects"}, nil)
	require.NoError(t, err)
	p, err := f.svc.welcomeSvc.Progress(ctx, uLena)
	require.NoError(t, err)
	assert.Equal(t, []welcome.Step{{Key: "projects", State: welcome.Left}}, p.Steps)
	assert.False(t, p.Dot)

	_, err = f.svc.rolesSvc.Update(as(uOwner), wsDefault, "role-newcomer", uOwner, "Newcomer", nil, nil, []string{}, nil)
	require.NoError(t, err)
	_, err = f.svc.rolesSvc.Update(as(uOwner), wsDefault, "role-newcomer", uOwner, "Newcomer", nil, nil, []string{"projects"}, nil)
	require.NoError(t, err)
	require.NoError(t, f.svc.welcomeSvc.HandleRoleUpdated(ctx, lastOutboxPayload(t, f, roles.TopicUpdated)))
	p, err = f.svc.welcomeSvc.Progress(ctx, uLena)
	require.NoError(t, err)
	assert.True(t, p.Dot)
}

// TestIntegration_ConfirmingAComputerPublishesTheProgressOfItsOwner runs the wired server: pairing and confirming a
// computer goes through the outbox to the welcome consumer, which writes welcome.progress_changed for its owner.
func TestIntegration_ConfirmingAComputerPublishesTheProgressOfItsOwner(t *testing.T) {
	t.Parallel()
	db := migrated.DB(t)
	_, store, svc := routerOver(t, db)
	f := permFixture{svc: svc, store: store, db: db}
	ctx := context.Background()
	_, _, err := store.Users.UpsertUser(ctx, &auth.Identity{UserID: uLena, Provider: auth.ProviderGitHub, ProviderUserID: uLena, Login: "lena"})
	require.NoError(t, err)
	saveComputer(t, f, "c-laptop", false, false)

	_, _ = svc.pairingSvc.ConfirmSetup(identity.WithActor(ctx, identity.Actor{ID: uLena}), uLena, "c-laptop") // reading the setup back asks the harness, which this test has none of
	want := fmt.Sprintf(`{"user_id":%q}`, uLena)
	require.Eventually(t, func() bool {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM outbox WHERE topic = ? AND CAST(payload AS TEXT) = ?`, welcome.TopicProgressChanged, want).Scan(&n))
		return n == 1
	}, 5*time.Second, 20*time.Millisecond, "the confirmation reaches the welcome consumer through the outbox")
}
