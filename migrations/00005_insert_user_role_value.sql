-- +goose Up
INSERT INTO
    user_role (
        `role_name`,
        `role_description`
    )
VALUES ('admin', 'Administrator'),
    ('user', 'Regular user')
ON DUPLICATE KEY UPDATE
    role_description = VALUES(role_description);

-- +goose Down
DELETE FROM user_role WHERE role_name IN ('admin', 'user');