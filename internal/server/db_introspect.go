package server

import (
	"context"
	"fmt"
	"os"

	"claudeops/internal/store"
)

func osStat(p string) (os.FileInfo, error) { return os.Stat(p) }

func countTable(ctx context.Context, s *store.Store, name string) int {
	var n int
	row := s.DB().QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", name))
	_ = row.Scan(&n)
	return n
}

func queryRows(ctx context.Context, s *store.Store, q string) []recentRow {
	rows, err := s.DB().QueryContext(ctx, q)
	if err != nil {
		return nil
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out []recentRow
	for rows.Next() {
		dest := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range dest {
			ptrs[i] = &dest[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		cells := make([]string, len(cols))
		for i, v := range dest {
			cells[i] = fmt.Sprintf("%v", v)
		}
		out = append(out, recentRow{Cells: cells})
	}
	return out
}
