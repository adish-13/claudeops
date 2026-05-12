import useSWR from "swr";
import { Link, useParams } from "react-router-dom";
import { api } from "@/lib/api";
import { Card, PageHeader } from "@/components/ui";
import { humanAgo, shortId } from "@/lib/utils";
import { Wrench, User, Bot } from "lucide-react";

export function TranscriptPage() {
  const { id } = useParams<{ id: string }>();
  const { data, error, isLoading } = useSWR(id ? `session-${id}` : null, () => api.sessionGet(id!));
  if (error) return <main className="p-8"><Card>Error: {String(error.message ?? error)}</Card></main>;
  if (isLoading || !data) return <main className="p-8 text-muted">Loading…</main>;
  const { session, messages } = data;
  return (
    <main className="overflow-y-auto p-8">
      <PageHeader
        crumb={<><Link to="/" className="text-accent hover:underline">home</Link> / <Link to="/sessions" className="text-accent hover:underline">sessions</Link> / {shortId(id!)}</>}
        title={`Session ${shortId(id!)}`}
        subtitle={<>cwd: <code className="text-fg">{session.cwd}</code> · branch: <code className="text-warm">{session.git_branch}</code></>}
      />
      <div className="max-w-4xl space-y-2">
        {messages.length === 0 && <p className="text-muted">No messages parsed.</p>}
        {messages.map((m, i) => {
          if (m.role === "tool") {
            return (
              <div key={i} className="rounded-md border border-dashed border-border bg-panel px-3 py-1.5 text-[11px] text-muted font-mono flex items-center justify-between max-w-md">
                <span><Wrench size={11} className="inline mr-1.5" />{m.tool_name}</span>
                <span>{humanAgo(m.timestamp)}</span>
              </div>
            );
          }
          const isUser = m.role === "user";
          return (
            <div key={i} className={`rounded-md p-3 ${isUser ? "bg-accent/[0.06] border-l-2 border-accent" : "bg-ok/[0.04] border-l-2 border-ok"}`}>
              <div className="flex justify-between text-[11px] uppercase tracking-wide text-muted mb-1.5 items-center">
                <span className="flex items-center gap-1.5">
                  {isUser ? <User size={11} /> : <Bot size={11} />}
                  {m.role}{m.model && ` · ${m.model}`}
                </span>
                <span>{humanAgo(m.timestamp)}</span>
              </div>
              <div className="whitespace-pre-wrap font-mono text-[12px] text-fg leading-relaxed">{m.text}</div>
            </div>
          );
        })}
      </div>
    </main>
  );
}
