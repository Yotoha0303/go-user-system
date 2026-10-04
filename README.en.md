# go-user-system

A self-hosted full-stack authentication and RBAC system built with Go, Gin, GORM, MySQL, Redis, React, TypeScript, and Vite.

[中文 README](README.md)

## Project Status

Current public delivery version: `v1.0.0-rc.3`. This is a release candidate for learning, extension, and non-critical validation. Complete `docs/deploy/production-checklist.md` before production use.

## Core Capabilities

- JWT Access / Refresh tokens with HttpOnly Cookie, hashing, rotation, and replay detection
- Global session invalidation with `auth_version` and Access JTI revocation
- Web Locks for multi-tab refresh serialization
- Redis account/IP login-failure rate limiting with fail-closed behavior
- Secure Cookie and trusted-proxy validation
- RBAC roles, permissions, and permission middleware
- One-time administrator bootstrap and configurable public registration
- Swagger, health/readiness endpoints, structured logs, Request ID, timeout, graceful shutdown
- Prometheus HTTP/runtime/readiness metrics and alert rules
- SHA-256 verified MySQL backup and isolated restore drills
- Docker Compose, Kubernetes, GitHub Actions, CodeQL, Dependabot, and GHCR release pipelines

## Quick Start

```bash
cp .env.example .env
docker compose up -d --build --wait
```

Bootstrap the first administrator:

```bash
export BOOTSTRAP_ADMIN_USERNAME=admin
export BOOTSTRAP_ADMIN_PASSWORD='replace-with-a-strong-password'
docker compose run --rm -e BOOTSTRAP_ADMIN_USERNAME -e BOOTSTRAP_ADMIN_PASSWORD app bootstrap-admin
```

## Technology Stack

| Area | Technology |
| --- | --- |
| Backend | Go 1.25.13, Gin, GORM, bcrypt, JWT |
| Data | MySQL 8.4, Redis 7.4, Goose |
| Frontend | React 18, TypeScript 5, Vite 8, Redux Toolkit, Tailwind CSS |
| Testing | Go testing, httptest, miniredis, MySQL integration, Vitest, Testing Library, Playwright |
| Delivery | Docker, Compose, Kubernetes, GitHub Actions, GHCR |
| Security | govulncheck, npm audit, CodeQL, Dependabot, secret scanning |

## Operations

Local observability uses Prometheus with TargetDown, NotReady, High5xxRate, and HighP95Latency rules. The repository also provides MySQL backup with checksum/manifest and isolated restore drills restricted to databases ending in `_restore_test`.

These capabilities are for local and validation environments. Production still requires external alert routing, managed backup/PITR, and MySQL/Redis high availability.

## Documentation

See the Chinese README and `docs/` for the complete API, deployment, testing, security, backup/recovery, and release documentation.

## License

MIT License.
