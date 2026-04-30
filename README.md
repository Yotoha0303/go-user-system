# go-user-system

## 1. 项目简介

基于 Go + Gin + GORM + MySQL 实现的用户系统，包含用户注册、登录、密码哈希、JWT 鉴权、受保护接口等基础后端能力。

## 2. 技术栈

- Go
- Gin
- GORM
- MySQL
- bcrypt
- JWT
- godotenv
- YAML 配置

## 3. 核心功能

- 用户注册
- 用户登录
- bcrypt 密码哈希
- JWT 签发
- JWT 中间件鉴权
- GET /api/v1/me
- 统一响应
- 环境变量配置

## 4. 项目结构

```text
api/        HTTP 接口层，负责参数绑定和响应返回
service/    业务逻辑层，负责注册、登录、鉴权相关业务规则
dao/        数据访问层，负责数据库操作
model/      数据模型层，定义 User、Response 等结构
router/     路由注册
utils/      工具函数，如统一响应、JWT 工具
config/     YAML 配置加载
global/     全局资源，如 DB

## 5. SQL结构

```text
CREATE TABLE `users`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `username` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `password_hash` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `nickname` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  `status` tinyint NOT NULL DEFAULT 1,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `idx_username`(`username` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 2 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci ROW_FORMAT = Dynamic;

```

## 6. 环境变量

```text
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=you_password
DB_NAME=go_user_system

JWT_SECRET=replace_with_a_long_random_secret
JWT_EXPIRE_HOURS=24
```

## 7. 启动方式
go mod tidy

go run main.go

## 8. 接口说明

POST /register

请求：

```
{
  "username": "yotoha",
  "password": "123456"
} 
```

响应：

```
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

POST /login

请求：

```
{
  "username": "yotoha",
  "password": "123456"
}
```

响应：

```
{
  "code": 0,
  "msg": "success",
  "data": {
    "token": "xxx"
  }
}
```

GET /me

Header：

Authorization: Bearer <token>

响应：

```
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": 1,
    "username": "yotoha",
    "nickname": "yotoha",
    "status": 1
  }
}
```

## 9. 项目亮点
使用分层结构拆分 API、Service、DAO、Model

使用 bcrypt 存储密码哈希，避免明文密码入库

使用 JWT 实现无状态登录鉴权

使用中间件保护用户信息接口

使用统一响应结构规范接口返回

使用环境变量管理数据库和 JWT 配置，避免敏感信息硬编码

## 10. 当前状态

该项目为 Go 后端基础项目，用于练习并展示用户系统、认证鉴权、接口设计、错误处理和基础工程结构。