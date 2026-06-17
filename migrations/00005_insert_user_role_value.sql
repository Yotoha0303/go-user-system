-- +goose Up
INSERT INTO
    user_role (
        `role_name`,
        `role_description`
    )
VALUES ('admin', '管理员'),
    ('user', '普通用户');

-- +goose Down
DELETE FROM user_role