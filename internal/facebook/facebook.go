// Package facebook provides a Messenger client that authenticates via browser
// cookie and delivers messages over the MQTT/WebSocket realtime protocol.
package facebook

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ── Task ──────────────────────────────────────────────────────────────────────

// Task is a single unit of work: one target page/thread processed with one
// authorized cookie credential.
// Credential is sensitive (browser session cookie). It must never be logged.
type Task struct {
	PageID                 string // target Facebook Page/User numeric ID (= Messenger thread_id after resolve)
	Credential             string // browser cookie string: c_user=...; xs=...; etc.
	LiveID                 string // uid extracted by the live check (not used internally by MessengerSender)
	Proxy                  string
	ImagePath              string
	Message                string
	ShareURL               string // if non-empty, share this Facebook post URL to the thread (send_type=6, wrapped /share/p/... URL)
	ClickMessenger         bool   // if true, register the account as a messenger_business_ads_sender for PageID before sending (graphql doc_id 6799089560137001)
	DeleteThread           bool   // if true, delete the thread after sending (label 146)
	DeleteMessageAfterSend bool   // if true, delete each sent message immediately after confirm (label 155)
	ProfileIndex           int    // 0-based worker index — used to pick a unique browser fingerprint per account
}

// ── Sender ────────────────────────────────────────────────────────────────────

// Sender opens a per-page send session that reuses one WebSocket pair for all
// steps of a single page.
type Sender interface {
	// Open starts a session bound to one worker's http.Client. It performs no
	// I/O; connections and tokens are established lazily on the first Send.
	Open(client *http.Client) Session
}

// Session sends the ordered steps for a single page over a reused WebSocket
// pair. It is used by one goroutine only and must be closed when the page's
// steps are done.
type Session interface {
	// Send returns nil on success. The error must not contain the credential value.
	Send(ctx context.Context, task Task) error
	// Close releases the WebSocket pair. It is idempotent.
	Close()
}

// ── LiveChecker ───────────────────────────────────────────────────────────────

// LiveCheckResult is the outcome of validating a credential before any action.
type LiveCheckResult int

const (
	// LiveCheckLive — credential valid; processing may continue.
	LiveCheckLive LiveCheckResult = iota
	// LiveCheckDie — credential invalid/expired; processing stops.
	LiveCheckDie
	// LiveCheckCheckpointed — account locked behind a Facebook security checkpoint;
	// user must log in manually and complete verification. Error is *CheckpointError.
	LiveCheckCheckpointed
	// LiveCheckError — network or unexpected failure; processing stops.
	LiveCheckError
)

// LiveChecker validates a credential before send actions run.
type LiveChecker interface {
	// CheckLive validates the credential. On LiveCheckLive it also returns the
	// account uid (empty for all other results).
	// On LiveCheckCheckpointed the error is *CheckpointError.
	// On LiveCheckError the error must already be credential-sanitized.
	CheckLive(ctx context.Context, client *http.Client, credential string) (LiveCheckResult, string, error)
}

// CheckpointError is returned (as *CheckpointError) when a Facebook account is
// locked behind a security checkpoint requiring manual login and verification.
type CheckpointError struct {
	URL string // full checkpoint URL the user must visit
}

func (e *CheckpointError) Error() string {
	return "tài khoản bị checkpoint — cần đăng nhập lại và xác minh tại " + e.URL
}

// PageRejectedError is returned when Facebook explicitly rejects a send task
// with mailboxTaskCompletionApiOnTaskCompletion=false ("Couldn't send.").
// This means the page/thread does not accept messages from this account
// (messaging disabled, not a follower, account restricted, etc.).
type PageRejectedError struct {
	ThreadID string
	Reason   string // e.g. "Couldn't send."
}

func (e *PageRejectedError) Error() string {
	return fmt.Sprintf("messenger: thread_id=%s: page rejected (%s)", e.ThreadID, e.Reason)
}

// ── SimulatedLiveChecker ──────────────────────────────────────────────────────

