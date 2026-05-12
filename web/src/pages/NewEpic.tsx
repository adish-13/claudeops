import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "@/lib/api";
import { Button, Card, Input, PageHeader, Textarea } from "@/components/ui";

export function NewEpicPage() {
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    const data = new FormData(e.currentTarget);
    try {
      const epic = await api.epicCreate({
        name: String(data.get("name") || "").trim(),
        slug: String(data.get("slug") || "").trim(),
        description: String(data.get("description") || "").trim(),
        repo_path: String(data.get("repo_path") || "").trim(),
        base_branch: String(data.get("base_branch") || "master").trim() || "master",
        context_md: String(data.get("context_md") || ""),
      });
      navigate(`/epics/${epic.slug}`);
    } catch (err: any) {
      setError(err.message || String(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="overflow-y-auto p-8">
      <PageHeader crumb={<a href="/" className="text-accent hover:underline">home</a>} title="New epic" />
      <form onSubmit={onSubmit} className="max-w-2xl">
        <Card className="space-y-3">
          {error && <div className="rounded-md border border-err/30 bg-err/10 px-3 py-2 text-err text-[12px]">{error}</div>}
          <Field label="Name"><Input name="name" placeholder="postgres recovery" required /></Field>
          <Field label="Slug (lowercase, dashes)"><Input name="slug" placeholder="postgres-recovery" required /></Field>
          <Field label="Description (optional)"><Input name="description" placeholder="One-line summary" /></Field>
          <Field label="Repo path (absolute, ~ ok)"><Input name="repo_path" placeholder="~/repos/myrepo" required /></Field>
          <Field label="Base branch"><Input name="base_branch" defaultValue="master" required /></Field>
          <Field label="Initial context (markdown)">
            <Textarea name="context_md" placeholder="Spec, links, decisions — anything you want every workspace under this epic to inherit." />
          </Field>
          <div className="pt-3">
            <Button type="submit" disabled={submitting}>{submitting ? "Creating…" : "Create epic"}</Button>
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
