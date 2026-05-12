// Package transcript renders a Claude Code session JSONL file into a clean
// stream of domain.Message values for display.
//
// Render reads the last N non-meta turns from the file and returns them in
// chronological order (oldest → newest). It pulls user text, assistant text,
// and a short tool-call summary out of the nested content blocks, dropping
// system events and sidechain noise.
//
// This package only knows how to read one file on demand for the transcript
// view. The bulk indexer (which scans every JSONL on a tick to populate the
// sessions table) lives in package indexer.
package transcript
