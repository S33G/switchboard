"use client";

import { useQueryState } from "next-usequerystate";

export type ViewMode = "grid" | "list";

const viewParser = {
  parse: (value: string | null): ViewMode =>
    value === "list" ? "list" : "grid",
  serialize: (value: ViewMode) => value,
};

export function useViewMode() {
  return useQueryState<ViewMode>("view", viewParser);
}
