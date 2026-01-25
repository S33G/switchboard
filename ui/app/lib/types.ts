export interface ContainerPort {
  IP?: string;
  PrivatePort: number;
  PublicPort?: number;
  Type: string;
}

export interface Container {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
  host: string;
  ports: ContainerPort[];
  labels: Record<string, string>;
  updated_at: string;
}

export interface Defaults {
  base_domain?: string;
  scheme?: string;
}

export interface Config {
  defaults: Defaults;
  proxy_mappings: Record<string, string>;
}

export interface HostGroup {
  host: string;
  containers: Container[];
}
