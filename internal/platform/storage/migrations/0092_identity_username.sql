-- A Discord identity's login is its email, which the allowlist matches; the username is what others know it by.
ALTER TABLE user_identities ADD COLUMN username TEXT NOT NULL DEFAULT '';
