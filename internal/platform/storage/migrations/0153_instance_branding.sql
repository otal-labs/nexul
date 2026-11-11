-- The instance's branding, one row: the product name, the accent as #rrggbb, each imported theme as JSON, and the logo
-- with its image type and a version that changes with its bytes. Whether it shows is decided at read time by the
-- license key, so an expired key hides the row without rewriting it.
CREATE TABLE IF NOT EXISTS instance_branding (
    id            INTEGER PRIMARY KEY CHECK (id = 1),
    product_name  TEXT    NOT NULL DEFAULT '',
    accent        TEXT    NOT NULL DEFAULT '',
    light_theme   TEXT    NOT NULL DEFAULT '',
    dark_theme    TEXT    NOT NULL DEFAULT '',
    logo          BLOB,
    logo_type     TEXT    NOT NULL DEFAULT '',
    logo_version  TEXT    NOT NULL DEFAULT '',
    updated_by    TEXT    NOT NULL DEFAULT '',
    updated_at    INTEGER NOT NULL
);
