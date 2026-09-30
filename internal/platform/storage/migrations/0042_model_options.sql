-- The model options (reasoning level, context window, fast mode, ...) picked beside each stored model, as a JSON array of {id, value}.
ALTER TABLE pairing_user_defaults ADD COLUMN model_options TEXT NOT NULL DEFAULT '[]';
ALTER TABLE pairing_project_links ADD COLUMN model_options TEXT NOT NULL DEFAULT '[]';
ALTER TABLE play_trails ADD COLUMN model_options TEXT NOT NULL DEFAULT '[]';
