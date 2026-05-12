import { Link, useLocation, useParams } from "react-router-dom";
import useSWR from "swr";
import { api, SidebarWorkspace } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Plus, Database, ListIcon, GitBranch, MessageSquareText, Activity } from "lucide-react";

export function Sidebar() {
  const { data, isLoading } = useSWR("sidebar", api.sidebar, { refreshInterval: 5000 });
  const params = useParams();
  const loc = useLocation();
  const activeEpic = params.slug;
  const activeWs = params.wsslug;

  return (
    <aside className="border-r border-border bg-panel2 overflow-y-auto py-3 px-2 flex flex-col">
      <div className="flex items-center justify-between px-2 py-1.5 mb-3">
        <Link to="/" className="font-semibold text-[14px] text-fg flex items-center gap-2">
          <Activity size={14} className="text-accent" />
          claudeops
        </Link>
        <Link to="/debug" className="text-[11px] text-muted hover:text-accent" title="DB inspector">
          <Database size={13} />
        </Link>
      </div>

      <div className="flex-1">
        {isLoading && <div className="px-3 text-muted text-[12px]">loading…</div>}
        {data?.epics?.map((e) => (
          <div key={e.slug} className="mb-3">
            <Link
              to={`/epics/${e.slug}`}
              className={cn(
                "flex items-center justify-between rounded-md px-2.5 py-1.5 font-semibold text-[13px]",
                activeEpic === e.slug ? "bg-hover text-accent" : "text-fg hover:bg-hover/60"
              )}
            >
              <span className="truncate">{e.name}</span>
            </Link>
            <div className="ml-2 mt-1 pl-3 border-l border-border-soft">
              {e.workspaces.map((w) => (
                <WorkspaceCard
                  key={w.slug}
                  ws={w}
                  active={activeEpic === e.slug && activeWs === w.slug}
                />
              ))}
              <Link
                to={`/epics/${e.slug}/workspaces/new`}
                className="block rounded-md px-2.5 py-1 text-[11px] text-muted hover:text-accent hover:bg-hover/60"
              >
                <Plus size={11} className="inline mr-1" />
                new workspace
              </Link>
            </div>
          </div>
        ))}
        {data && data.epics?.length === 0 && (
          <div className="px-3 text-muted text-[12px]">No epics yet.</div>
        )}
      </div>

      <div className="mt-2 border-t border-border pt-2 space-y-1">
        <Link
          to="/epics/new"
          className="block rounded-md px-2.5 py-1.5 text-[12px] text-accent hover:bg-hover/60"
        >
          <Plus size={12} className="inline mr-1" />
          new epic
        </Link>
        <Link
          to="/sessions"
          className={cn(
            "block rounded-md px-2.5 py-1.5 text-[12px] hover:bg-hover/60",
            loc.pathname === "/sessions" ? "text-accent bg-hover" : "text-muted"
          )}
        >
          <ListIcon size={12} className="inline mr-1" />
          all sessions
        </Link>
      </div>
      <div className="mt-2 px-3 text-[10px] text-dim">
        indexed {data?.indexed_at ?? "—"}
      </div>
    </aside>
  );
}

function WorkspaceCard({ ws, active }: { ws: SidebarWorkspace; active: boolean }) {
  return (
    <Link
      to={`/epics/${ws.epic_slug}/workspaces/${ws.slug}`}
      className={cn(
        "block rounded-md px-2.5 py-2 mb-1",
        active ? "bg-hover text-accent" : "text-fg hover:bg-hover/60"
      )}
    >
      <div className="flex items-center justify-between gap-1.5">
        <span className="text-[12px] font-medium truncate">{ws.name}</span>
        <span className="flex gap-1.5 shrink-0 text-[10px] font-mono">
          {ws.added > 0 && <span className="text-add">+{ws.added}</span>}
          {ws.removed > 0 && <span className="text-del">-{ws.removed}</span>}
        </span>
      </div>
      <div className="flex items-center justify-between mt-0.5">
        <span className="text-warm font-mono text-[10px] truncate flex items-center gap-1">
          <GitBranch size={9} />
          {ws.branch_name}
        </span>
        <span className="flex gap-1.5 items-center shrink-0 text-[10px]">
          {ws.has_pr && <span className="bg-ok/15 text-ok rounded-full px-1.5">PR</span>}
          {ws.num_sessions > 0 && (
            <span className="text-dim flex items-center gap-0.5">
              <MessageSquareText size={9} />
              {ws.num_sessions}
            </span>
          )}
        </span>
      </div>
    </Link>
  );
}
