-- Stack events written before they were redacted still carry env and branch-rule override values; blank them, keep the keys.

-- A delivery still waiting to go out is signed over the body being blanked and could not verify, so it ends here.
UPDATE integration_deliveries SET status = 'dead'
WHERE status IN ('pending', 'failed') AND topic IN ('service.created', 'service.updated') AND json_valid(CAST(payload AS TEXT))
  AND (EXISTS (SELECT 1 FROM json_each(CAST(payload AS TEXT), '$.data.stack.env') WHERE value IS NOT '')
       OR EXISTS (SELECT 1 FROM json_each(CAST(payload AS TEXT), '$.data.stack.branch_deploy_rules') r, json_each(r.value, '$.overrides') o WHERE o.value IS NOT ''));

UPDATE outbox SET payload = CAST(json_set(CAST(payload AS TEXT), '$.stack.env',
    json((SELECT json_group_object(key, '') FROM json_each(CAST(payload AS TEXT), '$.stack.env')))) AS BLOB)
WHERE topic IN ('service.created', 'service.updated') AND json_valid(CAST(payload AS TEXT))
  AND EXISTS (SELECT 1 FROM json_each(CAST(payload AS TEXT), '$.stack.env') WHERE value IS NOT '');

UPDATE outbox SET payload = CAST(json_set(CAST(payload AS TEXT), '$.stack.branch_deploy_rules',
    json((SELECT json_group_array(json(iif(json_type(rule.value, '$.overrides') = 'object',
        json_set(rule.value, '$.overrides', json((SELECT json_group_object(key, '') FROM json_each(rule.value, '$.overrides')))),
        rule.value)))
      FROM json_each(CAST(payload AS TEXT), '$.stack.branch_deploy_rules') rule))) AS BLOB)
WHERE topic IN ('service.created', 'service.updated') AND json_valid(CAST(payload AS TEXT))
  AND EXISTS (SELECT 1 FROM json_each(CAST(payload AS TEXT), '$.stack.branch_deploy_rules') r, json_each(r.value, '$.overrides') o WHERE o.value IS NOT '');

UPDATE dead_letters SET payload = CAST(json_set(CAST(payload AS TEXT), '$.stack.env',
    json((SELECT json_group_object(key, '') FROM json_each(CAST(payload AS TEXT), '$.stack.env')))) AS BLOB)
WHERE topic IN ('service.created', 'service.updated') AND json_valid(CAST(payload AS TEXT))
  AND EXISTS (SELECT 1 FROM json_each(CAST(payload AS TEXT), '$.stack.env') WHERE value IS NOT '');

UPDATE dead_letters SET payload = CAST(json_set(CAST(payload AS TEXT), '$.stack.branch_deploy_rules',
    json((SELECT json_group_array(json(iif(json_type(rule.value, '$.overrides') = 'object',
        json_set(rule.value, '$.overrides', json((SELECT json_group_object(key, '') FROM json_each(rule.value, '$.overrides')))),
        rule.value)))
      FROM json_each(CAST(payload AS TEXT), '$.stack.branch_deploy_rules') rule))) AS BLOB)
WHERE topic IN ('service.created', 'service.updated') AND json_valid(CAST(payload AS TEXT))
  AND EXISTS (SELECT 1 FROM json_each(CAST(payload AS TEXT), '$.stack.branch_deploy_rules') r, json_each(r.value, '$.overrides') o WHERE o.value IS NOT '');

UPDATE integration_deliveries SET payload = CAST(json_set(CAST(payload AS TEXT), '$.data.stack.env',
    json((SELECT json_group_object(key, '') FROM json_each(CAST(payload AS TEXT), '$.data.stack.env')))) AS BLOB)
WHERE topic IN ('service.created', 'service.updated') AND json_valid(CAST(payload AS TEXT))
  AND EXISTS (SELECT 1 FROM json_each(CAST(payload AS TEXT), '$.data.stack.env') WHERE value IS NOT '');

UPDATE integration_deliveries SET payload = CAST(json_set(CAST(payload AS TEXT), '$.data.stack.branch_deploy_rules',
    json((SELECT json_group_array(json(iif(json_type(rule.value, '$.overrides') = 'object',
        json_set(rule.value, '$.overrides', json((SELECT json_group_object(key, '') FROM json_each(rule.value, '$.overrides')))),
        rule.value)))
      FROM json_each(CAST(payload AS TEXT), '$.data.stack.branch_deploy_rules') rule))) AS BLOB)
WHERE topic IN ('service.created', 'service.updated') AND json_valid(CAST(payload AS TEXT))
  AND EXISTS (SELECT 1 FROM json_each(CAST(payload AS TEXT), '$.data.stack.branch_deploy_rules') r, json_each(r.value, '$.overrides') o WHERE o.value IS NOT '');
