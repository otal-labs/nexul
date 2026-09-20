-- Nexul schema. One file: the product has no installs to migrate yet.

CREATE TABLE docs (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,
    version     INTEGER NOT NULL DEFAULT 1,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
, archived INTEGER NOT NULL DEFAULT 0, body_md TEXT NOT NULL DEFAULT '', project_id TEXT REFERENCES projects(id));
CREATE TABLE tickets (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL,
    doc_id      TEXT REFERENCES docs(id),
    assignee    TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
, project_id TEXT REFERENCES projects(id), category_id TEXT REFERENCES categories(id) ON DELETE SET NULL, type_id TEXT REFERENCES ticket_types(id), finished_at INTEGER, position INTEGER NOT NULL DEFAULT 0, number INTEGER NOT NULL DEFAULT 0);
CREATE INDEX idx_tickets_doc_id ON tickets(doc_id);
CREATE TABLE topology (
    environment     TEXT PRIMARY KEY,
    schema_version  INTEGER NOT NULL,
    canvas          BLOB NOT NULL,
    updated_at      INTEGER NOT NULL
);
CREATE TABLE runners (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    last_seen   INTEGER NOT NULL,
    connected   INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL
, version TEXT NOT NULL DEFAULT '', machine_id TEXT NOT NULL DEFAULT '');
CREATE TABLE outbox (
    id          TEXT PRIMARY KEY,
    topic       TEXT NOT NULL,
    payload     BLOB NOT NULL,
    created_at  INTEGER NOT NULL DEFAULT (unixepoch('now')),
    published   INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_outbox_unpublished ON outbox(published, created_at) WHERE published = 0;
CREATE TABLE processed_events (
    event_id     TEXT PRIMARY KEY,
    processed_at INTEGER NOT NULL
);
CREATE TABLE dead_letters (
    id          TEXT PRIMARY KEY,
    topic       TEXT NOT NULL,
    payload     BLOB NOT NULL,
    error       TEXT NOT NULL,
    attempts    INTEGER NOT NULL,
    created_at  INTEGER NOT NULL DEFAULT (unixepoch('now'))
);
CREATE VIRTUAL TABLE tickets_fts USING fts5(title, body, content='tickets', content_rowid='rowid')
/* tickets_fts(title,body) */;
CREATE TRIGGER tickets_fts_ai AFTER INSERT ON tickets BEGIN
    INSERT INTO tickets_fts(rowid, title, body) VALUES (new.rowid, new.title, new.body);
END;
CREATE TRIGGER tickets_fts_ad AFTER DELETE ON tickets BEGIN
    INSERT INTO tickets_fts(tickets_fts, rowid, title, body) VALUES ('delete', old.rowid, old.title, old.body);
END;
CREATE TRIGGER tickets_fts_au AFTER UPDATE ON tickets BEGIN
    INSERT INTO tickets_fts(tickets_fts, rowid, title, body) VALUES ('delete', old.rowid, old.title, old.body);
    INSERT INTO tickets_fts(rowid, title, body) VALUES (new.rowid, new.title, new.body);
END;
CREATE TABLE doc_versions (
    doc_id     TEXT NOT NULL REFERENCES docs(id) ON DELETE CASCADE,
    version    INTEGER NOT NULL,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    created_at INTEGER NOT NULL, name TEXT NOT NULL DEFAULT '', author_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (doc_id, version)
);
CREATE TABLE ticket_pr_links (
    ticket_id TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    pr_owner  TEXT NOT NULL,
    pr_repo   TEXT NOT NULL,
    pr_number INTEGER NOT NULL,
    pr_title  TEXT NOT NULL DEFAULT '',
    pr_sha    TEXT NOT NULL DEFAULT '',
    linked_at INTEGER NOT NULL, pr_state TEXT NOT NULL DEFAULT 'open',
    PRIMARY KEY (ticket_id, pr_owner, pr_repo, pr_number)
);
CREATE TABLE IF NOT EXISTS "code_reviews" (
    id          TEXT PRIMARY KEY,
    pr_number   INTEGER NOT NULL,
    repo        TEXT NOT NULL,
    status      TEXT NOT NULL,
    reviewer    TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    UNIQUE (repo, pr_number)
);
CREATE INDEX idx_code_reviews_pr ON code_reviews(repo, pr_number);
CREATE TABLE users (
    id                TEXT PRIMARY KEY,
    provider          TEXT NOT NULL,
    provider_user_id  TEXT NOT NULL,
    login             TEXT NOT NULL,
    name              TEXT NOT NULL DEFAULT '',
    avatar_url        TEXT NOT NULL DEFAULT '',
    first_login_done  INTEGER NOT NULL DEFAULT 0,
    created_at        INTEGER NOT NULL,
    updated_at        INTEGER NOT NULL, can_create_workspace INTEGER NOT NULL DEFAULT 0, display_name TEXT, avatar_override_url TEXT,
    UNIQUE (provider, provider_user_id)
);
CREATE INDEX idx_users_login ON users(login);
CREATE TABLE allowlist (
    login     TEXT PRIMARY KEY,
    added_at  INTEGER NOT NULL
);
CREATE TABLE instance_settings (
    id                INTEGER PRIMARY KEY CHECK (id = 1),
    instance_url      TEXT NOT NULL DEFAULT '',
    settings_version  INTEGER NOT NULL DEFAULT 1,
    updated_at        INTEGER NOT NULL
, github_oauth_client_id TEXT NOT NULL DEFAULT '', github_oauth_client_secret TEXT NOT NULL DEFAULT '', mention_chip_template TEXT NOT NULL DEFAULT '{ticket.Ticket} {ticket.Status}', google_oauth_client_id TEXT NOT NULL DEFAULT '', google_oauth_client_secret TEXT NOT NULL DEFAULT '', discord_oauth_client_id TEXT NOT NULL DEFAULT '', discord_oauth_client_secret TEXT NOT NULL DEFAULT '', runner_secret TEXT NOT NULL DEFAULT '');
CREATE TABLE project_repos (
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    owner       TEXT NOT NULL,
    name        TEXT NOT NULL,
    full_name   TEXT NOT NULL DEFAULT '',
    added_at    INTEGER NOT NULL, connector_id TEXT NOT NULL DEFAULT 'github',
    PRIMARY KEY (owner, name)
);
CREATE INDEX idx_project_repos_project ON project_repos(project_id, name);
CREATE INDEX idx_tickets_project ON tickets(project_id);
CREATE TABLE categories (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    position    INTEGER NOT NULL,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
, color TEXT NOT NULL DEFAULT '');
CREATE INDEX idx_categories_project ON categories(project_id, position, id);
CREATE INDEX idx_tickets_category ON tickets(category_id);
CREATE TABLE ticket_labels (
    ticket_id   TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    label       TEXT NOT NULL,
    PRIMARY KEY (ticket_id, label)
);
CREATE INDEX idx_ticket_labels_label ON ticket_labels(label);
CREATE TABLE ticket_branch_links (
    ticket_id    TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    branch_owner TEXT NOT NULL,
    branch_repo  TEXT NOT NULL,
    branch_name  TEXT NOT NULL,
    linked_at    INTEGER NOT NULL,
    PRIMARY KEY (ticket_id, branch_owner, branch_repo, branch_name)
);
CREATE VIRTUAL TABLE docs_fts USING fts5(title, body_md, content='docs', content_rowid='rowid')
/* docs_fts(title,body_md) */;
CREATE TRIGGER docs_fts_ai AFTER INSERT ON docs BEGIN
    INSERT INTO docs_fts(rowid, title, body_md) VALUES (new.rowid, new.title, new.body_md);
END;
CREATE TRIGGER docs_fts_ad AFTER DELETE ON docs BEGIN
    INSERT INTO docs_fts(docs_fts, rowid, title, body_md) VALUES ('delete', old.rowid, old.title, old.body_md);
END;
CREATE TRIGGER docs_fts_au AFTER UPDATE ON docs BEGIN
    INSERT INTO docs_fts(docs_fts, rowid, title, body_md) VALUES ('delete', old.rowid, old.title, old.body_md);
    INSERT INTO docs_fts(rowid, title, body_md) VALUES (new.rowid, new.title, new.body_md);
END;
CREATE TABLE collab_updates (
    doc_id     TEXT NOT NULL REFERENCES docs(id) ON DELETE CASCADE,
    seq        INTEGER PRIMARY KEY AUTOINCREMENT,
    kind       TEXT NOT NULL CHECK (kind IN ('update', 'snapshot')),
    actor_id   TEXT NOT NULL,
    payload    TEXT NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX idx_collab_updates_doc ON collab_updates (doc_id, seq);
CREATE TABLE collab_sessions (
    doc_id         TEXT PRIMARY KEY REFERENCES docs(id) ON DELETE CASCADE,
    last_seq       INTEGER NOT NULL DEFAULT 0,
    last_commit_at INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE notifications (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind          TEXT NOT NULL,
    subject_type  TEXT NOT NULL,
    subject_id    TEXT NOT NULL,
    subject_title TEXT NOT NULL DEFAULT '',
    read          INTEGER NOT NULL DEFAULT 0,
    created_at    INTEGER NOT NULL
);
CREATE INDEX idx_notifications_user ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, read) WHERE read = 0;
CREATE TABLE personal_access_tokens (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    prefix       TEXT NOT NULL,
    created_at   INTEGER NOT NULL,
    last_used_at INTEGER,
    revoked_at   INTEGER
);
CREATE INDEX idx_pats_user_id ON personal_access_tokens(user_id);
CREATE TABLE integration_installs (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    trust_tier     TEXT NOT NULL,
    webhook_url    TEXT NOT NULL,
    webhook_secret TEXT NOT NULL,
    scopes         TEXT NOT NULL,
    created_by     TEXT NOT NULL,
    created_at     INTEGER NOT NULL,
    revoked_at     INTEGER
);
CREATE TABLE integration_tokens (
    id           TEXT PRIMARY KEY,
    install_id   TEXT NOT NULL REFERENCES integration_installs(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL UNIQUE,
    prefix       TEXT NOT NULL,
    created_at   INTEGER NOT NULL,
    last_used_at INTEGER,
    revoked_at   INTEGER
);
CREATE INDEX idx_integration_tokens_install ON integration_tokens(install_id);
CREATE TABLE integration_subscriptions (
    install_id TEXT NOT NULL REFERENCES integration_installs(id) ON DELETE CASCADE,
    topic      TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (install_id, topic)
);
CREATE TABLE integration_deliveries (
    id              TEXT PRIMARY KEY,
    install_id      TEXT NOT NULL REFERENCES integration_installs(id) ON DELETE CASCADE,
    topic           TEXT NOT NULL,
    event_id        TEXT NOT NULL,
    payload         BLOB NOT NULL,
    signature       TEXT NOT NULL,
    url             TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    attempts        INTEGER NOT NULL DEFAULT 0,
    next_attempt_at INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL,
    delivered_at    INTEGER
);
CREATE UNIQUE INDEX idx_integration_deliveries_event ON integration_deliveries(install_id, event_id, topic);
CREATE INDEX idx_integration_deliveries_due ON integration_deliveries(status, next_attempt_at);
CREATE TABLE event_schemas (
    topic      TEXT NOT NULL,
    version    INTEGER NOT NULL,
    schema     TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (topic, version)
);
CREATE TABLE audit_log (
    id         TEXT PRIMARY KEY,
    actor_type TEXT NOT NULL,
    actor_id   TEXT NOT NULL,
    token_id   TEXT,
    action     TEXT NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX idx_audit_actor ON audit_log(actor_type, actor_id, created_at);
CREATE INDEX idx_audit_token ON audit_log(token_id);
CREATE TABLE dns_service_hostnames (
    service     TEXT PRIMARY KEY,
    hostname    TEXT NOT NULL,
    zone_id     TEXT NOT NULL,
    zone        TEXT NOT NULL,
    record_id   TEXT NOT NULL,
    type        TEXT NOT NULL,
    content     TEXT NOT NULL,
    created_at  INTEGER NOT NULL
);
CREATE INDEX idx_dns_service_hostnames_hostname ON dns_service_hostnames(hostname);
CREATE TABLE dns_tunnels (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    account_id  TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT '',
    hostname    TEXT NOT NULL DEFAULT '',
    zone_id     TEXT NOT NULL DEFAULT '',
    zone        TEXT NOT NULL DEFAULT '',
    record_id   TEXT NOT NULL DEFAULT '',
    service     TEXT NOT NULL DEFAULT '',
    agent_service_id TEXT NOT NULL DEFAULT '',
    token       TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX idx_dns_tunnels_name ON dns_tunnels(name);
CREATE INDEX idx_tickets_status_category_position ON tickets(status, category_id, position);
CREATE TABLE workspaces (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS "projects" (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    prefix       TEXT NOT NULL DEFAULT '',
    position     INTEGER NOT NULL,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
, icon TEXT NOT NULL DEFAULT '');
CREATE INDEX idx_projects_position ON projects(position, id);
CREATE UNIQUE INDEX idx_projects_prefix ON projects(prefix) WHERE prefix != '';
CREATE INDEX idx_projects_workspace ON projects(workspace_id);
CREATE TABLE roles (
    id               TEXT PRIMARY KEY,
    workspace_id     TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    is_owner_role    INTEGER NOT NULL DEFAULT 0,
    created_at       INTEGER NOT NULL,
    updated_at       INTEGER NOT NULL
, permissions TEXT NOT NULL DEFAULT '[]');
CREATE INDEX idx_roles_workspace ON roles(workspace_id);
CREATE UNIQUE INDEX idx_roles_owner_singleton ON roles(workspace_id) WHERE is_owner_role = 1;
CREATE TABLE IF NOT EXISTS "workspace_members" (
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    role_id      TEXT NOT NULL REFERENCES roles(id),
    created_at   INTEGER NOT NULL,
    PRIMARY KEY (user_id, workspace_id)
);
CREATE INDEX idx_docs_project ON docs(project_id);
CREATE TABLE connector_credentials (
    connector_id  TEXT PRIMARY KEY,
    access_token  TEXT NOT NULL,
    refresh_token TEXT NOT NULL DEFAULT '',
    expires_at    INTEGER NOT NULL DEFAULT 0,
    connected_by  TEXT NOT NULL,
    connected_at  INTEGER NOT NULL
, manual_fields TEXT NOT NULL DEFAULT '');
CREATE TABLE workspace_invites (
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    login        TEXT NOT NULL,
    role_id      TEXT NOT NULL REFERENCES roles(id),
    invited_by   TEXT NOT NULL REFERENCES users(id),
    created_at   INTEGER NOT NULL,
    PRIMARY KEY (workspace_id, login)
);
CREATE INDEX idx_workspace_invites_login ON workspace_invites(login);
CREATE TABLE statuses (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    position    INTEGER NOT NULL,
    kind        TEXT NOT NULL DEFAULT 'active',
    icon        TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX idx_statuses_project ON statuses(project_id, position, id);
CREATE TABLE connector_app_config (
    connector_id  TEXT PRIMARY KEY,
    client_id     TEXT NOT NULL DEFAULT '',
    client_secret TEXT NOT NULL DEFAULT '',
    base_url      TEXT NOT NULL DEFAULT ''
, app_slug TEXT NOT NULL DEFAULT '');
CREATE TABLE ticket_types (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    position    INTEGER NOT NULL,
    color       TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX idx_ticket_types_project ON ticket_types(project_id, position, id);
CREATE TABLE label_colors (
    project_id    TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    label         TEXT NOT NULL,
    color         TEXT NOT NULL,
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    PRIMARY KEY (project_id, label)
);
CREATE TABLE conversations (
    id                TEXT PRIMARY KEY,
    workspace_id      TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind              TEXT NOT NULL,
    name              TEXT NOT NULL DEFAULT '',
    ticket_id         TEXT REFERENCES tickets(id) ON DELETE CASCADE,
    parent_message_id TEXT NOT NULL DEFAULT '',
    created_by        TEXT NOT NULL REFERENCES users(id),
    created_at        INTEGER NOT NULL,
    updated_at        INTEGER NOT NULL
, agent_thread_id TEXT NOT NULL DEFAULT '', agent_synced_at INTEGER NOT NULL DEFAULT 0, doc_id TEXT REFERENCES docs(id) ON DELETE CASCADE);
CREATE UNIQUE INDEX idx_conversations_ticket ON conversations(ticket_id) WHERE ticket_id IS NOT NULL;
CREATE UNIQUE INDEX idx_conversations_channel_name ON conversations(workspace_id, name) WHERE kind = 'channel';
CREATE INDEX idx_conversations_workspace_kind ON conversations(workspace_id, kind);
CREATE TABLE conversation_participants (
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at      INTEGER NOT NULL,
    PRIMARY KEY (conversation_id, user_id)
);
CREATE TABLE messages (
    id              TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    author_id       TEXT NOT NULL REFERENCES users(id),
    body            TEXT NOT NULL,
    mentions        TEXT NOT NULL DEFAULT '[]',
    attachment_id   TEXT,
    edited_at       INTEGER,
    deleted_at      INTEGER,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL
, author_kind TEXT NOT NULL DEFAULT 'user');
CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at);
CREATE TABLE conversation_unread_state (
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    last_read_at    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, conversation_id)
);
CREATE TABLE pairing_computers (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    server_url       TEXT NOT NULL,
    bearer_token     TEXT NOT NULL,
    token_expires_at INTEGER NOT NULL,
    harness_version       TEXT NOT NULL,
    created_at       INTEGER NOT NULL,
    updated_at       INTEGER NOT NULL
, kind TEXT NOT NULL DEFAULT 't3code');
CREATE INDEX idx_pairing_computers_user_id ON pairing_computers(user_id);
CREATE TABLE pairing_user_defaults (
    user_id              TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    default_computer_id  TEXT REFERENCES pairing_computers(id) ON DELETE SET NULL,
    fallback_project_id  TEXT NOT NULL DEFAULT '',
    provider             TEXT NOT NULL DEFAULT '',
    model                TEXT NOT NULL DEFAULT ''
);
CREATE TABLE pairing_project_links (
    project_id    TEXT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    computer_id   TEXT NOT NULL REFERENCES pairing_computers(id) ON DELETE CASCADE,
    harness_project_id TEXT NOT NULL,
    provider      TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT '',
    updated_at    INTEGER NOT NULL
);
CREATE TABLE dns_gateways (
    id             TEXT PRIMARY KEY,
    kind           TEXT NOT NULL,
    docker_network TEXT NOT NULL UNIQUE,
    service_id     TEXT NOT NULL DEFAULT '',
    service_name   TEXT NOT NULL DEFAULT '',
    tunnel_id      TEXT NOT NULL DEFAULT '',
    zone_id        TEXT NOT NULL DEFAULT '',
    zone           TEXT NOT NULL DEFAULT '',
    server_address TEXT NOT NULL DEFAULT '',
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL
, networks TEXT NOT NULL DEFAULT '[]', machine TEXT NOT NULL DEFAULT '');
CREATE TABLE dns_exposures (
    id          TEXT PRIMARY KEY,
    gateway_id  TEXT NOT NULL REFERENCES dns_gateways(id),
    hostname    TEXT NOT NULL,
    service     TEXT NOT NULL,
    port        INTEGER NOT NULL,
    zone_id     TEXT NOT NULL DEFAULT '',
    zone        TEXT NOT NULL DEFAULT '',
    record_id   TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
, service_id TEXT);
CREATE INDEX idx_dns_exposures_gateway ON dns_exposures(gateway_id);
CREATE INDEX idx_dns_exposures_service ON dns_exposures(service);
CREATE TABLE automations (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    kind             TEXT NOT NULL,
    enabled          INTEGER NOT NULL DEFAULT 0,
    subscriptions    BLOB NOT NULL DEFAULT '[]',
    config_schema    BLOB NOT NULL DEFAULT '{}',
    config_values    BLOB NOT NULL DEFAULT '{}',
    scopes           BLOB NOT NULL DEFAULT '[]',
    token_hash       TEXT NOT NULL,
    token_prefix     TEXT NOT NULL DEFAULT '',
    token_revoked_at INTEGER,
    created_at       INTEGER NOT NULL,
    updated_at       INTEGER NOT NULL
, created_by TEXT NOT NULL DEFAULT '');
CREATE TABLE automation_cursors (
    automation_id   TEXT PRIMARY KEY,
    last_created_at INTEGER NOT NULL,
    last_event_id   TEXT NOT NULL,
    updated_at      INTEGER NOT NULL
);
CREATE TABLE automation_runs (
    id            TEXT PRIMARY KEY,
    automation_id TEXT NOT NULL,
    event_topic   TEXT NOT NULL,
    event_id      TEXT NOT NULL,
    outcome       TEXT NOT NULL,
    error         TEXT NOT NULL DEFAULT '',
    started_at    INTEGER NOT NULL,
    finished_at   INTEGER NOT NULL,
    duration_ms   INTEGER NOT NULL DEFAULT 0,
    logs          TEXT NOT NULL DEFAULT '',
    created_at    INTEGER NOT NULL
);
CREATE INDEX idx_automation_runs_automation ON automation_runs(automation_id, started_at DESC, id DESC);
CREATE INDEX idx_automation_runs_created_at ON automation_runs(created_at);
CREATE INDEX idx_outbox_created_id ON outbox(created_at, id);
CREATE TABLE automation_versions (
    id            TEXT PRIMARY KEY,
    automation_id TEXT NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    sequence      INTEGER NOT NULL,
    code          BLOB NOT NULL,
    pusher_id     TEXT NOT NULL,
    message       TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL,
    created_at    INTEGER NOT NULL
);
CREATE INDEX idx_automation_versions_list ON automation_versions(automation_id, sequence DESC, id);
CREATE UNIQUE INDEX idx_automation_versions_one_pending ON automation_versions(automation_id) WHERE status = 'pending';
CREATE UNIQUE INDEX idx_automation_versions_one_active ON automation_versions(automation_id) WHERE status = 'active';
CREATE TABLE automation_secrets (
    name       TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE TABLE stacks (
    id                  TEXT PRIMARY KEY,
    project_id          TEXT NOT NULL REFERENCES projects(id),
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
CREATE INDEX idx_stacks_project ON stacks(project_id);
CREATE UNIQUE INDEX idx_stacks_machine_slug ON stacks(machine, slug);
CREATE TABLE IF NOT EXISTS "deploys" (
    id           TEXT PRIMARY KEY,
    service      TEXT NOT NULL,
    target       TEXT NOT NULL,
    image        TEXT NOT NULL,
    status       TEXT NOT NULL,
    strategy     TEXT NOT NULL,
    log          TEXT NOT NULL DEFAULT '',
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL,
    stack_id     TEXT REFERENCES stacks(id) ON DELETE SET NULL,
    service_id   TEXT NOT NULL DEFAULT '',
    triggered_by TEXT NOT NULL DEFAULT '',
    rule_id      TEXT NOT NULL DEFAULT '',
    rule_name    TEXT NOT NULL DEFAULT '',
    ticket_id    TEXT NOT NULL DEFAULT '',
    pr_number    INTEGER NOT NULL DEFAULT 0,
    kind         TEXT NOT NULL DEFAULT 'deploy',
    address      TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_deploys_status ON deploys(status);
CREATE INDEX idx_deploys_service ON deploys(service);
CREATE INDEX idx_deploys_stack_id ON deploys(stack_id);
CREATE INDEX idx_deploys_kind ON deploys(kind);
CREATE UNIQUE INDEX idx_deploys_one_active
    ON deploys(stack_id)
    WHERE status IN ('pending', 'running') AND stack_id IS NOT NULL AND stack_id != '';
CREATE TABLE services (
    id             TEXT PRIMARY KEY,
    stack_id       TEXT NOT NULL REFERENCES stacks(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    declared       TEXT NOT NULL DEFAULT '{}',
    container_name TEXT NOT NULL DEFAULT '',
    image          TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'pending',
    networks       TEXT NOT NULL DEFAULT '[]',
    ports          TEXT NOT NULL DEFAULT '[]',
    observed_at    INTEGER NOT NULL DEFAULT 0,
    UNIQUE (stack_id, name)
);
CREATE INDEX idx_services_stack ON services(stack_id);
CREATE TABLE machines (
    id                TEXT PRIMARY KEY,
    name              TEXT NOT NULL,
    stack_root        TEXT NOT NULL DEFAULT '/data/nexul',
    reported_hostname TEXT NOT NULL DEFAULT '',
    first_seen        INTEGER NOT NULL,
    last_seen         INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_machines_name ON machines(name);
CREATE INDEX idx_runners_machine ON runners(machine_id);
CREATE INDEX idx_dns_exposures_service_id ON dns_exposures(service_id);
CREATE INDEX idx_dns_gateways_machine ON dns_gateways(machine);
CREATE TABLE permission_overwrites (
    resource_type TEXT NOT NULL,
    resource_id   TEXT NOT NULL,
    user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    allow         TEXT NOT NULL DEFAULT '[]',
    deny          TEXT NOT NULL DEFAULT '[]',
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    PRIMARY KEY (resource_type, resource_id, user_id)
);
CREATE INDEX idx_permission_overwrites_user ON permission_overwrites(user_id);
CREATE TABLE instance_upgrades (
    id           TEXT PRIMARY KEY,
    from_version TEXT NOT NULL,
    to_version   TEXT NOT NULL,
    status       TEXT NOT NULL,
    error        TEXT NOT NULL DEFAULT '',
    requested_by TEXT NOT NULL,
    runner_id    TEXT NOT NULL,
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
);
CREATE INDEX idx_instance_upgrades_created ON instance_upgrades(created_at DESC);
CREATE UNIQUE INDEX idx_conversations_doc ON conversations(doc_id) WHERE doc_id IS NOT NULL;
CREATE TABLE plays (
    id                   TEXT PRIMARY KEY,
    workspace_id         TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    label                TEXT NOT NULL,
    type                 TEXT NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    instructions         TEXT NOT NULL DEFAULT '',
    enabled              INTEGER NOT NULL DEFAULT 1,
    show_when_stage      TEXT,
    excluded_project_ids TEXT NOT NULL DEFAULT '[]',
    created_by           TEXT NOT NULL DEFAULT '',
    created_at           INTEGER NOT NULL,
    updated_at           INTEGER NOT NULL
);
CREATE INDEX idx_plays_workspace ON plays(workspace_id);
CREATE TABLE memory_versions (
    id              TEXT PRIMARY KEY,
    memory_id       TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    version         INTEGER NOT NULL,
    title           TEXT NOT NULL,
    when_to_use     TEXT NOT NULL DEFAULT '',
    body            TEXT NOT NULL,
    always_included INTEGER NOT NULL DEFAULT 0,
    author_id       TEXT NOT NULL DEFAULT '',
    author_via      TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL,
    UNIQUE (memory_id, version)
);
CREATE INDEX idx_memory_versions_memory ON memory_versions(memory_id);
CREATE TABLE play_trails (
    id                  TEXT PRIMARY KEY,
    workspace_id        TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    play_id             TEXT NOT NULL,
    play_label          TEXT NOT NULL,
    target_type         TEXT NOT NULL,
    target_id           TEXT NOT NULL,
    project_id          TEXT NOT NULL,
    conversation_id     TEXT NOT NULL DEFAULT '',
    starter_id          TEXT NOT NULL,
    via                 TEXT NOT NULL,
    selected_memory_ids TEXT NOT NULL DEFAULT '[]',
    custom_instructions TEXT NOT NULL DEFAULT '',
    move_to_status_id   TEXT,
    harness_session_id  TEXT NOT NULL DEFAULT '',
    state               TEXT NOT NULL,
    started_at          INTEGER NOT NULL,
    ended_at            INTEGER,
    last_error          TEXT NOT NULL DEFAULT '',
    reply_message_id    TEXT NOT NULL DEFAULT '',
    activity            TEXT NOT NULL DEFAULT '[]'
, computer_id TEXT NOT NULL DEFAULT '', provider TEXT NOT NULL DEFAULT '', model TEXT NOT NULL DEFAULT '', question TEXT);
CREATE INDEX idx_play_trails_target ON play_trails(target_type, target_id, started_at);
CREATE INDEX idx_play_trails_choices ON play_trails(starter_id, play_id, project_id, started_at);
CREATE TABLE IF NOT EXISTS "attachments" (
    id              TEXT PRIMARY KEY,
    doc_id          TEXT REFERENCES docs(id) ON DELETE CASCADE,
    ticket_id       TEXT REFERENCES tickets(id) ON DELETE CASCADE,
    conversation_id TEXT REFERENCES conversations(id) ON DELETE CASCADE,
    memory_id       TEXT REFERENCES memories(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    content_type    TEXT NOT NULL,
    size            INTEGER NOT NULL,
    data            BLOB NOT NULL,
    uploaded_by     TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL,
    CHECK ((doc_id IS NOT NULL) + (ticket_id IS NOT NULL) + (conversation_id IS NOT NULL) + (memory_id IS NOT NULL) = 1)
);
CREATE INDEX idx_attachments_doc ON attachments(doc_id) WHERE doc_id IS NOT NULL;
CREATE INDEX idx_attachments_ticket ON attachments(ticket_id) WHERE ticket_id IS NOT NULL;
CREATE INDEX idx_attachments_conversation ON attachments(conversation_id) WHERE conversation_id IS NOT NULL;
CREATE INDEX idx_attachments_memory ON attachments(memory_id) WHERE memory_id IS NOT NULL;
CREATE TABLE IF NOT EXISTS "memories" (
    id              TEXT PRIMARY KEY,
    workspace_id    TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id      TEXT REFERENCES projects(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    when_to_use     TEXT NOT NULL DEFAULT '',
    body            TEXT NOT NULL,
    always_included INTEGER NOT NULL DEFAULT 0,
    version         INTEGER NOT NULL DEFAULT 1,
    created_by      TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL,
    updated_by      TEXT NOT NULL DEFAULT '',
    updated_at      INTEGER NOT NULL
);
CREATE INDEX idx_memories_project ON memories(project_id);
CREATE INDEX idx_memories_workspace ON memories(workspace_id);
CREATE INDEX idx_memories_workspace_scoped ON memories(workspace_id) WHERE project_id IS NULL;

-- Seed rows for a fresh instance.
INSERT INTO instance_settings VALUES(1,'',1,strftime('%s','now'),'','','{ticket.Ticket} {ticket.Status}','','','','','');
INSERT INTO workspaces VALUES('workspace-default','Default',strftime('%s','now'),strftime('%s','now'));
INSERT INTO projects VALUES('project-general','General','',0,'workspace-default',strftime('%s','now'),strftime('%s','now'),'');
INSERT INTO statuses VALUES('open','project-general','Open',0,'backlog','',strftime('%s','now'),strftime('%s','now'));
INSERT INTO statuses VALUES('in_progress','project-general','In progress',1,'progress','',strftime('%s','now'),strftime('%s','now'));
INSERT INTO statuses VALUES('done','project-general','Done',2,'done','',strftime('%s','now'),strftime('%s','now'));
INSERT INTO statuses VALUES('closed','project-general','Closed',3,'done','',strftime('%s','now'),strftime('%s','now'));
INSERT INTO ticket_types VALUES('ticket-type-task','project-general','task',0,'',strftime('%s','now'),strftime('%s','now'));
INSERT INTO ticket_types VALUES('ticket-type-bug','project-general','bug',1,'',strftime('%s','now'),strftime('%s','now'));
INSERT INTO ticket_types VALUES('ticket-type-feature','project-general','feature',2,'',strftime('%s','now'),strftime('%s','now'));
INSERT INTO plays VALUES('111f3b1d4fed4259556a7b016c23995b','workspace-default','Fix with AI','ticket','Reads the ticket, implements a fix on its own branch, and opens a pull request.','Confirm your Nexul access by reading the ticket with `ticket_get` and its links with `ticket_get_links`. Work in the project checkout you are running in. Create a branch named `<ticket key>-<short-slug>` from the default branch, implement the fix, run the project''s tests and linters, and commit. Push the branch, open a pull request whose title starts with the ticket key, then link the branch and the PR to the ticket with `ticket_link_branch` and `ticket_link_pr`. Do not merge unless a memory selected for this run explicitly permits merging; if one does, merge once checks pass. Reply with what changed, how it was verified, and the PR link. If you cannot complete the fix, say what blocked you instead of opening a partial PR.',1,'progress','[]','',strftime('%s','now'),strftime('%s','now'));
INSERT INTO plays VALUES('27206653223346fec10d980ab7cedb5c','workspace-default','To tickets via AI','doc','Splits a doc into tickets a developer could pick up independently.','Confirm your Nexul access by reading the doc with `doc_get`. List the doc''s project''s columns with `status_list` and its ticket types. Search existing tickets with `ticket_search` so you do not duplicate work already tracked. Split the doc into tickets a developer could pick up independently: one outcome per ticket, a title under eighty characters, a body with context, acceptance criteria, and a pointer to the doc section it came from. Create each with `ticket_create`, passing the doc id so the ticket links back. Put them in the backlog column. Reply with the list of tickets created and anything in the doc you deliberately did not turn into a ticket.',1,NULL,'[]','',strftime('%s','now'),strftime('%s','now'));
INSERT INTO memories VALUES('memory-project-general-working','workspace-default','project-general','Working in this project','Always included in this project''s turns.',replace('This memory is Agent''s shared context for this project — durable notes worth remembering across every chat turn.\n\nReplace this starter text with anything the team wants Agent to always know: conventions, gotchas, or standing preferences.\n\nEdit it any time; it stays always-included until someone turns that off.','\n',char(10)),1,1,'',strftime('%s','now'),'',strftime('%s','now'));
