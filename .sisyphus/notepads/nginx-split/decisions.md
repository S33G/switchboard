# Decisions: nginx-split

This file tracks architectural choices made during implementation.

---

## [2026-01-27T02:43:14.020Z] Initial Plan Decisions

- **nginx reload**: Go uses Docker API `ContainerExecCreate`/`ContainerExecAttach` to exec `nginx -t` and `nginx -s reload` in the separate nginx container
- **UI serving**: Enhanced Go `http.FileServer` with SPA fallback middleware and `/_next/` cache headers. No embedding.
- **Port change**: Go serves on port 80 by default (was 8069 behind nginx). API_PORT env var still allows override.
- **Shared config**: Docker named volume shared between Go (writer) and nginx (reader) for generated vhost config.
- **Backward compat**: `NGINX_ENABLED` still works but logs a deprecation warning pointing to `NGINX_CONF_GEN_ENABLED`.

---
