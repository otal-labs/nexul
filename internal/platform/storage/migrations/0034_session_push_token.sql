-- A phone registers its Expo push token on its own session, so deleting the row also stops its pushes.
ALTER TABLE sessions ADD COLUMN push_token TEXT;
