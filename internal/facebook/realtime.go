package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	messengerWSTimeout    = 20 * time.Second
	messengerReadDeadline = 10 * time.Second
)

// The per-page send flow (open the edge-chat + LightSpeed pair once, run every
// step over it, then close) lives in messengerSession (session.go). The task
// routing, delete-message and delete-thread payloads it builds mirror the
// former runTask/deleteMessage/deleteThread helpers.

// openEdgeChat connects MQTT WebSocket to edge-chat.facebook.com and completes
// CONNECT + SUBSCRIBE /t_rtc_multi handshake.
// Facebook requires this live MQTT session as proof of an active Messenger tab —
// without it, gateway.facebook.com rejects ALL tasks.
func openEdgeChat(ctx context.Context, cookie, uid, proxy, deviceID string) (*websocket.Conn, error) {
	sid := msgrRandInt64(1_000_000_000_000_000, 9_999_999_999_999_999)
	wsURL := fmt.Sprintf("wss://edge-chat.facebook.com/chat?region=pnb&sid=%d&cid=%s", sid, deviceID)

	p := profileForUID(uid)
	conn, _, err := newWSDialer(proxy, p.userAgent).DialContext(ctx, wsURL, wsHeaders(cookie, p))
	if err != nil {
		return nil, fmt.Errorf("messenger: edge-chat connect: %w", err)
	}

	// MQTT username JSON — mirrors Python username dict exactly.
	// All real auth comes from the Cookie header; this is only session metadata.
	appIDInt, _ := parseInt64(messengerAppID)
	username := map[string]interface{}{
		"a":            p.userAgent, // must match the WS handshake UA
		"asi":          nil,
		"aid":          appIDInt,
		"aids":         nil,
		"chat_on":      false,
		"cp":           3,
		"ct":           "websocket",
		"d":            deviceID,
		"dc":           "",
		"ecp":          10,
		"fg":           true,
		"gas":          nil,
		"mqtt_sid":     "",
		"no_auto_fg":   true,
		"p":            nil,
		"pack":         []interface{}{},
		"php_override": "",
		"pm":           []interface{}{},
		"s":            sid,
		"st":           []interface{}{},
		"u":            uid,
	}
	usernameJSON, err := json.Marshal(username)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// CONNECT → recv CONNACK
	if err := conn.WriteMessage(websocket.BinaryMessage, buildMQTTConnect(usernameJSON)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("messenger: MQTT CONNECT: %w", err)
	}
	conn.SetReadDeadline(time.Now().Add(messengerReadDeadline))
	_, connack, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("messenger: MQTT CONNACK recv: %w", err)
	}
	if len(connack) == 0 || connack[0] != 0x20 {
		conn.Close()
		return nil, fmt.Errorf("messenger: MQTT CONNACK invalid: %x", connack)
	}

	// SUBSCRIBE /t_rtc_multi → recv SUBACK (content not checked)
	if err := conn.WriteMessage(websocket.BinaryMessage, buildMQTTSubscribe([]byte("/t_rtc_multi"), 1)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("messenger: MQTT SUBSCRIBE: %w", err)
	}
	conn.SetReadDeadline(time.Now().Add(messengerReadDeadline))
	if _, _, err = conn.ReadMessage(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("messenger: MQTT SUBACK recv: %w", err)
	}

	conn.SetReadDeadline(time.Time{})
	return conn, nil
}

// openLightspeed connects to gateway.facebook.com/ws/lightspeed and reads the
// 1-byte ready signal (0x0a) the server sends immediately on open.
func openLightspeed(ctx context.Context, cookie, uid, proxy, deviceID string) (*websocket.Conn, error) {
	loggingID := uuid.New().String()
	wsURL := fmt.Sprintf(
		"wss://gateway.facebook.com/ws/lightspeed"+
			"?x-dgw-appid=%s&x-dgw-appversion=0&x-dgw-authtype=1:0"+
			"&x-dgw-version=5&x-dgw-uuid=%s&x-dgw-tier=prod"+
			"&x-dgw-loggingid=%s&x-dgw-regionhint=PNB&x-dgw-deviceid=%s",
		messengerAppID, uid, loggingID, deviceID,
	)

	p := profileForUID(uid)
	conn, _, err := newWSDialer(proxy, p.userAgent).DialContext(ctx, wsURL, wsHeaders(cookie, p))
	if err != nil {
		return nil, fmt.Errorf("messenger: lightspeed connect: %w", err)
	}
	conn.SetReadDeadline(time.Now().Add(messengerReadDeadline))
	if _, _, err = conn.ReadMessage(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("messenger: lightspeed ready recv: %w", err)
	}
	conn.SetReadDeadline(time.Time{})
	return conn, nil
}

