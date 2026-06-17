-- +goose Up
CREATE TABLE IF NOT EXISTS user_role (
    role_id BIGINT NOT NULL AUTO_INCREMENT,
    role_name VARCHAR(25) NOT NULL,
    role_description VARCHAR(255) NULL DEFAULT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (role_id),
    UNIQUE KEY uk_role_name (role_name)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE IF EXISTS user_role;