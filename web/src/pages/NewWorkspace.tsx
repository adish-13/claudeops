import { FormEvent, useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import useSWR, { mutate } from "swr";
import { api } from "@/lib/api";
import { Button, Card, Input, PageHeader } from "@/components/ui";

export function NewWorkspacePage() {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const { data: suggest } = useSWR(slug ? `suggest-${slug}` : null, () => api.workspaceSuggest(slug!));
  const { data: epicData } = useSWR(slug ? `epic-${slug}` : null, () => api.epicGet(slug!));
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // pre-populate when suggestion arrives
  const [name, setName] = useState("");
  const [wsSlug, setWsSlug] = useState("");
  const [branch, setBranch] = useState("");
  const [path, setPath] = useState("");
  useEffect(() => {
    if (suggest && !branch) setBranch(suggest.branch_name);
    if (suggest && !path) setPath(suggest.worktree_path);
    if (suggest && !wsSlug) setWsSlug(suggest.slug);
  }, [suggest]);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await api.workspaceCreate(slug!, { name, slug: wsSlug, branch_name: branch, worktree_path: path });
      mutate("sidebar");
      mutate(`epic-${slug}`);
      navigate(`/epics/${slug}/workspaces/${wsSlug}`);
    } catch (err: any) {
      setError(err.message || String(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="overflow-y-auto p-8">
      <PageHeader
        crumb={<><Link to="/" className="text-accent hover:underline">home</Link> / <Link to={`/epics/${slug}`} className="text-accent hover:underline">{epicData?.epic.name ?? slug}</Link> / new workspace</>}
        title="New workspace"
      />
      <form onSubmit={onSubmit} className="max-w-2xl">
        <Card className="space-y-3">
          {error && <div className="rounded-md border border-err/30 bg-err/10 px-3 py-2 text-err text-[12px]">{error}</div>}
          <Field label="Workspace name"><Input value={name} onChange={(e) => setName(e.target.value)} placeholder="snappable rewrite" required /></Field>
          <Field label="Slug"><Input value={wsSlug} onChange={(e) => setWsSlug(e.target.value)} required /></Field>
          <Field label="Branch name"><Input value={branch} onChange={(e) => setBranch(e.target.value)} required /></Field>
          <Field label="Worktree path"><Input value={path} onChange={(e) => setPath(e.target.value)} required /></Field>
          <p className="text-muted text-[12px]">
            On submit: <code>git worktree add -b {branch} {path} {epicData?.epic.base_branch ?? "master"}</code>
          </p>
          <div className="pt-3">
            <Button type="submit" disabled={submitting}>{submitting ? "Creating…" : "Create workspace"}</Button>
          </div>
        </Card>
      </form>
    </main>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <div className="text-muted text-[12px] mb-1">{label}</div>
      {children}
    </label>
  );
}
