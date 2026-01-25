# 🔀 Switchboard

> **Central nginx reverse proxy for all your Docker containers across multiple hosts**

Switchboard automatically discovers running containers across your Docker infrastructure and dynamically configures nginx to route traffic to them. Perfect for homelabs, dev environments, and multi-host container deployments.

[![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)](https://github.com/S33G/switchboard/pkgs/container/switchboard)
[![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-black?style=for-the-badge&logo=next.js&logoColor=white)](https://nextjs.org/)

---

## ✨ Features

- 🔍 **Auto-discovery**: Automatically detects containers across multiple Docker hosts
- 🌐 **Dynamic routing**: Generates nginx configs on-the-fly for `<container>.<host>.<domain>`
- 🔌 **Multi-host support**: Connect via Unix socket, TCP, TLS, SSH, or Docker contexts
- 📊 **Web UI**: Beautiful Next.js dashboard to monitor all containers
- 🔄 **Real-time updates**: WebSocket-powered live container state changes
- 🎯 **Custom mappings**: Override default routing with custom domain mappings
- 🐳 **Container-native**: Ships as a single Docker image with everything included

---

## 🚀 Quick Start

### Using Docker

Pull and run the latest image:

```bash
docker pull ghcr.io/s33g/switchboard:latest

docker run -d \
  --name switchboard \
  -p 80:80 \
  -p 8069:8069 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/config.yaml:/config/config.yaml:ro \
  -e CONFIG_PATH=/config/config.yaml \
  ghcr.io/s33g/switchboard:latest
```

### Using Docker Compose

```bash
# Create your config file
cat > config.yaml <<EOF
hosts:
  - name: local
    endpoint: unix:///var/run/docker.sock
defaults:
  base_domain: containers.local
  scheme: http
EOF

# Start Switchboard
docker compose up -d
```

**Access the UI**: http://localhost  
**Access the API**: http://localhost:8069

---

## 📦 Installation Options

### Option 1: Docker Run (Recommended)

**Basic setup with local Docker socket:**

```bash
docker run -d \
  --name switchboard \
  -p 80:80 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/s33g/switchboard:latest
```

**Production setup with custom config:**

```bash
docker run -d \
  --name switchboard \
  --restart unless-stopped \
  -p 80:80 \
  -p 443:443 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/config.yaml:/config/config.yaml:ro \
  -v $(pwd)/ssl:/etc/nginx/ssl:ro \
  -e CONFIG_PATH=/config/config.yaml \
  -e TZ=America/New_York \
  ghcr.io/s33g/switchboard:latest
```

### Option 2: Docker Compose

Create `compose.yml`:

```yaml
services:
  switchboard:
    image: ghcr.io/s33g/switchboard:latest
    container_name: switchboard
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      # - "8069:8069"  # Optional: Direct API access
    environment:
      CONFIG_PATH: /config/config.yaml
      NGINX_ENABLED: "1"
      TZ: America/New_York
    volumes:
      # Config file
      - ./config.yaml:/config/config.yaml:ro
      
      # Local Docker socket
      - /var/run/docker.sock:/var/run/docker.sock
      
      # Docker contexts (for remote hosts)
      - ~/.docker:/root/.docker:ro
      
      # SSH keys (if using SSH contexts)
      - ~/.ssh:/root/.ssh:ro
      
      # SSL certificates (optional)
      # - ./ssl:/etc/nginx/ssl:ro
```

Start with:

```bash
docker compose up -d
```

### Option 3: Build from Source

```bash
# Clone the repository
git clone https://github.com/S33G/switchboard.git
cd switchboard

# Build and run
docker compose up -d --build
```

---

## ⚙️ Configuration

Switchboard uses a YAML config file to define Docker hosts, routing rules, and defaults.

### Minimal Config

```yaml
hosts:
  - name: local
    endpoint: unix:///var/run/docker.sock

defaults:
  base_domain: containers.local
  scheme: http
```

### Multi-Host Config

```yaml
hosts:
  # Local Docker host
  - name: homelab
    endpoint: unix:///var/run/docker.sock
  
  # Remote host via SSH
  - name: remote-server
    endpoint: ssh://user@192.168.1.100
  
  # Docker context
  - name: cloud
    endpoint: context://aws-prod

# Map host names to network addresses (for nginx upstream)
host_addresses:
  homelab: 192.168.1.50
  remote-server: 192.168.1.100
  cloud: cloud.example.com

# Custom domain mappings
proxy_mappings:
  api.example.com: homelab/api-container
  app.example.com: remote-server/web-app

defaults:
  base_domain: containers.home
  scheme: https
```

### Configuration Reference

| Field | Type | Description |
|-------|------|-------------|
| `hosts` | array | List of Docker hosts to monitor |
| `hosts[].name` | string | Unique name for the Docker host |
| `hosts[].endpoint` | string | Docker daemon endpoint (unix://, tcp://, ssh://, context://) |
| `host_addresses` | map | Maps host names to network addresses reachable from nginx |
| `proxy_mappings` | map | Custom domain → container mappings |
| `defaults.base_domain` | string | Base domain for auto-generated container URLs |
| `defaults.scheme` | string | Default scheme for generated URLs (http or https) |

---

## 🌍 How Routing Works

Switchboard generates nginx configurations to route traffic to your containers.

### Default Routing Pattern

```
<container-name>.<host-name>.<base-domain>
```

**Example:**

- Container: `nginx-web`
- Host: `homelab`
- Base domain: `containers.home`
- Generated URL: `nginx-web.homelab.containers.home`

### Custom Mappings

Override the default pattern with `proxy_mappings`:

```yaml
proxy_mappings:
  api.example.com: homelab/backend-api
  app.example.com: cloud/frontend
```

Now:
- `api.example.com` → routes to `backend-api` on `homelab`
- `app.example.com` → routes to `frontend` on `cloud`

---

## 🔌 Docker Endpoint Types

Switchboard supports multiple ways to connect to Docker daemons:

### Unix Socket (Local)

```yaml
hosts:
  - name: local
    endpoint: unix:///var/run/docker.sock
```

**Requirements:**
- Mount the Docker socket: `-v /var/run/docker.sock:/var/run/docker.sock`

### TCP (Remote, Unencrypted)

```yaml
hosts:
  - name: remote
    endpoint: tcp://192.168.1.100:2375
```

⚠️ **Security Warning:** Only use in trusted networks. No encryption.

### TCP + TLS (Remote, Encrypted)

```yaml
hosts:
  - name: secure-remote
    endpoint: tcp://192.168.1.100:2376
```

**Requirements:**
- Docker daemon configured with TLS
- Mount certificates to `/root/.docker/` or set `DOCKER_CERT_PATH`

### SSH (Remote, via SSH)

```yaml
hosts:
  - name: ssh-host
    endpoint: ssh://user@192.168.1.100
```

**Requirements:**
- Mount SSH keys: `-v ~/.ssh:/root/.ssh:ro`
- User must have Docker socket access on remote host

### Docker Context (Any)

```yaml
hosts:
  - name: my-context
    endpoint: context://my-context-name
```

**Requirements:**
- Create Docker context first: `docker context create my-context --docker "host=ssh://user@host"`
- Mount Docker config: `-v ~/.docker:/root/.docker:ro`

---

## 🎯 Use Cases

### Homelab Setup

Route all your homelab containers through a single entry point:

```yaml
hosts:
  - name: pi1
    endpoint: ssh://pi@192.168.1.101
  - name: pi2
    endpoint: ssh://pi@192.168.1.102
  - name: nas
    endpoint: tcp://192.168.1.200:2376

host_addresses:
  pi1: 192.168.1.101
  pi2: 192.168.1.102
  nas: 192.168.1.200

proxy_mappings:
  home.example.com: pi1/homepage
  plex.example.com: nas/plex

defaults:
  base_domain: lab.home
  scheme: https
```

**Access containers:**
- `home.example.com` → Homepage on pi1
- `plex.example.com` → Plex on NAS
- `portainer.pi1.lab.home` → Portainer on pi1
- `grafana.pi2.lab.home` → Grafana on pi2

### Development Environment

Automatically route dev containers with zero configuration:

```yaml
hosts:
  - name: dev
    endpoint: unix:///var/run/docker.sock

defaults:
  base_domain: dev.local
  scheme: http
```

Start any container and it's instantly accessible:

```bash
docker run -d --name my-api my-api-image
# Automatically available at: my-api.dev.dev.local
```

### Multi-Cloud Deployment

Manage containers across different cloud providers:

```yaml
hosts:
  - name: aws
    endpoint: context://aws-prod
  - name: gcp
    endpoint: context://gcp-prod
  - name: azure
    endpoint: context://azure-prod

proxy_mappings:
  api.example.com: aws/api-service
  web.example.com: gcp/frontend
  admin.example.com: azure/admin-panel

defaults:
  base_domain: prod.example.com
  scheme: https
```

---

## 🖥️ API Reference

Switchboard exposes a REST API and WebSocket for real-time updates.

### REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Health check endpoint |
| `GET` | `/api/containers` | List all containers across all hosts |
| `GET` | `/api/config` | Get current configuration |
| `GET` | `/ws` | WebSocket endpoint for real-time updates |

### Example: List Containers

```bash
curl http://localhost:8069/api/containers | jq
```

**Response:**

```json
[
  {
    "id": "abc123",
    "name": "nginx-web",
    "host": "homelab",
    "state": "running",
    "ports": [{"private": 80, "public": 8080}],
    "labels": {"app": "web"},
    "created": "2026-01-20T10:30:00Z"
  }
]
```

### WebSocket Updates

Connect to `/ws` for real-time container state changes:

```javascript
const ws = new WebSocket('ws://localhost:8069/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Container update:', data);
};
```

---

## 🔧 Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CONFIG_PATH` | - | Path to YAML config file |
| `API_PORT` | `8069` | Backend API port |
| `NGINX_ENABLED` | `1` | Enable nginx config generation |
| `NGINX_RELOAD_DEBOUNCE` | `1500ms` | Debounce interval for nginx reloads |
| `NGINX_GENERATED_CONF` | `/etc/nginx/conf.d/switchboard.generated.conf` | Path to generated nginx config |
| `TZ` | `UTC` | Timezone for logs |

---

## 🌐 DNS Setup

To access containers via their generated URLs, configure DNS:

### Option 1: Wildcard DNS (Recommended)

Create wildcard DNS records for each host:

```
*.homelab.containers.home  → <switchboard-ip>
*.cloud.containers.home    → <switchboard-ip>
```

### Option 2: Local Hosts File

For testing, add entries to `/etc/hosts`:

```
127.0.0.1  nginx-web.homelab.containers.local
127.0.0.1  api.homelab.containers.local
```

### Option 3: dnsmasq

For local development, use dnsmasq:

```bash
# Add to dnsmasq.conf
address=/containers.local/192.168.1.50
```

---

## 🔐 Security Considerations

### Production Recommendations

1. **Use TLS**: Enable HTTPS and mount SSL certificates
2. **Restrict API access**: Use firewall rules to limit API port exposure
3. **Docker socket security**: Only mount Docker socket if necessary
4. **SSH key permissions**: Ensure SSH keys are read-only (`400` or `600`)
5. **Network isolation**: Run Switchboard in a dedicated Docker network

### Example: HTTPS Setup

```yaml
services:
  switchboard:
    image: ghcr.io/s33g/switchboard:latest
    ports:
      - "443:443"
    environment:
      CONFIG_PATH: /config/config.yaml
    volumes:
      - ./config.yaml:/config/config.yaml:ro
      - ./ssl/cert.pem:/etc/nginx/ssl/cert.pem:ro
      - ./ssl/key.pem:/etc/nginx/ssl/key.pem:ro
      - /var/run/docker.sock:/var/run/docker.sock
```

Update nginx config to use SSL:

```bash
# Add to deploy/nginx/00-switchboard.conf
server {
  listen 443 ssl;
  ssl_certificate /etc/nginx/ssl/cert.pem;
  ssl_certificate_key /etc/nginx/ssl/key.pem;
  # ... rest of config
}
```

---

## 🐛 Troubleshooting

### Containers not showing up

**Check logs:**

```bash
docker logs switchboard
```

**Verify Docker socket access:**

```bash
docker exec switchboard docker ps
```

**Verify config:**

```bash
curl http://localhost:8069/api/config
```

### nginx errors

**Check nginx logs:**

```bash
docker exec switchboard cat /var/log/nginx/error.log
```

**Reload nginx manually:**

```bash
docker exec switchboard nginx -s reload
```

### WebSocket connection fails

**Check API port is accessible:**

```bash
curl http://localhost:8069/healthz
```

**Verify WebSocket upgrade:**

```bash
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  http://localhost:8069/ws
```

---

## 📚 Advanced Usage

### Using Docker Contexts

Create contexts for remote hosts:

```bash
# SSH context
docker context create remote-host \
  --docker "host=ssh://user@192.168.1.100"

# TCP + TLS context
docker context create secure-host \
  --docker "host=tcp://192.168.1.200:2376"

# List contexts
docker context ls
```

Update `config.yaml`:

```yaml
hosts:
  - name: remote
    endpoint: context://remote-host
  - name: secure
    endpoint: context://secure-host
```

### Generating Initial Config

Generate a config file from environment variables:

```bash
export DOCKER_HOSTS="unix:///var/run/docker.sock"
docker run --rm ghcr.io/s33g/switchboard:latest \
  switchboard -init-config - > config.yaml
```

### Health Checks

Add health checks to your compose file:

```yaml
services:
  switchboard:
    image: ghcr.io/s33g/switchboard:latest
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8069/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

---

## 🛠️ Development

### Build from Source

```bash
# Clone repository
git clone https://github.com/S33G/switchboard.git
cd switchboard

# Build with Docker
docker build -t switchboard:local .

# Or build components separately
cd backend && go build -o switchboard .
cd ../ui && pnpm install && pnpm build
```

### Run Backend Locally

```bash
cd backend
export CONFIG_PATH=../config.yaml
go run .
```

### Run Frontend Locally

```bash
cd ui
pnpm install
pnpm dev
```

### Run Tests

```bash
# Backend tests
cd backend
go test ./...

# Frontend tests (if available)
cd ui
pnpm test
```

### API Documentation

Swagger/OpenAPI docs are available at:
- `backend/docs/swagger.yaml`
- `backend/docs/swagger.json`

Regenerate with:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
cd backend
swag init -g swagger.go -o docs --parseDependency --parseInternal
```

---

## 🤝 Contributing

Contributions are welcome! Please open an issue or pull request.

### Development Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Run tests: `go test ./... && pnpm test`
5. Commit: `git commit -m "feat: add amazing feature"`
6. Push: `git push origin feature/amazing-feature`
7. Open a pull request

---

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

---

## 🙏 Acknowledgments

Built with:
- [Go](https://go.dev/) - Backend API
- [Next.js](https://nextjs.org/) - Frontend UI
- [nginx](https://nginx.org/) - Reverse proxy
- [Docker](https://docker.com/) - Container platform

---

## 📞 Support

- 🐛 **Issues**: [GitHub Issues](https://github.com/S33G/switchboard/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/S33G/switchboard/discussions)
- 📖 **Documentation**: [Wiki](https://github.com/S33G/switchboard/wiki)

---

**Made with ❤️ for the homelab community**