// SimulatedLiveChecker always reports the credential as live without any
// network call. Safe for offline use and testing.
type SimulatedLiveChecker struct{}

func NewSimulatedLiveChecker() *SimulatedLiveChecker { return &SimulatedLiveChecker{} }

func (s *SimulatedLiveChecker) CheckLive(ctx context.Context, _ *http.Client, _ string) (LiveCheckResult, string, error) {
	if err := ctx.Err(); err != nil {
		return LiveCheckError, "", err
	}
	return LiveCheckLive, "", nil
}

// ── CookieLiveChecker ─────────────────────────────────────────────────────────

// CookieLiveChecker validates a Facebook browser cookie by GETting
// facebook.com/me and checking:
//  1. Final redirect URL contains /checkpoint/ → LiveCheckCheckpointed
//  2. Body contains "NAME" JS field → LiveCheckLive
//  3. Otherwise → LiveCheckDie
type CookieLiveChecker struct{}

func NewCookieLiveChecker() *CookieLiveChecker { return &CookieLiveChecker{} }

func (c *CookieLiveChecker) CheckLive(ctx context.Context, client *http.Client, cookie string) (LiveCheckResult, string, error) {
	if strings.TrimSpace(cookie) == "" {
		return LiveCheckDie, "", nil
	}
	page, finalURL, err := fbGetWithFinalURL(ctx, client, cookie, "https://www.facebook.com/", profileFromCookie(cookie), "none")
	if err != nil {
		if ctx.Err() != nil {
			return LiveCheckError, "", ctx.Err()
		}
		return LiveCheckError, "", err
	}

	// Checkpoint detection:
	//   1. Final URL redirected to /checkpoint/ — definitive.
	//   2. Page body contains both "/checkpoint/" AND "ServerRedirect" — this is the
	//      FB JS pattern for a forced redirect (same check as extractTokens).
	//      Using body-only `/checkpoint/` is too broad: normal FB pages include that
	//      string as a route definition in their JS bundle (false positive risk).
	bodyCheckpoint := strings.Contains(page, "/checkpoint/") && strings.Contains(page, "ServerRedirect")
	if strings.Contains(finalURL, "/checkpoint/") || bodyCheckpoint {
		cpURL := finalURL
		if !strings.Contains(finalURL, "/checkpoint/") {
			if idx := strings.Index(page, "/checkpoint/"); idx >= 0 {
				end := idx
				for end < len(page) && page[end] != '"' && page[end] != '\\' && page[end] != ' ' {
					end++
				}
				cpURL = "https://www.facebook.com" + page[idx:end]
			}
		}
		return LiveCheckCheckpointed, "", &CheckpointError{URL: cpURL}
	}

	if !strings.Contains(page, `"NAME":"`) {
		return LiveCheckDie, "", nil
	}
	uid := firstMatch(reCUser, cookie)
	return LiveCheckLive, uid, nil
}

// ── SanitizeToken ─────────────────────────────────────────────────────────────

// SanitizeToken removes the raw credential from a string before logging.
func SanitizeToken(s, credential string) string {
	if credential != "" {
		s = strings.ReplaceAll(s, credential, "***")
		s = strings.ReplaceAll(s, url.QueryEscape(credential), "***")
	}
	return redactAccessTokenParam(s)
}

func redactAccessTokenParam(s string) string {
	const key = "access_token="
	idx := strings.Index(s, key)
	if idx < 0 {
		return s
	}
	start := idx + len(key)
	end := start
	for end < len(s) && s[end] != '&' && s[end] != ' ' && s[end] != '"' {
		end++
	}
	return s[:start] + "***" + s[end:]
}

// ── validateTask ──────────────────────────────────────────────────────────────

// validateTask performs common non-sensitive validation shared by all senders.
// Never includes the credential in returned errors.
func validateTask(ctx context.Context, task Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(task.PageID) == "" {
		return errors.New("facebook: empty PageID")
	}
	return nil
}
