package runner

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"facebook-inbox/internal/domain"
	"facebook-inbox/internal/facebook"
	"facebook-inbox/internal/httpclient"
)

// checkLiveDelay is a short, context-aware pause so the "Check Live..." activity
// is visible before processing begins.
const checkLiveDelay = 400 * time.Millisecond

// accountWorker processes the pages for a single Account item. Each Account item has its
// own accountWorker goroutine with its own http.Client (own proxy and cookie jar);
// no mutable network state is shared between workers. Concurrency across Account
// items is capped by a semaphore so at most Threads run at once.
//
// The "Trạng thái" column shows this worker's current activity log (rt.CurrentLog),
// not a status enum.
type accountWorker struct {
	item                   domain.AccountItem
	proxy                  string
	link                   string
	maxErrors              int
	targetSuccess          int // stop after this many successful pages (PagesPerThread)
	delay                  time.Duration
	imagePath              string
	deleteThread           bool
	deleteMessageAfterSend bool

	sender      facebook.Sender
	liveChecker facebook.LiveChecker
	sched       *Scheduler
	sem         chan struct{}
	updates     chan<- domain.AccountSnapshot
	pageDone    chan<- string
	logger      *log.Logger
	act         activity

	rt          domain.WorkerRuntime
	started     bool
	taskCounter int
	liveID      string
}

func (w *accountWorker) snapshot() domain.AccountSnapshot {
	return domain.AccountSnapshot{
		ID:      w.item.ID,
		Value:   w.item.Value,
		Proxy:   w.rt.Proxy,
		Success: w.rt.Success,
		Errors:  w.rt.Errors,
		Log:     w.rt.CurrentLog,
		Running: w.rt.Running,
		Started: w.started,
	}
}

func (w *accountWorker) report() {
	w.updates <- w.snapshot()
}

func (w *accountWorker) setLog(msg string) {
	w.rt.CurrentLog = msg
	w.report()
}

// run executes the worker: it waits for a concurrency slot, checks the
// credential ("Check Live..."), then pulls page batches from the shared queue
// and sends messages, updating its activity log the whole time.
func (w *accountWorker) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.rt.AccountID = w.item.ID
	w.report() // seed as not started (Proxy "—")

	select {
	case w.sem <- struct{}{}:
		defer func() { <-w.sem }()
	case <-ctx.Done():
		return
	}
	if ctx.Err() != nil {
		return
	}

	client, err := httpclient.New(w.proxy, 30*time.Second)
	if err != nil {
		w.logger.Printf("account %s: proxy setup failed (%v); using direct connection", w.item.ID, err)
		client, _ = httpclient.New("", 30*time.Second)
		w.proxy = ""
	}
	// Release this worker's idle connections when it stops, so its pool is not
	// left holding sockets after the run finishes or is cancelled.
	defer client.CloseIdleConnections()

	// Begin: assign proxy, mark running, check the credential.
	w.started = true
	w.rt.Running = true
	w.rt.Proxy = w.proxy
	w.setLog(w.act.checkLive())

	if !w.wait(ctx, checkLiveDelay) {
		w.stop()
		return
	}

	// Step 1: Check Live. The outcome NEVER touches the Success/Error counters.
	result, liveID, checkErr := w.liveChecker.CheckLive(ctx, client, w.item.Value)
	if errors.Is(checkErr, context.Canceled) || errors.Is(checkErr, context.DeadlineExceeded) {
		w.stop()
		return
	}
	switch result {
	case facebook.LiveCheckLive:
		w.liveID = liveID
		w.setLog(w.act.live())
	case facebook.LiveCheckDie:
		w.rt.Running = false
		w.setLog(w.act.die())
		return
	case facebook.LiveCheckCheckpointed:
		cpURL := ""
		if checkErr != nil {
			if cp, ok := checkErr.(*facebook.CheckpointError); ok {
				cpURL = cp.URL
			}
		}
		w.logger.Printf("account %s: checkpoint detected: %s", w.item.ID, cpURL)
		w.rt.Running = false
		w.setLog(w.act.checkpoint(cpURL))
		return
	default: // LiveCheckError
		safe := "unexpected error"
		if checkErr != nil {
			safe = facebook.SanitizeToken(checkErr.Error(), w.item.Value)
		}
		w.logger.Printf("account %s: check live error: %s", w.item.ID, safe)
		w.rt.Running = false
		w.setLog(w.act.checkLiveError(safe))
		return
	}

	if ctx.Err() != nil {
		w.stop()
		return
	}

	// One send session per page reuses a single edge-chat + LightSpeed WebSocket
	// pair across all of that page's steps (text/photo/share/delete) instead of
	// reopening both per message. The deferred close covers the early-return
	// paths below (context cancelled, error limit); Close is idempotent.
	var sess facebook.Session
	defer func() {
		if sess != nil {
			sess.Close()
		}
	}()

	// Each worker pulls pages one at a time and stops when it reaches targetSuccess
	// successful pages, hits the error limit, or the queue is exhausted.
	// Using targetSuccess (= PagesPerThread) as the success target means failures
	// don't consume the quota — the account keeps trying until it has sent to the
	// required number of pages (or runs out of pages).
	for w.rt.Success < w.targetSuccess {
		page := w.sched.NextPage()
		if page == "" {
			break // queue exhausted
		}
		if ctx.Err() != nil {
			w.stop()
			return
		}

		content := w.sched.ContentForTask(w.taskCounter)
		shareContent := w.sched.ShareContentForTask(w.taskCounter)
		w.taskCounter++

		// Each page (recipient) gets an ordered sequence of Send API messages:
		// text -> photo -> link. Only the steps that have something to send are
		// included; each step is one separate message.
		steps := w.pageSteps(page, content, shareContent)
		if len(steps) == 0 {
			continue
		}

		sess = w.sender.Open(client)

		// Run all steps for this page as ONE atomic workflow: every real step
		// (click/text/photo/share) must succeed for the page to count as a success.
		// Any step failure — including a Facebook rejection ("Couldn't send.") —
		// fails the whole workflow and counts as one error. Delete cleanup is the
		// only exception: its failure is logged but never fails the workflow.
		attempted := false
		pageError := false
		for _, st := range steps {
			if ctx.Err() != nil {
				w.stop()
				return
			}
			// Once a step has failed, skip the remaining content steps — but still
			// run the delete cleanup if the user enabled it, so a ticked "delete
			// thread" always removes the thread even when the send failed.
			if pageError && !st.isDelete {
				continue
			}
			w.setLog(st.doing)
			sendErr := sess.Send(ctx, st.task)
			attempted = true
			switch {
			case sendErr == nil:
				w.setLog(st.done)
			case errors.Is(sendErr, context.Canceled), errors.Is(sendErr, context.DeadlineExceeded):
				// Consume the page we already started before stopping.
				if w.pageDone != nil {
					w.pageDone <- page
				}
				w.stop()
				return
			case st.isDelete:
				// Delete-thread/delete-message failure: log and continue — main send already done.
				safe := facebook.SanitizeToken(sendErr.Error(), w.item.Value)
				w.logger.Printf("account %s: page %s %s failed (ignored): %s", w.item.ID, page, st.kind, safe)
			default:
				safe := facebook.SanitizeToken(sendErr.Error(), w.item.Value)
				w.logger.Printf("account %s: page %s %s failed: %s", w.item.ID, page, st.kind, safe)
				pageError = true
				var rejected *facebook.PageRejectedError
				if errors.As(sendErr, &rejected) {
					w.setLog(w.act.pageRejected(page, rejected.Reason))
				} else {
					w.setLog(w.act.requestError(safe))
				}
			}
		}

		// All of this page's steps are done — release its WebSocket pair before
		// the inter-page delay so sockets aren't held idle (and past edge-chat's
		// 15s keep-alive) while waiting.
		sess.Close()
		sess = nil

		// Count once per page (whole workflow), not per step.
		if attempted {
			if pageError {
				w.rt.Errors++
				w.report() // update error count immediately
				if w.rt.Errors >= w.maxErrors {
					w.rt.Running = false
					w.setLog(w.act.errorLimit())
					if w.pageDone != nil {
						w.pageDone <- page
					}
					return
				}
			} else {
				w.rt.Success++
				w.rt.Errors = 0 // a success resets the error streak — maxErrors is now consecutive errors
				w.report()      // update success+error counts immediately
			}
			if w.pageDone != nil {
				w.pageDone <- page
			}
		}

		// Delay between pages (not between steps within the same page).
		if !w.wait(ctx, w.delay) {
			w.stop()
			return
		}
	}

	w.rt.Running = false
	w.setLog(w.act.done())
}

