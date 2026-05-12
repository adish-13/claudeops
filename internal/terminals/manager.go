// Package terminals owns the lifecycle of pty-backed shell sessions, one per
// workspace. Each session spawns `claude` (or any configured command) inside
// the workspace's worktree directory, multiplexes output to subscribers, and
// keeps a small ring-buffer backlog so newly attaching browsers see context.
package terminals

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

const backlogBytes = 64 * 1024

// Manager owns all live pty Sessions, keyed by workspace ID.
type Manager struct {
	cmd     string   // command to run, default "claude"
	cmdArgs []string // optional args
	mu      sync.Mutex
	byID    map[int64]*Session
}

func NewManager(cmd string, args ...string) *Manager {
	if cmd == "" {
		cmd = "claude"
	}
	return &Manager{cmd: cmd, cmdArgs: args, byID: map[int64]*Session{}}
}

// Get returns the Session for workspaceID if one exists, else nil.
func (m *Manager) Get(workspaceID int64) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.byID[workspaceID]
}

// Spawn starts a new pty session in worktreePath if none exists; reuses the
// existing one otherwise. extraArgs are appended after the manager's base args
// — used e.g. to pass `--resume <session-id>` for resuming prior sessions.
// Safe to call repeatedly.
func (m *Manager) Spawn(workspaceID int64, worktreePath string, extraArgs ...string) (*Session, error) {
	m.mu.Lock()
	if s, ok := m.byID[workspaceID]; ok && s.Alive() {
		m.mu.Unlock()
		return s, nil
	}
	m.mu.Unlock()

	args := append([]string{}, m.cmdArgs...)
	args = append(args, extraArgs...)
	cmd := exec.Command(m.cmd, args...)
	cmd.Dir = worktreePath
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	pf, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("pty start: %w", err)
	}
	s := &Session{
		workspaceID: workspaceID,
		cmd:         cmd,
		pty:         pf,
		backlog:     newRing(backlogBytes),
	}
	go s.pump()
	m.mu.Lock()
	m.byID[workspaceID] = s
	m.mu.Unlock()
	return s, nil
}

// Kill stops the session for workspaceID if present.
func (m *Manager) Kill(workspaceID int64) {
	m.mu.Lock()
	s := m.byID[workspaceID]
	delete(m.byID, workspaceID)
	m.mu.Unlock()
	if s != nil {
		s.close()
	}
}

// Session is one pty + its multiplexed subscribers.
type Session struct {
	workspaceID int64
	cmd         *exec.Cmd
	pty         *os.File

	mu          sync.Mutex
	subscribers []chan []byte // each gets a copy of stdout
	closed      bool
	backlog     *ring
}

// Backlog returns a snapshot of recent output for new subscribers.
func (s *Session) Backlog() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backlog.bytes()
}

// Subscribe registers a channel that will receive subsequent stdout bytes.
// Caller must drain it; if it blocks the session keeps writing to others.
func (s *Session) Subscribe(buf int) (<-chan []byte, func()) {
	ch := make(chan []byte, buf)
	s.mu.Lock()
	s.subscribers = append(s.subscribers, ch)
	s.mu.Unlock()
	cancel := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, c := range s.subscribers {
			if c == ch {
				s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
				close(ch)
				return
			}
		}
	}
	return ch, cancel
}

// Write sends bytes (typically keystrokes) into the pty.
func (s *Session) Write(p []byte) (int, error) {
	if s == nil || s.pty == nil {
		return 0, errors.New("session not started")
	}
	return s.pty.Write(p)
}

// Resize resizes the pty's viewport.
func (s *Session) Resize(rows, cols uint16) error {
	if s == nil || s.pty == nil {
		return errors.New("session not started")
	}
	return pty.Setsize(s.pty, &pty.Winsize{Rows: rows, Cols: cols})
}

// Alive reports whether the underlying process is still running.
func (s *Session) Alive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.closed
}

func (s *Session) pump() {
	buf := make([]byte, 4096)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			s.broadcast(chunk)
		}
		if err != nil {
			if err != io.EOF {
				// log if you want; non-fatal
			}
			s.close()
			return
		}
	}
}

func (s *Session) broadcast(chunk []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backlog.write(chunk)
	for _, ch := range s.subscribers {
		select {
		case ch <- chunk:
		default:
			// slow subscriber — drop chunk to avoid backpressure
		}
	}
}

func (s *Session) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	subs := s.subscribers
	s.subscribers = nil
	s.mu.Unlock()
	if s.pty != nil {
		_ = s.pty.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	for _, c := range subs {
		close(c)
	}
}

// ring is a fixed-size, lossy byte ring buffer for backlog replay.
type ring struct {
	buf    []byte
	pos    int
	filled bool
}

func newRing(size int) *ring { return &ring{buf: make([]byte, size)} }

func (r *ring) write(p []byte) {
	for _, b := range p {
		r.buf[r.pos] = b
		r.pos++
		if r.pos == len(r.buf) {
			r.pos = 0
			r.filled = true
		}
	}
}

func (r *ring) bytes() []byte {
	if !r.filled {
		out := make([]byte, r.pos)
		copy(out, r.buf[:r.pos])
		return out
	}
	out := make([]byte, len(r.buf))
	copy(out, r.buf[r.pos:])
	copy(out[len(r.buf)-r.pos:], r.buf[:r.pos])
	return out
}
