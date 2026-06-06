ALTER TABLE users
  DROP INDEX idx_users_deleted_at,
  DROP COLUMN deleted_at,
  DROP COLUMN last_login_at;
