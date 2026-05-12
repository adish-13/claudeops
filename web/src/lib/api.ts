// All HTTP calls to the Go backend live here.
// Anything else in the app uses these functions instead of `fetch` directly.

export type Epic = {
  id: number;
  slug: string;
  name: string;
  description: string;
  repo_path: string;
  base_branch: string;
  context_md: string;
  created_at: string;
};

export type Workspace = {
  id: number;
  epic_id: number;
  slug: string;
  name: string;
  branch_name: string;
  worktree_path: string;
  pr_url: string;
  created_at: string;
};

export type SidebarWorkspace = {
  epic_slug: string;
  slug: string;
  name: string;
  branch_name: string;
  num_sessions: number;
  added: number;
  removed: number;
  has_pr: boolean;
};

export type SidebarEpic = {
  slug: string;
  name: string;
  workspaces: SidebarWorkspace[];
};

export type Session = {
  session_id: string;
  project_dir: string;
  cwd: string;
  git_branch: string;
  model: string;
  last_activity: string;
  last_user_preview: string;
  last_assistant_text: string;
  num_events: number;
  workspace_id: number | null;
  // enrichments from server
  workspace_link?: string;
  workspace_label?: string;
};

export type DiffStat = {
  path: string;
  status: string;
  added: number;
  removed: number;
};

export type DiffSummary = {
  files_changed: number;
  added: number;
  removed: number;
};

export type WorkspaceDetail = {
  workspace: Workspace;
  epic: Epic;
  sessions: Session[];
  diff: DiffSummary;
  files: DiffStat[];
  terminal_live: boolean;
  worktree_short: string;
};

export type EpicDetail = {
  epic: Epic;
  workspaces: Array<SidebarWorkspace & { worktree_path: string; pr_url: string }>;
  unbound_sessions: Session[];
};

export type Message = {
  role: "user" | "assistant" | "tool";
  text: string;
  tool_name?: string;
  model?: string;
  timestamp: string;
};

export type SessionDetail = {
  session: Session;
  messages: Message[];
};

export type DebugInfo = {
  db_path: string;
  db_size_bytes: number;
  wal_size_bytes: number;
  counts: { epics: number; workspaces: number; sessions: number };
  recent_epics: Record<string, unknown>[];
  recent_workspaces: Record<string, unknown>[];
  recent_sessions: Record<string, unknown>[];
};

const API = "/api";

async function jsonOr<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const txt = await res.text();
    throw new Error(`${res.status}: ${txt || res.statusText}`);
  }
  return res.json();
}

export const api = {
  sidebar: () => fetch(`${API}/sidebar`).then(jsonOr<{ epics: SidebarEpic[]; indexed_at: string }>),
  home: () => fetch(`${API}/home`).then(jsonOr<{ total_sessions: number; project_count: number }>),

  epicGet: (slug: string) => fetch(`${API}/epics/${slug}`).then(jsonOr<EpicDetail>),
  epicCreate: (body: Partial<Epic>) =>
    fetch(`${API}/epics`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).then(jsonOr<Epic>),
  epicSaveContext: (slug: string, context_md: string) =>
    fetch(`${API}/epics/${slug}/context`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ context_md }),
    }).then(jsonOr<{ ok: true }>),
  epicArchive: (slug: string) =>
    fetch(`${API}/epics/${slug}/archive`, { method: "POST" }).then(jsonOr<{ ok: true }>),

  workspaceGet: (epicSlug: string, wsSlug: string) =>
    fetch(`${API}/epics/${epicSlug}/workspaces/${wsSlug}`).then(jsonOr<WorkspaceDetail>),
  workspaceCreate: (
    epicSlug: string,
    body: { name: string; slug: string; branch_name: string; worktree_path: string }
  ) =>
    fetch(`${API}/epics/${epicSlug}/workspaces`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).then(jsonOr<Workspace>),
  workspaceSuggest: (epicSlug: string) =>
    fetch(`${API}/epics/${epicSlug}/workspaces/suggest`).then(
      jsonOr<{ branch_name: string; worktree_path: string; slug: string }>
    ),
  workspaceSavePR: (epicSlug: string, wsSlug: string, pr_url: string) =>
    fetch(`${API}/epics/${epicSlug}/workspaces/${wsSlug}/pr`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ pr_url }),
    }).then(jsonOr<{ ok: true }>),
  workspaceArchive: (epicSlug: string, wsSlug: string) =>
    fetch(`${API}/epics/${epicSlug}/workspaces/${wsSlug}/archive`, { method: "POST" }).then(
      jsonOr<{ ok: true }>
    ),
  workspaceLaunchITerm: (epicSlug: string, wsSlug: string) =>
    fetch(`${API}/epics/${epicSlug}/workspaces/${wsSlug}/launch`, { method: "POST" }).then(
      jsonOr<{ ok: true }>
    ),
  workspaceTermStart: (epicSlug: string, wsSlug: string, sessionId?: string) =>
    fetch(`${API}/epics/${epicSlug}/workspaces/${wsSlug}/term/start`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: sessionId ? JSON.stringify({ session_id: sessionId }) : undefined,
    }).then(jsonOr<{ ok: true }>),
  workspaceTermKill: (epicSlug: string, wsSlug: string) =>
    fetch(`${API}/epics/${epicSlug}/workspaces/${wsSlug}/term/kill`, { method: "POST" }).then(
      jsonOr<{ ok: true }>
    ),

  sessionsList: () => fetch(`${API}/sessions`).then(jsonOr<{ sessions: Session[]; project_count: number }>),
  sessionGet: (id: string) => fetch(`${API}/sessions/${id}`).then(jsonOr<SessionDetail>),

  debug: () => fetch(`${API}/debug`).then(jsonOr<DebugInfo>),
};
