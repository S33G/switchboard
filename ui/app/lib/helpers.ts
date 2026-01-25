import type { Config, Container, ContainerPort, HostGroup } from "./types";

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
  const sortedPorts = sortPorts(container.ports);

  if (!sortedPorts.length) {
    return "—";
  }

  return sortedPorts
    .map((port) => {
      const privatePort = `${port.PrivatePort}/${port.Type}`;
      if (port.PublicPort) {
        return `${port.PublicPort} → ${privatePort}`;
      }
      return privatePort;
    })
    .join(", ");
}

export function sortPorts(ports: ContainerPort[]): ContainerPort[] {
  return [...ports].sort((a, b) => {
    const aPublic = a.PublicPort ?? Number.POSITIVE_INFINITY;
    const bPublic = b.PublicPort ?? Number.POSITIVE_INFINITY;
    if (aPublic !== bPublic) {
      return aPublic - bPublic;
    }
    if (a.PrivatePort !== b.PrivatePort) {
      return a.PrivatePort - b.PrivatePort;
    }
    const typeCompare = a.Type.localeCompare(b.Type);
    if (typeCompare) {
      return typeCompare;
    }
    return (a.IP ?? "").localeCompare(b.IP ?? "");
  });
}

type ImageReference = {
  registry?: string;
  path: string;
};

export type ImageLinks = {
  dockerHub?: string;
  github?: string;
};

function parseImageReference(image: string): ImageReference {
  const withoutDigest = image.split("@")[0] ?? "";
  let name = withoutDigest;
  const lastColon = withoutDigest.lastIndexOf(":");
  const lastSlash = withoutDigest.lastIndexOf("/");
  if (lastColon > lastSlash) {
    name = withoutDigest.slice(0, lastColon);
  }

  const segments = name.split("/").filter(Boolean);
  let registry: string | undefined;

  if (segments.length > 1) {
    const firstSegment = segments[0];
    if (
      firstSegment.includes(".") ||
      firstSegment.includes(":") ||
      firstSegment === "localhost"
    ) {
      registry = firstSegment;
      segments.shift();
    }
  }

  return {
    registry,
    path: segments.join("/"),
  };
}

export function getImageLinks(image: string): ImageLinks {
  const { registry, path } = parseImageReference(image);
  const links: ImageLinks = {};

  if (path && (!registry || registry === "docker.io")) {
    links.dockerHub = path.includes("/")
      ? `https://hub.docker.com/r/${path}`
      : `https://hub.docker.com/_/${path}`;
  }

  if (registry === "ghcr.io") {
    const [owner, repo] = path.split("/");
    if (owner && repo) {
      links.github = `https://github.com/${owner}/${repo}`;
    }
  }

  return links;
}

export function resolveHostAddress(host: string, config: Config): string {
  return (config.host_addresses?.[host] ?? host).trim();
}

export function buildPortLink(
  host: string,
  port: number,
  config: Config
): string | null {
  const hostAddress = resolveHostAddress(host, config);
  if (!hostAddress || port <= 0) {
    return null;
  }
  const scheme = config.defaults.scheme ?? "http";
  return `${scheme}://${hostAddress}:${port}`;
}

export function buildWebUiLinks(container: Container, config: Config): string[] {
  const scheme = config.defaults.scheme ?? "http";
  const baseDomain = config.defaults.base_domain ?? "";

  let links: string[] = [];

  const containerKey = `${container.host}/${container.name}`;
  if (config.proxy_routes?.[containerKey]?.domains) {
    links = [...config.proxy_routes[containerKey].domains];
  }

  const publishedPorts = container.ports.filter(
    (port) => (port.PublicPort ?? 0) > 0
  );
  const canGuessPort = publishedPorts.length === 1;
  const containerName = container.name?.trim();

  const fallbackLink =
    !links.length && baseDomain && containerName && canGuessPort
      ? `${scheme}://${containerName}.${container.host}.${baseDomain}`
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
