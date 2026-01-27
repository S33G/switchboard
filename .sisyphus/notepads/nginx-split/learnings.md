# Learnings: nginx-split

This file tracks conventions, patterns, and wisdom discovered during implementation.

---

## [2026-01-27] Task 1: SPA file server + port change

- Go's `http.Dir.Open()` is the clean way to check file existence before deciding on SPA fallback — it respects the same root dir as `http.FileServer`.
- `http.ServeFile` is used for the SPA fallback (serving index.html for unknown paths) rather than the file server, because the file server would try to serve the original path.
- Go's `ServeMux` matches more specific patterns first, so API routes (`/api/`, `/healthz`, `/ws`) registered before the catch-all `/` handler naturally take priority — no special ordering needed.
- Cache headers must be set BEFORE calling `ServeHTTP` or `ServeFile` since those write the response.
- The `f.Close()` after `fs.Open()` is important — we only open to check existence, not to read.
- Default port changed from 8069 to 80 — Go now serves directly instead of behind nginx for UI.


## [2026-01-27] Task 6: Rename NGINX_ENABLED → NGINX_CONF_GEN_ENABLED

- Backward compatibility pattern: Check new env var first, fall back to old var with deprecation log if old var is set.
- The fallback logic uses `os.Getenv("NGINX_ENABLED") != ""` to detect if the old var was explicitly set, then logs deprecation warning before using it.
- Updated 4 files: `backend/nginx_gen.go` (with backward compat), `deploy/entrypoint.sh`, `Dockerfile`, `compose.yml`.
- Grep verification confirms only backward compat code references old `NGINX_ENABLED` name — no other usages remain.
- Go build and tests pass cleanly with no errors or warnings.
- The new name `NGINX_CONF_GEN_ENABLED` better reflects purpose: controls nginx config generation, not nginx itself (which runs in separate container).

## Task 4: Docker API-based nginx reload (2026-01-27)

### Key findings:
- Docker SDK v28.5.2 uses `container.ExecOptions` (not `types.ExecConfig`) for exec create
- `ExecCreateResponse` is an alias for `common.IDResponse` with `.ID` field
- `ContainerExecAttach` returns `types.HijackedResponse` — must use `stdcopy.StdCopy` to demux stdout/stderr
- `ContainerExecInspect` returns `container.ExecInspect` with `.ExitCode` field
- Import aliases needed: `dockercontainer` and `dockerclient` to avoid collision with existing `container` and `client` imports in the package
- `dockerclient.APIClient` interface used for the parameter type (not concrete `*client.Client`) — enables testability
- `os/exec` import removed from nginx_gen.go (only used for nginx commands, now replaced)
- `os/exec` still used in `docker_context.go` for `docker context inspect` — separate concern
- Env var `NGINX_CONF_GEN_ENABLED` was already renamed from `NGINX_ENABLED` (with backward compat) in a prior task
- Graceful degradation: if Docker client init fails or is nil, config is still written to shared volume, just not reloaded

## [2026-01-27] Task 5: Integration verification

### Key Findings:
- Both containers (`switchboard` and `switchboard-nginx`) build and start successfully
- Shared volume `switchboard_nginx-conf` is created correctly
- Go backend serves UI on port 80 - HTML loads correctly
- All API endpoints work:
  - `/api/containers` returns JSON array ✓
  - `/api/config` returns JSON object ✓
  - `/healthz` returns 200 ✓
- SPA fallback works perfectly - `/nonexistent-route` returns HTML with 200 status ✓
- Cache headers work for `/_next/` assets: `Cache-Control: public, max-age=31536000, immutable` ✓
- nginx container runs and successfully reloads via Docker API exec:
  - nginx logs show SIGHUP signal received and graceful reload ✓
  - Generated config at `/etc/nginx/conf.d/switchboard.generated.conf` contains proper vhost blocks ✓
- No errors in either container logs ✓
- nginx is accessible on port 8090 (mapped from container port 80)

### Architecture Validation:
- Two-container topology working as designed
- Go serves UI+API directly on port 80 (no nginx in front)
- nginx runs separately for reverse proxy duties only
- Docker API exec successfully triggers nginx reload across container boundary
- Shared volume enables Go to write config that nginx reads

### Test Results Summary:
```
✓ docker compose up --build -d - Both containers started
✓ docker compose ps - Both "Up"
✓ curl http://localhost/ - Returns HTML (Switchboard UI)
✓ curl http://localhost/api/containers - Returns JSON array
✓ curl http://localhost/api/config - Returns JSON object
✓ curl http://localhost/healthz - Returns 200
✓ curl -I http://localhost/_next/...css - Has cache headers
✓ curl http://localhost/nonexistent - Returns HTML (SPA fallback)
✓ docker volume ls | grep nginx-conf - Volume exists
✓ docker exec switchboard-nginx cat /etc/nginx/conf.d/switchboard.generated.conf - Config generated
✓ docker compose logs nginx - Shows successful reload (SIGHUP)
✓ docker compose logs switchboard - No errors
```

All acceptance criteria met. Integration test PASSED.

## [2026-01-27] Task 7: Documentation updates

### Files Updated:
1. **README.md**:
   - Changed `NGINX_ENABLED` → `NGINX_CONF_GEN_ENABLED` in compose example
   - Updated environment variables table:
     * `API_PORT` default: `8069` → `80`
     * `NGINX_ENABLED` → `NGINX_CONF_GEN_ENABLED` (updated description)
     * Added `NGINX_CONTAINER_NAME` (default: `switchboard-nginx`)
   
2. **examples/README.md**:
   - Line 17: `NGINX_ENABLED=1` → `NGINX_CONF_GEN_ENABLED=1`

3. **backend/README.md**:
   - Added note: "Go backend serves the Switchboard UI directly on port 80 (configurable via API_PORT)"
   - Clarified: "nginx runs in a separate container for reverse proxy duties"

4. **plan.md**:
   - Updated "Deployment Topology" section (lines 47-52):
     * Go backend container serves UI+API on port 80
     * nginx container is separate reverse proxy
     * Shared volume for config generation
     * Docker API exec for reload signals

### Verification:
- ✓ All `NGINX_ENABLED` references removed from documentation
- ✓ All files use `NGINX_CONF_GEN_ENABLED` consistently
- ✓ `API_PORT` default documented as `80`
- ✓ `NGINX_CONTAINER_NAME` documented with default value
- ✓ Two-container architecture properly described
- ✓ No broken markdown formatting

### Commit:
- Commit e834821: "docs: update documentation for nginx split and NGINX_CONF_GEN_ENABLED rename"
- 4 files changed, 15 insertions(+), 9 deletions(-)

All documentation updates complete and committed.
