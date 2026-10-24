package runner

import "sync"

// Queue is a thread-safe dispenser of page IDs. Workers pull batches from it;
// because a single mutex-guarded cursor advances across the shared slice, no
// page is ever handed to two workers.
type Queue struct {
	mu     sync.Mutex
	pages  []string
	cursor int
}

// NewQueue copies pages into a new queue.
func NewQueue(pages []string) *Queue {
	cp := make([]string, len(pages))
	copy(cp, pages)
	return &Queue{pages: cp}
}

// Next returns up to n pages, advancing the cursor. It returns an empty slice
// when the queue is exhausted.
func (q *Queue) Next(n int) []string {
	if n < 1 {
		n = 1
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.cursor >= len(q.pages) {
		return nil
	}
	end := q.cursor + n
	if end > len(q.pages) {
		end = len(q.pages)
	}
	batch := make([]string, end-q.cursor)
	copy(batch, q.pages[q.cursor:end])
	q.cursor = end
	return batch
}

// Total returns the total number of pages.
func (q *Queue) Total() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.pages)
}

// Remaining returns how many pages have not yet been dispensed.
func (q *Queue) Remaining() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.pages) - q.cursor
}
