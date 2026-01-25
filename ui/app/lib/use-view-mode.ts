"use client";

import { parseAsStringLiteral, useQueryState } from "next-usequerystate";

export type ViewMode = "grid" | "list";

const viewModeParser = parseAsStringLiteral(["grid", "list"] as const).withDefault("grid");

export function useViewMode() {
  return useQueryState("view", viewModeParser);
}
