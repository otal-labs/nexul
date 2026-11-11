-- The instance's enterprise license key, one row holding the signed text as pasted. Every read checks it again, so a
-- changed instance URL or a passed expiry takes effect without rewriting the row; removing the key deletes the row.
CREATE TABLE IF NOT EXISTS license_key (
    id      INTEGER PRIMARY KEY CHECK (id = 1),
    key     TEXT    NOT NULL,
    set_by  TEXT    NOT NULL DEFAULT '',
    set_at  INTEGER NOT NULL
);
