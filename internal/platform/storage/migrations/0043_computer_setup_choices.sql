-- The rest of the Set up step's saved choices beside setup_skipped_providers: the model and its options per driver kind, and the folder.
ALTER TABLE pairing_computers ADD COLUMN setup_models TEXT NOT NULL DEFAULT '{}';
ALTER TABLE pairing_computers ADD COLUMN setup_model_options TEXT NOT NULL DEFAULT '{}';
ALTER TABLE pairing_computers ADD COLUMN setup_folder TEXT NOT NULL DEFAULT '';
