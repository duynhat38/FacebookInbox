package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"facebook-inbox/internal/domain"
	"facebook-inbox/internal/utils"
)

// SettingsStore persists domain.Settings as JSON using atomic writes.
type SettingsStore struct {
	path string
}

func NewSettingsStore(p Paths) *SettingsStore {
	return &SettingsStore{path: p.Settings}
}

// Load reads settings.json. When the file does not exist it returns the
// defaults (and no error) so a first run works without setup.
func (s *SettingsStore) Load() (domain.Settings, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return domain.DefaultSettings(), nil
		}
		return domain.Settings{}, fmt.Errorf("read settings: %w", err)
	}

	// Start from defaults so missing JSON fields keep sane values.
	out := domain.DefaultSettings()
	if err := json.Unmarshal(raw, &out); err != nil {
		return domain.Settings{}, fmt.Errorf("parse settings: %w", err)
	}
	// Migrate the legacy "dataPerLink" key (renamed to "accountPerLink"). Only
	// apply it when the new key is absent, so a file already using the new key
	// is never overwritten by a stale legacy value.
	var legacy struct {
		AccountPerLink *int `json:"accountPerLink"`
		DataPerLink    *int `json:"dataPerLink"`
	}
	if json.Unmarshal(raw, &legacy) == nil &&
		legacy.AccountPerLink == nil && legacy.DataPerLink != nil {
		out.AccountPerLink = *legacy.DataPerLink
	}
	return out, nil
}

// Save writes settings.json atomically.
func (s *SettingsStore) Save(settings domain.Settings) error {
	raw, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := utils.AtomicWrite(s.path, raw, 0o644); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}
