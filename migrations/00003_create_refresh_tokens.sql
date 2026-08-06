-- +goose NO TRANSACTION

-- +goose Up
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id BIGINT NOT NULL AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    jti VARCHAR(64) NOT NULL,
    token_hash CHAR(64) NOT NULL,
    expires_at DATETIME(3) NOT NULL,
    revoked_at DATETIME(3) NULL DEFAULT NULL,
    replaced_by_jti VARCHAR(64) NULL DEFAULT NULL,
    created_at DATETIME(3) NULL DEFAULT NULL,
    updated_at DATETIME(3) NULL DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_refresh_tokens_jti (jti),
    UNIQUE KEY uk_refresh_tokens_hash (token_hash),
    KEY idx_refresh_tokens_user_id (user_id),
    KEY idx_refresh_tokens_expires_at (expires_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
