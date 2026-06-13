-- +goose NO TRANSACTION

-- +goose Up
-- alter table users
--	add column last_login_at
-- 	add column deleted_at
--	add index idx_users_deleted_at
ALTER TABLE users
ADD COLUMN last_login_at DATETIME(3) NULL DEFAULT NULL,
ADD COLUMN deleted_at DATETIME(3) NULL DEFAULT NULL,
ADD INDEX idx_users_deleted_at (deleted_at);

-- +goose Down
-- alter table users
--	drop column last_login_at
-- 	drop column deleted_at
--	drop index idx_users_deleted_at
ALTER TABLE users
DROP INDEX idx_users_deleted_at,
DROP COLUMN deleted_at,
DROP COLUMN last_login_at;