package runner

import "fmt"

// activity builds the runtime log messages shown in the Account table's
// "Trạng thái" column, in the user's selected language. These are activity
// strings, not status labels.
type activity struct {
	vi bool
}

func newActivity(lang string) activity {
	return activity{vi: lang != "en"}
}

func (a activity) checkLive() string {
	if a.vi {
		return "Check Live..."
	}
	return "Check Live..."
}

func (a activity) live() string {
	return "Live"
}

func (a activity) die() string {
	return "Cookie Die"
}

func (a activity) checkpoint(url string) string {
	if a.vi {
		return "Checkpoint — vào " + url + " để xác minh"
	}
	return "Checkpoint — verify at " + url
}

func (a activity) checkLiveError(msg string) string {
	return "Check Live Error: " + msg
}

func (a activity) clickingMessenger(pageID string) string {
	if a.vi {
		return fmt.Sprintf("Click Messenger %s", pageID)
	}
	return fmt.Sprintf("Click Messenger %s", pageID)
}

func (a activity) clickedMessenger(pageID string) string {
	if a.vi {
		return fmt.Sprintf("Đã Click Messenger %s", pageID)
	}
	return fmt.Sprintf("Clicked Messenger %s", pageID)
}

func (a activity) sending(pageID string) string {
	if a.vi {
		return fmt.Sprintf("Send Mess đến %s", pageID)
	}
	return fmt.Sprintf("Send message to %s", pageID)
}

func (a activity) sent(pageID string) string {
	if a.vi {
		return fmt.Sprintf("Send Mess thành công đến %s", pageID)
	}
	return fmt.Sprintf("Sent to %s", pageID)
}

func (a activity) sendingPhoto(pageID string) string {
	if a.vi {
		return fmt.Sprintf("Send Photo đến %s", pageID)
	}
	return fmt.Sprintf("Send photo to %s", pageID)
}

func (a activity) sentPhoto(pageID string) string {
	if a.vi {
		return fmt.Sprintf("Send Photo thành công đến %s", pageID)
	}
	return fmt.Sprintf("Sent photo to %s", pageID)
}

func (a activity) requestError(msg string) string {
	if msg == "" {
		return "Error: Request failed"
	}
	return "Error: " + msg
}

func (a activity) pageRejected(pageID, reason string) string {
	if a.vi {
		return fmt.Sprintf("Page %s từ chối tin nhắn: %s", pageID, reason)
	}
	return fmt.Sprintf("Page %s rejected: %s", pageID, reason)
}

func (a activity) errorLimit() string {
	if a.vi {
		return "Đã đạt giới hạn lỗi"
	}
	return "Error limit reached"
}

func (a activity) stopped() string {
	if a.vi {
		return "Đã dừng"
	}
	return "Stopped"
}

func (a activity) sharingPost(pageID string) string {
	if a.vi {
		return fmt.Sprintf("Chia sẻ bài viết đến %s", pageID)
	}
	return fmt.Sprintf("Sharing post to %s", pageID)
}

func (a activity) sharedPost(pageID string) string {
	if a.vi {
		return fmt.Sprintf("Chia sẻ bài viết thành công đến %s", pageID)
	}
	return fmt.Sprintf("Shared post to %s", pageID)
}

func (a activity) deletingThread(pageID string) string {
	if a.vi {
		return fmt.Sprintf("Xoá tin nhắn với %s", pageID)
	}
	return fmt.Sprintf("Deleting thread with %s", pageID)
}

func (a activity) deletedThread(pageID string) string {
	if a.vi {
		return fmt.Sprintf("Đã xoá tin nhắn với %s", pageID)
	}
	return fmt.Sprintf("Deleted thread with %s", pageID)
}

func (a activity) done() string {
	if a.vi {
		return "Hoàn tất"
	}
	return "Done"
}
