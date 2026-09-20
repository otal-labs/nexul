-- name: GetTopologyCanvas :one
SELECT canvas FROM topology WHERE environment = ?;

-- name: SaveTopology :exec
INSERT INTO topology (environment, schema_version, canvas, updated_at) VALUES (?, ?, ?, ?)
ON CONFLICT(environment) DO UPDATE SET
    schema_version = excluded.schema_version,
    canvas = excluded.canvas,
    updated_at = excluded.updated_at;
