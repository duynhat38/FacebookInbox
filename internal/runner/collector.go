package runner

import "time"

const (
	pageFlushInterval = 400 * time.Millisecond
	pageBatchMax      = 500
)

// collectPages batches successfully-processed page IDs from ch and delivers
// them to onDone, flushing on a timer or when the batch is large. It returns
// after a final flush when ch is closed. This keeps page removal (and its disk
// write) off the worker goroutines.
func collectPages(ch <-chan string, onDone func([]string)) {
	ticker := time.NewTicker(pageFlushInterval)
	defer ticker.Stop()

	var batch []string
	flush := func() {
		if len(batch) == 0 {
			return
		}
		ids := make([]string, len(batch))
		copy(ids, batch)
		batch = batch[:0]
		onDone(ids)
	}

	for {
		select {
		case id, ok := <-ch:
			if !ok {
				flush()
				return
			}
			batch = append(batch, id)
			if len(batch) >= pageBatchMax {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}
