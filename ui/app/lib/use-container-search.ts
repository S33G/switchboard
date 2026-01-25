"use client";

import { parseAsString, useQueryState } from "next-usequerystate";

const searchQueryParser = parseAsString.withDefault("");

export function useContainerSearch() {
  return useQueryState("q", searchQueryParser);
}
