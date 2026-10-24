package runner

import (
	"context"
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"facebook-inbox/internal/domain"
	"facebook-inbox/internal/facebook"
)

// ErrAlreadyRunning is returned when Start is called while a run is in progress.
var ErrAlreadyRunning = errors.New("runner.alreadyRunning")

// ErrNoPages is returned when there are no pages to process.
var ErrNoPages = errors.New("runner.noPages")

// ErrNoAccounts is returned when there are no Accounts items to run.
var ErrNoAccounts = errors.New("runner.noAccounts")

// ErrNoSender is returned when Start is called without a configured Sender.
var ErrNoSender = errors.New("runner.noSender")

// emitInterval throttles how often aggregated stats are pushed to the UI.
const emitInterval = 150 * time.Millisecond

// Config describes a single run.
type Config struct {
	Settings      domain.Settings
	Pages         []string
	Proxies       []string
	Accounts      []domain.AccountItem
	Links         []string
	Contents      []string
	ShareContents []string
	ImagePath     string
	Language      string
	Sender        facebook.Sender
	LiveChecker   facebook.LiveChecker
	// OnPagesDone, if set, is called (off the worker goroutines, in batches)
	// with the page IDs that were successfully processed, so they can be removed
	// from the persistent page list.
	OnPagesDone func(pageIDs []string)
}

// Runner owns the lifecycle of a run: Accounts workers, the scheduler, the proxy
// allocator, the stats aggregator and context cancellation.
type Runner struct {
	emit      func(domain.Stats)
	logger    *log.Logger
	allocator ProxyAllocator

	mu        sync.Mutex
	running   bool
	wasStop   bool
	cancel    context.CancelFunc
	lastStats domain.Stats
}

// New creates a Runner. emit is called (from a single goroutine) whenever the
// aggregated stats change; logger receives non-sensitive operational logs.
func New(emit func(domain.Stats), logger *log.Logger) *Runner {
	if emit == nil {
		emit = func(domain.Stats) {}
	}
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &Runner{emit: emit, logger: logger, allocator: RoundRobinAllocator{}}
}

// IsRunning reports whether a run is currently in progress.
func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// LastStats returns the most recently aggregated stats snapshot.
func (r *Runner) LastStats() domain.Stats {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastStats
}

// Start validates the config, allocates a proxy to each worker (round-robin)
// and begins processing. It is non-blocking: processing continues in background
// goroutines and progress is reported through emit.
func (r *Runner) Start(cfg Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return ErrAlreadyRunning
	}
	if len(cfg.Pages) == 0 {
		return ErrNoPages
	}
	if len(cfg.Accounts) == 0 {
		return ErrNoAccounts
	}

	sender := cfg.Sender
	if sender == nil {
		return ErrNoSender
	}

	liveChecker := cfg.LiveChecker
	if liveChecker == nil {
		// Default: the safe simulated checker (reports live, no network).
		liveChecker = facebook.NewSimulatedLiveChecker()
	}

	threads := cfg.Settings.Threads
	if threads < 1 {
		threads = 1
	}

	sched := NewScheduler(
		cfg.Pages, cfg.Accounts, cfg.Links, cfg.Contents, cfg.ShareContents,
		cfg.Settings.PagesPerThread, cfg.Settings.AccountPerLink,
	)
	delay := time.Duration(cfg.Settings.DelaySeconds) * time.Second
	act := newActivity(cfg.Language)

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.wasStop = false
	r.running = true

	updates := make(chan domain.AccountSnapshot, len(cfg.Accounts)+threads*2)
	sem := make(chan struct{}, threads)
	// pageDone carries page IDs that succeeded, for removal from the page list.
	pageDone := make(chan string, 512)
	var wg sync.WaitGroup

	// Collector: batches succeeded pages and hands them to OnPagesDone off the
	// worker goroutines. When no handler is set, it simply drains the channel.
	if cfg.OnPagesDone != nil {
		go collectPages(pageDone, cfg.OnPagesDone)
	} else {
		go func() {
			for range pageDone {
			}
		}()
	}

	order := make([]string, len(cfg.Accounts))
	for i, item := range cfg.Accounts {
		order[i] = item.ID
		// Proxy is allocated per worker index (round-robin over the pool).
		proxy := r.allocator.Assign(i, cfg.Proxies)
		w := &accountWorker{
			item:                   item,
			proxy:                  proxy,
			link:                   sched.LinkForAccount(i),
			maxErrors:              cfg.Settings.MaxErrors,
			targetSuccess:          cfg.Settings.PagesPerThread,
			delay:                  delay,
			imagePath:              cfg.ImagePath,
			deleteThread:           cfg.Settings.DeleteThreadAfterSend,
			deleteMessageAfterSend: cfg.Settings.DeleteMessageAfterSend,
			sender:                 sender,
			liveChecker:            liveChecker,
			sched:                  sched,
			sem:                    sem,
			updates:                updates,
			pageDone:               pageDone,
			logger:                 r.logger,
			act:                    act,
		}
		w.rt.ID = i + 1
		wg.Add(1)
		go w.run(ctx, &wg)
	}

	go r.aggregate(sched, cfg.Accounts, order, updates)

	go func() {
		wg.Wait()
		close(updates)
		close(pageDone)
	}()

	r.logger.Printf("run started: threads=%d pages=%d accounts=%d proxies=%d",
		threads, len(cfg.Pages), len(cfg.Accounts), len(cfg.Proxies))
	return nil
}

// Stop requests a graceful shutdown. Workers observe the cancelled context and
// finish their current step before exiting. It is safe to call when not running.
func (r *Runner) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running || r.cancel == nil {
		return
	}
	r.wasStop = true
	r.cancel()
	r.logger.Printf("stop requested")
}

func (r *Runner) aggregate(sched *Scheduler, seed []domain.AccountItem, order []string, updates <-chan domain.AccountSnapshot) {
	// Seed every Accounts row so it appears immediately as not-started (proxy "—").
	accounts := make(map[string]domain.AccountSnapshot, len(seed))
	for _, item := range seed {
		accounts[item.ID] = domain.AccountSnapshot{ID: item.ID, Value: item.Value}
	}

	ticker := time.NewTicker(emitInterval)
	defer ticker.Stop()

	dirty := true
	flush := func(state domain.RunnerState) {
		stats := buildStats(state, accounts, order, sched.TotalPages(), sched.Remaining())
		r.mu.Lock()
		r.lastStats = stats
		r.mu.Unlock()
		r.emit(stats)
		dirty = false
	}

	flush(domain.RunnerRunning)

	for {
		select {
		case snap, ok := <-updates:
			if !ok {
				r.mu.Lock()
				state := domain.RunnerDone
				if r.wasStop {
					state = domain.RunnerStopped
				}
				r.mu.Unlock()
				flush(state)
				r.mu.Lock()
				r.running = false
				r.cancel = nil
				r.mu.Unlock()
				return
			}
			accounts[snap.ID] = snap
			dirty = true
		case <-ticker.C:
			if dirty {
				flush(domain.RunnerRunning)
			}
		}
	}
}
