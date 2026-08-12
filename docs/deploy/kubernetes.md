# Kubernetes 部署

清单默认部署 `v1.0.0-rc.3` 的 GHCR 固定标签，包含 MySQL、Redis、单例 migration Job、两个后端副本、两个前端副本和启用 TLS 跳转的 Nginx Ingress。

## 前置条件

- Kubernetes 集群和 `kubectl`。
- 默认 StorageClass。
- 已安装 Nginx Ingress Controller。
- 已签发域名证书，可创建 `go-user-system-tls` TLS Secret。
- 集群可以拉取 `ghcr.io/yotoha0303` 的公开镜像；私有包需配置 `imagePullSecrets`。

## 创建 Secret

先创建命名空间和本地 Secret 文件：

```bash
kubectl apply -f k8s/namespace.yaml
cp k8s/secret.example.yaml k8s/secret.yaml
```

编辑 `k8s/secret.yaml`，为 `DB_ROOT_PASSWORD`、`DB_PASSWORD` 和 `JWT_SECRET` 设置独立强随机值，然后应用：

```bash
kubectl apply -f k8s/secret.yaml
```

`k8s/secret.yaml` 已被 Git 忽略。不要直接提交或复用示例值。

## 部署

修改 `k8s/ingress.yaml` 的域名，并创建清单引用的 TLS Secret：

```bash
kubectl create secret tls go-user-system-tls \
  -n go-user-system \
  --cert=/path/to/tls.crt \
  --key=/path/to/tls.key
```

确认 `k8s/configmap.yaml` 的 `TRUSTED_PROXIES` 只覆盖 Ingress Controller 所在网络；仓库中的私网 CIDR 是自托管示例，不应直接照搬到公网边界。然后运行：

```bash
make k8s-deploy
```

该目标会先等待 MySQL 和 Redis，再执行并等待 migration Job，最后部署前后端和 Ingress。不要使用 `kubectl apply -f k8s/ --recursive`，否则示例 Secret 和业务 Deployment 可能在 migration 完成前被应用。

查看状态：

```bash
make k8s-status
kubectl get job,pod,deploy,svc,ingress -n go-user-system
kubectl logs job/go-user-system-migrate-v1-0-0-rc-3 -n go-user-system
```

后端 Pod 带 `/metrics` Prometheus 抓取注解，但仓库不安装集群监控组件。使用已有 Prometheus/Operator 时，应通过服务发现抓取 ClusterIP，并使用 NetworkPolicy 限制监控访问；不要为 `/metrics` 增加公网 Ingress。详见 `observability.md`。

## 初始化管理员

普通注册不会获得管理员权限。创建短期 bootstrap Secret 并运行一次性 Job：

```bash
kubectl create secret generic go-user-system-bootstrap \
  -n go-user-system \
  --from-literal=BOOTSTRAP_ADMIN_USERNAME=admin \
  --from-literal=BOOTSTRAP_ADMIN_PASSWORD='replace-with-a-strong-password'
kubectl apply -f k8s/bootstrap-admin-job.example.yaml
kubectl wait --for=condition=complete job/go-user-system-bootstrap-admin -n go-user-system --timeout=300s
kubectl logs job/go-user-system-bootstrap-admin -n go-user-system
kubectl delete job/go-user-system-bootstrap-admin secret/go-user-system-bootstrap -n go-user-system
```

示例清单默认开放普通注册。需要关闭时，把 `k8s/configmap.yaml` 中两处 `registration`/`REGISTRATION_ENABLED` 改为 `false`，应用 ConfigMap 后重启后端 Deployment；当前发布前端仍会显示注册入口，提交时会由后端拒绝请求，因此正式产品可按策略同步定制前端入口。

## 升级

1. 把前后端镜像标签改为同一个新版本。
2. 更新 migration Job 名称，确保 Kubernetes 创建新 Job；Makefile 直接等待清单中的 Job，不再重复硬编码名称。当前 Job 对 `rc.3` 的 Goose CLI 与后续镜像的内置 `migrate up` 命令做兼容选择，升级镜像后应验证实际执行分支。
3. 先应用基础设施并等待 migration 成功。
4. 再滚动更新前后端。
5. 验证 `/readyz`、登录和管理员权限。

## 生产差异

仓库内 MySQL 和 Redis 适合演示及单集群自托管。生产环境建议使用托管服务、网络策略、TLS、外部 Secret 管理、PodDisruptionBudget、备份与监控，并完成 `production-checklist.md`。
