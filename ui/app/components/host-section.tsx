import type { Config, HostGroup } from "../lib/types";
import { ContainerCard, type AnimationState } from "./container-card";

interface AnimatingIds {
  added: Set<string>;
  updated: Set<string>;
  removed: Set<string>;
}

interface HostSectionProps {
  group: HostGroup;
  config: Config;
  animatingIds?: AnimatingIds;
  hasSearch?: boolean;
}

function getAnimationState(
  containerId: string,
  animatingIds?: AnimatingIds
): AnimationState {
  if (!animatingIds) return "none";
  if (animatingIds.added.has(containerId)) return "entering";
  if (animatingIds.updated.has(containerId)) return "updated";
  return "none";
}

export function HostSection({
  group,
  config,
  animatingIds,
  hasSearch = false,
}: HostSectionProps) {
  if (!group.containers.length) {
    return (
      <section className="rounded-3xl border border-dashed border-slate-800 bg-slate-900/40 p-8 text-center text-slate-300">
        {hasSearch
          ? "No running containers match your search yet."
          : "No running containers detected yet."}
      </section>
    );
  }

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
            animationState={getAnimationState(container.id, animatingIds)}
          />
        ))}
      </div>
    </section>
  );
}
