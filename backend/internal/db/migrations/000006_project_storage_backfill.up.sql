UPDATE projects
SET storage_provider = 'filesystem'
WHERE storage_provider = '';

UPDATE projects
SET storage_prefix = id
WHERE storage_prefix = '';
