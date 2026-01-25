"use client";

import { useEffect } from "react";

interface ErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function Error({ error, reset }: ErrorProps) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main className="min-h-screen bg-slate-950 px-6 py-10 text-slate-100 lg:px-16">
      <div className="max-w-xl rounded-3xl border border-rose-500/40 bg-rose-950/40 p-8">
        <h1 className="text-2xl font-semibold text-white">Something went wrong</h1>
        <p className="mt-3 text-sm text-rose-100">
          We could not load container data from the Switchboard backend. Please
          verify the API is reachable and try again.
        </p>
        <button
          type="button"
          onClick={reset}
          className="mt-6 rounded-full border border-rose-300/40 px-5 py-2 text-sm font-semibold text-rose-100 transition hover:border-rose-200"
        >
          Try again
        </button>
      </div>
    </main>
  );
}
