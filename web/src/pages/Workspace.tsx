import { Link, useParams } from "react-router-dom";
import useSWR, { mutate } from "swr";
import { useState } from "react";
import { api, DiffStat } from "@/lib/api";
import { Button, Card, CardTitle, Input, KV, KVGrid, PageHeader, Pill } from "@/components/ui";
import { Terminal } from "@/components/Terminal";
import { humanAgo } from "@/lib/utils";
import { ExternalLink, Play, RotateCcw, Square, GitBranch as GitBranchIcon, Folder } from "lucide-react";

export function WorkspacePage() {
  const { slug, wsslug } = useParams<{ slug: string; wsslug: string }>();
  const key = `workspace-${slug}-${wsslug}`;
  const { data, error, isLoading } = useSWR(key, () => api.workspaceGet(slug!, wsslug!), {
    refreshInterval: 5000,
  });

  if (error) return <main className="p-8"><Card>Error: {String(error.message ?? error)}</Card></main>;
  if (isLoading || !data) return <main className="p-8 text-muted">Loading…</main>;

  const { workspace, epic, sessions, diff, files, terminal_live, worktree_short } = data;

  async function startTerm(sessionId?: string) {
    await api.workspaceTermStart(slug!, wsslug!, sessionId);
    mutate(key);
  }
  async function killTerm() {
    if (!confirm("Stop the embedded claude session?")) return;
    await api.workspaceTermKill(slug!, wsslug!);
    mutate(key);
  }
  async function launchITerm() {
    await api.workspaceLaunchITerm(slug!, wsslug!);
  }

  return (
    <div className="grid h-full overflow-hidden" style={{ gridTemplateColumns: "1fr 380px" }}>
      <div className="flex flex-col overflow-hidden border-r border-border">
        <header className="flex items-center justify-between gap-3 border-b border-border px-5 py-3">
          <div>
            <div className="text-[12px] text-muted">
              <Link to="/" className="text-accent hover:underline">home</Link> /{" "}
              <Link to={`/epics/${epic.slug}`} className="text-accent hover:underline">{epic.name}</Link>
            </div>
            <h2 className="text-[16px] font-semibold mt-0.5 flex items-center gap-2">
              {workspace.name}
              <span className="font-mono text-warm text-[12px] font-normal flex items-center gap-1">
                <GitBranchIcon size={11} />
                {workspace.branch_name}
              </span>
            </h2>
          </div>
          <div className="flex items-center gap-2">
            {terminal_live ? (
              <>
                <Pill variant="ok">terminal live</Pill>
                <Button variant="ghost" onClick={killTerm}><Square size={12} />stop</Button>
              </>
            ) : (
              <>
                <Button onClick={() => startTerm()}><Play size={12} />launch fresh</Button>
                {sessions.length > 0 && (
                  <Button variant="ghost" onClick={() => startTerm(sessions[0].session_id)} title={`claude --resume ${sessions[0].session_id.slice(0, 8)}`}>
                    <RotateCcw size={12} />resume last
                  </Button>
                )}
              </>
            )}
            <Button variant="ghost" onClick={launchITerm}><ExternalLink size={12} />iTerm</Button>
          </div>
        </header>

        {terminal_live ? (
          <div className="flex-1 min-h-0 min-w-0 bg-black overflow-hidden">
            <Terminal workspaceId={workspace.id} />
          </div>
        ) : (
          <div className="overflow-y-auto p-6 space-y-3">
            <Card>
              <CardTitle>Workspace</CardTitle>
              <KVGrid>
                <KV k="Worktree" v={worktree_short} />
                <KV k="Created" v={humanAgo(workspace.created_at)} />
                <KV k="PR" v={
                  workspace.pr_url
                    ? <a href={workspace.pr_url} target="_blank" rel="noreferrer" className="text-accent hover:underline">{workspace.pr_url}</a>
                    : <Pill variant="muted">none yet</Pill>
                } />
              </KVGrid>
              <div className="mt-3">
                <PRForm initial={workspace.pr_url} onSave={async (url) => {
                  await api.workspaceSavePR(slug!, wsslug!, url);
                  mutate(key);
                  mutate("sidebar");
                }} />
              </div>
            </Card>

            <Card>
              <CardTitle>Sessions ({sessions.length})</CardTitle>
              {sessions.length > 0 ? (
                <table className="w-full text-[12px]">
                  <thead>
                    <tr className="text-muted text-[11px] uppercase tracking-wide">
                      <th className="text-left py-2 pr-3 font-medium">When</th>
                      <th className="text-left py-2 pr-3 font-medium">Session</th>
                      <th className="text-left py-2 pr-3 font-medium">Last messages</th>
                      <th className="text-left py-2 pr-3 font-medium">Events</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {sessions.map((s) => (
                      <tr key={s.session_id} className="border-t border-border-soft">
                        <td className="py-2 pr-3 text-muted">{humanAgo(s.last_activity)}</td>
                        <td className="py-2 pr-3 font-mono">
                          <Link to={`/sessions/${s.session_id}`} className="text-accent hover:underline">{s.session_id.slice(0, 8)}</Link>
                        </td>
                        <td className="py-2 pr-3">
                          {s.last_user_preview && <div className="text-fg"><span className="text-muted mr-1.5">→</span>{s.last_user_preview}</div>}
                          {s.last_assistant_text && <div className="text-fg"><span className="text-muted mr-1.5">←</span>{s.last_assistant_text}</div>}
                        </td>
                        <td className="py-2 pr-3">{s.num_events}</td>
                        <td className="py-2 pr-3 text-right">
                          {!terminal_live && (
                            <button
                              onClick={() => startTerm(s.session_id)}
                              className="text-accent text-[11px] hover:underline inline-flex items-center gap-1"
                              title={`claude --resume ${s.session_id.slice(0, 8)}`}
                            >
                              <RotateCcw size={10} />resume
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : (
                <p className="text-muted">No sessions yet — click "launch claude" above to start one in the browser, or "iTerm" for a real iTerm tab.</p>
              )}
            </Card>

            {epic.context_md && (
              <Card>
                <CardTitle>Epic context</CardTitle>
                <pre className="whitespace-pre-wrap font-mono text-[12px] text-fg">{epic.context_md}</pre>
              </Card>
            )}
          </div>
        )}
      </div>

      <aside className="overflow-y-auto bg-panel2 p-4">
        <CardTitle>Diff vs {epic.base_branch}</CardTitle>
        <div className="text-[12px] text-muted mb-3">
          {diff.files_changed} files <span className="text-add">+{diff.added}</span> <span className="text-del">-{diff.removed}</span>
        </div>
        <FileList files={files} />

        <CardTitle className="mt-6">PR</CardTitle>
        {workspace.pr_url ? (
          <a href={workspace.pr_url} target="_blank" rel="noreferrer" className="text-[12px] text-accent break-all hover:underline">
            {workspace.pr_url} ↗
          </a>
        ) : (
          <p className="text-muted text-[12px]">No PR linked yet.</p>
        )}

        <CardTitle className="mt-6">Path</CardTitle>
        <code className="text-[11px] text-muted break-all flex items-center gap-1.5">
          <Folder size={11} />{worktree_short}
        </code>
      </aside>
    </div>
  );
}

function FileList({ files }: { files: DiffStat[] }) {
  if (!files || files.length === 0) {
    return <div className="text-muted text-[12px]">No changes</div>;
  }
  return (
    <div>
      {files.map((f) => (
        <div key={f.path} className="flex items-center justify-between rounded px-2 py-1.5 hover:bg-hover">
          <span className="inline-block w-5 text-center font-mono text-[10px] text-muted">{f.status}</span>
          <span className="font-mono text-[12px] text-fg flex-1 truncate min-w-0" title={f.path}>{f.path}</span>
          <span className="flex gap-1.5 shrink-0 ml-2 font-mono text-[11px]">
            {f.added > 0 && <span className="text-add">+{f.added}</span>}
            {f.removed > 0 && <span className="text-del">-{f.removed}</span>}
          </span>
        </div>
      ))}
    </div>
  );
}

function PRForm({ initial, onSave }: { initial: string; onSave: (url: string) => Promise<void> }) {
  const [val, setVal] = useState(initial);
  const [saving, setSaving] = useState(false);
  return (
    <div className="flex gap-2">
      <Input value={val} onChange={(e) => setVal(e.target.value)} placeholder="https://github.com/.../pull/123" />
      <Button variant="ghost" disabled={saving} onClick={async () => { setSaving(true); try { await onSave(val); } finally { setSaving(false); } }}>
        {saving ? "Saving…" : "Save PR"}
      </Button>
    </div>
  );
}
