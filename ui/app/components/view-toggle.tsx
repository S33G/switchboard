"use client";

import type { ViewMode } from "../lib/use-view-mode";

interface ViewToggleProps {
  view: ViewMode;
  onChange: (view: ViewMode) => void;
}

export function ViewToggle({ view, onChange }: ViewToggleProps) {
  const currentView: ViewMode = view === "list" ? "list" : "grid";

  return (
    <div className="inline-flex items-center gap-2 rounded-full border border-slate-800 bg-slate-900/60 p-1">
      <button
        type="button"
        onClick={() => onChange("grid")}
        aria-pressed={currentView === "grid"}
        className={`rounded-full px-4 py-2 text-xs font-semibold uppercase tracking-[0.2em] transition ${
          currentView === "grid"
            ? "bg-slate-100 text-slate-950"
            : "text-slate-400 hover:text-slate-200"
        }`}
      >
        Grid
      </button>
      <button
        type="button"
        onClick={() => onChange("list")}
        aria-pressed={currentView === "list"}
        className={`rounded-full px-4 py-2 text-xs font-semibold uppercase tracking-[0.2em] transition ${
          currentView === "list"
            ? "bg-slate-100 text-slate-950"
            : "text-slate-400 hover:text-slate-200"
        }`}
      >
        List
      </button>
    </div>
  );
}
