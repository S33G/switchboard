"use client";

import { useState, useEffect } from "react";
import type { ColumnConfig, ColumnId } from "./types";
import {
  getDefaultColumnConfig,
  loadColumnConfigFromStorage,
  saveColumnConfigToStorage,
} from "./column-config";

type UseColumnConfigReturn = [
  ColumnConfig,
  (visibleColumns: ColumnId[]) => void,
  () => void
];

export function useColumnConfig(): UseColumnConfigReturn {
  const [config, setConfig] = useState<ColumnConfig>(() => {
    const loaded = loadColumnConfigFromStorage();
    return loaded;
  });

  const saveConfig = (visibleColumns: ColumnId[]) => {
    const newConfig: ColumnConfig = {
      ...config,
      visibleColumns,
    };
    setConfig(newConfig);
    saveColumnConfigToStorage(newConfig);
  };

  const resetToDefaults = () => {
    const defaultConfig = getDefaultColumnConfig();
    setConfig(defaultConfig);
    saveColumnConfigToStorage(defaultConfig);
  };

  return [config, saveConfig, resetToDefaults];
}
