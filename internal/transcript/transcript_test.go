package transcript_test

import (
	"path/filepath"
	"testing"

	"claudeops/internal/transcript"
)

func TestRenderSample(t *testing.T) {
	msgs, err := transcript.Render(filepath.Join("testdata", "sample.jsonl"), 0)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// Expect: user "hello claude", assistant text "hi!...", user "run a tool",
	//         assistant text "sure", tool "Bash". The isMeta line is skipped.
	if len(msgs) != 5 {
		for i, m := range msgs {
			t.Logf("msg[%d] role=%s tool=%s text=%q", i, m.Role, m.ToolName, m.Text)
		}
		t.Fatalf("expected 5 messages (meta skipped), got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Text != "hello claude" {
		t.Errorf("msg 0: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Model != "claude-opus-4-7" {
		t.Errorf("msg 1: %+v", msgs[1])
	}
	if msgs[4].Role != "tool" || msgs[4].ToolName != "Bash" {
		t.Errorf("msg 4: %+v", msgs[4])
	}
}

func TestRenderRespectsMax(t *testing.T) {
	msgs, err := transcript.Render(filepath.Join("testdata", "sample.jsonl"), 2)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 (last) messages, got %d", len(msgs))
	}
}
