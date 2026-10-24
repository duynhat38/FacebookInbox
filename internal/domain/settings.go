package domain

// Settings holds the user-configurable runner configuration. It is persisted
// to data/settings.json and auto-saved on every change from the UI.
type Settings struct {
	Threads                int    `json:"threads"`
	DelaySeconds           int    `json:"delaySeconds"`
	MaxErrors              int    `json:"maxErrors"`
	PagesPerThread         int    `json:"pagesPerThread"`
	AccountPerLink         int    `json:"accountPerLink"`
	ImagePath              string `json:"imagePath"`
	Language               string `json:"language"`
	DeleteThreadAfterSend  bool   `json:"deleteThreadAfterSend"`
	DeleteMessageAfterSend bool   `json:"deleteMessageAfterSend"`
}

// DefaultSettings returns a sensible baseline used when no settings file exists
// yet or when a loaded file is missing fields.
func DefaultSettings() Settings {
	return Settings{
		Threads:        5,
		DelaySeconds:   3,
		MaxErrors:      5,
		PagesPerThread: 10,
		AccountPerLink: 1,
		ImagePath:      "",
		Language:       "vi",
		// DeleteThreadAfterSend defaults to false (bool zero value).
	}
}
