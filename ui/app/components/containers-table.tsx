"use client";

import { useMemo, useState } from "react";

import {
  buildPortLink,
  buildWebUiLinks,
  getImageLinks,
  sortPorts,
} from "../lib/helpers";
import { StatusPill } from "./status-pill";
import type { Config, Container } from "../lib/types";

type SortKey = "host" | "container" | "image" | "status" | "ports" | "web";
type SortDirection = "asc" | "desc";
type SortValue = string | number | null;

const SORT_COLUMNS: { key: SortKey; label: string }[] = [
  { key: "host", label: "Host" },
  { key: "container", label: "Container" },
  { key: "image", label: "Image" },
  { key: "status", label: "Status" },
  { key: "ports", label: "Ports" },
  { key: "web", label: "Web UI" },
];

function compareText(a: string, b: string): number {
  return a.localeCompare(b, undefined, { numeric: true, sensitivity: "base" });
}

function compareSortValues(
  a: SortValue,
  b: SortValue,
  direction: SortDirection
): number {
  const aEmpty = a === null || a === "";
  const bEmpty = b === null || b === "";
  if (aEmpty || bEmpty) {
    if (aEmpty && bEmpty) {
      return 0;
    }
    return aEmpty ? 1 : -1;
  }

  if (typeof a === "number" && typeof b === "number") {
    return direction === "asc" ? a - b : b - a;
  }

  const comparison = compareText(String(a), String(b));
  return direction === "asc" ? comparison : -comparison;
}

function getPrimaryPort(container: Container): number | null {
  const sortedPorts = sortPorts(container.ports);
  if (!sortedPorts.length) {
    return null;
  }
  return sortedPorts[0].PublicPort ?? sortedPorts[0].PrivatePort ?? null;
}

function getSortValue(
  container: Container,
  sortKey: SortKey,
  config: Config
): SortValue {
  switch (sortKey) {
    case "host":
      return container.host;
    case "container":
      return container.name || container.id;
    case "image":
      return container.image;
    case "status":
      return container.status;
    case "ports":
      return getPrimaryPort(container);
    case "web": {
      const links = buildWebUiLinks(container, config).sort();
      return links[0] ?? null;
    }
    default:
      return null;
  }
}

function isStatusUp(status: string): boolean {
  const normalized = status.toLowerCase();
  if (!normalized.startsWith("up")) {
    return false;
  }
  if (
    normalized.includes("unhealthy") ||
    normalized.includes("starting") ||
    normalized.includes("restarting")
  ) {
    return false;
  }
  return true;
}

function PortsList({ container, config }: { container: Container; config: Config }) {
  const ports = sortPorts(container.ports);

  if (!ports.length) {
    return <span className="text-sm text-slate-500">—</span>;
  }

  return (
    <div className="flex flex-col gap-1 text-slate-300">
      {ports.map((port, index) => {
        const privateLabel = `${port.PrivatePort}/${port.Type}`;
        const href = port.PublicPort
          ? buildPortLink(container.host, port.PublicPort, config)
          : null;

        return (
          <div key={`${port.PrivatePort}-${port.PublicPort ?? "internal"}-${index}`}>
            {port.PublicPort ? (
              <span className="inline-flex flex-wrap items-center gap-2">
                {href ? (
                  <a
                    href={href}
                    target="_blank"
                    rel="noreferrer"
                    className="font-semibold text-sky-300 transition hover:text-sky-200"
                  >
                    {port.PublicPort}
                  </a>
                ) : (
                  <span className="font-semibold text-slate-100">{port.PublicPort}</span>
                )}
                <span className="text-slate-500">-&gt;</span>
                <span className="text-slate-400">{privateLabel}</span>
              </span>
            ) : (
              <span className="text-slate-400">{privateLabel}</span>
            )}
          </div>
        );
      })}
    </div>
  );
}

interface ContainersTableProps {
  containers: Container[];
  config: Config;
}