// pageStep is one Send API message directed at a page in the per-page sequence.
// isDelete marks cleanup steps whose failure is logged but never counted as an error.
type pageStep struct {
	kind     string
	task     facebook.Task
	doing    string
	done     string
	isDelete bool
}

// pageSteps builds the ordered send steps for a page: text, photo, share, then optionally delete.
// Only steps with content are included. The delete step (if enabled) always runs last
// and failures are logged but not counted against the error limit.
func (w *accountWorker) pageSteps(page, content, shareContent string) []pageStep {
	base := facebook.Task{
		PageID:                 page,
		Credential:             w.item.Value,
		LiveID:                 w.liveID,
		Proxy:                  w.proxy,
		DeleteMessageAfterSend: w.deleteMessageAfterSend,
	}
	var steps []pageStep
	if content != "" {
		t := base
		t.Message = content
		steps = append(steps, pageStep{kind: "text", task: t, doing: w.act.sending(page), done: w.act.sent(page)})
	}
	if w.imagePath != "" {
		t := base
		t.ImagePath = w.imagePath
		steps = append(steps, pageStep{kind: "photo", task: t, doing: w.act.sendingPhoto(page), done: w.act.sentPhoto(page)})
	}
	if w.link != "" {
		// send_type=6 carries the caption in its own "text" field alongside "url",
		// so the shared-post card and the caption arrive in ONE message (matches the
		// browser). No separate text step needed.
		t := base
		t.ShareURL = w.link
		t.Message = shareContent
		steps = append(steps, pageStep{kind: "share", task: t, doing: w.act.sharingPost(page), done: w.act.sharedPost(page)})
	}
	hasContent := len(steps) > 0
	if w.deleteThread {
		// Delete-thread runs whenever the user enabled it — even with no content
		// to send — so a ticked "delete thread" always removes the thread.
		t := base
		t.DeleteThread = true
		steps = append(steps, pageStep{
			kind:     "delete",
			task:     t,
			doing:    w.act.deletingThread(page),
			done:     w.act.deletedThread(page),
			isDelete: true,
		})
	}
	// Register as ad-click sender first, but only when there is real content to
	// send — a pure delete-thread needs no ad-click. Prepended so it runs first.
	if hasContent {
		t := base
		t.ClickMessenger = true
		click := pageStep{
			kind:  "click",
			task:  t,
			doing: w.act.clickingMessenger(page),
			done:  w.act.clickedMessenger(page),
		}
		steps = append([]pageStep{click}, steps...)
	}
	return steps
}

// stop marks the worker stopped and records the stopped activity.
func (w *accountWorker) stop() {
	w.rt.Running = false
	w.setLog(w.act.stopped())
}

// wait pauses for d, returning false if the context is cancelled during the
// wait. A non-positive duration returns immediately.
func (w *accountWorker) wait(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
