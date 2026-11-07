package facebook

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// wsMaxAge caps how long one edge-chat + LightSpeed pair is reused before being
// reopened. Edge-chat's MQTT keep-alive is 15s (see buildMQTTConnect in wire.go):
// we send nothing on that socket during a page, so Facebook drops it once the
// keep-alive elapses. Reopening the pair before 15s keeps the "active Messenger
// tab" proof alive without a PINGREQ loop.
const wsMaxAge = 12 * time.Second

var (
	errWSClosed  = errors.New("messenger: websocket closed")
	errWSTimeout = errors.New("messenger: websocket read timeout")
)

// wsReader owns all reads on one *websocket.Conn from a dedicated goroutine and
// forwards frames over a channel. This is the only safe way to reuse a gorilla
// connection for multiple tasks: a read that hits a socket deadline sets a
// permanent c.readErr (gorilla/websocket conn.go), poisoning the connection for
// every future read. By never setting a read deadline and turning "no response
// yet" into a select-timeout on the channel, the socket is never poisoned and
// stale frames can be drained from the channel (not the socket) before a send.
type wsReader struct {
	conn   *websocket.Conn
	frames chan []byte
	done   chan struct{}
	once   sync.Once
}

func startWSReader(conn *websocket.Conn) *wsReader {
	r := &wsReader{
		conn:   conn,
		frames: make(chan []byte, 32),
		done:   make(chan struct{}),
	}
	go func() {
		defer close(r.frames)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			select {
			case r.frames <- data:
			case <-r.done:
				return
			}
		}
	}()
	return r
}

