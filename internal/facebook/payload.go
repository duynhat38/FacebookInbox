package facebook

import (
	"strconv"
	"time"
)

// msgrOtid builds the one-time message ID.
// Mirrors Python: epoch = int(time.time()*1000)<<22; otid = str(epoch + randbelow(2**22))
func msgrOtid() string {
	ms := time.Now().UnixMilli()
	r := msgrRandInt(0, (1<<22)-1)
	return strconv.FormatInt((ms<<22)+int64(r), 10)
}

// buildTextPayload mirrors Python _build_send_task_payload with send_type=_SEND_TEXT.
// thread_id is a JSON string per capture (not int64). sync_group=1 confirmed from capture.
func buildTextPayload(threadID, text string) map[string]interface{} {
	return map[string]interface{}{
		"thread_id":         threadID,
		"otid":              msgrOtid(),
		"source":            0,
		"send_type":         messengerSendText,
		"text":              text,
		"initiating_source": 1,
		"sync_group":        1,
	}
}

// buildSharePayload builds the send_type=6 URL-share task payload — exactly the
// request the real browser sends (captured 2026-10-07, Reqable record 6002). The
// post URL goes in a plain "url" field; Facebook renders the shared-post card
// server-side. There is NO XMA/url_attachment — that earlier approach was rejected
// with "Couldn't send." postURL is the wrapped /share/p/... URL; text is "" or a
// user-supplied caption. thread_id is an int64 and queue_name=thread_id (DGW), like
// the text/photo sends.
func buildSharePayload(threadID, text, postURL string) map[string]interface{} {
	tid, _ := strconv.ParseInt(threadID, 10, 64)
	return map[string]interface{}{
		"thread_id":         tid,
		"otid":              msgrOtid(),
		"source":            1572866,
		"send_type":         messengerSendShare,
		"sync_group":        1,
		"mark_thread_read":  0,
		"url":               postURL,
		"text":              text,
		"initiating_source": 0,
	}
}

// buildAttachPayload mirrors Python _build_send_task_payload with send_type=_SEND_ATTACHMENT.
// thread_id is a JSON string per capture (not int64). sync_group=1 confirmed from capture.
func buildAttachPayload(threadID, fileID string) map[string]interface{} {
	return map[string]interface{}{
		"thread_id":         threadID,
		"otid":              msgrOtid(),
		"source":            0,
		"send_type":         messengerSendAttach,
		"attachment_fbids":  []string{fileID},
		"initiating_source": 1,
		"sync_group":        1,
	}
}
