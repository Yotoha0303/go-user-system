# Minimum Operability Delivery Record

## Goal

Move the project from repeatable deployment to a verifiable local and acceptance-environment operations loop without representing the reference stack as production high availability.

## Problems And Changes

| Problem | Root cause | Change | Verification |
| --- | --- | --- | --- |
| A running instance could not identify its source | Build metadata was not injected into the binary or images | Added `/version`, startup fields, OCI labels, and `go_user_system_build_info` | A metadata-injected Compose image returned the expected version, commit, and build time |
| Health checks and logs could not calculate RED signals | No metrics exporter or bounded route labels | Added Prometheus client collectors and Gin route-template labels | Prometheus scraped HTTP, readiness, runtime, process, and build metrics; unmatched URLs used one `unmatched` label |
| Alerts existed only as a design document | No runnable rule configuration or validation | Added a fixed Prometheus Compose overlay and four baseline rules | `promtool` accepted the configuration and rules; stopping the app moved TargetDown to firing and restart cleared it |
| Backup recovery depended on unsafe manual commands | No checksum manifest, target guard, or evidence output | Added PowerShell backup and restore-drill scripts | A real SQL backup restored 7 tables at Goose version 6 into an isolated `_restore_test` database |
| Migration images depended on a build-time Goose CLI download | The container installed a second executable and all CLI drivers | Added the application-owned `migrate up` subcommand with migration-only DB config | Fresh Compose startup completed migration before the backend became healthy |
| Release waits and local images lost traceability | Job names and metadata were duplicated or omitted | Wait from the manifest and pass build args through Compose, Make, and release CI | Manifest validation and metadata-injected image startup passed |

## Safety Boundaries

- HTTP metric route labels use Gin templates; raw IDs, usernames, tokens, IP addresses, and query strings are not labels.
- `/metrics` is not exposed through the Kubernetes Ingress. Production access still requires network policy and a cluster monitoring system.
- Restore drills reject targets that do not end in `_restore_test`, require explicit confirmation, verify the manifest first, and never delete the restored database automatically.
- Backup data and drill artifacts are Git-ignored and must not be attached to public issues or interview materials.
- The repository does not provide Alertmanager receivers, managed MySQL PITR, data-store failover, centralized logs, traces, real TLS, or an on-call rotation.

## Evidence Summary

- Prometheus config: one scrape target and four accepted rules.
- Runtime: frontend, backend, MySQL, Redis, migration, and Prometheus started in an isolated Compose project.
- Alert: `GoUserSystemTargetDown` reached firing after the configured delay and cleared after application recovery.
- Backup: non-empty SQL with matching container/host SHA-256 and JSON manifest.
- Restore: 7 tables, migration version 6, no temporary restore file left in the container; the restored application passed login, refresh, password change, logout, token revocation, and RBAC denial checks.
- Destructive guard: a non-`_restore_test` database name failed before any Docker operation.

Detailed commands and operating procedures are in [observability.md](deploy/observability.md) and [backup-recovery.md](deploy/backup-recovery.md).

## Review Hardening

- HTTP metrics wrap the client-facing timeout handler; timeout requests are counted once as 503 and release in-flight state when the response is sent.
- The checked-in Kubernetes Job uses Goose when running the pinned rc.3 image and automatically selects the built-in migration command for later images.
- Backup metadata treats Git as optional, so release-archive hosts still receive a valid manifest with an `unknown` commit.
