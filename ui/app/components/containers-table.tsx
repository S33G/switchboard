import { buildWebUiLinks, formatPorts } from "../lib/helpers";
import type { Config, Container } from "../lib/types";

interface ContainersTableProps {
  containers: Container[];
  config: Config;
}

export function ContainersTable({ containers, config }: ContainersTableProps) {
  const runningContainers = containers
    .filter((container) => container.state === "running")
    .sort((a, b) => a.host.localeCompare(b.host) || a.name.localeCompare(b.name));

  if (!runningContainers.length) {
    return (
      <div className="rounded-3xl border border-dashed border-slate-800 bg-slate-900/40 p-8 text-center text-slate-300">
        No running containers detected yet.
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-3xl border border-slate-800 bg-slate-900/60 shadow-soft">
      <table className="w-full border-collapse text-left text-sm text-slate-200">
        <thead className="bg-slate-900/90 text-[11px] uppercase tracking-[0.2em] text-slate-400">
          <tr>
            <th className="px-4 py-3">Host</th>
            <th className="px-4 py-3">Container</th>
            <th className="px-4 py-3">Image</th>
            <th className="px-4 py-3">Status</th>
            <th className="px-4 py-3">Ports</th>
            <th className="px-4 py-3">Web UI</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800">
          {runningContainers.map((container) => {
            const links = buildWebUiLinks(container, config);
            return (
              <tr key={container.id} className="align-top">
                <td className="px-4 py-3 text-slate-300">{container.host}</td>
                <td className="px-4 py-3">
                  <div className="font-semibold text-white">
                    {container.name || container.id.slice(0, 12)}
                  </div>
                  <div className="mt-1 text-xs uppercase tracking-[0.2em] text-slate-500">
                    {container.state}
                  </div>
                </td>
                <td className="px-4 py-3 text-slate-300">{container.image}</td>
                <td className="px-4 py-3">
                  <span className="rounded-full bg-slate-800 px-3 py-1 text-xs font-semibold text-slate-200">
                    {container.status}
                  </span>
                </td>
                <td className="px-4 py-3 text-slate-300">
                  {formatPorts(container)}
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
