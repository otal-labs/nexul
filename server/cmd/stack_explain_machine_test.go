package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// deployAnswer is userID's explain answer for deploys:write on stackID.
func deployAnswer(t *testing.T, f permFixture, userID, stackID string) access.Answer {
	t.Helper()
	explained, err := f.svc.accessSvc.Explain(as(userID), userID, "stack", stackID, "")
	require.NoError(t, err)
	for _, a := range explained.Effective {
		if a.Action == permissions.DeploysWrite {
			return a
		}
	}
	require.FailNow(t, "explain has no deploys:write answer")
	return access.Answer{}
}

// Explain and a deploy start must refuse and allow together, so the buttons never disagree with the click.
func TestMachines_ExplainRefusesADeployWithoutDeployToOnTheStacksMachine(t *testing.T) {
	f := machinesFixture(t)
	seenOnly, hidden, deployable := newStackOn(t, f, "m2"), newStackOn(t, f, "m3"), newStackOn(t, f, "m1")
	start := func(stackID string) int {
		status, _ := deployOverHTTP(t, f, uShipper, fmt.Sprintf(`{"stack_id":%q,"image":"api:1"}`, stackID))
		return status
	}

	answer := deployAnswer(t, f, uShipper, deployable.ID)
	assert.True(t, answer.Allowed, "Deploy to on m1")
	assert.Equal(t, http.StatusCreated, start(deployable.ID))

	answer = deployAnswer(t, f, uShipper, seenOnly.ID)
	assert.False(t, answer.Allowed)
	assert.False(t, answer.Fallback.Allowed)
	assert.Equal(t, "No Deploy to on machine m2", answer.Source.Sentence)
	assert.Equal(t, http.StatusForbidden, start(seenOnly.ID), "the click agrees")

	answer = deployAnswer(t, f, uShipper, hidden.ID)
	assert.False(t, answer.Allowed)
	assert.Equal(t, "No Deploy to on machine m3", answer.Source.Sentence)
	assert.Equal(t, http.StatusNotFound, start(hidden.ID), "the click agrees")

	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "machine", []string{machineM2}, access.Targets{RoleIDs: []string{"role-shipper"}}, []permissions.Action{permissions.MachinesDeploy}, access.StateAllow))
	assert.True(t, deployAnswer(t, f, uShipper, seenOnly.ID).Allowed, "a rule gives Deploy to")
	assert.Equal(t, http.StatusCreated, start(seenOnly.ID))

	assert.True(t, deployAnswer(t, f, uOwner, hidden.ID).Allowed, "the instance Owner deploys to every machine")
}

// roleDeployAnswer is the permissions panel's answer for a holder of roleID alone on deploys:write of stackID.
func roleDeployAnswer(t *testing.T, f permFixture, roleID, stackID string) access.Answer {
	t.Helper()
	explained, err := f.svc.accessSvc.ExplainRole(as(uOwner), uOwner, "stack", stackID, roleID)
	require.NoError(t, err)
	for _, a := range explained.Effective {
		if a.Action == permissions.DeploysWrite {
			return a
		}
	}
	require.FailNow(t, "explain has no deploys:write answer")
	return access.Answer{}
}

// The per-role panel must agree with a holder's own answer on the stack's machine.
func TestMachines_ExplainRoleAgreesWithAHoldersAnswerOnTheStacksMachine(t *testing.T) {
	f := machinesFixture(t)
	stacks := map[string]*deploy.Stack{"m1": newStackOn(t, f, "m1"), "m2": newStackOn(t, f, "m2"), "m3": newStackOn(t, f, "m3")}
	check := func(machine string, allowed bool, sentence string) {
		role := roleDeployAnswer(t, f, "role-shipper", stacks[machine].ID)
		person := deployAnswer(t, f, uShipper, stacks[machine].ID)
		assert.Equal(t, allowed, role.Allowed, machine)
		assert.Equal(t, allowed, role.Fallback.Allowed, machine)
		assert.Equal(t, person.Allowed, role.Allowed, "the panel agrees with the person on "+machine)
		if sentence != "" {
			assert.Equal(t, sentence, role.Source.Sentence, machine)
		}
	}
	check("m1", true, "")
	check("m2", false, "No Deploy to on machine m2")
	check("m3", false, "No Deploy to on machine m3")

	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "machine", []string{machineM2}, access.Targets{RoleIDs: []string{"role-shipper"}}, []permissions.Action{permissions.MachinesDeploy}, access.StateAllow))
	check("m2", true, "")

	owner := roleDeployAnswer(t, f, "role-owner", stacks["m3"].ID)
	assert.True(t, owner.Allowed, "the instance Owner's role deploys to every machine")
}

// Cancel takes the same Deploy to as a start, so its button locks with the stack's other deploy actions.
func TestMachines_CancellingADeployTakesDeployToOnItsMachine(t *testing.T) {
	f := machinesFixture(t)
	stack := newStackOn(t, f, "m2")
	d, err := f.svc.deploySvc.Deploy(as(uOwner), deploy.DeployRequest{StackID: stack.ID, Image: "api:1"})
	require.NoError(t, err)

	err = f.svc.deploySvc.Cancel(as(uShipper), d.ID)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "m2 is seen but not deployed to")
	assert.Contains(t, err.Error(), "machines:deploy")

	require.NoError(t, f.svc.accessSvc.SetRules(as(uOwner), uOwner, "machine", []string{machineM2}, access.Targets{RoleIDs: []string{"role-shipper"}}, []permissions.Action{permissions.MachinesDeploy}, access.StateAllow))
	require.NoError(t, f.svc.deploySvc.Cancel(as(uShipper), d.ID))
}

// newStackOn is a run stack in the default project on the named machine, made by the instance Owner.
func newStackOn(t *testing.T, f permFixture, machine string) *deploy.Stack {
	t.Helper()
	stack, _, err := f.svc.deploySvc.CreateStackWithOptions(as(uOwner), deploy.Stack{ProjectID: pGeneral, Name: "api-" + machine, Machine: machine, Strategy: deploy.StrategyRun, DockerNetwork: "edge"}, nil, deploy.CreateStackOptions{})
	require.NoError(t, err)
	return stack
}
