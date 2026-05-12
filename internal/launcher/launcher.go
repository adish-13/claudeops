package launcher

import (
	"fmt"
	"os/exec"
	"strings"
)

// LaunchClaude opens a new iTerm tab (or Terminal as fallback), cds into worktreePath,
// and runs `claude` with an optional initial prompt. macOS only.
func LaunchClaude(worktreePath, initialPrompt string) error {
	if hasApp("iTerm") {
		return launchInITerm(worktreePath, initialPrompt)
	}
	return launchInTerminal(worktreePath, initialPrompt)
}

func hasApp(name string) bool {
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`tell application "System Events" to (name of processes) contains "%s"`, name))
	out, err := cmd.Output()
	if err != nil {
		// Fall back to checking /Applications + ~/Applications
		for _, prefix := range []string{"/Applications/", "/Applications/iTerm.app", "/System/Applications/"} {
			if _, err := exec.Command("test", "-d", prefix+name+".app").CombinedOutput(); err == nil {
				return true
			}
		}
		return false
	}
	return strings.TrimSpace(string(out)) == "true" || appExists(name)
}

func appExists(name string) bool {
	_, err := exec.Command("osascript", "-e", fmt.Sprintf(`exists application "%s"`, name)).Output()
	return err == nil
}

func launchInITerm(worktreePath, prompt string) error {
	cmdLine := fmt.Sprintf("cd %s && claude", shellQuote(worktreePath))
	if prompt != "" {
		cmdLine = fmt.Sprintf("cd %s && claude %s", shellQuote(worktreePath), shellQuote(prompt))
	}
	script := fmt.Sprintf(`
tell application "iTerm"
  activate
  if (count of windows) = 0 then
    create window with default profile
  else
    tell current window to create tab with default profile
  end if
  tell current session of current window
    write text %s
  end tell
end tell`, appleScriptString(cmdLine))
	return exec.Command("osascript", "-e", script).Run()
}

func launchInTerminal(worktreePath, prompt string) error {
	cmdLine := fmt.Sprintf("cd %s && claude", shellQuote(worktreePath))
	if prompt != "" {
		cmdLine = fmt.Sprintf("cd %s && claude %s", shellQuote(worktreePath), shellQuote(prompt))
	}
	script := fmt.Sprintf(`
tell application "Terminal"
  activate
  do script %s
end tell`, appleScriptString(cmdLine))
	return exec.Command("osascript", "-e", script).Run()
}

// shellQuote returns a single-quoted shell string with embedded single quotes escaped.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// appleScriptString returns an AppleScript string literal with embedded backslashes/quotes escaped.
func appleScriptString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
