type StatusTone = {
  surface: string;
  dot: string;
  detail: string;
};

const STATUS_TONES: Record<string, StatusTone> = {
  healthy: {
    surface: "border-emerald-400/40 bg-emerald-500/15 text-emerald-50",
    dot: "bg-emerald-300 shadow-[0_0_10px_rgba(16,185,129,0.75)]",
    detail: "border-emerald-300/30 bg-emerald-500/20 text-emerald-100",
  },
  warning: {
    surface: "border-amber-400/40 bg-amber-500/15 text-amber-50",
    dot: "bg-amber-300 shadow-[0_0_10px_rgba(251,191,36,0.75)]",
    detail: "border-amber-300/30 bg-amber-500/20 text-amber-100",
  },
  danger: {
    surface: "border-rose-400/40 bg-rose-500/15 text-rose-50",
    dot: "bg-rose-300 shadow-[0_0_10px_rgba(251,113,133,0.75)]",
    detail: "border-rose-300/30 bg-rose-500/20 text-rose-100",
  },
  info: {
    surface: "border-sky-400/40 bg-sky-500/15 text-sky-50",
    dot: "bg-sky-300 shadow-[0_0_10px_rgba(56,189,248,0.75)]",
    detail: "border-sky-300/30 bg-sky-500/20 text-sky-100",
  },
  neutral: {
    surface: "border-slate-500/40 bg-slate-500/15 text-slate-50",
    dot: "bg-slate-300 shadow-[0_0_10px_rgba(148,163,184,0.6)]",
    detail: "border-slate-400/30 bg-slate-500/20 text-slate-100",
  },
};

function pickStatusTone(status: string): StatusTone {
  const normalized = status.toLowerCase();

  if (
    normalized.includes("unhealthy") ||
    normalized.includes("dead") ||
    normalized.includes("exited")
  ) {
    return STATUS_TONES.danger;
  }

  if (normalized.includes("starting") || normalized.includes("restarting")) {
    return STATUS_TONES.warning;
  }

  if (normalized.includes("healthy")) {
    return STATUS_TONES.healthy;
  }

  if (normalized.includes("paused") || normalized.includes("created")) {
    return STATUS_TONES.neutral;
  }

  if (normalized.includes("up") || normalized.includes("running")) {
    return STATUS_TONES.info;
  }

  return STATUS_TONES.neutral;
}

function splitStatus(status: string): { main: string; detail?: string } {
  const match = status.match(/^(.*?)(?:\s*\(([^)]+)\))?$/);
  if (!match) {
    return { main: status };
  }

  const main = match[1].trim() || status;
  const rawDetail = match[2]?.trim();
  const detail = rawDetail?.replace(/^health:\s*/i, "").trim();

  return detail ? { main, detail } : { main };
}

interface StatusPillProps {
  status: string;
}

export function StatusPill({ status }: StatusPillProps) {
  const tone = pickStatusTone(status);
  const { main, detail } = splitStatus(status);

  return (
    <span
      className={`inline-flex flex-wrap items-center gap-2 rounded-full border px-3 py-1 text-[11px] font-semibold leading-none shadow-[0_10px_24px_rgba(15,23,42,0.45)] backdrop-blur ${tone.surface}`}
    >
      <span
        aria-hidden="true"
        className={`h-2 w-2 rounded-full ${tone.dot}`}
      />
      <span>{main}</span>
      {detail ? (
        <span
          className={`rounded-full border px-2 py-0.5 text-[10px] font-medium uppercase tracking-[0.08em] ${tone.detail}`}
        >
          {detail}
        </span>
      ) : null}
    </span>
  );
}
