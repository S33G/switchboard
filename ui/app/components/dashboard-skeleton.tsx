export function DashboardSkeleton() {
  return (
    <div className="space-y-6">
      {[0, 1].map((index) => (
        <section
          key={index}
          className="rounded-3xl border border-slate-800 bg-slate-900/40 p-6"
        >
          <div className="h-5 w-40 animate-pulse rounded-full bg-slate-800" />
          <div className="mt-6 grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
            {[0, 1, 2].map((card) => (
              <div
                key={card}
                className="h-40 animate-pulse rounded-2xl border border-slate-800 bg-slate-900/60"
              />
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}
