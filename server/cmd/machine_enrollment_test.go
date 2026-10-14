package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// mintRunnerCode asks for a runner enrollment code as token's person and returns the response.
func mintRunnerCode(t *testing.T, c privacyCast, token, runnerName, machine string) (code string, status int, body string) {
	t.Helper()
	rec := callAPI(t, c.h, http.MethodPost, "/api/runners/enrollments", token, `{"name":"`+runnerName+`","machine":"`+machine+`"}`)
	var got struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &got) // an error response has no code to read
	return got.Code, rec.Code, rec.Body.String()
}

// enrollRunner trades code for a runner as its installer would, with no sign-in.
func enrollRunner(t *testing.T, c privacyCast, code, runnerName string) (status int, body string) {
	t.Helper()
	rec := callAPI(t, c.h, http.MethodPost, "/api/runners/enroll", "", `{"code":"`+code+`","name":"`+runnerName+`"}`)
	return rec.Code, rec.Body.String()
}

// enrollmentCast is edge-box and core-box with carol on edge, and the instance URL set so a code can be minted.
func enrollmentCast(t *testing.T) privacyCast {
	t.Helper()
	c, db := newPrivacyCast(t)
	seedEdgeAndCore(t, db)
	_, err := c.store.Settings.Set(t.Context(), "https://nexul.example.com")
	require.NoError(t, err)
	return c
}

// TestEnrollment_ANewMachineGivesItsAdderEveryAction: a runner enrolled under a machine name nobody has makes the
// machine and gives the person who made the code a rule with See, Deploy to, Manage and Delete, so the machine they
// added is not one they lose sight of; the instance Owner needs no rule and a member who may not add machines gets none.
func TestEnrollment_ANewMachineGivesItsAdderEveryAction(t *testing.T) {
	c := enrollmentCast(t)

	_, status, body := mintRunnerCode(t, c, c.alice, "fresh-runner", "fresh-box")
	assert.Equal(t, http.StatusForbidden, status, "a member without Add machines: %s", body)

	code, status, body := mintRunnerCode(t, c, c.everyBits, "fresh-runner", "fresh-box")
	require.Equal(t, http.StatusCreated, status, body)
	status, body = enrollRunner(t, c, code, "fresh-runner")
	require.Equal(t, http.StatusCreated, status, body)

	m, err := c.store.Machines.GetByName(t.Context(), "fresh-box")
	require.NoError(t, err)
	held, err := c.svc.accessSvc.MachinePermissions(t.Context(), "u-carol", m.ID)
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.MachineActions...), held, "all four actions, from the rule written with the machine")
	var rules int
	require.NoError(t, c.db.QueryRow(`SELECT COUNT(*) FROM permission_overwrites WHERE resource_type = 'machine' AND resource_id = ?`, m.ID).Scan(&rules))
	assert.Equal(t, 1, rules, "one rule, naming the person who made the code")

	assert.Contains(t, callAPI(t, c.h, http.MethodGet, "/api/machines", c.everyBits, "").Body.String(), "fresh-box")
	assert.NotContains(t, callAPI(t, c.h, http.MethodGet, "/api/machines", c.alice, "").Body.String(), "fresh-box", "nobody else is on it")
	assert.Contains(t, callAPI(t, c.h, http.MethodGet, "/api/machines", c.owner, "").Body.String(), "fresh-box", "the instance Owner sees every machine")
}

// TestEnrollment_AnExistingMachineTakesManage: a code for a runner on a machine that exists takes Manage on it, not just
// seeing it, and a machine the person can't see is not found; neither adds a machine, so Add machines doesn't help.
func TestEnrollment_AnExistingMachineTakesManage(t *testing.T) {
	c := enrollmentCast(t)
	_, err := c.db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at)
