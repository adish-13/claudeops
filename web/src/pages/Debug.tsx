import useSWR from "swr";
import { Link } from "react-router-dom";
import { api } from "@/lib/api";
import { Card, CardTitle, KV, KVGrid, PageHeader } from "@/components/ui";

export function DebugPage() {
  const { data } = useSWR("debug", api.debug, { refreshInterval: 10000 });
  if (!data) return <main className="p-8 text-muted">Loading…</main>;
  return (
    <main className="overflow-y-auto p-8">
      <PageHeader
        crumb={<><Link to="/" className="text-accent hover:underline">home</Link> / debug</>}
        title="Database debug"
      />
      <div className="space-y-3 max-w-3xl">
        <Card>
          <CardTitle>SQLite file</CardTitle>
          <KVGrid>
            <KV k="Path" v={data.db_path} />
            <KV k="Size" v={`${data.db_size_bytes} bytes (main) + ${data.wal_size_bytes} bytes (WAL)`} />
          </KVGrid>
        </Card>
        <Card>
          <CardTitle>Row counts</CardTitle>
          <table className="w-full text-[12px]">
            <tbody>
              <tr className="border-t border-border-soft"><td className="py-2 pr-3">epics</td><td className="py-2 pr-3 font-mono">{data.counts.epics}</td></tr>
              <tr className="border-t border-border-soft"><td className="py-2 pr-3">workspaces</td><td className="py-2 pr-3 font-mono">{data.counts.workspaces}</td></tr>
              <tr className="border-t border-border-soft"><td className="py-2 pr-3">sessions</td><td className="py-2 pr-3 font-mono">{data.counts.sessions}</td></tr>
            </tbody>
          </table>
        </Card>
        <DebugTable title="Recent epics" rows={data.recent_epics} />
        <DebugTable title="Recent workspaces" rows={data.recent_workspaces} />
        <DebugTable title="Recent sessions (last 10 by activity)" rows={data.recent_sessions} />
      </div>
    </main>
  );
}

function DebugTable({ title, rows }: { title: string; rows: Record<string, unknown>[] }) {
  if (!rows || rows.length === 0) return null;
  const cols = Object.keys(rows[0]);
  return (
    <Card>
      <CardTitle>{title}</CardTitle>
      <table className="w-full text-[12px]">
        <thead>
          <tr className="text-muted text-[11px] uppercase tracking-wide">
            {cols.map((c) => <th key={c} className="text-left py-2 pr-3 font-medium">{c}</th>)}
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i} className="border-t border-border-soft">
              {cols.map((c) => <td key={c} className="py-2 pr-3 font-mono">{String(r[c] ?? "")}</td>)}
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  );
}
