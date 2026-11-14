-- name: GetInstanceBranding :one
SELECT product_name, accent, light_theme, dark_theme, css, length(logo) AS logo_size, logo_type, logo_version, updated_by, updated_at
FROM instance_branding WHERE id = 1;

-- name: SaveInstanceBranding :exec
INSERT INTO instance_branding (id, product_name, accent, light_theme, dark_theme, css, updated_by, updated_at) VALUES (1, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET product_name = excluded.product_name, accent = excluded.accent, light_theme = excluded.light_theme,
    dark_theme = excluded.dark_theme, css = excluded.css, updated_by = excluded.updated_by, updated_at = excluded.updated_at;

-- name: GetInstanceBrandingLogo :one
SELECT logo, logo_type FROM instance_branding WHERE id = 1 AND logo IS NOT NULL;

-- name: SetInstanceBrandingLogo :exec
INSERT INTO instance_branding (id, logo, logo_type, logo_version, updated_by, updated_at) VALUES (1, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET logo = excluded.logo, logo_type = excluded.logo_type, logo_version = excluded.logo_version,
    updated_by = excluded.updated_by, updated_at = excluded.updated_at;

-- name: ClearInstanceBrandingLogo :exec
UPDATE instance_branding SET logo = NULL, logo_type = '', logo_version = '', updated_by = ?, updated_at = ? WHERE id = 1;

-- name: DeleteInstanceBranding :exec
DELETE FROM instance_branding WHERE id = 1;
