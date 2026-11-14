-- Marks a session, and the connect code a phone session is minted from, as started through single sign-on, which turning SSO-only on keeps.
ALTER TABLE sessions ADD COLUMN sso INTEGER NOT NULL DEFAULT 0;
ALTER TABLE connect_codes ADD COLUMN sso INTEGER NOT NULL DEFAULT 0;
