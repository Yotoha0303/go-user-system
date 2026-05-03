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
# 1、注册
  示例

curl -X POST http://localhost:8082/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"yotoha","password":"123456"}'

# 2、登录
curl -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"yotoha","password":"123456"}'


# 3、获取用户信息(备注：获取access_token需要先调用登录接口，再替换下面命令中的值)
curl http://localhost:8082/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"


# 4、修改昵称
curl -X PUT http://localhost:8082/api/v1/users/me/profile \
   -H "Content-Type: application/json" \
   -H "Authorization: Bearer <access_token>" \
   -d '{"nickname":"new_name"}'

# 5、再次获取用户信息
curl http://localhost:8082/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"

```

## 10. 项目亮点

使用分层结构拆分 API、Service、DAO、Model

使用 bcrypt 存储密码哈希，避免明文密码入库

使用 JWT 实现无状态登录鉴权

使用中间件保护用户信息接口

使用统一响应结构规范接口返回

使用环境变量管理数据库和 JWT 配置，避免敏感信息硬编码

## 11. 最终自测清单

### 服务与环境

- [x] `.env` 配置正确，项目可正常启动
- [x] `GET /ping` 返回 Http 200，且响应体 code = 0，响应结构符合统一格式

### 注册

- [x] 正常注册成功
- [x] 用户名为空时返回正确错误
- [x] 用户名过短时返回正确错误
- [x] 重复注册时返回正确错误

### 登录

- [x] 正常登录成功，返回 `access_token`
- [x] 用户不存在时返回正确错误
- [x] 密码错误时返回正确错误
- [x] 被禁用用户无法登录

### JWT 鉴权

- [x] 不带 token 访问 `/api/v1/users/me` 被拦截
- [x] 错误格式的 Authorization 头被拦截
- [x] 无效 token 被拦截
- [x] 正确 token 可访问 `/api/v1/users/me`

### 用户信息

- [x] `/api/v1/users/me` 返回当前用户信息
- [x] `PUT /api/v1/users/me/profile` 修改昵称成功
- [x] 修改昵称后再次查询，返回最新昵称

## 12. 设计与实现要点

### 1. 分层结构设计

```
本项目采用 `api/ service/ dao/ model/ middleware/ utils/ config/`的分层结构，将 HTTP 请求处理、业务逻辑、数据访问和通用工具拆开。

这样做的原因是避免所有逻辑都堆积在 handler 中，方便后续的扩展。

`api`层只负责参数绑定和响应返回；
`service`层负责处理业务逻辑，如注册、登录、用户状态判定和修改昵称等业务规则；
`dao`层只负责数据库访问；
`model`层只负责定义实体和通用响应结构；
`middleware`层是中间件层，用于`router`层挂载中间件;
`utils`层用于声明各层之间的通用工具;
`config`是项目的配置入口，用于读取项目配置所需的参数。

这种拆分避免项目各层之间耦合，让代码职责更清晰。

```

### 2. 路由分组设计

```
本项目采用 `/api/v1` 作为接口前缀，并将认证接口 `auth` 和用户资源 `users` 进行接口分组：

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/users/me`
- `PUT /api/v1/users/me/profile`

其中，`/auth` 负责注册和登录，`/users`负责当前用户相关的操作。需要登录的用户接口统一挂载JWT鉴权中间件，
避免每个 handler 中重复编写 access_token 校验逻辑。
```

### 3. 用户注册与密码哈希

```
在本项目的注册流程中，`service` 层会先校验用户名和密码，再通过 `dao` 层检查用户名是否已经存在。
密码不会以明文的形式保存在数据库中，而是通过 bcrypt 生成哈希后写入 `password_hash` 字段。

注册流程：
客户端提交用户名和密码：
-> api 层绑定 JSON 参数
-> service 层校验参数
-> dao 层检查用户名是否已经存在
-> bcrypt 生成密码哈希
-> dao 层创建用户记录
-> mysql 保存数据
-> api 统一响应
```

### 4. 用户登录与 JWT 鉴权

```
在本项目的用户登录流程中，`service` 层不会直接查询账户和密码是否符合登录需求，而是会依次查询用户名是否存在、判断用户是否禁用、bcrypt 校验用户密码是否和数据库中的密码哈希是否相互匹配。
`api` 层会调用 `middleware` 层生成 JWT ，并返回 access_token ，最后会将部分用户信息和 access_token 进行返回。

登录流程：
客户端提交用户名和密码：
-> api 层绑定 JSON 参数
-> service 层校验参数，依次查询用户名、判断用户状态、bcrypt 校验密码
-> middleware 层生成 JWT 返回 access_token
-> api 将部分用户信息和 access_token 返回

```

### 5. 受保护接口与用户上下文

```

```

### 6. 统一响应与错误处理

### 7. 配置与敏感信息管理
