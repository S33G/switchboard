import type { Config, HostGroup } from "../lib/types";
import { ContainerCard } from "./container-card";

interface HostSectionProps {
  group: HostGroup;
  config: Config;
}

export function HostSection({ group, config }: HostSectionProps) {
  return (
    <section className="rounded-3xl border border-slate-800 bg-slate-900/60 p-6 shadow-soft">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold text-white">{group.host}</h2>
          <p className="text-sm text-slate-400">
            {group.containers.length} running container
            {group.containers.length === 1 ? "" : "s"}
          </p>
        </div>
      </div>

      <div className="mt-6 grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
        {group.containers.map((container) => (
          <ContainerCard
            key={container.id}
            container={container}
            config={config}
          />
        ))}
      </div>
    </section>
  );
}
