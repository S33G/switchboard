# Switchboard Backend

## Requirements
- Go 1.22+
- Docker daemon accessible via local socket or TCP+TLS

## Quick Start
```bash
cd backend
export DOCKER_HOSTS="unix:///var/run/docker.sock"
go run .
```

The API listens on `http://localhost:8069`. Set `PORT` to override the default.

## Configuration
Use a YAML config file and set `CONFIG_PATH`:

```bash
export CONFIG_PATH=./config.yaml
go run .
```

Example config:
```yaml
hosts:
  - name: host1
    endpoint: unix:///var/run/docker.sock
proxy_mappings:
  api.example.com: web-api
defaults:
  base_domain: containers.example.com
  scheme: https
```

## API Endpoints
- `GET /healthz`
- `GET /api/containers`
- `GET /api/config`
- `GET /ws` (WebSocket)

## Tests
```bash
cd backend
go test ./...
```

## Swagger / OpenAPI
Swagger files are checked in at:
- `backend/docs/swagger.yaml`
- `backend/docs/swagger.json`

To regenerate them:
```bash
go install github.com/swaggo/swag/cmd/swag@latest
cd backend
swag init -g swagger.go -o docs --parseDependency --parseInternal
```
