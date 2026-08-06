-- +goose NO TRANSACTION

-- +goose Up
INSERT IGNORE INTO user_roles (user_id, role_id, created_at)
SELECT u.id, r.id, NOW(3)
FROM users u
JOIN roles r ON r.code = 'user';

INSERT IGNORE INTO user_roles (user_id, role_id, created_at)
SELECT first_user.id, r.id, NOW(3)
FROM (
    SELECT MIN(id) AS id
    FROM users
) AS first_user
JOIN roles r ON r.code = 'admin'
WHERE first_user.id IS NOT NULL;

-- +goose Down
-- Role backfill is intentionally not deleted on rollback because these rows may
-- have been modified by administrators after migration.
SELECT 1;
