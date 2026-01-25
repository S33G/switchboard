import { z } from "zod";

import { ContainerSchema, apiBaseUrl } from "./api";
import type { Container } from "./types";

const SnapshotMessageSchema = z.object({
  type: z.literal("containers_snapshot"),
  payload: z.array(ContainerSchema),
});

export type WebSocketStatus = "connecting" | "live" | "offline";

export function buildWebSocketUrl(): string {
  const url = new URL("/ws", apiBaseUrl);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}

export function parseSnapshotMessage(data: unknown): Container[] | null {
  const parsed = SnapshotMessageSchema.safeParse(data);
  if (!parsed.success) {
    return null;
  }
  return parsed.data.payload;
}
