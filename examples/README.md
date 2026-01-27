# Examples

## docker-compose

From repo root:

```bash
docker compose up --build
```

Edit `examples/config.yaml` to match your homelab:

- `hosts[].endpoint`: how Switchboard talks to each Docker daemon (e.g. `context://<name>`, `tcp://...`, `ssh://...`, `unix://...`).
- `host_addresses`: how nginx reaches each Docker host's **published ports** (LAN IP / hostname).
- `defaults.base_domain`: the domain suffix used to generate routes.

With `NGINX_CONF_GEN_ENABLED=1`, running containers get vhosts at:

`<container-name>.<host-name>.<defaults.base_domain>`

Example:

`whoami.deathstar.containers.home` -> `192.168.1.11:<published-port>`

## Docker contexts (optional)

If your `hosts[].endpoint` uses `context://...`, mount your Docker config into the container:

```yaml
volumes:
  - ~/.docker:/root/.docker:ro
  - ~/.ssh:/root/.ssh:ro
```

And create the contexts on the machine running the container:

```bash
docker context create deathstar --docker "host=ssh://user@192.168.1.11"
docker context create c3po --docker "host=ssh://user@192.168.1.10"
```
