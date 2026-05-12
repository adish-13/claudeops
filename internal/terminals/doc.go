// Package terminals owns the pty-backed shell sessions that the embedded
// xterm.js terminal in the workspace UI attaches to.
//
// Manager keeps one *Session per workspace id. A Session owns:
//
//   - The *os.File returned by pty.Start.
//   - A mutex-protected list of subscriber channels (one per attached browser tab).
//   - A 64KB ring buffer of recent output, replayed to new subscribers on attach
//     so that refreshing the page doesn't lose context.
//
// A read goroutine pumps pty output → ring buffer → every subscriber. Writes
// (keyboard input from the browser) go straight to the pty. Subscriber
// channels have a small buffer; if a subscriber falls behind, writes drop
// rather than block the read goroutine — the ring buffer is the safety net.
//
// This package is intentionally workspace-agnostic: workspace ids are opaque
// int64s. The HTTP/WebSocket layer in package server is what turns them into
// URLs and resolves them back to workspace records.
package terminals
