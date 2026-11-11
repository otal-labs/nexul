-- The variable names a runner's secret check found changed in the secrets manager since the stack's last deploy, as a JSON
-- array; '[]' for every stack that exists. UpdateStack never writes it, so editing a stack cannot clear or restore a notice.
ALTER TABLE stacks ADD COLUMN secrets_changed TEXT NOT NULL DEFAULT '[]';
