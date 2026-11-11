package storage

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
)

func storedStackSecrets(t *testing.T, db *sql.DB, id string) (env, rules string) {
	t.Helper()
	require.NoError(t, db.QueryRow(`SELECT env, branch_deploy_rules FROM stacks WHERE id = ?`, id).Scan(&env, &rules))
	return env, rules
}

// TestStacksRepo_EnvAndRuleOverrides_AreSealedAtRest: neither the stack's env values nor its rules' override values
// reach the column, on create or update, nor a backup of the database, yet a read gives them back.
func TestStacksRepo_EnvAndRuleOverrides_AreSealedAtRest(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	stack := newTestStack("svc-1")
	stack.Env = map[string]string{"DATABASE_URL": "postgres://hunter2@db"}
	stack.BranchDeployRules = []deploy.BranchDeployRule{{Pattern: "feature/*", DockerNetwork: "qa-net", Overrides: map[string]string{"API_KEY": "override-s3cret"}}}
	require.NoError(t, s.Stacks.Create(ctx, stack))
	stack.Env["TOKEN"] = "updated-t0ken"
	require.NoError(t, s.Stacks.Update(ctx, stack))

	backup, err := Backup(s.db, t.TempDir(), "test")
	require.NoError(t, err)
	copied, err := OpenDB(backup)
	require.NoError(t, err)
	t.Cleanup(func() { _ = copied.Close() })

	for _, db := range []*sql.DB{s.db, copied} {
		env, rules := storedStackSecrets(t, db, "svc-1")
		for _, plain := range []string{"hunter2", "override-s3cret", "updated-t0ken"} {
			assert.NotContains(t, env+rules, plain)
		}
	}
	got, err := s.Stacks.GetByID(ctx, "svc-1")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"DATABASE_URL": "postgres://hunter2@db", "TOKEN": "updated-t0ken"}, got.Env)
	assert.Equal(t, map[string]string{"API_KEY": "override-s3cret"}, got.BranchDeployRules[0].Overrides)
}

// TestStacksRepo_SealStackSecrets_SealsRowsFromBeforeTheUpgradeOnce: a row stored as plain JSON still reads back
// whole, is sealed by the boot pass, and a second pass leaves it untouched.
func TestStacksRepo_SealStackSecrets_SealsRowsFromBeforeTheUpgradeOnce(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	require.NoError(t, s.Stacks.Create(ctx, newTestStack("svc-1")))
	_, err := s.db.Exec(`UPDATE stacks SET env = '{"PORT":"8080","SECRET":"legacy-s3cret"}',
		branch_deploy_rules = '[{"pattern":"feature/*","docker_network":"qa-net","overrides":{"K":"legacy-override"}}]' WHERE id = 'svc-1'`)
	require.NoError(t, err)

	before, err := s.Stacks.GetByID(ctx, "svc-1")
	require.NoError(t, err)
	require.NoError(t, s.Stacks.SealStackSecrets(ctx))
	env, rules := storedStackSecrets(t, s.db, "svc-1")
	assert.NotContains(t, env+rules, "legacy-")

	require.NoError(t, s.Stacks.SealStackSecrets(ctx))
	env2, rules2 := storedStackSecrets(t, s.db, "svc-1")
	assert.Equal(t, []string{env, rules}, []string{env2, rules2})
	after, err := s.Stacks.GetByID(ctx, "svc-1")
	require.NoError(t, err)
	assert.Equal(t, before, after)
	assert.Equal(t, "legacy-s3cret", after.Env["SECRET"])
	assert.Equal(t, "legacy-override", after.BranchDeployRules[0].Overrides["K"])
}
