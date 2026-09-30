-- Workspaces get the URL slug tenancy.Slugify derives, and project prefixes become unique per workspace (ADR 0089).
ALTER TABLE workspaces ADD COLUMN slug TEXT NOT NULL DEFAULT '';

CREATE TEMP TABLE workspace_slug_base AS
WITH RECURSIVE walk(id, created_at, name, pos, out) AS (
    SELECT id, created_at, lower(name), 1, '' FROM workspaces
    UNION ALL
    SELECT id, created_at, name, pos + 1, out || CASE
        WHEN substr(name, pos, 1) BETWEEN 'a' AND 'z' OR substr(name, pos, 1) BETWEEN '0' AND '9' THEN substr(name, pos, 1)
        WHEN out = '' OR substr(out, -1) = '-' THEN ''
        ELSE '-'
    END
    FROM walk WHERE pos <= length(name)
)
SELECT id, created_at, COALESCE(NULLIF(rtrim(substr(rtrim(out, '-'), 1, 48), '-'), ''), 'workspace') AS base
FROM walk WHERE pos = length(name) + 1;

UPDATE workspaces SET slug = (
    SELECT CASE
        WHEN r.reserved = 0 AND r.n = 1 THEN r.base
        ELSE r.base || '-' || (r.n + r.reserved)
    END
    FROM (
        SELECT id, base,
            ROW_NUMBER() OVER (PARTITION BY base ORDER BY created_at, id) AS n,
            base IN ('api', 'assets', 'auth', 'hooks', 'invite', 'login', 'logout', 'mcp', 'onboarding',
                     'openobserve', 'settings', 'setup', 'static', 'swagger', 'wizard', 'ws') AS reserved
        FROM workspace_slug_base
    ) r
    WHERE r.id = workspaces.id
);

-- A name that already ends in "-2" can meet a suffixed twin; the later one takes its id's first characters instead.
UPDATE workspaces SET slug = slug || '-' || rtrim(lower(substr(id, 1, 8)), '-')
WHERE EXISTS (
    SELECT 1 FROM workspaces earlier
    WHERE earlier.slug = workspaces.slug
      AND (earlier.created_at < workspaces.created_at OR (earlier.created_at = workspaces.created_at AND earlier.id < workspaces.id))
);

DROP TABLE workspace_slug_base;

CREATE UNIQUE INDEX idx_workspaces_slug ON workspaces(slug) WHERE slug != '';

DROP INDEX idx_projects_prefix;
CREATE UNIQUE INDEX idx_projects_workspace_prefix ON projects(workspace_id, prefix) WHERE prefix != '';
