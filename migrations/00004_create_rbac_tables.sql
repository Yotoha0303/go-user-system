-- +goose NO TRANSACTION

-- +goose Up
CREATE TABLE IF NOT EXISTS roles (
    id BIGINT NOT NULL AUTO_INCREMENT,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(64) NOT NULL,
    created_at DATETIME(3) NULL DEFAULT NULL,
    updated_at DATETIME(3) NULL DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_roles_code (code)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS permissions (
    id BIGINT NOT NULL AUTO_INCREMENT,
    code VARCHAR(128) NOT NULL,
    name VARCHAR(64) NOT NULL,
    method VARCHAR(16) NOT NULL,
    path VARCHAR(255) NOT NULL,
    created_at DATETIME(3) NULL DEFAULT NULL,
    updated_at DATETIME(3) NULL DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_permissions_code (code),
    UNIQUE KEY uk_permissions_method_path (method, path)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    created_at DATETIME(3) NULL DEFAULT NULL,
    PRIMARY KEY (user_id, role_id),
    KEY idx_user_roles_role_id (role_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id BIGINT NOT NULL,
    permission_id BIGINT NOT NULL,
    created_at DATETIME(3) NULL DEFAULT NULL,
    PRIMARY KEY (role_id, permission_id),
    KEY idx_role_permissions_permission_id (permission_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

INSERT INTO roles (code, name, created_at, updated_at)
VALUES
    ('admin', '管理员', NOW(3), NOW(3)),
    ('user', '普通用户', NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    updated_at = VALUES(updated_at);

INSERT INTO permissions (code, name, method, path, created_at, updated_at)
VALUES
    ('profile:read', '读取个人资料', 'GET', '/api/v1/users/me', NOW(3), NOW(3)),
    ('profile:update', '更新个人资料', 'PUT', '/api/v1/users/me/profile', NOW(3), NOW(3)),
    ('password:update', '修改登录密码', 'PATCH', '/api/v1/users/me/update/password', NOW(3), NOW(3)),
    ('admin:roles:read', '读取角色列表', 'GET', '/api/v1/admin/roles', NOW(3), NOW(3)),
    ('admin:permissions:read', '读取权限列表', 'GET', '/api/v1/admin/permissions', NOW(3), NOW(3)),
    ('admin:user_roles:update', '分配用户角色', 'PUT', '/api/v1/admin/users/:id/roles', NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    method = VALUES(method),
    path = VALUES(path),
    updated_at = VALUES(updated_at);

INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW(3)
FROM roles r
JOIN permissions p
WHERE r.code = 'user'
  AND p.code IN ('profile:read', 'profile:update', 'password:update');

INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW(3)
FROM roles r
JOIN permissions p
WHERE r.code = 'admin';

-- +goose Down
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
