-- +goose NO TRANSACTION

-- +goose Up
ALTER TABLE users
    ADD COLUMN auth_version BIGINT NOT NULL DEFAULT 1 AFTER status;

ALTER TABLE refresh_tokens
    ADD COLUMN family_id VARCHAR(64) NULL DEFAULT NULL AFTER jti,
    ADD COLUMN revoked_reason VARCHAR(32) NULL DEFAULT NULL AFTER revoked_at;

UPDATE refresh_tokens
SET family_id = jti
WHERE family_id IS NULL;

ALTER TABLE refresh_tokens
    MODIFY COLUMN family_id VARCHAR(64) NOT NULL,
    ADD KEY idx_refresh_tokens_family_id (family_id);

-- +goose Down
ALTER TABLE refresh_tokens
    DROP KEY idx_refresh_tokens_family_id,
    DROP COLUMN revoked_reason,
    DROP COLUMN family_id;

ALTER TABLE users
    DROP COLUMN auth_version;