export function ContainersTable({ containers, config }: ContainersTableProps) {
  const [sortKey, setSortKey] = useState<SortKey>("host");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");

  const runningContainers = useMemo(
    () => containers.filter((container) => container.state === "running"),
    [containers]
  );

  const sortedContainers = useMemo(() => {
    const items = runningContainers.map((container) => ({
      container,
      sortValue: getSortValue(container, sortKey, config),
      host: container.host,
      name: container.name || container.id,
    }));

    items.sort((a, b) => {
      let comparison = compareSortValues(
        a.sortValue,
        b.sortValue,
        sortDirection
      );
      if (comparison === 0) {
        comparison = compareText(a.host, b.host) || compareText(a.name, b.name);
      }
      return comparison;
    });

    return items.map((item) => item.container);
  }, [runningContainers, sortKey, sortDirection, config]);

  if (!runningContainers.length) {
    return (
      <div className="rounded-3xl border border-dashed border-slate-800 bg-slate-900/40 p-8 text-center text-slate-300">
        No running containers detected yet.
      </div>
    );
  }

  const handleSort = (key: SortKey) => {
    setSortDirection((prev) =>
      sortKey === key ? (prev === "asc" ? "desc" : "asc") : "asc"
    );
    setSortKey(key);
  };

  return (
    <div className="overflow-hidden rounded-3xl border border-slate-800 bg-slate-900/60 shadow-soft">
      <table className="w-full border-collapse text-left text-sm text-slate-200">
        <thead className="bg-slate-900/90 text-[11px] uppercase tracking-[0.2em] text-slate-400">
          <tr>
            {SORT_COLUMNS.map((column) => {
              const isActive = sortKey === column.key;
              const indicator = isActive
                ? sortDirection === "asc"
                  ? "^"
                  : "v"
                : "";

              return (
                <th
                  key={column.key}
                  scope="col"
                  aria-sort={
                    isActive
                      ? sortDirection === "asc"
                        ? "ascending"
                        : "descending"
                      : "none"
                  }
                  className="px-4 py-3"
                >
                  <button
                    type="button"
                    onClick={() => handleSort(column.key)}
                    className="group inline-flex items-center gap-2 text-left transition hover:text-slate-200"
                  >
                    <span>{column.label}</span>
                    <span
                      className={`text-[10px] font-semibold ${
                        isActive ? "text-slate-200" : "text-slate-600"
                      }`}
                    >
                      {indicator}
                    </span>
                  </button>
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800">
          {sortedContainers.map((container) => {
            const links = buildWebUiLinks(container, config);
            const imageLinks = getImageLinks(container.image);
            const statusUp = isStatusUp(container.status);
            return (
              <tr
                key={container.id}
                className={`align-top transition ${
                  statusUp
                    ? "hover:bg-slate-900/40"
                    : "bg-rose-500/5 hover:bg-rose-500/10"
                }`}
              >
                <td className="px-4 py-3 text-slate-300">{container.host}</td>
                <td className="px-4 py-3">
                  <div className="font-semibold text-white">
                    {container.name || container.id.slice(0, 12)}
                  </div>
                  <div className="mt-1 text-xs uppercase tracking-[0.2em] text-slate-500">
                    {container.state}
                  </div>
                </td>
                <td className="px-4 py-3 text-slate-300">
                  <div className="text-slate-300">{container.image}</div>
                  {(imageLinks.dockerHub || imageLinks.github) && (
                    <div className="mt-2 flex flex-wrap gap-2 text-[11px] uppercase tracking-[0.2em]">
                      {imageLinks.dockerHub ? (
                        <a
                          href={imageLinks.dockerHub}
                          target="_blank"
                          rel="noreferrer"
                          className="text-slate-400 transition hover:text-slate-200"
                        >
                          Docker Hub
                        </a>
                      ) : null}
                      {imageLinks.github ? (
                        <a
                          href={imageLinks.github}
                          target="_blank"
                          rel="noreferrer"
                          className="text-slate-400 transition hover:text-slate-200"
                        >
                          GitHub
                        </a>
                      ) : null}
                    </div>
                  )}
                </td>
                <td className="px-4 py-3">
                  <StatusPill status={container.status} />
                </td>
                <td className="px-4 py-3">
                  <PortsList container={container} config={config} />
                </td>
                <td className="px-4 py-3">
                  {links.length ? (
                    <div className="flex flex-col gap-1">
                      {links.map((link) => (
                        <a
                          key={link}
                          href={link}
                          target="_blank"
                          rel="noreferrer"
                          className="text-sky-300 transition hover:text-sky-200"
                        >
                          {link.replace(/^https?:\/\//, "")}
                        </a>
                      ))}
                    </div>
                  ) : (
                    <span className="text-sm text-slate-500">No mapping</span>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
