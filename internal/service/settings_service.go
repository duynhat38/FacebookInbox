package service

import (
	"sync"

	"facebook-inbox/internal/domain"
	"facebook-inbox/internal/storage"
	"facebook-inbox/internal/utils"
)

// SettingsService owns the in-memory settings and persists every change. All
// access is guarded by a mutex so concurrent calls from the UI and runner are
// race-free.
type SettingsService struct {
	mu    sync.RWMutex
	store *storage.SettingsStore
	cur   domain.Settings
}

func NewSettingsService(store *storage.SettingsStore) (*SettingsService, error) {
	s, err := store.Load()
	if err != nil {
		return nil, err
	}
	return &SettingsService{store: store, cur: s}, nil
}

// Get returns the current settings.
func (s *SettingsService) Get() domain.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cur
}

// Save validates nothing destructive but persists the provided settings and
// updates the in-memory copy. Numeric validation for Start is performed
// separately via Validate so the UI can still auto-save in-progress edits.
func (s *SettingsService) Save(settings domain.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if settings.Language == "" {
		settings.Language = s.cur.Language
	}
	if err := s.store.Save(settings); err != nil {
		return err
	}
	s.cur = settings
	return nil
}

// SetImagePath updates only the image path and persists the change.
func (s *SettingsService) SetImagePath(path string) (domain.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cur.ImagePath = path
	if err := s.store.Save(s.cur); err != nil {
		return domain.Settings{}, err
	}
	return s.cur, nil
}

// Validate enforces the numeric constraints required before a run can start.
func (s *SettingsService) Validate() error {
	return utils.ValidateSettings(s.Get())
}
