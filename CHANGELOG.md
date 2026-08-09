# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

See `ROADMAP.md` for planned work.

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
