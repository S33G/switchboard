import { z } from "zod";

import { ContainerSchema, apiBaseUrl } from "./api";
import type { Container } from "./types";

const SnapshotMessageSchema = z.object({
  type: z.literal("containers_snapshot"),
  payload: z.array(ContainerSchema),
});

const DiffMessageSchema = z.object({
  type: z.literal("containers_diff"),
  payload: z.object({
    added: z.array(ContainerSchema).nullable().optional().transform(v => v ?? []),
    updated: z.array(ContainerSchema).nullable().optional().transform(v => v ?? []),
    removed: z.array(z.string()).nullable().optional().transform(v => v ?? []),
  }),
});

export type WebSocketStatus = "connecting" | "live" | "offline";

export type ContainerDiff = {
  added: Container[];
  updated: Container[];
  removed: string[];
};

export type WebSocketMessage =
  | { type: "snapshot"; containers: Container[] }
  | { type: "diff"; diff: ContainerDiff };

export function buildWebSocketUrl(): string {
  const url = new URL("/ws", apiBaseUrl);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}

export function parseWebSocketMessage(data: unknown): WebSocketMessage | null {
  const snapshotParsed = SnapshotMessageSchema.safeParse(data);
  if (snapshotParsed.success) {
    return { type: "snapshot", containers: snapshotParsed.data.payload };
  }

  const diffParsed = DiffMessageSchema.safeParse(data);
  if (diffParsed.success) {
    return {
      type: "diff",
      diff: {
        added: diffParsed.data.payload.added,
        updated: diffParsed.data.payload.updated,
        removed: diffParsed.data.payload.removed,
      },
    };
  }

  return null;
}

export function applyContainerDiff(
  current: Container[],
  diff: ContainerDiff
): {
  containers: Container[];
  addedIds: Set<string>;
  updatedIds: Set<string>;
  removedIds: Set<string>;
} {
  const containerMap = new Map(current.map((c) => [c.id, c]));
  const addedIds = new Set<string>();
  const updatedIds = new Set<string>();
  const removedIds = new Set(diff.removed);

  for (const id of diff.removed) {
    containerMap.delete(id);
  }

  for (const container of diff.added) {
    containerMap.set(container.id, container);
    addedIds.add(container.id);
  }

  for (const container of diff.updated) {
    const existing = containerMap.get(container.id);
    const stateChanged = existing && existing.state !== container.state;
    if (stateChanged) {
      updatedIds.add(container.id);
    }
    containerMap.set(container.id, container);
  }

  return {
    containers: Array.from(containerMap.values()),
    addedIds,
    updatedIds,
    removedIds,
  };
}
