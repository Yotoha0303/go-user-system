# 文件结构
```
/project
  /api        // 接口层
  /service    // 业务逻辑
  /dao        // 数据库操作
  /model      // 数据结构
  /middleware // 中间件
  /config     // 配置
  main.go
```
能力点：工程意识
面试考点：为什么这样分层；各个文件夹的作用是什么

# 基础目录职责

api：接收请求、参数校验、调用 service、返回响应
service：业务逻辑
dao：数据库操作
model：请求体、响应体、数据库模型
middleware：JWT、日志、恢复
config：配置加载
router：路由注册
utils：工具函数