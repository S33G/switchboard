# Project Plan: Multi-Host Docker Monitoring & Auto-Proxy System

## Executive Summary
Build a Go service that monitors Docker containers across multiple hosts, generates nginx reverse-proxy config dynamically, and broadcasts live status updates to a React dashboard over WebSockets. The system is read-only against Docker APIs and focuses on near real-time visibility and safe, automated proxy updates.

## Goals
- Monitor containers across multiple Docker hosts in near real-time.
- Generate and reload nginx reverse-proxy configs based on discovered containers.
- Provide a live dashboard for container status and proxy mappings.
- Keep Docker access read-only where possible and minimize operational risk.

## Non-Goals
- Orchestration (scheduling, scaling, or deployment management).
- Persistent historical metrics (MVP uses in-memory state only).
- Multi-tenant auth/roles beyond simple read-only access.

## Success Criteria
- Container state updates appear in the UI within 1–3 seconds of Docker events.
- nginx config reloads occur within 5 seconds of relevant container changes.
- System can track at least 5 hosts and 200 containers without errors.
- WebSocket fan-out handles 100 concurrent dashboard clients.

## Assumptions
- Docker daemons are reachable via Unix socket or TCP+TLS.
- nginx runs in the same pod/host or sidecar with shared config volume.
- Single-tenant deployment; minimal access controls in MVP.

## Architecture Overview

### Components
**Backend (Go)**
- Docker Monitor: connects to multiple Docker hosts and watches events.
- State Store: in-memory container state by host, exposes snapshot + delta.
- Nginx Generator: renders templates and triggers reloads.
- API + WebSocket Server: initial state via REST, live updates via WS.

**Frontend (React/TypeScript)**
- Dashboard with container list, status badges, and proxy mappings.
- WebSocket-driven updates without page reload.

### Data Flow
1. Docker host events → event listener per host.
2. Events update state store → broadcast deltas.
3. Nginx config generator renders → write + validate + reload.
4. Clients load snapshot → subscribe to deltas via WebSocket.

### Deployment Topology
- Monitoring service container with optional mounts:
  - Docker sockets (local/remote)
  - nginx config volume (shared)
- nginx container receives config updates + reload signal.
- React build served via nginx alongside reverse proxy.

## Interfaces & Data Model

### REST Endpoints (v1)
- `GET /api/containers`: snapshot of container states.
- `GET /api/config`: current mapping config.
- `POST /api/config`: optional (future) mapping updates.

### WebSocket Events (v1)
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

### Config Schema (draft)
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

## Milestones & Phases

### Phase 0 — Project Setup
**Deliverables**
- Repository layout and module boundaries defined.
- Baseline lint/format/test config (Go + TS).
- Local dev instructions.

### Phase 1 — Core Docker Monitoring
**Deliverables**
- Multi-host Docker client layer.
- In-memory state store and snapshot endpoint.
- Event listeners for start/stop/restart.

**Exit Criteria**
- Events update state within 3 seconds.
- Snapshot endpoint responds < 200ms for 200 containers.

### Phase 2 — Nginx Auto-Configuration
**Deliverables**
- Config parser for URL → container mappings.
- Nginx template rendering and validation (`nginx -t`).
- Reload strategy with retry/backoff.

**Exit Criteria**
- Config reloads succeed after container changes.
- Invalid templates do not replace current config.

### Phase 3 — API + WebSocket Layer
**Deliverables**
- WebSocket hub with broadcast of deltas.
- REST endpoints for config + state.
- CORS defaults + basic rate limiting.

**Exit Criteria**
- 100 concurrent clients receive updates without disconnects.

### Phase 4 — React Dashboard
**Deliverables**
- Container list/grid with status badges.
- Proxy mapping view per container.
- Filtering/search and host grouping.

**Exit Criteria**
- UI updates within 1–3 seconds of events.

### Phase 5 — Integration & Hardening
**Deliverables**
- Multi-stage Dockerfile for Go + React.
- Integration tests for multi-host setup.
- Load tests for WebSocket fan-out.
- Deployment docs and configuration reference.

## Workstreams

### Backend (Go)
- Docker client manager with host pooling and retries.
- Event ingestion pipeline with reconnect/resync.
- State store with snapshot + delta APIs.
- WebSocket hub for broadcasts.

### Nginx Generator
- Template-based rendering with upstream/server blocks.
- Default subdomain fallback for unmapped containers.
- Atomic file writes + validation before reload.

### Frontend (React)
- `react-use-websocket` connection + reconnection logic.
- React Query snapshot + in-memory live updates.
- Filtering/search UI.

### DevOps & Deployment
- Docker-compose for local dev (multi-host simulation).
- TLS configuration guidance for remote daemons.
- Volume mount strategy for sockets and nginx configs.

## Testing & Validation
- Unit tests: config parsing, template rendering, mapping rules.
- Integration tests: multi-host events → state → nginx reload.
- Load tests: WebSocket fan-out and reconnect storms.

## Security Considerations
- Read-only Docker socket access where possible.
- TLS for remote Docker daemon connections.
- CORS restrictions for API and WS.
- Rate limiting on WS connections.

## Observability
- Structured logs for event stream, render, reload.
- Health endpoints: `/healthz`, `/readyz`.
- Optional metrics: event lag, connected WS clients.

## Risks & Mitigations
- **Event stream drops**: reconnect with backoff + resync on reconnect.
- **Bad config reload**: validate with `nginx -t` before swap.
- **TLS setup complexity**: provide examples and validation steps.
- **Host scale**: limit per-host connections and batch refresh.

## Open Questions
- Should proxy mappings use labels, config file, or both?
- How to handle multiple containers matching a hostname?
- Is in-memory state sufficient for v1, or do we need optional Redis?

## References
- Docker Go client: https://pkg.go.dev/github.com/docker/docker/client
- Docker Engine API: https://docs.docker.com/engine/api/
- nginx-proxy: https://github.com/nginx-proxy/nginx-proxy
- docker-gen pattern: http://romkevandermeulen.nl/2015/02/19/docker-gen-automatic-nginx-config-with-a-human-touch.html
- gorilla/websocket: https://github.com/gorilla/websocket
- react-use-websocket: https://github.com/robtaussig/react-use-websocket
