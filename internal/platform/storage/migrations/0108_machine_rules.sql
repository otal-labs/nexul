-- A machine's rules go with it: resource_id names any resource type, so it has no foreign key.
CREATE TRIGGER IF NOT EXISTS machines_rules_ad AFTER DELETE ON machines BEGIN
    DELETE FROM permission_overwrites WHERE resource_type = 'machine' AND resource_id = old.id;
END;