// read waits for the next frame, a timeout, or context cancellation. A timeout
// does NOT damage the connection (the reader goroutine keeps running with no
// socket deadline), so the caller may keep using it.
func (r *wsReader) read(ctx context.Context, timeout time.Duration) ([]byte, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case data, ok := <-r.frames:
		if !ok {
			return nil, errWSClosed
		}
		return data, nil
	case <-timer.C:
		return nil, errWSTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// drain discards any frames buffered from a previous task (e.g. a late
// "instant reply" push) so they can't be mistaken for the next task's response.
func (r *wsReader) drain() {
	for {
		select {
		case <-r.frames:
		default:
			return
		}
	}
}

func (r *wsReader) close() {
	r.once.Do(func() {
		close(r.done)
		safeCloseWS(r.conn) // unblocks the goroutine's ReadMessage
	})
}

// messengerSession reuses one edge-chat + LightSpeed WebSocket pair for every
// step of a single page instead of reopening both per message. It is driven by
// exactly one worker goroutine, so it needs no locking.
//
// WARNING: reusing one gateway socket for multiple tasks is live-protocol
// behaviour that cannot be verified offline (the gateway's acceptance of a
// second task on the same socket, and whether it pushes post-ack frames).
// Confirm with a live multi-step send (text+photo+share) before trusting it.
type messengerSession struct {
	sender *MessengerSender
	client *http.Client

	inited     bool
	cookie     string
	uid        string
	proxy      string
	pageID     string
	deviceID   string // stable per session — used in both edge-chat and LightSpeed
	profileIdx int
	tok        messengerTokens
	threadID string // cached once resolved; "" until a real resolution succeeds

	pageName     string // display name fetched from facebook.com/<page_id>; cached per page
	pageNameDone bool   // true once a fetch was attempted (so a miss isn't retried)

	edgeConn *websocket.Conn
	lsConn   *websocket.Conn
	lsReader *wsReader
	openedAt time.Time
	seq      byte   // LightSpeed task sequence (byte wraps at 256 — fine for one session)
	httpReq  int    // AJAX __req counter; incremented before each HTTP AJAX call
}

// nextHTTPReq increments the AJAX request counter and returns it as a string.
// Mirrors what the browser does: each AJAX call on the page gets the next seq.
func (m *messengerSession) nextHTTPReq() string {
	m.httpReq++
	return strconv.Itoa(m.httpReq)
}

// Open returns a per-page session. It performs no I/O; tokens, thread
// resolution and WebSocket connects all happen lazily on the first Send, so a
// failure surfaces through Send and the worker's existing error handling.
func (s *MessengerSender) Open(client *http.Client) Session {
	return &messengerSession{sender: s, client: client}
}

func (m *messengerSession) Send(ctx context.Context, task Task) error {
	if err := validateTask(ctx, task); err != nil {
		return err
	}
	if task.Credential == "" {
		return errors.New("messenger: missing cookie credential")
	}
	if firstMatch(reCUser, task.Credential) == "" {
		return errors.New("messenger: cookie has no c_user=<id> field")
	}
	if err := m.init(ctx, task); err != nil {
		return err
	}

	// Ad-click registration uses the raw page_id over HTTP and needs no thread
	// resolution or WebSocket, so handle it before either runs.
	if task.ClickMessenger {
		m.tok.reqSeq = m.nextHTTPReq()
		return clickMessenger(ctx, m.client, "", m.tok, m.pageID)
	}

	tid := m.resolveThread(ctx)

	if task.DeleteThread {
		return m.deleteThread(ctx, tid)
	}
	if task.ShareURL != "" {
		// URL post share (send_type=6): wrap the post URL into a /share/p/... link,
		// then send it in a plain "url" field over DGW — exactly what the browser
		// does. Facebook renders the shared-post card server-side; no XMA fetch.
		m.tok.reqSeq = m.nextHTTPReq()
		wrapped := createWrappedURL(ctx, m.client, "", m.tok, task.ShareURL)
		log.Printf("messenger: share[1/2] wrapped thread_id=%s url=%q wrapped_ok=%v", tid, wrapped, wrapped != task.ShareURL)

		log.Printf("messenger: share[2/2] sending share payload thread_id=%s queue=%s", tid, tid)
		msgID, err := m.lightspeedSend(ctx, tid, tid, messengerTaskLabel, buildSharePayload(tid, m.applyPageName(ctx, task.Message), wrapped), "send")
		if err != nil {
			return err
		}
		return m.maybeDeleteMessage(ctx, tid, msgID, task.DeleteMessageAfterSend)
	}
	if task.ImagePath != "" {
		m.tok.reqSeq = m.nextHTTPReq()
		fileID, err := uploadFile(ctx, m.client, "", m.tok, task.ImagePath)
		if err != nil {
			return err
		}
		msgID, err := m.lightspeedSend(ctx, tid, tid, messengerTaskLabel, buildAttachPayload(tid, fileID), "send")
		if err != nil {
			return err
		}
		return m.maybeDeleteMessage(ctx, tid, msgID, task.DeleteMessageAfterSend)
	}
	if task.Message != "" {
		msgID, err := m.lightspeedSend(ctx, tid, tid, messengerTaskLabel, buildTextPayload(tid, m.applyPageName(ctx, task.Message)), "send")
		if err != nil {
			return err
		}
		return m.maybeDeleteMessage(ctx, tid, msgID, task.DeleteMessageAfterSend)
	}
	return nil
}

// Close releases the WebSocket pair. Idempotent — safe to call from both the
// worker's per-page cleanup and a deferred stop path.
func (m *messengerSession) Close() {
	m.closeConns()
}

func (m *messengerSession) init(ctx context.Context, task Task) error {
	if m.inited {
		return nil
	}
	// Seed the jar before any HTTP request so the http.Client sends the credential
	// cookies automatically on every request and captures Set-Cookie updates.
	// All subsequent calls within this session pass "" as the cookie argument;
	// the jar is the single source of truth from this point on.
	seedJar(m.client.Jar, task.Credential)

	m.profileIdx = task.ProfileIndex
	tok, err := m.sender.getTokens(ctx, m.client, task.Credential, m.profileIdx)
	if err != nil {
		return err
	}
	m.tok = tok
	m.uid = tok.uid
	m.cookie = task.Credential
	m.proxy = task.Proxy
	m.pageID = task.PageID
	m.deviceID = uuid.New().String() // one device ID per session, shared across all WS connections
	m.inited = true
	return nil
}

// resolveThread resolves the Messenger thread ID once and caches it. A transient
// failure (bulk-route fails and scrape returns the raw ID) is NOT cached, so a
// later step in the same page retries instead of reusing a bad value.
func (m *messengerSession) resolveThread(ctx context.Context) string {
	if m.threadID != "" {
		return m.threadID
	}
	m.tok.reqSeq = m.nextHTTPReq()
	threadID, bulkOK := resolveBulkRoute(ctx, m.client, "", m.tok, m.pageID)
	if !bulkOK {
		scraped := resolvePageID(ctx, m.client, "", m.tok, m.pageID)
		if scraped != m.pageID {
			threadID = scraped
		}
	}
	if bulkOK || threadID != m.pageID {
		m.threadID = threadID
	}
	return threadID
}

// applyPageName replaces the {namepage} placeholder in msg with the target page's
// display name. If msg has no placeholder, it returns msg untouched (no fetch).
// If the name can't be fetched, the literal placeholder is left in place.
func (m *messengerSession) applyPageName(ctx context.Context, msg string) string {
	if !strings.Contains(msg, pageNamePlaceholder) {
		return msg
	}
	name := m.resolvePageName(ctx)
	if name == "" {
		return msg
	}
	return strings.ReplaceAll(msg, pageNamePlaceholder, name)
}

// resolvePageName fetches the page display name once per session and caches it
// (including a miss, so a failed fetch isn't retried for every step).
func (m *messengerSession) resolvePageName(ctx context.Context) string {
	if !m.pageNameDone {
		m.pageName = fetchPageName(ctx, m.client, "", m.tok, m.pageID)
		m.pageNameDone = true
	}
	return m.pageName
}

func (m *messengerSession) deleteThread(ctx context.Context, tid string) error {
	id, _ := strconv.ParseInt(tid, 10, 64)
	payload := map[string]interface{}{
		"thread_key":            id,
		"remove_type":           0,
		"sync_group":            1,
		"offline_threading_id":  "",
	}
	_, err := m.lightspeedSend(ctx, tid, tid, "146", payload, "delete")
	return err
}

// maybeDeleteMessage deletes msgID if del is true. An empty msgID means the send
// was unconfirmed, which is surfaced as an error so the counter increments.
func (m *messengerSession) maybeDeleteMessage(ctx context.Context, tid, msgID string, del bool) error {
	if !del {
		return nil
	}
	if msgID == "" {
		return fmt.Errorf("messenger: thread_id=%s: send unconfirmed — no message ID returned, cannot delete", tid)
	}
	id, _ := strconv.ParseInt(tid, 10, 64)
	payload := map[string]interface{}{
		"message_id": msgID,
		"thread_key": id, // int64 per capture (label 155)
	}
	_, err := m.lightspeedSend(ctx, tid, "155", "155", payload, "deleteMessage")
	return err
}

// lightspeedSend writes one task on the reused LightSpeed socket and validates
// the response. The connection is dropped (and reopened on the next call) only
// when the socket is actually dead or the context ends — never on a plain
// timeout/empty response, which leaves the socket healthy. It never retries a
// send, to avoid delivering the message twice.
func (m *messengerSession) lightspeedSend(ctx context.Context, tid, queueName, label string, payload map[string]interface{}, action string) (string, error) {
	if err := m.ensureConns(ctx); err != nil {
		return "", err
	}
	seq := m.seq
	m.seq++
	resp, err := sendLightspeedTask(ctx, m.lsConn, m.lsReader, seq, tid, queueName, label, payload)
	if err != nil {
		if errors.Is(err, errWSClosed) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			m.closeConns()
		}
		return "", err
	}
	return assertTaskAccepted(resp, tid, action)
}

