-- A rule names a role or a person (ADR 0148): target says which, and exactly one of user_id and role_id is set, each
-- going with the account or role it names. Every existing row names a person and is copied as one, unchanged.
CREATE TABLE permission_overwrites_new (
    resource_type TEXT NOT NULL,
    resource_id   TEXT NOT NULL,
    target        TEXT NOT NULL CHECK (target IN ('person', 'role')),
    user_id       TEXT REFERENCES users(id) ON DELETE CASCADE,
    role_id       TEXT REFERENCES roles(id) ON DELETE CASCADE,
    allow         TEXT NOT NULL DEFAULT '[]',
    deny          TEXT NOT NULL DEFAULT '[]',
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    CHECK ((target = 'person' AND user_id IS NOT NULL AND role_id IS NULL)
        OR (target = 'role' AND role_id IS NOT NULL AND user_id IS NULL))
);
INSERT INTO permission_overwrites_new (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at)
SELECT resource_type, resource_id, 'person', user_id, allow, deny, created_at, updated_at FROM permission_overwrites;
DROP TABLE permission_overwrites;
ALTER TABLE permission_overwrites_new RENAME TO permission_overwrites;

-- One rule per target on an entity: the upserts' conflict target. Serves ListOverwritesByResource and the resource half
-- of ListOverwriteRules, an entity's rules for a person and the roles they hold.
CREATE UNIQUE INDEX IF NOT EXISTS idx_permission_overwrites_target
    ON permission_overwrites(resource_type, resource_id, target, coalesce(user_id, role_id));
-- Serves GetOverwrite, DeleteOverwrite, the person half of ListOverwriteRulesOfType and CountOverwriteAllowAny, and an
-- account's delete.
CREATE INDEX IF NOT EXISTS idx_permission_overwrites_user ON permission_overwrites(user_id, resource_type, resource_id);
-- Serves the role half of ListOverwriteRulesOfType and CountOverwriteAllowAny, and a role's delete.
CREATE INDEX IF NOT EXISTS idx_permission_overwrites_role ON permission_overwrites(role_id, resource_type);
