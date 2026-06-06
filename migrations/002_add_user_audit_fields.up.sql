ALTER TABLE users
  ADD COLUMN last_login_at DATETIME(3) NULL DEFAULT NULL,
  ADD COLUMN deleted_at DATETIME(3) NULL DEFAULT NULL,
  ADD INDEX idx_users_deleted_at (deleted_at);
