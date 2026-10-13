-- The project wizard's "Who's in it" step (ADR 0148): every project made before it counts the step as done.
UPDATE projects SET setup_steps = json_set(setup_steps, '$.people', 'done');
