import type { ColumnConfig, Config, Container } from "../lib/types";
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
import { isColumnVisible } from "../lib/column-config";
import { StatusPill } from "./status-pill";

export type AnimationState = "entering" | "updated" | "none";

interface ContainerCardProps {
  container: Container;
  config: Config;
  columnConfig: ColumnConfig;
  animationState?: AnimationState;
}

export function ContainerCard({
  container,
  config,
  columnConfig,
  animationState = "none",
}: ContainerCardProps) {
  const links = buildWebUiLinks(container, config);
  const imageLinks = getImageLinks(container.image);
  const sortedPorts = sortPorts(container.ports);

  const animationClass =
    animationState === "entering"
      ? "animate-container-enter"
      : animationState === "updated"
        ? "animate-container-pulse"
        : "";

  return (
    <article
      className={`flex h-full flex-col gap-4 rounded-2xl border border-slate-800 bg-slate-900/40 p-4 shadow-soft transition-all duration-300 ${animationClass}`}
    >
      <header className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-lg font-semibold text-white">
            {container.name || container.id.slice(0, 12)}
          </h3>
          <p className="text-xs uppercase tracking-[0.2em] text-slate-400">
            {container.state}
          </p>
        </div>
        <StatusPill status={container.status} />
      </header>

      <dl className="space-y-4 text-sm text-slate-300">
        {isColumnVisible("host", columnConfig) && (
          <div className="flex flex-col gap-1">
            <dt className="text-xs uppercase tracking-[0.2em] text-slate-500">Host</dt>
            <dd className="break-words text-slate-100">{container.host}</dd>
          </div>
        )}

        {isColumnVisible("image", columnConfig) && (
          <div className="flex flex-col gap-1">
            <dt className="text-xs uppercase tracking-[0.2em] text-slate-500">Image</dt>
            <dd className="break-words text-slate-100">{container.image}</dd>
            {(() => {
              const orderedLinks = getOrderedImageLinks(imageLinks);
              return orderedLinks.length > 0 ? (
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
              ) : null;
            })()}
          </div>
        )}

        {isColumnVisible("image_id", columnConfig) && (
          <div className="flex flex-col gap-1">
            <dt className="text-xs uppercase tracking-[0.2em] text-slate-500">Image ID</dt>
            <dd className="font-mono text-xs text-slate-100">
              {formatImageId(container.image_id)}
            </dd>
          </div>
        )}

        {isColumnVisible("command", columnConfig) && (
          <div className="flex flex-col gap-1">
            <dt className="text-xs uppercase tracking-[0.2em] text-slate-500">Command</dt>
            <dd className="font-mono text-xs text-slate-100">
              {formatCommand(container.command)}
            </dd>
          </div>
        )}

        {isColumnVisible("created", columnConfig) && (
          <div className="flex flex-col gap-1">
            <dt className="text-xs uppercase tracking-[0.2em] text-slate-500">Created</dt>
            <dd className="text-slate-100" title={formatAbsoluteTime(container.created_at)}>
              {formatRelativeTime(container.created_at)}
            </dd>
          </div>
        )}

        {isColumnVisible("size_rw", columnConfig) && (
          <div className="flex flex-col gap-1">
            <dt className="text-xs uppercase tracking-[0.2em] text-slate-500">Size (RW)</dt>
            <dd className="text-slate-100">{formatBytes(container.size_rw)}</dd>
          </div>
        )}

        {isColumnVisible("size_rootfs", columnConfig) && (
          <div className="flex flex-col gap-1">
            <dt className="text-xs uppercase tracking-[0.2em] text-slate-500">Root FS Size</dt>
            <dd className="text-slate-100">{formatBytes(container.size_rootfs)}</dd>
          </div>
        )}

        {isColumnVisible("networks", columnConfig) && (
          <div className="flex flex-col gap-1">
            <dt className="text-xs uppercase tracking-[0.2em] text-slate-500">Networks</dt>
            <dd className="text-slate-100">{formatNetworks(container.networks)}</dd>
          </div>
        )}

        {isColumnVisible("mounts", columnConfig) && (
          <div className="flex flex-col gap-1">
            <dt className="text-xs uppercase tracking-[0.2em] text-slate-500">Mounts</dt>
            <dd
              className="text-slate-100 cursor-help"
              title={formatMountsTooltip(container.mounts)}
            >
              {formatMountsCount(container.mounts)}
            </dd>
          </div>
        )}

        {isColumnVisible("ports", columnConfig) && (
          <div className="flex flex-col gap-2">
            <dt className="text-xs uppercase tracking-[0.2em] text-slate-500">Ports</dt>
            {sortedPorts.length ? (
              <div className="overflow-hidden rounded-xl border border-slate-800">
                <table className="w-full text-left text-xs text-slate-300">
                  <thead className="bg-slate-900/70 text-[10px] uppercase tracking-[0.2em] text-slate-500">
                    <tr>
                      <th className="px-3 py-2">Host IP</th>
                      <th className="px-3 py-2">Public</th>
                      <th className="px-3 py-2">Private</th>
                      <th className="px-3 py-2">Type</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800">
                    {sortedPorts.map((port, index) => {
                      const href = port.public
                        ? buildPortLink(container.host, port.public, config)
                        : null;
                      return (
                        <tr key={`${port.private}-${port.public ?? "internal"}-${index}`}>
                          <td className="px-3 py-2 text-slate-100">
                            —
                          </td>
                          <td className="px-3 py-2 text-slate-100">
                            {port.public ? (
                              href ? (
                                <a
                                  href={href}
                                  target="_blank"
                                  rel="noreferrer"
                                  className="font-semibold text-sky-300 transition hover:text-sky-200"
                                >
                                  {port.public}
                                </a>
                              ) : (
                                port.public
                              )
                            ) : (
                              "—"
                            )}
                          </td>
                          <td className="px-3 py-2 text-slate-100">
                            {port.private}
                          </td>
                          <td className="px-3 py-2 text-slate-100">
                            {port.type}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            ) : (
              <dd className="text-sm text-slate-500">No exposed ports</dd>
            )}
          </div>
        )}
      </dl>

      {isColumnVisible("web_ui", columnConfig) && (
        <div className="mt-auto flex flex-col gap-2 text-sm">
          <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Web UI</p>
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
            <p className="text-sm text-slate-500">No Web UI mapping</p>
          )}
        </div>
      )}
    </article>
  );
}
