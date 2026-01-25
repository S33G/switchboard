import { z } from "zod";

import type { Config, Container } from "./types";

export const apiBaseUrl =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  (typeof window !== "undefined" ? window.location.origin : "http://localhost:8069");

const ContainerPortSchema = z.object({
  IP: z.string().optional(),
  PrivatePort: z.number(),
  PublicPort: z.number().optional(),
  Type: z.string(),
});

export const ContainerSchema = z.object({
  id: z.string(),
  name: z.string(),
  image: z.string(),
  state: z.string(),
  status: z.string(),
  host: z.string(),
  ports: z.array(ContainerPortSchema).optional().default([]),
  labels: z.record(z.string(), z.string()).optional().default({}),
  updated_at: z.string(),
});

const ConfigSchema = z.object({
  defaults: z
    .object({
      base_domain: z.string().optional(),
      scheme: z.string().optional(),
    })
    .optional(),
  proxy_mappings: z.record(z.string(), z.string()).optional(),
});

async function fetchJson(path: string): Promise<unknown> {
  const url = new URL(path, apiBaseUrl);
  const response = await fetch(url, {
    headers: { Accept: "application/json" },
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);
  }

  return response.json();
}

export async function getContainers(): Promise<Container[]> {
  const payload = await fetchJson("/api/containers");
  const parsed = z.array(ContainerSchema).safeParse(payload);
  if (!parsed.success) {
    throw new Error("Invalid containers payload");
  }
  return parsed.data;
}

export async function getConfig(): Promise<Config> {
  const payload = await fetchJson("/api/config");
  const parsed = ConfigSchema.safeParse(payload);
  if (!parsed.success) {
    throw new Error("Invalid config payload");
  }

  return {
    defaults: parsed.data.defaults ?? {},
    proxy_mappings: parsed.data.proxy_mappings ?? {},
  };
}
