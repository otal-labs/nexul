-- Automations are workspace actions and automation hosts follow their machine (ADR 0154): the "automations" instance
-- area and its rules go, roles keep their automations:* bits, and a host code remembers who made it so it is checked
-- again when it is used.
DELETE FROM permission_overwrites WHERE resource_type = 'instance_area' AND resource_id = 'automations';

ALTER TABLE automation_host_enrollment_codes ADD COLUMN enrolled_by TEXT NOT NULL DEFAULT '';
