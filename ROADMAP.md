# Roadmap

## 1.0 stable

- Validate `v1.0.0-rc.3` on a clean host and a real Kubernetes ingress with TLS.
- Complete restore testing for MySQL and Redis persistent data.
- Add an administrative user search/list workflow before stable release.
- Optimize the backend multi-architecture image build to avoid compiling Go tools under QEMU.
- Resolve release-candidate feedback and publish `v1.0.0`.

## Later

- Login device metadata and user-visible session management.
- Revoke one device or all other sessions.
- Optional email verification and password reset.
- OpenTelemetry traces and deployment dashboards.
- RS256/JWKS signing and key rotation for multi-service deployments.

Detailed session-management design remains in `docs/iteration-plan-redis-device-token.md`.
