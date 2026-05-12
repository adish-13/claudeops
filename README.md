# claudeops

Local dashboard for all your Claude Code sessions across every project.

P0 MVP: read-only crawler over `~/.claude/projects/*/*.jsonl` →
SQLite index → web dashboard at http://127.0.0.1:7777.

## Run

```sh
go build -o claudeops ./cmd/claudeops
./claudeops
```

Flags:

- `-projects` — path to projects dir (default `~/.claude/projects`)
- `-db` — sqlite db path (default `~/.claude/claudeops.db`)
- `-addr` — http listen address (default `127.0.0.1:7777`)
- `-scan` — rescan interval (default `5s`)

## What it shows

| Column | Source |
|---|---|
| When | `last_activity` (max of file mtime + parsed timestamps) |
| Project | `cwd` from session events, falls back to project dir name |
| Branch | `gitBranch` from session events |
| Last messages | last `user` and `assistant` text content from the tail |
| Events | line count |
| Session | first 8 chars of session UUID |

## What's next (not in MVP)

- Hook-based status (running / idle / needs-input)
- Per-session detail page
- `claudeops new <task>` worktree spawn
- Diff view + PR button
