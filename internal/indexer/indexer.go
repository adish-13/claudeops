package indexer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"claudeops/internal/store"
)

const (
	headLines    = 50
	tailReadSize = 64 * 1024
	previewMax   = 240
)

type event struct {
	Type        string          `json:"type"`
	SessionID   string          `json:"sessionId"`
	Cwd         string          `json:"cwd"`
	GitBranch   string          `json:"gitBranch"`
	Version     string          `json:"version"`
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
	Type string `json:"type"`
	Text string `json:"text"`
}

type Indexer struct {
	root  string
	store *store.Store
}

func New(root string, s *store.Store) *Indexer {
	return &Indexer{root: root, store: s}
}

func (ix *Indexer) Run(ctx context.Context, every time.Duration) {
	if err := ix.scan(ctx); err != nil {
		log.Printf("initial scan: %v", err)
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := ix.scan(ctx); err != nil {
				log.Printf("scan: %v", err)
			}
		}
	}
}

func (ix *Indexer) scan(ctx context.Context) error {
	workspaces, err := ix.store.AllWorkspaces(ctx)
	if err != nil {
		return fmt.Errorf("load workspaces: %w", err)
	}
	wsByPath := buildWorkspaceMatcher(workspaces)

	entries, err := os.ReadDir(ix.root)
	if err != nil {
		return fmt.Errorf("read root: %w", err)
	}
	var count int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		projectDir := filepath.Join(ix.root, e.Name())
		files, err := os.ReadDir(projectDir)
		if err != nil {
			log.Printf("skip project %s: %v", e.Name(), err)
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			path := filepath.Join(projectDir, f.Name())
			sess, err := parseFile(path, e.Name())
			if err != nil {
				log.Printf("parse %s: %v", path, err)
				continue
			}
			if sess.SessionID == "" {
				continue
			}
			if id := wsByPath.match(sess.Cwd); id != 0 {
				sess.WorkspaceID = &id
			}
			if err := ix.store.UpsertSession(ctx, sess); err != nil {
				log.Printf("upsert %s: %v", sess.SessionID, err)
				continue
			}
			count++
		}
	}
	log.Printf("scan done: %d sessions, %d workspaces", count, len(workspaces))
	return nil
}

type wsMatcher struct{ items []wsItem }
type wsItem struct {
	id     int64
	prefix string
}

func buildWorkspaceMatcher(ws []store.Workspace) *wsMatcher {
	m := &wsMatcher{}
	for _, w := range ws {
		p := strings.TrimRight(w.WorktreePath, "/") + "/"
		m.items = append(m.items, wsItem{id: w.ID, prefix: p})
	}
	return m
}

// match returns workspace ID whose worktree_path is the longest prefix of cwd, or 0.
func (m *wsMatcher) match(cwd string) int64 {
	if cwd == "" {
		return 0
	}
	cwdSlash := strings.TrimRight(cwd, "/") + "/"
	var bestID int64
	var bestLen int
	for _, it := range m.items {
		if strings.HasPrefix(cwdSlash, it.prefix) && len(it.prefix) > bestLen {
			bestID = it.id
			bestLen = len(it.prefix)
		}
	}
	return bestID
}

func parseFile(path, projectDir string) (store.Session, error) {
	var sess store.Session
	info, err := os.Stat(path)
	if err != nil {
		return sess, err
	}
	sess.FilePath = path
	sess.FileSizeBytes = info.Size()
	sess.ProjectDir = projectDir
	sess.LastActivity = info.ModTime()

	f, err := os.Open(path)
	if err != nil {
		return sess, err
	}
	defer f.Close()

	br := bufio.NewReader(f)
	var lineCount int64
	for i := 0; i < headLines; i++ {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			lineCount++
			applyHeader(&sess, line)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}

	tail := readTail(f, info.Size(), tailReadSize)
	tailLines, totalAfter := splitLines(tail)
	sess.NumEvents = lineCount + totalAfter
	applyTail(&sess, tailLines)

	return sess, nil
}

func applyHeader(s *store.Session, line []byte) {
	var ev event
	if err := json.Unmarshal(line, &ev); err != nil {
		return
	}
	if ev.SessionID != "" && s.SessionID == "" {
		s.SessionID = ev.SessionID
	}
	if ev.Cwd != "" && s.Cwd == "" {
		s.Cwd = ev.Cwd
	}
	if ev.GitBranch != "" && s.GitBranch == "" {
		s.GitBranch = ev.GitBranch
	}
	if ev.Version != "" && s.Version == "" {
		s.Version = ev.Version
	}
	if ts, ok := parseTime(ev.Timestamp); ok && ts.After(s.LastActivity) {
		s.LastActivity = ts
	}
}

func applyTail(s *store.Session, lines [][]byte) {
	for i := len(lines) - 1; i >= 0; i-- {
		var ev event
		if err := json.Unmarshal(lines[i], &ev); err != nil {
			continue
		}
		if s.SessionID == "" && ev.SessionID != "" {
			s.SessionID = ev.SessionID
		}
		if s.Cwd == "" && ev.Cwd != "" {
			s.Cwd = ev.Cwd
		}
		if s.GitBranch == "" && ev.GitBranch != "" {
			s.GitBranch = ev.GitBranch
		}
		if ts, ok := parseTime(ev.Timestamp); ok && ts.After(s.LastActivity) {
			s.LastActivity = ts
		}
		if ev.IsMeta || ev.IsSidechain {
			continue
		}
		switch ev.Type {
		case "user":
			if s.LastUserPreview == "" {
				if t := extractText(ev.Message); t != "" {
					s.LastUserPreview = trim(t, previewMax)
				}
			}
		case "assistant":
			if s.LastAssistantText == "" {
				if t := extractText(ev.Message); t != "" {
					s.LastAssistantText = trim(t, previewMax)
				}
			}
			if s.Model == "" {
				var mb messageBody
				if err := json.Unmarshal(ev.Message, &mb); err == nil {
					s.Model = mb.Model
				}
			}
		}
		if s.LastUserPreview != "" && s.LastAssistantText != "" && s.Model != "" {
			return
		}
	}
}

func extractText(raw json.RawMessage) string {
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
		return asString
	}
	var blocks []contentBlock
	if err := json.Unmarshal(mb.Content, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return b.Text
			}
		}
	}
	return ""
}

func readTail(f *os.File, size, n int64) []byte {
	if size == 0 {
		return nil
	}
	if n > size {
		n = size
	}
	buf := make([]byte, n)
	if _, err := f.ReadAt(buf, size-n); err != nil && err != io.EOF {
		return nil
	}
	return buf
}

func splitLines(buf []byte) ([][]byte, int64) {
	if len(buf) == 0 {
		return nil, 0
	}
	if buf[0] != '\n' {
		if i := bytesIndex(buf, '\n'); i >= 0 {
			buf = buf[i+1:]
		} else {
			return nil, 0
		}
	} else {
		buf = buf[1:]
	}
	var out [][]byte
	var count int64
	start := 0
	for i, b := range buf {
		if b == '\n' {
			if i > start {
				out = append(out, buf[start:i])
				count++
			}
			start = i + 1
		}
	}
	if start < len(buf) {
		out = append(out, buf[start:])
		count++
	}
	return out, count
}

func bytesIndex(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

func parseTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func trim(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
