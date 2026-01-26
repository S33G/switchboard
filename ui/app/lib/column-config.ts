import type { ColumnDefinition, ColumnId, ColumnConfig } from "./types";

export const COLUMN_DEFINITIONS: Record<ColumnId, ColumnDefinition> = {
  host: {
    id: "host",
    label: "Host",
    group: "basic",
    sortable: true,
    width: "medium",
  },
  container: {
    id: "container",
    label: "Container",
    group: "basic",
    sortable: true,
    width: "medium",
  },
  image: {
    id: "image",
    label: "Image",
    group: "image",
    sortable: true,
    width: "large",
  },
  image_id: {
    id: "image_id",
    label: "Image ID",
    group: "image",
    sortable: true,
    width: "small",
  },
  command: {
    id: "command",
    label: "Command",
    group: "basic",
    sortable: false,
    width: "large",
  },
  status: {
    id: "status",
    label: "Status",
    group: "basic",
    sortable: true,
    width: "small",
  },
  created: {
    id: "created",
    label: "Created",
    group: "basic",
    sortable: true,
    width: "medium",
  },
  size_rw: {
    id: "size_rw",
    label: "Size (RW)",
    group: "resources",
    sortable: true,
    width: "small",
  },
  size_rootfs: {
    id: "size_rootfs",
    label: "Root FS",
    group: "resources",
    sortable: true,
    width: "small",
  },
  networks: {
    id: "networks",
    label: "Networks",
    group: "network",
    sortable: false,
    width: "medium",
  },
  mounts: {
    id: "mounts",
    label: "Mounts",
    group: "resources",
    sortable: false,
    width: "small",
  },
  ports: {
    id: "ports",
    label: "Ports",
    group: "network",
    sortable: false,
    width: "large",
  },
  web_ui: {
    id: "web_ui",
    label: "Web UI",
    group: "network",
    sortable: false,
    width: "small",
  },
};

const STORAGE_KEY = "switchboard:column-config";
const CONFIG_VERSION = 1;

const DEFAULT_VISIBLE_COLUMNS: ColumnId[] = [
  "host",
  "container",
  "image",
  "status",
  "created",
  "ports",
  "web_ui",
];

export function getDefaultColumnConfig(): ColumnConfig {
  return {
    version: CONFIG_VERSION,
    visibleColumns: [...DEFAULT_VISIBLE_COLUMNS],
    lastUpdated: new Date().toISOString(),
  };
}

export function loadColumnConfigFromStorage(): ColumnConfig {
  if (typeof window === "undefined") {
    return getDefaultColumnConfig();
  }

  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) {
      return getDefaultColumnConfig();
    }

    const parsed = JSON.parse(stored) as ColumnConfig;

    if (parsed.version !== CONFIG_VERSION) {
      return getDefaultColumnConfig();
    }

    const validColumns = parsed.visibleColumns.filter(
      (id) => id in COLUMN_DEFINITIONS
    );

    if (validColumns.length === 0) {
      return getDefaultColumnConfig();
    }

    return {
      ...parsed,
      visibleColumns: validColumns,
    };
  } catch {
    return getDefaultColumnConfig();
  }
}

export function saveColumnConfigToStorage(config: ColumnConfig): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    const toSave: ColumnConfig = {
      ...config,
      lastUpdated: new Date().toISOString(),
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(toSave));
  } catch {
    console.error("Failed to save column config to localStorage");
  }
}

export function getColumnsByGroup(): Record<string, ColumnDefinition[]> {
  const grouped: Record<string, ColumnDefinition[]> = {
    basic: [],
    image: [],
    resources: [],
    network: [],
  };

  Object.values(COLUMN_DEFINITIONS).forEach((col) => {
    if (grouped[col.group]) {
      grouped[col.group].push(col);
    }
  });

  return grouped;
}

export function isColumnVisible(
  columnId: ColumnId,
  config: ColumnConfig
): boolean {
  return config.visibleColumns.includes(columnId);
}