// ensureConns opens the edge-chat + LightSpeed pair if absent or older than
// wsMaxAge. Edge-chat must be connected first — the gateway rejects LightSpeed
// tasks without a live Messenger session behind them.
func (m *messengerSession) ensureConns(ctx context.Context) error {
	if m.lsConn != nil && m.edgeConn != nil && time.Since(m.openedAt) < wsMaxAge {
		return nil
	}
	m.closeConns()

	// Use the jar's current cookies for WebSocket handshake headers — the jar may
	// have fresher values from prior Set-Cookie responses than the original credential.
	wsCookie := jarCookieHeader(m.client.Jar, "https://www.facebook.com", m.cookie)
	edge, err := openEdgeChat(ctx, wsCookie, m.tok.profile, m.proxy, m.deviceID)
	if err != nil {
		return err
	}
	ls, err := openLightspeed(ctx, wsCookie, m.uid, m.tok.profile, m.proxy, m.deviceID)
	if err != nil {
		safeCloseWS(edge)
		return err
	}
	m.edgeConn = edge
	m.lsConn = ls
	m.lsReader = startWSReader(ls)
	m.openedAt = time.Now()
	m.seq = 0
	return nil
}

func (m *messengerSession) closeConns() {
	if m.lsReader != nil {
		m.lsReader.close() // also closes m.lsConn
		m.lsReader = nil
	} else {
		safeCloseWS(m.lsConn)
	}
	safeCloseWS(m.edgeConn)
	m.lsConn = nil
	m.edgeConn = nil
}
