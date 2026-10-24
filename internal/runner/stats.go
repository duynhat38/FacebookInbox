package runner

import (
	"sort"

	"facebook-inbox/internal/domain"
)

// buildStats aggregates the latest per-Accounts snapshots into a single Stats
// value. The `order` slice fixes the row order to match the Accounts list. It is
// called only from the aggregator goroutine, so it needs no locking.
func buildStats(state domain.RunnerState, accounts map[string]domain.AccountSnapshot, order []string, total, remaining int) domain.Stats {
	stats := domain.Stats{
		State:        state,
		TotalWorkers: len(order),
		TotalPages:   total,
		Remaining:    remaining,
		Accounts:     make([]domain.AccountSnapshot, 0, len(order)),
	}

	seen := make(map[string]bool, len(order))
	appendSnap := func(d domain.AccountSnapshot) {
		stats.Success += d.Success
		stats.Errors += d.Errors
		if d.Running {
			stats.Running++
		}
		stats.Accounts = append(stats.Accounts, d)
	}

	for _, id := range order {
		if d, ok := accounts[id]; ok {
			appendSnap(d)
			seen[id] = true
		}
	}
	extra := make([]string, 0)
	for id := range accounts {
		if !seen[id] {
			extra = append(extra, id)
		}
	}
	sort.Strings(extra)
	for _, id := range extra {
		appendSnap(accounts[id])
	}

	stats.Processed = stats.Success + stats.Errors
	return stats
}
