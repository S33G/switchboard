import { DashboardSkeleton } from "./components/dashboard-skeleton";

export default function Loading() {
  return (
    <main className="min-h-screen bg-slate-950 px-6 py-10 text-slate-100 lg:px-16">
      <div className="mb-10 space-y-3">
        <div className="h-4 w-24 animate-pulse rounded-full bg-slate-800" />
        <div className="h-8 w-72 animate-pulse rounded-full bg-slate-800" />
        <div className="h-4 w-96 animate-pulse rounded-full bg-slate-800" />
      </div>
      <DashboardSkeleton />
    </main>
  );
}
