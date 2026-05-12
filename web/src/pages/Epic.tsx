import { useParams, Link, useNavigate } from "react-router-dom";
import useSWR, { mutate } from "swr";
import { useState, useEffect } from "react";
import { api } from "@/lib/api";
import { Button, Card, CardTitle, KV, KVGrid, PageHeader, Pill, Textarea } from "@/components/ui";
import { humanAgo } from "@/lib/utils";

export function EpicPage() {
  const { slug } = useParams<{ slug: string }>();
  const { data, error, isLoading } = useSWR(slug ? `epic-${slug}` : null, () => api.epicGet(slug!));
  const navigate = useNavigate();
  const [contextDraft, setContextDraft] = useState("");
  const [contextDirty, setContextDirty] = useState(false);

  useEffect(() => {
    if (data) setContextDraft(data.epic.context_md);
  }, [data?.epic.context_md]);

  if (error) return <main className="p-8"><Card>Error: {String(error.message ?? error)}</Card></main>;
  if (isLoading || !data) return <main className="p-8 text-muted">Loading…</main>;

  const { epic, workspaces, unbound_sessions } = data;

  async function saveContext() {
    await api.epicSaveContext(slug!, contextDraft);
    setContextDirty(false);
    mutate(`epic-${slug}`);
  }

  async function archiveEpic() {
    if (!confirm("Archive this epic?")) return;
    await api.epicArchive(slug!);
    mutate("sidebar");
    navigate("/");
  }

  return (
    <main className="overflow-y-auto p-8">
      <PageHeader
        crumb={<><Link to="/" className="text-accent hover:underline">home</Link> / epic</>}
        title={epic.name}
        subtitle={epic.description}
        right={<Button variant="muted" onClick={archiveEpic}>archive epic</Button>}
      />

      <div className="space-y-3 max-w-4xl">
        <Card>
          <CardTitle>Repo</CardTitle>
          <KVGrid>
            <KV k="Path" v={epic.repo_path} />
            <KV k="Base branch" v={epic.base_branch} />
          </KVGrid>
        </Card>

        <Card>
          <CardTitle>Context</CardTitle>
          <Textarea
            value={contextDraft}
            onChange={(e) => { setContextDraft(e.target.value); setContextDirty(true); }}
            placeholder="Shared context, spec, links — markdown."
          />
          <div className="mt-3">
            <Button onClick={saveContext} disabled={!contextDirty}>Save context</Button>
          </div>
        </Card>

        <Card>
          <CardTitle>Workspaces ({workspaces.length})</CardTitle>
          {workspaces.length > 0 ? (
            <table className="w-full text-[12px] mt-1">
              <thead>
                <tr className="text-muted text-[11px] uppercase tracking-wide">
                  <th className="text-left py-2 pr-3 font-medium">Name</th>
                  <th className="text-left py-2 pr-3 font-medium">Branch</th>
                  <th className="text-left py-2 pr-3 font-medium">Diff</th>
                  <th className="text-left py-2 pr-3 font-medium">Sessions</th>
                  <th className="text-left py-2 pr-3 font-medium">PR</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {workspaces.map((w) => (
                  <tr key={w.slug} className="border-t border-border-soft hover:bg-hover">
                    <td className="py-2 pr-3">
                      <Link to={`/epics/${epic.slug}/workspaces/${w.slug}`} className="text-accent hover:underline">{w.name}</Link>
                    </td>
                    <td className="py-2 pr-3 font-mono text-warm">{w.branch_name}</td>
                    <td className="py-2 pr-3 font-mono">
                      {w.added > 0 || w.removed > 0 ? (
                        <span><span className="text-add">+{w.added}</span> <span className="text-del">-{w.removed}</span></span>
                      ) : <span className="text-muted">—</span>}
                    </td>
                    <td className="py-2 pr-3">{w.num_sessions}</td>
                    <td className="py-2 pr-3">
                      {w.pr_url ? <a href={w.pr_url} target="_blank" rel="noreferrer" className="text-accent hover:underline">PR ↗</a> : <Pill variant="muted">none</Pill>}
                    </td>
                    <td className="py-2 pr-3 text-right">
                      <button
                        onClick={async () => {
                          if (!confirm("Archive this workspace?")) return;
                          await api.workspaceArchive(epic.slug, w.slug);
                          mutate(`epic-${slug}`);
                          mutate("sidebar");
                        }}
                        className="text-muted text-[11px] hover:text-err"
                      >
                        archive
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="text-muted">No workspaces yet.</p>
          )}
          <div className="mt-3">
            <Link to={`/epics/${epic.slug}/workspaces/new`}><Button>+ New workspace</Button></Link>
          </div>
        </Card>

        {unbound_sessions.length > 0 && (
          <Card>
            <CardTitle>Unbound sessions in this repo ({unbound_sessions.length})</CardTitle>
            <p className="text-muted text-[12px] mb-2">
              Sessions whose cwd is inside <code>{epic.repo_path}</code> but not inside any workspace worktree.
            </p>
            <table className="w-full text-[12px]">
              <thead>
                <tr className="text-muted text-[11px] uppercase tracking-wide">
                  <th className="text-left py-2 pr-3 font-medium">When</th>
                  <th className="text-left py-2 pr-3 font-medium">Cwd</th>
                  <th className="text-left py-2 pr-3 font-medium">Branch</th>
                  <th className="text-left py-2 pr-3 font-medium">Last user message</th>
                </tr>
              </thead>
              <tbody>
                {unbound_sessions.map((s) => (
                  <tr key={s.session_id} className="border-t border-border-soft">
                    <td className="py-2 pr-3 text-muted">{humanAgo(s.last_activity)}</td>
                    <td className="py-2 pr-3 font-mono">{s.cwd}</td>
                    <td className="py-2 pr-3 font-mono text-warm">{s.git_branch}</td>
                    <td className="py-2 pr-3">{s.last_user_preview}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>
        )}
      </div>
    </main>
  );
}
