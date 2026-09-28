-- An instance's own stacks, its gateways, belong to no project (ADR 0079), so a stack's project becomes optional.
CREATE TABLE stacks_new (
    id                  TEXT PRIMARY KEY,
    project_id          TEXT REFERENCES projects(id),
    name                TEXT NOT NULL,
    slug                TEXT NOT NULL DEFAULT '',
    machine             TEXT NOT NULL,
    strategy            TEXT NOT NULL,
    compose_path        TEXT NOT NULL DEFAULT '',
    env                 TEXT NOT NULL DEFAULT '{}',
    docker_network      TEXT NOT NULL DEFAULT '',
    ports               TEXT NOT NULL DEFAULT '[]',
    mounts              TEXT NOT NULL DEFAULT '[]',
    command             TEXT NOT NULL DEFAULT '[]',
    build_repo_owner    TEXT NOT NULL DEFAULT '',
    build_repo_name     TEXT NOT NULL DEFAULT '',
    build_branch        TEXT NOT NULL DEFAULT '',
    build_dockerfile    TEXT NOT NULL DEFAULT '',
    build_compose_path  TEXT NOT NULL DEFAULT '',
    branch_deploy_rules TEXT NOT NULL DEFAULT '[]',
    derived_from        TEXT NOT NULL DEFAULT '',
    branch              TEXT NOT NULL DEFAULT '',
    managed             INTEGER NOT NULL DEFAULT 1,
    created_at          INTEGER NOT NULL,
    updated_at          INTEGER NOT NULL
);
INSERT INTO stacks_new (id, project_id, name, slug, machine, strategy, compose_path, env, docker_network, ports, mounts,
    command, build_repo_owner, build_repo_name, build_branch, build_dockerfile, build_compose_path, branch_deploy_rules,
    derived_from, branch, managed, created_at, updated_at)
SELECT id, project_id, name, slug, machine, strategy, compose_path, env, docker_network, ports, mounts,
    command, build_repo_owner, build_repo_name, build_branch, build_dockerfile, build_compose_path, branch_deploy_rules,
    derived_from, branch, managed, created_at, updated_at
FROM stacks;
DROP TABLE stacks;
ALTER TABLE stacks_new RENAME TO stacks;
-- Serves ListStacksByProject and the project delete impact's service count.
CREATE INDEX IF NOT EXISTS idx_stacks_project ON stacks(project_id);
-- Serves GetStackBySlugAndMachine and refuses a second stack with the same slug on a machine.
CREATE UNIQUE INDEX IF NOT EXISTS idx_stacks_machine_slug ON stacks(machine, slug);

UPDATE stacks SET project_id = NULL
WHERE id IN (SELECT service_id FROM dns_gateways) OR id IN (SELECT agent_service_id FROM dns_tunnels);

-- A project only comes from the project wizard: an instance nobody has signed in to loses the seeded one.
DELETE FROM statuses WHERE project_id = 'project-general' AND NOT EXISTS (SELECT 1 FROM users);
DELETE FROM ticket_types WHERE project_id = 'project-general' AND NOT EXISTS (SELECT 1 FROM users);
DELETE FROM memory_versions
WHERE memory_id IN (SELECT id FROM memories WHERE project_id = 'project-general') AND NOT EXISTS (SELECT 1 FROM users);
DELETE FROM memories WHERE project_id = 'project-general' AND NOT EXISTS (SELECT 1 FROM users);
DELETE FROM projects WHERE id = 'project-general' AND NOT EXISTS (SELECT 1 FROM users);
