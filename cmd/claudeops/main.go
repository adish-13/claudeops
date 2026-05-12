package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"claudeops/internal/indexer"
	"claudeops/internal/server"
	"claudeops/internal/store"
	"claudeops/internal/terminals"
)

func main() {
	home, _ := os.UserHomeDir()
	defaultProjects := filepath.Join(home, ".claude", "projects")
	defaultDB := filepath.Join(home, ".claude", "claudeops.db")
	defaultWorktrees := filepath.Join(home, "worktrees")

	projectsDir := flag.String("projects", defaultProjects, "path to ~/.claude/projects")
	dbPath := flag.String("db", defaultDB, "sqlite db path")
	addr := flag.String("addr", "127.0.0.1:7777", "http listen address")
	scanEvery := flag.Duration("scan", 5*time.Second, "rescan interval")
	worktreeRoot := flag.String("worktrees", defaultWorktrees, "root dir for new worktrees")
	termCmd := flag.String("term-cmd", "claude", "command run inside the embedded terminal pty")
	flag.Parse()

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ix := indexer.New(*projectsDir, st)
	go ix.Run(ctx, *scanEvery)

	tm := terminals.NewManager(*termCmd)

	srv := server.New(st, tm, *worktreeRoot)

	go func() {
		log.Printf("claudeops listening on http://%s", *addr)
		log.Printf("  projects = %s", *projectsDir)
		log.Printf("  db       = %s", *dbPath)
		log.Printf("  worktrees= %s", *worktreeRoot)
		log.Printf("  term cmd = %s", *termCmd)
		if err := srv.ListenAndServe(*addr); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	log.Printf("shutting down")
	cancel()
}
