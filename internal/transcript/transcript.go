// Package transcript renders a session JSONL file into a clean Message stream
// suitable for display. Tool calls are summarized rather than dumped raw.
package transcript

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"claudeops/internal/domain"
)

type rawEvent struct {
	Type        string          `json:"type"`
	Timestamp   string          `json:"timestamp"`
	IsMeta      bool            `json:"isMeta"`
	IsSidechain bool            `json:"isSidechain"`
	Message     json.RawMessage `json:"message"`
}

type messageBody struct {
	Role    string          `json:"role"`
	Model   string          `json:"model"`
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
}

// Render reads up to maxMessages of the most recent non-meta turns from path.
// It returns them oldest→newest so the UI can render top→bottom naturally.
func Render(path string, maxMessages int) ([]domain.Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	var msgs []domain.Message
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		var ev rawEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		if ev.IsMeta || ev.IsSidechain {
			continue
		}
		ts, _ := time.Parse(time.RFC3339Nano, ev.Timestamp)
		switch ev.Type {
		case "user":
			text := extractUserText(ev.Message)
			if text == "" {
				continue
			}
			msgs = append(msgs, domain.Message{Role: "user", Text: text, Timestamp: ts})
		case "assistant":
			var mb messageBody
			if err := json.Unmarshal(ev.Message, &mb); err != nil {
				continue
			}
			text, tools := extractAssistantContent(mb.Content)
			if text != "" {
				msgs = append(msgs, domain.Message{Role: "assistant", Text: text, Model: mb.Model, Timestamp: ts})
			}
			for _, t := range tools {
				msgs = append(msgs, domain.Message{Role: "tool", ToolName: t, Timestamp: ts})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return msgs, err
	}
	if maxMessages > 0 && len(msgs) > maxMessages {
		msgs = msgs[len(msgs)-maxMessages:]
	}
	return msgs, nil
}

func extractUserText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var mb messageBody
	if err := json.Unmarshal(raw, &mb); err != nil {
		return ""
	}
	if len(mb.Content) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(mb.Content, &asString); err == nil {
		return cleanText(asString)
	}
	var blocks []contentBlock
	if err := json.Unmarshal(mb.Content, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return cleanText(b.Text)
			}
		}
	}
	return ""
}

func extractAssistantContent(raw json.RawMessage) (text string, toolNames []string) {
	if len(raw) == 0 {
		return "", nil
	}
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		var asString string
		if err2 := json.Unmarshal(raw, &asString); err2 == nil {
			return cleanText(asString), nil
		}
		return "", nil
	}
	var sb strings.Builder
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(b.Text)
		case "tool_use":
			toolNames = append(toolNames, b.Name)
		}
	}
	return cleanText(sb.String()), toolNames
}

func cleanText(s string) string {
	s = strings.TrimSpace(s)
	// Strip noisy hidden tags that show up in command output.
	for _, prefix := range []string{"<local-command-caveat>", "<system-reminder>"} {
		if i := strings.Index(s, prefix); i == 0 {
			if end := strings.Index(s, ">"); end > 0 {
				s = strings.TrimSpace(s[end+1:])
			}
		}
	}
	return s
}
