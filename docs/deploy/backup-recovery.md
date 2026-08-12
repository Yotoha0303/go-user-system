# Compose MySQL 备份与恢复演练

该工具用于开发和验收环境，目标是证明 SQL 备份可校验、可恢复。生产环境优先使用托管 MySQL 自动备份、binlog/PITR、加密异地保留和独立恢复实例。

## 安全边界

- 备份脚本在 MySQL 容器内读取 `MYSQL_ROOT_PASSWORD`，不把密码展开到宿主机命令参数。
- 输出目录 `backups/` 被 Git 忽略。
- 每份 SQL 必须同时存在 `.manifest.json`，并校验大小和 SHA-256。
- 恢复数据库只能包含字母、数字、下划线并以 `_restore_test` 结尾。
- 恢复必须传入 `-ConfirmRestore`；脚本不会自动删除演练库。
- 工具不执行生产 restore、migration down、PVC 删除或 Secret 轮换。

## 创建备份

先确保 Compose MySQL 正在运行：

```powershell
docker compose ps mysql
powershell -NoProfile -File scripts/ops/backup-mysql.ps1
```

使用隔离 Compose project 时：

```powershell
powershell -NoProfile -File scripts/ops/backup-mysql.ps1 `
  -ComposeProjectName go-user-system-ops-test
```

输出包含：

```text
backups/go_user_system-<UTC timestamp>.sql
backups/go_user_system-<UTC timestamp>.sql.manifest.json
```

Manifest 只记录来源、时间、应用提交、文件名、大小和 SHA-256，不记录数据库密码或业务行。从 Git checkout 运行时记录当前提交；发布归档或未安装 Git 的主机使用 `application_commit: unknown`，不会影响备份与恢复。

## 恢复演练

```powershell
powershell -NoProfile -File scripts/ops/restore-mysql.ps1 `
  -BackupPath backups/go_user_system-<timestamp>.sql `
  -RestoreDatabase go_user_system_restore_test `
  -ConfirmRestore
```

脚本会：

1. 在 Docker 操作前拒绝不安全数据库名和缺失确认参数。
2. 校验 SQL 文件与 manifest。
3. 只重建指定 `_restore_test` 数据库，并只向 Compose `MYSQL_USER` 授予该演练库权限。
4. 导入 SQL，并查询表数量与 Goose migration 版本。
5. 在 `artifacts/operations/` 写入不含业务数据的 JSON 证据。

使用隔离 project 时，备份与恢复必须传入相同的 `-ComposeProjectName`。

Windows PowerShell 使用 `powershell`；安装 PowerShell 7 的 Windows 或 Linux/macOS 使用 `pwsh`。Makefile 会根据操作系统选择默认命令，也可以通过 `POWERSHELL=<command>` 覆盖。

## 验收

最低验收条件：

- SQL 文件非空，容器与宿主机 SHA-256 一致。
- 非 `_restore_test` 目标在任何 Docker 操作前被拒绝。
- 恢复数据库表数量大于 0。
- 恢复 migration 版本与源库一致。
- 使用指向恢复库的隔离应用完成登录、刷新、改密和 RBAC 测试；只检查表数量不算完整演练。
- 记录实际耗时，并在不再需要时单独审批清理演练库。

备份文件包含用户与认证状态，必须移入受控、加密存储，不能提交 Git、上传公开 Issue 或作为面试附件。
