-- Which job a setup turn did: a full setup of its provider, or a skills update that rewrote the shared skill folders.
ALTER TABLE pairing_setup_turns ADD COLUMN kind TEXT NOT NULL DEFAULT 'setup';
