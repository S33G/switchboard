# Project Plan: Multi-Host Docker Monitoring & Auto-Proxy System

## Goals
- Monitor Docker containers across multiple hosts in near real-time.
- Auto-generate and reload nginx reverse-proxy configs based on container discovery.
- Provide a live React dashboard for container status and proxy mappings.
- Keep the system read-only against Docker APIs with minimal operational risk.

## Non-Goals
- Container orchestration (no scheduling, scaling, or deployment).
- Persistent historical metrics storage (initial release is in-memory only).
- User-facing auth/roles beyond read-only access (optional later phase).

## Assumptions
- Docker daemons are reachable via Unix sockets or TCP+TLS.
- nginx runs in the same container or a reachable sidecar with a shared config volume.
- Single-tenant deployment; multi-tenant access control is out of scope initially.

## Architecture Summary
**Backend (Go)**
- Docker Monitor Service: Connects to multiple Docker hosts, watches events, and maintains in-memory container state.
- Nginx Config Generator: Renders upstream/server blocks from container state and config mappings.
- WebSocket + REST API: Serves initial state and pushes live updates to clients.

**Frontend (React/TypeScript)**
- Live dashboard with container status, proxy mappings, filters, and search.
- WebSocket-driven UI updates without page refresh.

## Milestones & Phases

### Phase 0 — Project Setup & Baselines
**Deliverables**
- Repository layout and module boundaries defined.
- Baseline configs for linting/formatting/testing (Go + TS).
- Dev container or local run instructions.

### Phase 1 — Core Docker Monitoring
**Deliverables**
- Multi-host Docker client layer with connection pooling.
- Container state model and in-memory store.
- Event listener (start/stop/restart) for each host.
- REST endpoint: `GET /api/containers` returns current state.

### Phase 2 — Nginx Auto-Configuration
**Deliverables**
- Config parser for URL→container mappings + defaults.
- Nginx template rendering (upstream + server blocks).
- File writer to shared config volume.
- Nginx reload strategy (signal or exec) with retries.

### Phase 3 — WebSocket + API Layer
**Deliverables**
- WebSocket hub with broadcast of container changes.
- REST endpoints for config read/update (read-only in MVP).
- CORS + rate-limiting defaults.

### Phase 4 — React Dashboard
**Deliverables**
- Container list/grid with status indicators.
- Proxy mapping view per container.
- WebSocket hook integration (`react-use-websocket`).
- Responsive layout + search/filter UI.

### Phase 5 — Integration, Testing, and Hardening
**Deliverables**
- Multi-stage Dockerfile building Go + React.
- End-to-end integration test with multiple docker hosts.
- Load test for WebSocket fan-out.
- Deployment documentation and configuration reference.

## Workstreams

### Backend (Go)
1. **Docker Client Manager**
   - Parse `DOCKER_HOSTS` or config file into host entries.
   - Initialize per-host Docker clients.
   - Reconnect on transient failures.

2. **State Store**
   - Normalize container info: ID, name, image, state, host, ports.
   - Merge host-level data into global view.
   - Provide snapshot + delta APIs.

3. **Events Pipeline**
   - Subscribe to Docker Events API per host.
   - Update store and broadcast deltas.
   - Backoff/retry on stream errors.

4. **API Layer**
   - `GET /api/containers` (snapshot)
   - `GET /api/config` (read-only)
   - `POST /api/config` (optional later)

### Nginx Generator
1. Parse mappings from config file.
2. Generate default subdomain mapping `<container>.<base_domain>`.
3. Render nginx config template.
4. Atomically write config and reload nginx.

### Frontend (React)
1. WebSocket connection + reconnect handling.
2. Data model types + query layer (React Query + axios).
3. Container list components + status badges.
4. Filters (host, status, name, image).

### DevOps & Deployment
1. Docker-compose for local dev with fake multiple hosts.
2. TLS handling for remote Docker endpoints.
3. Volume mounts for docker sockets and nginx configs.

## Interfaces & Data Model

### Container Model (draft)
```json
{
  "id": "<container-id>",
  "name": "web-api",
  "image": "repo/web-api:latest",
  "state": "running",
  "status": "Up 12 minutes",
  "host": "host1",
  "ports": [
    {"private": 8080, "public": 49153, "type": "tcp"}
  ],
  "labels": {"com.example.proxy": "api.example.com"}
}
```

### WebSocket Events (draft)
```json
{
  "type": "container_updated",
  "payload": {
    "id": "...",
    "state": "restarting",
    "host": "host2"
  }
}
```

## Configuration

### YAML Config (draft)
```yaml
hosts:
  - name: host1
    endpoint: unix:///var/run/docker.sock
  - name: host2
    endpoint: tcp://192.168.1.10:2376
    tls:
      ca: /certs/ca.pem
      cert: /certs/cert.pem
      key: /certs/key.pem

proxy_mappings:
  api.example.com: web-api
  dashboard.example.com: admin-dashboard

defaults:
  base_domain: containers.example.com
  scheme: https
```

## Nginx Reload Strategy
- **Preferred**: `docker exec nginx nginx -s reload` for isolated reload behavior.
- **Fallback**: SIGHUP to nginx PID if exec is unavailable.

## Security Considerations
- Read-only Docker socket access where possible.
- TLS for remote Docker daemon connections.
- CORS restrictions for API + WebSocket.
- Rate limiting for WebSocket connections.

## Observability
- Structured logs for event stream, config renders, and reloads.
- Basic health endpoints: `/healthz`, `/readyz`.
- Metrics (optional): event lag, connected websocket clients.

## Testing Strategy
- Unit tests for config parsing and template rendering.
- Integration tests with multiple Docker daemons.
- WebSocket load tests and reconnect scenarios.

## Risks & Mitigations
- **Docker event stream reliability**: Reconnect with backoff and resync on reconnect.
- **Config reload errors**: Render to temp file and validate with `nginx -t` before reload.
- **Remote TLS setup complexity**: Provide docs and sample configs.

## Open Questions
- Should proxy mappings be label-based, config-based, or both?
- How to handle multiple containers matching the same hostname?
- Do we need persistent state or is in-memory sufficient for v1?

## References
- Docker Go client: https://pkg.go.dev/github.com/docker/docker/client
- nginx-proxy: https://github.com/nginx-proxy/nginx-proxy
- docker-gen pattern: http://romkevandermeulen.nl/2015/02/19/docker-gen-automatic-nginx-config-with-a-human-touch.html
- gorilla/websocket: https://www.jonathan-petitcolas.com/2015/01/27/playing-with-websockets-in-go.html
- react-use-websocket: https://github.com/robtaussig/react-use-websocket
