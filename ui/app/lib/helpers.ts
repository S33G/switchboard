import type { Config, Container, HostGroup } from "./types";

export function groupRunningByHost(containers: Container[]): HostGroup[] {
  const grouped = containers
    .filter((container) => container.state === "running")
    .reduce<Record<string, Container[]>>((acc, container) => {
      const hostName = container.host || "unknown";
      if (!acc[hostName]) {
        acc[hostName] = [];
      }
      acc[hostName].push(container);
      return acc;
    }, {});

  return Object.entries(grouped)
    .map(([host, hostContainers]) => ({
      host,
      containers: hostContainers.sort((a, b) => a.name.localeCompare(b.name)),
    }))
    .sort((a, b) => a.host.localeCompare(b.host));
}

export function formatPorts(container: Container): string {
  if (!container.ports.length) {
    return "—";
  }

  return container.ports
    .map((port) => {
      const privatePort = `${port.PrivatePort}/${port.Type}`;
      if (port.PublicPort) {
        return `${port.PublicPort} → ${privatePort}`;
      }
      return privatePort;
    })
    .join(", ");
}

export function buildWebUiLinks(container: Container, config: Config): string[] {
  const scheme = config.defaults.scheme ?? "http";
  const baseDomain = config.defaults.base_domain ?? "";
  
  let links: string[] = [];
  
  if (config.proxy_routes?.[container.name]?.domains) {
    links = [...config.proxy_routes[container.name].domains];
  }

  const fallbackLink = baseDomain
    ? `${scheme}://${container.name}.${baseDomain}`
    : "";

  return Array.from(new Set([...links, fallbackLink].filter(Boolean)));
}

export function latestUpdate(containers: Container[]): string {
  const timestamps = containers
    .map((container) => new Date(container.updated_at))
    .filter((date) => !Number.isNaN(date.valueOf()))
    .map((date) => date.valueOf());

  if (!timestamps.length) {
    return "—";
  }

  const latest = new Date(Math.max(...timestamps));
  return latest.toLocaleString();
}
