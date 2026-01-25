import type { Config, Container } from "../lib/types";
import {
  buildPortLink,
  buildWebUiLinks,
  getImageLinks,
  sortPorts,
} from "../lib/helpers";
import { StatusPill } from "./status-pill";

interface ContainerCardProps {
  container: Container;
  config: Config;
}

export function ContainerCard({ container, config }: ContainerCardProps) {
  const links = buildWebUiLinks(container, config);
  const imageLinks = getImageLinks(container.image);
  const sortedPorts = sortPorts(container.ports);

  return (
    <article className="flex h-full flex-col gap-4 rounded-2xl border border-slate-800 bg-slate-900/40 p-4 shadow-soft">
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
        <div className="flex flex-col gap-1">
          <dt className="text-xs uppercase tracking-[0.2em] text-slate-500">Image</dt>
          <dd className="break-words text-slate-100">{container.image}</dd>
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
        </div>
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
                    const href = port.PublicPort
                      ? buildPortLink(container.host, port.PublicPort, config)
                      : null;
                    return (
                      <tr key={`${port.PrivatePort}-${port.PublicPort ?? "internal"}-${index}`}>
                        <td className="px-3 py-2 text-slate-100">
                          {port.IP || "—"}
                        </td>
                        <td className="px-3 py-2 text-slate-100">
                          {port.PublicPort ? (
                            href ? (
                              <a
                                href={href}
                                target="_blank"
                                rel="noreferrer"
                                className="font-semibold text-sky-300 transition hover:text-sky-200"
                              >
                                {port.PublicPort}
                              </a>
                            ) : (
                              port.PublicPort
                            )
                          ) : (
                            "—"
                          )}
                        </td>
                        <td className="px-3 py-2 text-slate-100">
                          {port.PrivatePort}
                        </td>
                        <td className="px-3 py-2 text-slate-100">
                          {port.Type}
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
      </dl>

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
    </article>
  );
}
