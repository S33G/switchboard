export interface ContainerPort {
  private: number;
  public?: number;
  type: string;
  proxied?: boolean;
}

export interface MountInfo {
  type: string;
  source: string;
  destination: string;
  mode: string;
}

export type ColumnId =
  | "host"
  | "container"
  | "image"
  | "image_id"
  | "command"
  | "status"
  | "created"
  | "size_rw"
  | "size_rootfs"
  | "networks"
  | "mounts"
  | "ports"
  | "web_ui";

export interface ColumnDefinition {
  id: ColumnId;
  label: string;
  group: "basic" | "image" | "resources" | "network";
  sortable: boolean;
  width?: "small" | "medium" | "large";
}

export interface ColumnConfig {
  version: number;
  visibleColumns: ColumnId[];
  lastUpdated: string;
}

export interface Container {
  id: string;
  name: string;
  image: string;
  image_id: string;
  command: string;
  state: string;
  status: string;
  host: string;
  ports: ContainerPort[];
  labels: Record<string, string>;
  created_at: string;
  updated_at: string;
  size_rw: number;
  size_rootfs: number;
  networks: string[];
  mounts: MountInfo[];
}

export interface Defaults {
  base_domain?: string;
  scheme?: string;
}

export interface Config {
  defaults: Defaults;
  proxy_mappings: Record<string, string>;
  proxy_routes: Record<string, Record<string, string[]>>;
  host_addresses?: Record<string, string>;
}

export interface HostGroup {
  host: string;
  containers: Container[];
}
