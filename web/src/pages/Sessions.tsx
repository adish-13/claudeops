import useSWR from "swr";
import { Link } from "react-router-dom";
import { api } from "@/lib/api";
import { PageHeader, Pill } from "@/components/ui";
import { humanAgo } from "@/lib/utils";

export function SessionsPage() {
  const { data } = useSWR("sessions", api.sessionsList, { refreshInterval: 10000 });
  return (
    <main className="overflow-y-auto p-8">
      <PageHeader
        crumb={<><Link to="/" className="text-accent hover:underline">home</Link> / all sessions</>}
        title="All sessions"
        subtitle={data ? `${data.sessions.length} sessions across ${data.project_count} projects` : "Loading…"}
      />
      {data && (
        <table className="w-full text-[12px]">
          <thead>
            <tr className="text-muted text-[11px] uppercase tracking-wide">
              <th className="text-left py-2 pr-3 font-medium">When</th>
              <th className="text-left py-2 pr-3 font-medium">Cwd</th>
              <th className="text-left py-2 pr-3 font-medium">Branch</th>
              <th className="text-left py-2 pr-3 font-medium">Workspace</th>
              <th className="text-left py-2 pr-3 font-medium">Last messages</th>
              <th className="text-left py-2 pr-3 font-medium">Events</th>
            </tr>
          </thead>
          <tbody>
            {data.sessions.map((s) => (
              <tr key={s.session_id} className="border-t border-border-soft hover:bg-hover">
                <td className="py-2 pr-3 text-muted">{humanAgo(s.last_activity)}</td>
                <td className="py-2 pr-3 font-mono">{s.cwd || s.project_dir}</td>
                <td className="py-2 pr-3 font-mono text-warm">{s.git_branch}</td>
                <td className="py-2 pr-3">
                  {s.workspace_link ? (
                    <Link to={s.workspace_link} className="text-accent hover:underline">{s.workspace_label}</Link>
                  ) : (
                    <Pill variant="muted">unbound</Pill>
                  )}
                </td>
                <td className="py-2 pr-3">
                  {s.last_user_preview && <div><span className="text-muted mr-1.5">→</span>{s.last_user_preview}</div>}
                  {s.last_assistant_text && <div><span className="text-muted mr-1.5">←</span>{s.last_assistant_text}</div>}
                </td>
                <td className="py-2 pr-3"><Link to={`/sessions/${s.session_id}`} className="text-accent hover:underline">{s.num_events}</Link></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </main>
  );
}
