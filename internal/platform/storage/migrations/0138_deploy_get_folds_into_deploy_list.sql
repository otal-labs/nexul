-- deploy_get folded into deploy_list with an id, so stored play instructions that name it are rewritten.
UPDATE plays SET instructions = replace(replace(instructions, '`deploy_get`', '`deploy_list` with an `id`'), 'deploy_get', 'deploy_list')
WHERE instructions LIKE '%deploy_get%';
