-- A computer's rules go with it and with its owner's account: resource_id names any resource type, so it has no foreign key.
CREATE TRIGGER IF NOT EXISTS pairing_computers_rules_ad AFTER DELETE ON pairing_computers BEGIN
    DELETE FROM permission_overwrites WHERE resource_type = 'computer' AND resource_id = old.id;
END;
