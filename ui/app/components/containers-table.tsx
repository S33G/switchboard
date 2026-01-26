"use client";

import { useMemo, useState } from "react";
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  flexRender,
  type ColumnDef,
  type SortingState,
} from "@tanstack/react-table";

import {
  buildPortLink,
  buildWebUiLinks,
  formatAbsoluteTime,
  formatBytes,
  formatCommand,
  formatImageId,
  formatMountsCount,
  formatMountsTooltip,
  formatNetworks,
  formatRelativeTime,
  getImageLinks,
  getOrderedImageLinks,
  sortPorts,
} from "../lib/helpers";
import { COLUMN_DEFINITIONS } from "../lib/column-config";
import { StatusPill } from "./status-pill";
import type { ColumnConfig, ColumnId, Config, Container } from "../lib/types";

interface AnimatingIds {
  added: Set<string>;
  updated: Set<string>;
  removed: Set<string>;
}

type RowAnimationState = "entering" | "updated" | "none";

function compareText(a: string, b: string): number {
  return a.localeCompare(b, undefined, { numeric: true, sensitivity: "base" });
}

function getPrimaryPort(container: Container): number | null {
  const sortedPorts = sortPorts(container.ports);
  if (!sortedPorts.length) {
    return null;
  }
  return sortedPorts[0].public ?? sortedPorts[0].private ?? null;
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
        const privateLabel = `${port.private}/${port.type}`;
        const href = port.public
          ? buildPortLink(container.host, port.public, config)
          : null;

        return (
          <div key={`${port.private}-${port.public ?? "internal"}-${index}`}>
            {port.public ? (
              <span className="inline-flex flex-wrap items-center gap-2">
                {href ? (
                  <a
                    href={href}
                    target="_blank"
                    rel="noreferrer"
                    className="font-semibold text-sky-300 transition hover:text-sky-200"
                  >
                    {port.public}
                  </a>
                ) : (
                  <span className="font-semibold text-slate-100">{port.public}</span>
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
  columnConfig: ColumnConfig;
  animatingIds?: AnimatingIds;
  hasSearch?: boolean;
}

function getRowAnimationState(
  containerId: string,
  animatingIds?: AnimatingIds
): RowAnimationState {
  if (!animatingIds) return "none";
  if (animatingIds.added.has(containerId)) return "entering";
  if (animatingIds.updated.has(containerId)) return "updated";
  return "none";
}

export function ContainersTable({
  containers,
  config,
  columnConfig,
  animatingIds,
  hasSearch = false,
}: ContainersTableProps) {
  const visibleColumns = columnConfig.visibleColumns.filter(
    (id) => COLUMN_DEFINITIONS[id]
  );

  const [sorting, setSorting] = useState<SortingState>([
    { id: visibleColumns.find((id) => COLUMN_DEFINITIONS[id].sortable) ?? "host", desc: false }
  ]);

  const columns = useMemo<ColumnDef<Container>[]>(() => {
    return visibleColumns.map((columnId): ColumnDef<Container> => {
      const colDef = COLUMN_DEFINITIONS[columnId];
      
      return {
        id: columnId,
        header: colDef.label,
        enableSorting: colDef.sortable,
        accessorFn: (row) => {
          switch (columnId) {
            case "host":
              return row.host;
            case "container":
              return row.name || row.id;
            case "image":
              return row.image;
            case "image_id":
              return row.image_id;
            case "command":
              return row.command;
            case "status":
              return row.status;
            case "created":
              return row.created_at;
            case "size_rw":
              return row.size_rw;
            case "size_rootfs":
              return row.size_rootfs;
            case "networks":
              return row.networks.join(", ");
            case "mounts":
              return row.mounts.length;
            case "ports":
              return getPrimaryPort(row);
            case "web_ui": {
              const links = buildWebUiLinks(row, config).sort();
              return links[0] ?? null;
            }
            default:
              return null;
          }
        },
        sortingFn: (rowA, rowB, columnId) => {
          const a = rowA.getValue(columnId);
          const b = rowB.getValue(columnId);

          const aEmpty = a === null || a === "";
          const bEmpty = b === null || b === "";
          
          if (aEmpty && bEmpty) return 0;
          if (aEmpty) return 1;
          if (bEmpty) return -1;

          if (typeof a === "number" && typeof b === "number") {
            return a - b;
          }

          return compareText(String(a), String(b));
        },
        cell: ({ row }) => {
          const container = row.original;
          
          switch (columnId) {
            case "host":
              return <span className="text-slate-300">{container.host}</span>;

            case "container":
              return (
                <div>
                  <div className="font-semibold text-white">
                    {container.name || container.id.slice(0, 12)}
                  </div>
                  <div className="mt-1 text-xs uppercase tracking-[0.2em] text-slate-500">
                    {container.state}
                  </div>
                </div>
              );

            case "image": {
              const imageLinks = getImageLinks(container.image);
              const orderedLinks = getOrderedImageLinks(imageLinks);
              return (
                <div>
                  <div className="text-slate-300">{container.image}</div>
                  {orderedLinks.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-2 text-[11px] uppercase tracking-[0.2em]">
                      {orderedLinks.map(({ key, label, url }) => (
                        <a
                          key={key}
                          href={url}
                          target="_blank"
                          rel="noreferrer"
                          className="text-slate-400 transition hover:text-slate-200"
                        >
                          {label}
                        </a>
                      ))}
                    </div>
                  )}
                </div>
              );
            }

            case "image_id":
              return (
                <span className="font-mono text-xs text-slate-300">
                  {formatImageId(container.image_id)}
                </span>
              );

            case "command":
              return (
                <span className="font-mono text-xs text-slate-300">
                  {formatCommand(container.command, 40)}
                </span>
              );

            case "status":
              return <StatusPill status={container.status} />;

            case "created":
              return (
                <span
                  className="text-slate-300 cursor-help"
                  title={formatAbsoluteTime(container.created_at)}
                >
                  {formatRelativeTime(container.created_at)}
                </span>
              );

            case "size_rw":
              return (
                <span className="text-slate-300">
                  {formatBytes(container.size_rw)}
                </span>
              );

            case "size_rootfs":
              return (
                <span className="text-slate-300">
                  {formatBytes(container.size_rootfs)}
                </span>
              );

            case "networks":
              return (
                <span className="text-slate-300">
                  {formatNetworks(container.networks)}
                </span>
              );

            case "mounts":
              return (
                <span
                  className="text-slate-300 cursor-help"
                  title={formatMountsTooltip(container.mounts)}
                >
                  {formatMountsCount(container.mounts)}
                </span>
              );

            case "ports":
              return <PortsList container={container} config={config} />;

            case "web_ui": {
              const links = buildWebUiLinks(container, config);
              return links.length ? (
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
              );
            }

            default:
              return <span className="text-slate-500">—</span>;
          }
        },
      };
    });
  }, [visibleColumns, config]);

  const table = useReactTable({
    data: containers,
    columns,
    state: {
      sorting,
    },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  if (!containers.length) {
    return (
      <div className="rounded-3xl border border-dashed border-slate-800 bg-slate-900/40 p-8 text-center text-slate-300">
        {hasSearch
          ? "No running containers match your search yet."
          : "No running containers detected yet."}
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-3xl border border-slate-800 bg-slate-900/60 shadow-soft">
      <table className="w-full border-collapse text-left text-sm text-slate-200">
        <thead className="bg-slate-900/90 text-[11px] uppercase tracking-[0.2em] text-slate-400">
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => {
                const isSortable = header.column.getCanSort();
                const sortDirection = header.column.getIsSorted();
                const isActive = sortDirection !== false;
                const indicator = isActive
                  ? sortDirection === "asc"
                    ? "^"
                    : "v"
                  : "";

                return (
                  <th
                    key={header.id}
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
                    {isSortable ? (
                      <button
                        type="button"
                        onClick={header.column.getToggleSortingHandler()}
                        className="group inline-flex items-center gap-2 text-left transition hover:text-slate-200"
                      >
                        <span>
                          {flexRender(
                            header.column.columnDef.header,
                            header.getContext()
                          )}
                        </span>
                        <span
                          className={`text-[10px] font-semibold ${
                            isActive ? "text-slate-200" : "text-slate-600"
                          }`}
                        >
                          {indicator}
                        </span>
                      </button>
                    ) : (
                      <span>
                        {flexRender(
                          header.column.columnDef.header,
                          header.getContext()
                        )}
                      </span>
                    )}
                  </th>
                );
              })}
            </tr>
          ))}
        </thead>
        <tbody className="divide-y divide-slate-800">
          {table.getRowModel().rows.map((row) => {
            const container = row.original;
            const statusUp = isStatusUp(container.status);
            const rowAnimation = getRowAnimationState(container.id, animatingIds);
            const animationClass =
              rowAnimation === "entering"
                ? "animate-row-enter"
                : rowAnimation === "updated"
                  ? "animate-row-pulse"
                  : "";
            return (
              <tr
                key={row.id}
                className={`align-top transition ${animationClass} ${
                  statusUp
                    ? "hover:bg-slate-900/40"
                    : "bg-rose-500/5 hover:bg-rose-500/10"
                }`}
              >
                {row.getVisibleCells().map((cell) => (
                  <td key={cell.id} className="px-4 py-3">
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
