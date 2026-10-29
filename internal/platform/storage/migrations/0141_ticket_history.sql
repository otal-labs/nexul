-- A ticket's description versions: the kept (newest ten) texts of its body, with who wrote each (a JSON array of user ids) and the automation or "mcp" it came through.
CREATE TABLE IF NOT EXISTS ticket_description_versions (
    ticket_id   TEXT    NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    version     INTEGER NOT NULL,
    body        TEXT    NOT NULL,
    editor_ids  TEXT    NOT NULL DEFAULT '[]',
    via         TEXT    NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    PRIMARY KEY (ticket_id, version)
);

-- A ticket's field changes: one row per changed field with its display-ready before and after and the actor as JSON.
CREATE TABLE IF NOT EXISTS ticket_changes (
    id          TEXT    PRIMARY KEY,
    ticket_id   TEXT    NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    field       TEXT    NOT NULL,
    from_value  TEXT    NOT NULL,
    to_value    TEXT    NOT NULL,
    actor       TEXT    NOT NULL,
    created_at  INTEGER NOT NULL
);

-- Serves a ticket's history list newest first, and the prune that keeps its newest hundred (id breaks a second's tie).
CREATE INDEX IF NOT EXISTS idx_ticket_changes_ticket ON ticket_changes(ticket_id, created_at DESC, id DESC);

-- History starts here: each ticket's current description is version 1.
INSERT OR IGNORE INTO ticket_description_versions (ticket_id, version, body, created_at)
SELECT id, 1, body, updated_at FROM tickets;

-- Each ticket gets one "created" change, by its stored reporter (an automation, or a person looked up by login).
INSERT INTO ticket_changes (id, ticket_id, field, from_value, to_value, actor, created_at)
SELECT lower(hex(randomblob(16))), t.id, 'created', '', '',
    json_patch(json_object('kind', t.reporter_kind), json_object(
        'automation_id', NULLIF(t.reporter_automation_id, ''),
        'automation_name', NULLIF(t.reporter_automation_name, ''),
        'user_id', (SELECT u.id FROM users u WHERE t.reporter_kind != 'automation' AND lower(u.login) = lower(t.reporter_login)
                    ORDER BY u.login = t.reporter_login DESC, u.rowid LIMIT 1))),
    t.created_at
FROM tickets t
WHERE NOT EXISTS (SELECT 1 FROM ticket_changes c WHERE c.ticket_id = t.id AND c.field = 'created');