VALUES ('machine', 'm-edge', 'person', 'u-alice', '["machines:see"]', '[]', 1, 1)`)
	require.NoError(t, err)

	_, status, body := mintRunnerCode(t, c, c.alice, "edge-two", "edge-box")
	assert.Equal(t, http.StatusForbidden, status, "seeing the machine is not managing it: %s", body)
	_, status, body = mintRunnerCode(t, c, c.alice, "core-two", "core-box")
	assert.Equal(t, http.StatusNotFound, status, "a machine they can't see: %s", body)

	_, err = c.db.Exec(`UPDATE permission_overwrites SET allow = '["machines:see","machines:write"]' WHERE resource_id = 'm-edge' AND user_id = 'u-alice'`)
	require.NoError(t, err)
	code, status, body := mintRunnerCode(t, c, c.alice, "edge-two", "edge-box")
	require.Equal(t, http.StatusCreated, status, "Manage on the machine: %s", body)
	status, body = enrollRunner(t, c, code, "edge-two")
	require.Equal(t, http.StatusCreated, status, body)

	r, err := c.store.Runners.GetByName(t.Context(), "edge-two")
	require.NoError(t, err)
	assert.Equal(t, "m-edge", r.MachineID, "the runner joined the machine that was there")
	var machines int
	require.NoError(t, c.db.QueryRow(`SELECT COUNT(*) FROM machines`).Scan(&machines))
	assert.Equal(t, 2, machines)
}

// TestEnrollment_ACodeCantLandOnAMachineTakenSince: a code made for a machine name nobody had is refused once someone
// else's machine bears that name, so it can't put a runner on a machine its maker doesn't manage; nothing is enrolled.
func TestEnrollment_ACodeCantLandOnAMachineTakenSince(t *testing.T) {
	c := enrollmentCast(t)
	code, status, body := mintRunnerCode(t, c, c.everyBits, "late-runner", "late-box")
	require.Equal(t, http.StatusCreated, status, body)
	_, err := c.db.Exec(`INSERT INTO machines (id, name, stack_root, reported_hostname, first_seen, last_seen) VALUES ('m-late', 'late-box', '/data/nexul', '', 1, 1)`)
	require.NoError(t, err)

	status, body = enrollRunner(t, c, code, "late-runner")
	assert.Equal(t, http.StatusConflict, status, body)
	assert.Contains(t, body, "machine_taken")
	_, err = c.store.Runners.GetByName(t.Context(), "late-runner")
	assert.ErrorIs(t, err, apperrs.ErrNotFound, "no runner landed on it")
	var rules int
	require.NoError(t, c.db.QueryRow(`SELECT COUNT(*) FROM permission_overwrites WHERE resource_id = 'm-late'`).Scan(&rules))
	assert.Zero(t, rules, "and its maker got no say over it")

	_, err = c.db.Exec(`UPDATE permission_overwrites SET allow = '["machines:see","machines:write"]' WHERE resource_id = 'm-edge' AND user_id = 'u-carol'`)
	require.NoError(t, err)
	_, err = c.db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at)
VALUES ('machine', 'm-late', 'person', 'u-carol', '["machines:see"]', '[]', 1, 1)`)
	require.NoError(t, err)
	status, body = enrollRunner(t, c, code, "late-runner")
	assert.Equal(t, http.StatusCreated, status, "once its maker manages the machine the same code lands: %s", body)
}

// TestEnrollment_ACodeOfSomeoneWhoCanNoLongerAddMachinesIsRefused: a code reads its maker's rights when it is used, so
// taking Add machines from them stops a code they made earlier from creating a machine.
func TestEnrollment_ACodeOfSomeoneWhoCanNoLongerAddMachinesIsRefused(t *testing.T) {
	c := enrollmentCast(t)
	code, status, body := mintRunnerCode(t, c, c.everyBits, "gone-runner", "gone-box")
	require.Equal(t, http.StatusCreated, status, body)
	_, err := c.db.Exec(`UPDATE roles SET permissions = '[]' WHERE id = 'role-everything'`)
	require.NoError(t, err)

	status, body = enrollRunner(t, c, code, "gone-runner")
	assert.Equal(t, http.StatusForbidden, status, body)
	assert.Contains(t, body, "code_cant_add")
	_, err = c.store.Machines.GetByName(t.Context(), "gone-box")
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
}
