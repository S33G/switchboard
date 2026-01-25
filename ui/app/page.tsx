"use client";

import { useEffect, useState, useTransition } from "react";

import { getConfig, getContainers } from "./lib/api";
import { groupRunningByHost, latestUpdate } from "./lib/helpers";
import type { Config, Container } from "./lib/types";
import {
  buildWebSocketUrl,
  parseSnapshotMessage,
  type WebSocketStatus,
} from "./lib/websocket";
import { DashboardSkeleton } from "./components/dashboard-skeleton";
import { HostSection } from "./components/host-section";
import { ContainersTable } from "./components/containers-table";
import { ViewToggle } from "./components/view-toggle";
import { useViewMode } from "./lib/use-view-mode";

type LoadStatus = "idle" | "loading" | "success" | "error";
const RECONNECT_DELAY_MS = 5000;

export default function HomePage() {
  const [refreshKey, setRefreshKey] = useState(0);
  const [containers, setContainers] = useState<Container[]>([]);
  const [config, setConfig] = useState<Config | null>(null);
  const [lastUpdated, setLastUpdated] = useState<string>("—");
  const [status, setStatus] = useState<LoadStatus>("loading");
  const [socketStatus, setSocketStatus] = useState<WebSocketStatus>(
    "connecting"
  );
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const [viewMode, setViewMode] = useViewMode();

  const handleRefresh = () => {
    startTransition(() => {
      setRefreshKey((prev) => prev + 1);
    });
  };

  useEffect(() => {
    let isActive = true;

    const loadData = async () => {
      if (refreshKey === 0) {
        setStatus("loading");
      }

      try {
        const [containersResponse, configResponse] = await Promise.all([
          getContainers(),
          getConfig(),
        ]);

        if (!isActive) {
          return;
        }

        setContainers(containersResponse);
        setConfig(configResponse);
        setLastUpdated(latestUpdate(containersResponse));
        setStatus("success");
        setErrorMessage(null);
      } catch (error) {
        if (!isActive) {
          return;
        }
        console.error(error);
        setStatus("error");
        setErrorMessage(
          "Unable to load container data. Confirm the backend is running."
        );
      }
    };

    void loadData();

    return () => {
      isActive = false;
    };
  }, [refreshKey]);

  useEffect(() => {
    const wsUrl = buildWebSocketUrl();
    let socket: WebSocket | null = null;
    let reconnectTimer: number | null = null;
    let isActive = true;

    const connect = () => {
      if (!isActive) {
        return;
      }
      setSocketStatus("connecting");
      socket = new WebSocket(wsUrl);

      socket.addEventListener("open", () => {
        if (!isActive) {
          return;
        }
        setSocketStatus("live");
      });

      socket.addEventListener("message", (event) => {
        if (!isActive) {
          return;
        }

        try {
          const parsed = parseSnapshotMessage(JSON.parse(event.data));
          if (!parsed) {
            return;
          }

          setContainers(parsed);
          setLastUpdated(latestUpdate(parsed));
        } catch (error) {
          console.error(error);
        }
      });

      socket.addEventListener("close", () => {
        if (!isActive) {
          return;
        }
        setSocketStatus("offline");
        reconnectTimer = window.setTimeout(connect, RECONNECT_DELAY_MS);
      });

      socket.addEventListener("error", () => {
        if (!isActive) {
          return;
        }
        setSocketStatus("offline");
        socket?.close();
      });
    };

    connect();

    return () => {
      isActive = false;
      if (reconnectTimer) {
        window.clearTimeout(reconnectTimer);
      }
      socket?.close();
    };
  }, []);

  const groups = groupRunningByHost(containers);

  return (
    <main className="min-h-screen bg-slate-950 px-6 py-10 text-slate-100 lg:px-16">
      <header className="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.4em] text-slate-500">
            Switchboard
          </p>
          <h1 className="mt-3 text-3xl font-semibold text-white">
            Running containers by host
          </h1>
          <p className="mt-2 text-sm text-slate-400">
            Monitor currently running containers and jump straight to the Web UI.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <ViewToggle view={viewMode} onChange={setViewMode} />
          <span
            className={`rounded-full px-4 py-2 text-xs font-semibold uppercase tracking-[0.2em] ${
              status === "error"
                ? "bg-rose-950 text-rose-200"
                : socketStatus === "offline"
                  ? "bg-amber-950 text-amber-200"
                  : isPending || socketStatus === "connecting"
                    ? "bg-sky-950 text-sky-200"
                    : "bg-emerald-950 text-emerald-200"
            }`}
          >
            {status === "error"
              ? "Offline"
              : socketStatus === "offline"
                ? "Disconnected"
                : socketStatus === "connecting"
                  ? "Connecting"
                  : isPending
                    ? "Refreshing"
                    : "Live"}
          </span>
          <button
            type="button"
            onClick={handleRefresh}
            className="rounded-full border border-slate-700 bg-slate-900 px-5 py-2 text-sm font-semibold text-slate-100 transition hover:border-slate-500"
          >
            Refresh now
          </button>
        </div>
      </header>

      <section className="mt-10">
        {status === "loading" && containers.length === 0 ? (
          <DashboardSkeleton />
        ) : status === "error" && containers.length === 0 ? (
          <div className="rounded-3xl border border-rose-500/40 bg-rose-950/40 p-8 text-center text-rose-100">
            <p className="text-lg font-semibold">Data unavailable</p>
            <p className="mt-2 text-sm text-rose-200">{errorMessage}</p>
          </div>
        ) : config ? (
          <div className="space-y-6">
            {viewMode === "list" ? (
              <ContainersTable containers={containers} config={config} />
            ) : groups.length ? (
              groups.map((group) => (
                <HostSection
                  key={group.host}
                  group={group}
                  config={config}
                />
              ))
            ) : (
              <div className="rounded-3xl border border-dashed border-slate-800 bg-slate-900/40 p-8 text-center text-slate-300">
                No running containers detected yet.
              </div>
            )}
            <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
              Last updated {lastUpdated}
            </p>
          </div>
        ) : null}
      </section>
    </main>
  );
}
