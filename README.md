# go-user-system（Go 认证与基础用户系统项目）

## 1. 项目简介

基于 Go + Gin + GORM + MySQL 实现用户认证系统，支持注册、登录、bcrypt 密码哈希、JWT 鉴权、当前用户信息查询和昵称修改。项目采用 api/service/ dao/ model/ 分层结构，使用环境变量管理数据库与 JWT 配置，并通过统一响应结构和业务错误映射提升接口规范性。

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

- 修改昵称

- bcrypt 密码哈希存储

- 用户状态校验

- JWT 生成

- JWT 中间件鉴权

- GET /api/v1/users/me 当前用户信息

- 统一响应结构

- 环境变量配置

- README + 接口文档

## 4. 项目结构

```text
api/        HTTP 接口层，负责参数绑定和响应返回
service/    业务逻辑层，负责注册、登录、鉴权相关业务规则
dao/        数据访问层，负责数据库操作
model/      数据模型层，定义 User、Response 等结构
router/     路由注册、接口归类、版本管理、中间件挂载
utils/      工具函数，如统一响应、JWT 工具
config/     YAML 配置加载
global/     全局资源，如 DB
middleware/ 路由中间件
```

## 5. SQL 结构

```text
CREATE TABLE `users`  (
  `id` bigint NOT NULL AUTO_INCREMENT,
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

```.env
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=go_user_system

JWT_SECRET=replace_with_a_long_random_secret
JWT_EXPIRE_HOURS=24
```

## 7. 启动方式

go mod tidy

go run main.go

## 8. 接口说明

GET /ping

示例

```
curl http://localhost:8082/ping
```

响应

```
{
    "code": 0,
    "msg": "success",
    "data": {
        "message": "success"
    }
}
```

POST /api/v1/auth/register

示例

```
curl -X POST http://localhost:8082/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"yotoha","password":"123456"}'
```

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

POST /api/v1/auth/login

示例

```
curl -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"yotoha","password":"123456"}'
```

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
    "access_token": "xxx",
    "user": {
      "id": 1,
      "username": "yotoha",
      "nickname": "yotoha",
      "status": 1
    }
  }
}
```

GET /api/v1/users/me

示例

```
curl http://localhost:8082/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"
```

Header：

Authorization: Bearer <access_token>

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

PUT /api/v1/users/me/profile

示例

```
curl -X PUT http://localhost:8082/api/v1/users/me/profile \
   -H "Content-Type: application/json" \
   -H "Authorization: Bearer <access_token>" \
   -d '{"nickname":"new_name"}'
```

Header：

Authorization: Bearer <access_token>

请求

```
{
  "nickname":"new_name"
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

## 9. 手动测试流程

```
ping测试服务器状态->注册用户->登录用户->获取access_token中的数据->修改昵称->登录用户，查看更新后的昵称
```

## 10. 项目亮点

使用分层结构拆分 API、Service、DAO、Model

使用 bcrypt 存储密码哈希，避免明文密码入库

使用 JWT 实现无状态登录鉴权

使用中间件保护用户信息接口

使用统一响应结构规范接口返回

使用环境变量管理数据库和 JWT 配置，避免敏感信息硬编码
