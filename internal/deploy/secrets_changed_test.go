package deploy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func secretsReport(machine string, changes ...secretChange) secretsChangedReport {
	return secretsChangedReport{Machine: machine, Changes: changes}
}

func (e *outageEnv) report(t *testing.T, r secretsChangedReport) {
	t.Helper()
	require.NoError(t, e.svc.HandleSecretsChanged(t.Context(), busEvent(t, TopicRunnerSecretsChanged, r)))
}

func (e *outageEnv) announced() []SecretsChangedEvent {
	var out []SecretsChangedEvent
	for _, evt := range e.stacks.outbox {
		if p, ok := evt.Payload.(SecretsChangedEvent); ok {
			out = append(out, p)
		}
	}
	return out
}

func (e *outageEnv) notice(t *testing.T) []string {
	t.Helper()
	stack, err := e.stacks.GetByID(t.Context(), outageStack)
	require.NoError(t, err)
	return stack.SecretsChanged
}

// A runner's finding becomes the stack's notice; a variable is announced when it first joins the notice, and a
// report that adds nothing announces nothing.
func TestHandleSecretsChanged_KeepsTheNoticeAndAnnouncesEachNewVariableOnce(t *testing.T) {
	env := newOutageEnv(t)

	env.report(t, secretsReport(outageMachine, secretChange{Slug: "api", Variables: []string{"TOKEN", "DB_PASSWORD"}}))
	assert.Equal(t, []string{"DB_PASSWORD", "TOKEN"}, env.notice(t))
	assert.Equal(t, []SecretsChangedEvent{
		{ProjectID: "proj-1", StackID: outageStack, StackName: "api", Variables: []string{"DB_PASSWORD", "TOKEN"}},
	}, env.announced())

	env.report(t, secretsReport(outageMachine, secretChange{Slug: "api", Variables: []string{"TOKEN"}}))
	assert.Len(t, env.announced(), 1, "the same variable again is not news")

	env.report(t, secretsReport(outageMachine, secretChange{Slug: "api", Variables: []string{"API_KEY"}}))
	assert.Equal(t, []string{"API_KEY", "DB_PASSWORD", "TOKEN"}, env.notice(t))
	require.Len(t, env.announced(), 2)
	assert.Equal(t, []string{"API_KEY", "DB_PASSWORD", "TOKEN"}, env.announced()[1].Variables, "the event names every variable of the notice")
}

func TestHandleSecretsChanged_IgnoresWhatItCannotPlace(t *testing.T) {
	t.Run("a stack this machine does not have, or another machine's", func(t *testing.T) {
		env := newOutageEnv(t)
		env.report(t, secretsReport(outageMachine, secretChange{Slug: "ghost", Variables: []string{"A"}}))
		env.report(t, secretsReport("10.0.0.2:22", secretChange{Slug: "api", Variables: []string{"A"}}))
		env.report(t, secretsReport("", secretChange{Slug: "api", Variables: []string{"A"}}))
		assert.Empty(t, env.notice(t))
		assert.Empty(t, env.announced())
	})
	t.Run("a stack with a deploy under way, which resolves the secrets afresh", func(t *testing.T) {
		env := newOutageEnv(t)
		seedDeploy(t, env.deploys, &Deploy{ID: "d1", StackID: outageStack, Status: StatusRunning})
		env.report(t, secretsReport(outageMachine, secretChange{Slug: "api", Variables: []string{"A"}}))
		assert.Empty(t, env.notice(t))
		assert.Empty(t, env.announced())
	})
	t.Run("an undecodable report is fatal", func(t *testing.T) {
		env := newOutageEnv(t)
		err := env.svc.HandleSecretsChanged(t.Context(), eventbus.Event{Topic: TopicRunnerSecretsChanged, Payload: []byte(`{`)})
		assert.ErrorIs(t, err, apperrs.ErrFatal)
	})
}

// A deploy, retry or rollback that comes up healthy resolved every secret again, so the notice ends with it; one that
// failed changed nothing.
func TestHandleStatusChanged_AHealthyDeployClearsTheSecretsNotice(t *testing.T) {
	env := newOutageEnv(t)
	env.report(t, secretsReport(outageMachine, secretChange{Slug: "api", Variables: []string{"DB_PASSWORD"}}))
	seedDeploy(t, env.deploys, &Deploy{ID: "d1", StackID: outageStack, Status: StatusPending})
	before := len(env.stacks.outbox)

	require.NoError(t, env.svc.HandleStatusChanged(t.Context(), busEvent(t, "deploy.status_changed", DeployStatusChangedEvent{ID: "d1", Status: string(StatusFailed), Error: "boom"})))
	assert.Equal(t, []string{"DB_PASSWORD"}, env.notice(t), "a failed deploy leaves the notice")

	seedDeploy(t, env.deploys, &Deploy{ID: "d2", StackID: outageStack, Status: StatusPending})
	require.NoError(t, env.svc.HandleStatusChanged(t.Context(), busEvent(t, "deploy.status_changed", DeployStatusChangedEvent{ID: "d2", Status: string(StatusHealthy)})))
	assert.Empty(t, env.notice(t))
	require.Len(t, env.stacks.outbox, before+1)
	cleared, ok := env.stacks.outbox[before].Payload.(StackEvent)
	require.True(t, ok, "clients that follow the stack hear that the notice ended")
	assert.Equal(t, TopicStackUpdated, env.stacks.outbox[before].Topic)
	assert.Empty(t, cleared.Stack.SecretsChanged)

	seedDeploy(t, env.deploys, &Deploy{ID: "d3", StackID: outageStack, Status: StatusPending})
	require.NoError(t, env.svc.HandleStatusChanged(t.Context(), busEvent(t, "deploy.status_changed", DeployStatusChangedEvent{ID: "d3", Status: string(StatusHealthy)})))
	assert.Len(t, env.stacks.outbox, before+1, "a stack with no notice is not written to again")
}
