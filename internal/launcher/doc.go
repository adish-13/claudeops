// Package launcher opens a new iTerm or Terminal.app tab on macOS, cd's into
// a worktree, and runs `claude` with an optional initial prompt.
//
// This is the macOS-only "escape hatch" alongside the embedded WebSocket
// terminal — useful when a user wants the real native terminal (full
// keybindings, scrollback, copy/paste) instead of xterm.js in the browser.
//
// Implementation: builds an AppleScript string and runs it via `osascript`.
// iTerm is preferred when present; Terminal.app is the fallback.
//
// On non-macOS platforms LaunchClaude returns an error. Guard calls into
// this package with `runtime.GOOS == "darwin"` if you're adding new ones.
package launcher
