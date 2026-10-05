-- Each workspace draws its own topology canvas, stored under its id, while the shared row keyed 'default' becomes the
-- registry of every service node (ADR 0125). The first workspace keeps the shared canvas's positions, drawn nodes,
-- edges, and camera; the others start empty and lay their services out on first view.
INSERT OR IGNORE INTO topology (environment, schema_version, canvas, updated_at)
SELECT 'workspace-default', schema_version, canvas, updated_at FROM topology
WHERE environment = 'default' AND EXISTS (SELECT 1 FROM workspaces WHERE id = 'workspace-default');
