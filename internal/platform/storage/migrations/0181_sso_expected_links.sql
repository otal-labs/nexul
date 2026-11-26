-- Expected sign-ins: the account a person's first sign-in through the single sign-on provider lands on, by the email it
-- brings, saved while moving people from one provider to the other. One per account and one per email; using it deletes it.
CREATE TABLE IF NOT EXISTS sso_expected_links (
    user_id    TEXT    PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    -- The unique index also serves GetSSOExpectedLinkByEmail.
    email      TEXT    NOT NULL UNIQUE COLLATE NOCASE,
    created_by TEXT    NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);
