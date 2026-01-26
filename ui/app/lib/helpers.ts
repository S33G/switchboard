import type { Config, Container, ContainerPort, HostGroup, MountInfo } from "./types";

export function groupRunningByHost(containers: Container[]): HostGroup[] {
  const grouped = containers.reduce<Record<string, Container[]>>((acc, container) => {
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
      const privatePort = `${port.private}/${port.type}`;
      if (port.public) {
        return `${port.public} → ${privatePort}`;
      }
      return privatePort;
    })
    .join(", ");
}

export function sortPorts(ports: ContainerPort[]): ContainerPort[] {
  return [...ports].sort((a, b) => {
    const aPublic = a.public ?? Number.POSITIVE_INFINITY;
    const bPublic = b.public ?? Number.POSITIVE_INFINITY;
    if (aPublic !== bPublic) {
      return aPublic - bPublic;
    }
    if (a.private !== b.private) {
      return a.private - b.private;
    }
    const typeCompare = a.type.localeCompare(b.type);
    if (typeCompare) {
      return typeCompare;
    }
    return 0;
  });
}

type ImageReference = {
  registry?: string;
  path: string;
};

export type ImageLinks = {
  dockerHub?: string;
  github?: string;
  linuxserver?: string;
  quay?: string;
  gitlab?: string;
  gcr?: string;
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

  if (!path) return links;

  const segments = path.split("/");
  const [namespace, repo] = segments;

  // Docker Hub: default registry or docker.io
  if (!registry || registry === "docker.io") {
    links.dockerHub = path.includes("/")
      ? `https://hub.docker.com/r/${path}`
      : `https://hub.docker.com/_/${path}`;
  }

  // LinuxServer.io Container Registry
  if (registry === "lscr.io" && namespace === "linuxserver" && repo) {
    links.linuxserver = `https://docs.linuxserver.io/images/docker-${repo}`;
    links.github = `https://github.com/linuxserver/docker-${repo}`;
  }

  // GitHub Container Registry
  if (registry === "ghcr.io" && namespace && repo) {
    links.github = `https://github.com/${namespace}/${repo}`;
  }

  // Quay.io
  if (registry === "quay.io" && namespace && repo) {
    links.quay = `https://quay.io/repository/${path}`;
  }

  // GitLab Container Registry
  if (registry === "registry.gitlab.com" && segments.length >= 2) {
    const [group, project] = segments;
    if (group && project) {
      links.gitlab = `https://gitlab.com/${group}/${project}/container_registry`;
    }
  }

  // Google Container Registry (gcr.io, us.gcr.io, eu.gcr.io, asia.gcr.io)
  if (registry?.includes("gcr.io")) {
    links.gcr = `https://gcr.io/${path}`;
  }

  return links;
}

export function getOrderedImageLinks(imageLinks: ImageLinks): Array<{
  key: keyof ImageLinks;
  label: string;
  url: string;
}> {
  const linkLabels: Record<keyof ImageLinks, string> = {
    github: "GitHub",
    gitlab: "GitLab",
    linuxserver: "LinuxServer",
    dockerHub: "Docker Hub",
    quay: "Quay.io",
    gcr: "GCR",
  };

  const linkOrder: (keyof ImageLinks)[] = [
    "github",
    "gitlab",
    "linuxserver",
    "dockerHub",
    "quay",
    "gcr",
  ];

  return linkOrder
    .filter((key) => imageLinks[key])
    .map((key) => ({
      key,
      label: linkLabels[key],
      url: imageLinks[key]!,
    }));
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
    (port) => (port.public ?? 0) > 0
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

// === New Column Formatting Functions ===

export function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  if (Number.isNaN(date.valueOf())) {
    return "—";
  }

  const now = Date.now();
  const diff = now - date.valueOf();
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  const weeks = Math.floor(days / 7);
  const months = Math.floor(days / 30);
  const years = Math.floor(days / 365);

  if (years > 0) return `${years}y ago`;
  if (months > 0) return `${months}mo ago`;
  if (weeks > 0) return `${weeks}w ago`;
  if (days > 0) return `${days}d ago`;
  if (hours > 0) return `${hours}h ago`;
  if (minutes > 0) return `${minutes}m ago`;
  return "just now";
}

export function formatAbsoluteTime(dateString: string): string {
  const date = new Date(dateString);
  if (Number.isNaN(date.valueOf())) {
    return "—";
  }
  return date.toLocaleString();
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  if (bytes < 0) return "—";

  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  if (i >= sizes.length) {
    return `${(bytes / Math.pow(k, sizes.length - 1)).toFixed(1)} ${sizes[sizes.length - 1]}`;
  }

  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

export function formatImageId(imageId: string): string {
  if (!imageId) return "—";
  // Docker image IDs are sha256 hashes, typically prefixed with "sha256:"
  const hash = imageId.replace(/^sha256:/, "");
  return hash.slice(0, 12);
}

export function formatCommand(command: string, maxLength = 50): string {
  if (!command) return "—";
  
  const trimmed = command.trim();
  if (trimmed.length <= maxLength) return trimmed;
  
  return trimmed.slice(0, maxLength - 1) + "…";
}

export function formatNetworks(networks: string[]): string {
  if (!networks || networks.length === 0) return "—";
  return networks.join(", ");
}

export function formatMountsCount(mounts: MountInfo[]): string {
  if (!mounts || mounts.length === 0) return "—";
  const count = mounts.length;
  return `${count} mount${count !== 1 ? "s" : ""}`;
}

export function formatMountsTooltip(mounts: MountInfo[]): string {
  if (!mounts || mounts.length === 0) return "No mounts";
  
  return mounts
    .map((mount) => {
      const type = mount.type || "bind";
      const mode = mount.mode || "rw";
      return `[${type}] ${mount.source || "?"} → ${mount.destination} (${mode})`;
    })
    .join("\n");
}
