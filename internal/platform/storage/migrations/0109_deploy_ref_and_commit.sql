-- What a deploy was asked to build and the commit the runner checked out, so Retry rebuilds the same commit.
ALTER TABLE deploys ADD COLUMN ref TEXT NOT NULL DEFAULT '';
ALTER TABLE deploys ADD COLUMN commit_sha TEXT NOT NULL DEFAULT '';
