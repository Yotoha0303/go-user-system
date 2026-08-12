# 最小可观测性

本项目提供本地/验收环境可运行的 Prometheus 闭环：应用指标、固定版本 Prometheus、持久化时序数据和基础告警规则。它不包含生产级外部通知、长期存储、集中日志或 Trace。

## 指标端点

| 端点 | 说明 |
| --- | --- |
| `/version` | version、commit、build time |
| `/metrics` | Prometheus exposition；默认不经 Kubernetes Ingress 暴露 |
| `/livez` | 进程存活 |
| `/readyz` | MySQL 与启用的 Redis 就绪状态，并更新 readiness gauge |

关键指标：

- `go_user_system_build_info`
- `go_user_system_readiness`
- `go_user_system_http_requests_total`
- `go_user_system_http_request_duration_seconds`
- `go_user_system_http_requests_in_flight`
- Go Runtime 与 Process Collector 默认指标

HTTP 标签只使用归一化 `method`、Gin 路由模板和 `status`。标准方法之外的输入统一为 `OTHER`，动态 ID 不进入标签，未匹配请求统一使用 `route="unmatched"`。

HTTP 计时与 in-flight 统计位于请求超时包装器外层，Gin 在处理开始时写入路由模板。因此超过应用 timeout 的请求只记录一次客户端实际收到的 `503` 和实际等待时间，响应发出后 in-flight 立即归零，不等待忽略取消信号的内层 handler 返回。

## 配置校验

```powershell
make observability-validate
```

该目标同时验证 Compose overlay、Prometheus 主配置和 4 条规则。CI 执行相同检查。

## 启动

先按 README 创建 `.env`，然后执行：

```powershell
make observability-up
```

也可以直接运行：

```powershell
docker compose -f compose.yaml -f compose.observability.yaml up -d --build --wait
```

访问：

| 地址 | 用途 |
| --- | --- |
| `http://127.0.0.1:9090/targets` | 抓取状态，应显示 `go-user-system` 为 UP |
| `http://127.0.0.1:9090/alerts` | 告警状态 |
| `http://127.0.0.1:9090/graph` | PromQL 查询 |

建议查询：

```promql
go_user_system_build_info
go_user_system_readiness
sum by (route, status) (rate(go_user_system_http_requests_total[5m]))
histogram_quantile(0.95, sum by (le) (rate(go_user_system_http_request_duration_seconds_bucket[10m])))
```

## 告警规则

| 告警 | 条件 | 默认等待 |
| --- | --- | --- |
| `GoUserSystemTargetDown` | Prometheus 无法抓取后端 | 1 分钟 |
| `GoUserSystemNotReady` | 最近 readiness 检查失败 | 2 分钟 |
| `GoUserSystemHigh5xxRate` | 5 分钟 5xx 比例超过 5% | 5 分钟 |
| `GoUserSystemHighP95Latency` | 10 分钟 p95 超过 1 秒 | 10 分钟 |

本地规则只在 Prometheus 中形成 Pending/Firing 状态，不发送邮件或 IM。生产需要在平台侧配置 Alertmanager 和 Secret 管理的接收器。

## 安全验证告警

在隔离环境启动完整栈后，可以停止 app 触发 TargetDown：

```powershell
docker compose -f compose.yaml -f compose.observability.yaml stop app
Start-Sleep -Seconds 90
Invoke-RestMethod http://127.0.0.1:9090/api/v1/alerts
docker compose -f compose.yaml -f compose.observability.yaml start app
```

只在隔离环境执行。恢复后确认 target 回到 UP、`/readyz` 为 200 且告警自动解除。

## Kubernetes

后端 Pod 模板带 `prometheus.io/scrape/path/port` 注解，供支持注解发现的集群 Prometheus 使用。仓库不安装 Prometheus Operator，也不创建公网 `/metrics` Ingress。生产环境应使用 NetworkPolicy 限制只有监控命名空间可以访问指标端口。

## 停止与数据

```powershell
make observability-down
```

默认保留 `prometheus_data` 命名卷和 7 天数据。只有确认指标历史可删除时，才可在隔离环境显式删除该卷。
