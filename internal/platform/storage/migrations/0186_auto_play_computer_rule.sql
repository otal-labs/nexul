-- An auto play may narrow the computers its runs land on (ADR 0214): JSON {"labels_in":[],"labels_out":[],"ids":[]}.
ALTER TABLE auto_plays ADD COLUMN computer_rule TEXT NOT NULL DEFAULT '{}';
