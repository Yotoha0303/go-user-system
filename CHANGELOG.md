# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

See `ROADMAP.md` for planned work.

## [1.0.0-rc.3] - 2026-08-10

### Changed

- Raised the password policy to at least 12 characters with the bcrypt 72-byte ceiling enforced.
- Added explicit runtime environment, secure refresh-cookie, and trusted-proxy configuration.
- Kubernetes defaults now require TLS and production-safe Redis-backed authentication state.

### Security

- Browser logout now presents and revokes the current Access Token JTI.
- Refresh rotation is serialized across browser tabs with the Web Locks API.
- Login IP rate limiting now honors forwarded addresses only from configured trusted proxies.
- Disabled, missing, and wrong-password login attempts now share the same external response and rate-limit path.
- Configuration rejects unsupported RS256 instead of silently issuing HS256 tokens.

## [1.0.0-rc.2] - 2026-08-10

### Changed

- Synchronized public delivery records and deployment defaults after the initial release.

### Security

- Updated `filippo.io/edwards25519` to 1.1.1 to resolve the low-severity Dependabot alert reported after `rc.1`.

## [1.0.0-rc.1] - 2026-08-10

### Added

- React frontend packaged in the main repository with a production Nginx image.
- Full-stack Compose startup, automatic one-shot migrations, and Playwright authentication flow.
- Explicit `bootstrap-admin` command and configurable public registration.
- GHCR backend/frontend image release workflow with provenance, SBOM, binaries, and checksums.
- CodeQL, Dependabot, Go vulnerability scanning, npm auditing, and Kubernetes manifest validation.
- Public contribution, security, conduct, issue, pull request, deployment, and roadmap documentation.

### Changed

- Updated Go to 1.25.12 and patched vulnerable backend and frontend dependencies.
- Registered users now receive only the `user` role; the first visitor can no longer claim administrator access.
- Kubernetes uses fixed release images, a non-root application account, a singleton migration Job, and path-preserving ingress routing.

### Security

- Removed all known reachable Go vulnerabilities reported by `govulncheck`.
- Removed all npm audit findings in production and development dependencies.
