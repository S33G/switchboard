# Switchboard Backend

This service watches one-or-more Docker daemons (local socket, TCP, TLS, SSH) and exposes their container state over an HTTP + WebSocket API.

## Configure Docker hosts via Docker Context

Switchboard accepts host endpoints in two places:

- `DOCKER_HOSTS` (comma-separated endpoints)
- `CONFIG_PATH` (YAML config file)

In addition to raw endpoints like `unix://`, `tcp://`, `ssh://`, you can use:

- `context://<context-name>`

When `context://...` is used, Switchboard runs `docker context inspect <name> --format '{{json .}}'` and uses the context's `Endpoints.docker.Host`.

### 1) Create a Docker context that points at a remote daemon

Example (SSH):

```bash
docker context create my-remote \
  --description "Remote engine" \
  --docker "host=ssh://user@192.168.1.50"

docker context inspect my-remote --format '{{json .}}'
```

Docker docs:

- https://docs.docker.com/reference/cli/docker/context/create/
- https://docs.docker.com/engine/security/protect-access/ (SSH example)

### 2) Point Switchboard at the context

#### Option A: `DOCKER_HOSTS`

```bash
export DOCKER_HOSTS="context://my-remote"
go run .
```

Multiple daemons:

```bash
export DOCKER_HOSTS="context://my-remote,unix:///var/run/docker.sock"
go run .
```

#### Option B: `config.yaml`

```yaml
hosts:
  - name: remote
    endpoint: context://my-remote
  - name: local
    endpoint: unix:///var/run/docker.sock
proxy_mappings: {}
defaults:
  base_domain: local
  scheme: http
```

```bash
export CONFIG_PATH=./config.yaml
go run .
```

## Notes / gotchas

- `context://...` requires the `docker` CLI to be available at runtime.
- Switchboard needs access to your Docker context store (by default under `~/.docker`). If running Switchboard in a container, mount your Docker config directory (or set `DOCKER_CONFIG`).
- SSH contexts require that `ssh` is available and that the configured user can access the remote Docker socket.

## DNS for per-container hostnames (central nginx)

If you want the default URL format to be:

`<container-name>.<host-name>.<domain>`

and you run a **single central nginx**, point DNS at the nginx IP/hostname.

### Wildcard records (one per host)

Create a wildcard record per Docker host name:

- `*.c3po.<domain>`  ->  `<nginx-ip>`
- `*.deathstar.<domain>`  ->  `<nginx-ip>`

Example for `domain=containers.example.com`:

- `*.c3po.containers.example.com` -> `203.0.113.10`
- `*.deathstar.containers.example.com` -> `203.0.113.10`

This works because DNS wildcards must be the **leftmost label** (so you cannot create a single wildcard like `*.*.containers.example.com` to cover all hosts) and a wildcard like `*.deathstar.containers.example.com` covers all names below that label.


### TLS note (if you serve HTTPS)

DNS wildcards and TLS wildcards are different:

- A TLS cert for `*.containers.example.com` typically **won't** cover `app.deathstar.containers.example.com` (two labels below).
- You'll usually need `*.deathstar.containers.example.com` and `*.c3po.containers.example.com` on the certificate (SANs), or issue per-host wildcard certs.

(In practice: wildcard TLS certs usually only cover a single label, so plan for per-host wildcards or SANs.)
