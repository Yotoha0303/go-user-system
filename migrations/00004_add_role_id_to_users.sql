-- +goose Up
ALTER TABLE users
ADD COLUMN role_id BIGINT NULL,
ADD CONSTRAINT fk_users_role_id FOREIGN KEY (role_id) REFERENCES user_role (role_id),
ADD INDEX idx_users_role_id (role_id);

-- +goose Down
ALTER TABLE users
DROP FOREIGN KEY fk_users_role_id,
DROP INDEX idx_users_role_id,
DROP COLUMN role_id;