// sendLightspeedTask builds the triple-nested JSON envelope, writes it on the
// (reused) LightSpeed WebSocket, and returns as soon as the response frame (the
// one containing a "payload" field) arrives. It reads through the wsReader's
// channel, never off the socket directly, so a timeout can't poison a
// connection we intend to reuse. Stale frames from a previous task are drained
// before the write.
//
// seq is the per-connection DGW frame sequence; the session increments it per
// task so a reused socket doesn't send two seq=0 frames.
//
// Frame filter note: filter by '"payload"', NOT '"step"'. The "step" field lives
// inside a nested JSON string so on the wire it appears as \"step\" (escaped).
// Filtering '"step"' plain never matches and makes the response look empty.
//
// queueName is the task's queue_name field (threadID for send/delete-thread,
// "messenger_delete_message_v2" for delete-message). threadID is used only for logging.
func sendLightspeedTask(ctx context.Context, conn *websocket.Conn, reader *wsReader, seq byte, threadID, queueName, label string, taskPayload map[string]interface{}) (string, error) {
	taskPayloadJSON, err := json.Marshal(taskPayload)
	if err != nil {
		return "", err
	}

	epochID := time.Now().UnixMilli() << 22
	innerJSON, err := json.Marshal(map[string]interface{}{
		"epoch_id": epochID,
		"tasks": []map[string]interface{}{
			{
				"failure_count": nil,
				"label":         label,
				"payload":       string(taskPayloadJSON),
				"queue_name":    queueName,
				"task_id":       msgrRandInt(100, 999), // [100,999] mirrors Python randbelow(900)+100
			},
		},
		"version_id": messengerVersionID,
	})
	if err != nil {
		return "", err
	}

	outerJSON, err := json.Marshal(map[string]interface{}{
		"app_id":     messengerAppID,
		"payload":    string(innerJSON),
		"request_id": msgrRandInt(100, 9099), // [100,9099] mirrors Python randbelow(9000)+100
		"type":       3,
	})
	if err != nil {
		return "", err
	}

	reader.drain() // discard any leftover push from a previous task on this socket
	log.Printf("messenger: lightspeed send thread_id=%s queue=%s label=%s payload=%s", threadID, queueName, label, taskPayloadJSON)
	if err := conn.WriteMessage(websocket.BinaryMessage, buildLightspeedWSFrame(seq, outerJSON)); err != nil {
		return "", fmt.Errorf("messenger: send lightspeed task: %w", err)
	}

	// Return on the first "payload" frame. No post-ack wait: the old 3s read for
	// an "instant reply" was diagnostic-only and always ended in a timeout, which
	// is both slow and (on a reused socket) fatal to the connection.
	for i := 0; i < 10; i++ {
		chunk, err := reader.read(ctx, messengerReadDeadline)
		if err != nil {
			if errors.Is(err, errWSTimeout) {
				log.Printf("messenger: lightspeed thread_id=%s: no payload frame (timeout)", threadID)
				return "", nil
			}
			return "", err
		}
		idx := bytes.Index(chunk, []byte(`{"`))
		if idx < 0 {
			continue
		}
		text := string(chunk[idx:])
		log.Printf("messenger: lightspeed recv[%d]: %s", i, text)
		if strings.Contains(text, `"payload"`) {
			return text, nil
		}
	}
	log.Printf("messenger: lightspeed thread_id=%s: no payload frame in 10 messages", threadID)
	return "", nil
}

