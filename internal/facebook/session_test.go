package facebook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// wsEcho upgrades the connection and, for each binary frame it reads, writes
// back the frames produced by respond(i) where i is the 0-based task index.
// It models the LightSpeed gateway closely enough to exercise connection reuse:
// multiple tasks on one socket, with optional trailing "push" frames.
func wsEchoServer(t *testing.T, respond func(i int) [][]byte) *httptest.Server {
	t.Helper()
	up := websocket.Upgrader{}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for i := 0; ; i++ {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			for _, frame := range respond(i) {
				if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
					return
				}
			}
		}
	}))
}

func dialWS(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

func payloadFrame(body string) []byte {
	// Binary prefix + JSON, mirroring the DGW framing that sendLightspeedTask
	// skips past via bytes.Index(chunk, `{"`).
	return append([]byte{0x0d, 0x00, 0x80}, []byte(`{"payload":"`+body+`"}`)...)
}

// TestSendLightspeedTaskReusesConnection is the regression guard for the
// per-page WebSocket reuse: two tasks sent on one socket must both get their
// response, and the call must not block on a post-ack timeout. The old
// implementation waited ~3s after each ack and poisoned the socket on timeout,
// so the second send returned empty — this test fails against that version.
func TestSendLightspeedTaskReusesConnection(t *testing.T) {
	srv := wsEchoServer(t, func(i int) [][]byte {
		return [][]byte{payloadFrame("ack-" + string(rune('0'+i)))}
	})
	defer srv.Close()

	conn := dialWS(t, srv)
	reader := startWSReader(conn)
	defer reader.close()

	ctx := context.Background()
	start := time.Now()
	for seq := byte(0); seq < 2; seq++ {
		resp, err := sendLightspeedTask(ctx, conn, reader, seq, "123", "123", "46",
			map[string]interface{}{"n": int(seq)})
		if err != nil {
			t.Fatalf("send %d: unexpected error: %v", seq, err)
		}
		if !strings.Contains(resp, `"payload"`) {
			t.Fatalf("send %d: empty/invalid response: %q", seq, resp)
		}
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("two sends took %v — a post-ack wait was reintroduced", elapsed)
	}
}

// TestSendLightspeedTaskDrainsStaleFrame guards item (2): a trailing push left
// by task 1 must not be mistaken for task 2's response. The server emits the
// ack plus an extra "payload" frame for task 0; drain() must discard the
// leftover so task 1 reads its own ack.
func TestSendLightspeedTaskDrainsStaleFrame(t *testing.T) {
	srv := wsEchoServer(t, func(i int) [][]byte {
		if i == 0 {
			return [][]byte{payloadFrame("ack-0"), payloadFrame("stale-push")}
		}
		return [][]byte{payloadFrame("ack-1")}
	})
	defer srv.Close()

	conn := dialWS(t, srv)
	reader := startWSReader(conn)
	defer reader.close()

	ctx := context.Background()

	resp0, err := sendLightspeedTask(ctx, conn, reader, 0, "123", "123", "46", map[string]interface{}{})
	if err != nil {
		t.Fatalf("send 0: %v", err)
	}
	if !strings.Contains(resp0, "ack-0") {
		t.Fatalf("send 0: want ack-0, got %q", resp0)
	}

	// Let the trailing push land in the reader channel before the next send.
	time.Sleep(50 * time.Millisecond)

	resp1, err := sendLightspeedTask(ctx, conn, reader, 1, "123", "123", "46", map[string]interface{}{})
	if err != nil {
		t.Fatalf("send 1: %v", err)
	}
	if strings.Contains(resp1, "stale-push") {
		t.Fatalf("send 1: read a stale frame from task 0: %q", resp1)
	}
	if !strings.Contains(resp1, "ack-1") {
		t.Fatalf("send 1: want ack-1, got %q", resp1)
	}
}
