"use client";

import { useCallback, useEffect, useMemo, useRef, useState, useTransition } from "react";
import uFuzzy from "@leeoniya/ufuzzy";

import { getConfig, getContainers } from "./lib/api";
import { groupRunningByHost, latestUpdate } from "./lib/helpers";
import type { Config, Container } from "./lib/types";
import {
  applyContainerDiff,
  buildWebSocketUrl,
  parseWebSocketMessage,
  type WebSocketStatus,
} from "./lib/websocket";
import { DashboardSkeleton } from "./components/dashboard-skeleton";
import { HostSection } from "./components/host-section";
import { ContainersTable } from "./components/containers-table";
import { ViewToggle } from "./components/view-toggle";
import { useViewMode } from "./lib/use-view-mode";
import { useContainerSearch } from "./lib/use-container-search";
import { useColumnConfig } from "./lib/use-column-config";
import { ColumnConfigurator } from "./components/column-configurator";

type LoadStatus = "idle" | "loading" | "success" | "error";
const RECONNECT_DELAY_MS = 5000;
const ANIMATION_DURATION_MS = 600;

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
  const [searchQuery, setSearchQuery] = useContainerSearch();
  const [columnConfig, saveColumnConfig, resetColumnConfig] = useColumnConfig();

  const [animatingIds, setAnimatingIds] = useState<{
    added: Set<string>;
    updated: Set<string>;
    removed: Set<string>;
  }>({ added: new Set(), updated: new Set(), removed: new Set() });

  const animationTimeoutRef = useRef<number | null>(null);

  const clearAnimations = useCallback(() => {
    if (animationTimeoutRef.current) {
      window.clearTimeout(animationTimeoutRef.current);
    }
    animationTimeoutRef.current = window.setTimeout(() => {
      setAnimatingIds({ added: new Set(), updated: new Set(), removed: new Set() });
    }, ANIMATION_DURATION_MS);
  }, []);

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
          const message = parseWebSocketMessage(JSON.parse(event.data));
          if (!message) {
            return;
          }

          if (message.type === "snapshot") {
            setContainers(message.containers);
            setLastUpdated(latestUpdate(message.containers));
          } else if (message.type === "diff") {
            setContainers((current) => {
              const result = applyContainerDiff(current, message.diff);

              if (result.addedIds.size > 0 || result.updatedIds.size > 0 || result.removedIds.size > 0) {
                setAnimatingIds({
                  added: result.addedIds,
                  updated: result.updatedIds,
                  removed: result.removedIds,
                });
                clearAnimations();
              }

              setLastUpdated(latestUpdate(result.containers));
              return result.containers;
            });
          }
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
      if (animationTimeoutRef.current) {
        window.clearTimeout(animationTimeoutRef.current);
      }
      socket?.close();
    };
  }, [clearAnimations]);

  const runningContainers = useMemo(
    () => containers.filter((container) => container.state === "running"),
    [containers]
  );

  const filteredContainers = useMemo(() => {
    const trimmedQuery = searchQuery.trim();
    if (!trimmedQuery) {
      return runningContainers;
    }

    const haystack = runningContainers.map((container) =>
      [container.name, container.image, container.host, container.status]
        .filter(Boolean)
        .join(" ")
    );

    const matcher = new uFuzzy();
    const result = matcher.filter(haystack, trimmedQuery);
    const matchIndexes = result ?? [];
    const matchSet = new Set(matchIndexes);

    return runningContainers.filter((_, index) => matchSet.has(index));
  }, [runningContainers, searchQuery]);

  const groups = groupRunningByHost(filteredContainers);
  const hasSearch = searchQuery.trim().length > 0;

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
          <div className="flex-1 min-w-[220px]">
            <label className="sr-only" htmlFor="container-search">
              Search running containers
            </label>
            <input
              id="container-search"
              type="search"
              value={searchQuery}
              onChange={(event) => {
                void setSearchQuery(event.target.value);
              }}
              placeholder="Search running containers"
              className="w-full rounded-full border border-slate-800 bg-slate-900/70 px-4 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-slate-500 focus:outline-none"
            />
          </div>
          <ViewToggle view={viewMode} onChange={setViewMode} />
          <ColumnConfigurator
            config={columnConfig}
            onSave={saveColumnConfig}
            onReset={resetColumnConfig}
          />
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
            Refresh
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
              <ContainersTable
                containers={filteredContainers}
                config={config}
                columnConfig={columnConfig}
                animatingIds={animatingIds}
                hasSearch={hasSearch}
              />
            ) : groups.length ? (
              groups.map((group) => (
                <HostSection
                  key={group.host}
                  group={group}
                  config={config}
                  columnConfig={columnConfig}
                  animatingIds={animatingIds}
                  hasSearch={hasSearch}
                />
              ))
            ) : (
              <div className="rounded-3xl border border-dashed border-slate-800 bg-slate-900/40 p-8 text-center text-slate-300">
                {hasSearch
                  ? "No running containers match your search yet."
                  : "No running containers detected yet."}
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