// assertTaskAccepted validates the LightSpeed response.
// Returns the message ID (mid.$...) extracted from replaceOptimsiticMessage on success,
// or "" when unconfirmed or when action != "send".
// Check order MUST match Python _assert_accepted exactly:
// markOptimisticMessageFailed variants BEFORE issueNewTask/issueNewError,
// because E2EE responses contain both — checking issueNewTask first would
// swallow the more specific E2EE error.
func assertTaskAccepted(response, threadID, action string) (string, error) {
	if response == "" {
		return "", fmt.Errorf("messenger: thread_id=%s: no LightSpeed response for task %s", threadID, action)
	}

	// Unwrap one JSON level to get inner payload string (unescaped).
	// All subsequent checks and regex matches use inner, not response —
	// the raw response has quotes escaped as \" so regex patterns with literal "
	// would never match the escaped form.
	inner := response
	var outer map[string]interface{}
	if err := json.Unmarshal([]byte(response), &outer); err == nil {
		if p, ok := outer["payload"].(string); ok && p != "" {
			inner = p
		}
	}

	if action == "send" && strings.Contains(inner, "markOptimisticMessageFailed") {
		// E2EE hard block — never retry
		if strings.Contains(inner, "Cannot create open thread") && strings.Contains(inner, "E2EE") {
			return "", fmt.Errorf("messenger: thread_id=%s: E2EE block (Cannot create open thread between users with default E2EE)", threadID)
		}
		if strings.Contains(inner, "cutoverHandleInvalidSendToOpen") || strings.Contains(inner, "invalid_send_task_queue") {
			return "", fmt.Errorf("messenger: thread_id=%s: E2EE cutover — thread migrated to E2EE", threadID)
		}

		// mailboxTaskCompletionApiOnTaskCompletion: false = server explicitly completed
		// the task with failure. Combined with markOptimisticMessageFailed, this is a
		// definitive send failure (not ambiguous). Checked BEFORE ambiguous signals.
		if reTaskCompletionFailed.MatchString(inner) {
			reason := firstMatch(reMarkFailed, inner)
			if reason == "" {
				reason = "Couldn't send."
			}
			return "", &PageRejectedError{ThreadID: threadID, Reason: reason}
		}

		// insertMessageSendErrorForMessage / "Write error response from server":
		// Python verified 2026-09-04 that message STILL ARRIVES despite this signal
		// when mailboxTaskCompletionApiOnTaskCompletion is absent.
		if strings.Contains(inner, "insertMessageSendErrorForMessage") || strings.Contains(inner, "Write error response from server") {
			reason := firstMatch(reMarkFailed, inner)
			if reason == "" {
				reason = "send failed"
			}
			log.Printf("messenger: thread_id=%s: ambiguous delivery — %q (tin có thể đã tới, không retry)", threadID, reason)
			return "", nil
		}

		// markOptimisticMessageFailed without specific reason — treat as noise
		log.Printf("messenger: thread_id=%s: markOptimisticMessageFailed (no specific reason, ignoring)", threadID)
	}

	// issueNewTask/issueNewError: server hard-rejected the task — checked LAST
	if strings.Contains(inner, "issueNewTask") || strings.Contains(inner, "issueNewError") {
		return "", fmt.Errorf("messenger: thread_id=%s: server rejected task %s (issueNewTask/issueNewError). Response: %.400s",
			threadID, action, inner)
	}

	// Extract message ID from confirmed-delivery signals.
	// Apply regex to inner (unescaped payload) — the raw response has \" escaped quotes
	// which would prevent the regex from matching literal " delimiters.
	// replaceOptimsiticMessage (text/photo): ["replaceOptimsiticMessage","<otid>","mid.$..."]
	// lsInsertMessage (share/XMA):           ["lsInsertMessage","mid.$...","<otid>",...]
	var msgID string
	switch {
	case strings.Contains(inner, "replaceOptim"):
		msgID = firstMatch(reMessageIDReplace, inner)
		log.Printf("messenger: thread_id=%s: task %s confirmed delivered (replaceOptimisticMessage) msgID=%s", threadID, action, msgID)
	case strings.Contains(inner, "lsInsertMessage"):
		msgID = firstMatch(reMessageIDInsert, inner)
		log.Printf("messenger: thread_id=%s: task %s confirmed delivered (lsInsertMessage) msgID=%s", threadID, action, msgID)
	default:
		log.Printf("messenger: thread_id=%s: task %s accepted (no error signals; delivery unconfirmed)", threadID, action)
	}
	return msgID, nil
}
