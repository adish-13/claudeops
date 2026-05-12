import useSWR from "swr";
import { Link } from "react-router-dom";
import { api } from "@/lib/api";
import { Button, Card, CardTitle, PageHeader } from "@/components/ui";

export function HomePage() {
  const { data: home } = useSWR("home", api.home);
  const { data: sidebar } = useSWR("sidebar", api.sidebar);
  const hasEpics = (sidebar?.epics?.length ?? 0) > 0;

  return (
    <main className="overflow-y-auto p-8">
      <PageHeader crumb="Home" title="claudeops" />

      {hasEpics ? (
        <div className="space-y-3 max-w-2xl">
          <Card>
            <CardTitle>Stats</CardTitle>
            <p className="text-[12px] text-muted">
              {home?.total_sessions ?? 0} sessions across {home?.project_count ?? 0} projects.
            </p>
            <p className="mt-3 space-x-3 text-[12px]">
              <Link to="/sessions" className="text-accent hover:underline">View all sessions →</Link>
              <Link to="/debug" className="text-accent hover:underline">DB debug →</Link>
            </p>
          </Card>
          <p className="text-muted text-[12px]">
            Pick an epic from the sidebar, or <Link to="/epics/new" className="text-accent hover:underline">start a new one</Link>.
          </p>
        </div>
      ) : (
        <Card className="max-w-xl">
          <CardTitle>No epics yet</CardTitle>
          <p>
            An <strong>epic</strong> is a top-level project — like "postgres recovery" or "oncall tasks".
            Each epic owns a repo, shared context, and any number of <strong>workspaces</strong> (worktrees)
            where Claude sessions actually run.
          </p>
          <div className="mt-4">
            <Link to="/epics/new"><Button>Create your first epic</Button></Link>
          </div>
        </Card>
      )}
    </main>
  );
}
