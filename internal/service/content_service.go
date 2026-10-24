package service

import (
	"sync"

	"facebook-inbox/internal/storage"
	"facebook-inbox/internal/utils"
)

// ContentService manages the content list: parsing pipe/newline-delimited text,
// trimming, dropping empties and persisting. Duplicates are preserved.
type ContentService struct {
	mu       sync.RWMutex
	store    *storage.ContentStore
	contents []string
}

func NewContentService(store *storage.ContentStore) (*ContentService, error) {
	contents, err := store.Load()
	if err != nil {
		return nil, err
	}
	return &ContentService{store: store, contents: contents}, nil
}

// Get returns the current content items.
func (s *ContentService) Get() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.contents))
	copy(out, s.contents)
	return out
}

// SetFromText parses raw text (pipe and newline delimited) and auto-saves.
func (s *ContentService) SetFromText(raw string) ([]string, error) {
	parsed := utils.ParseContent(raw)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Save(parsed); err != nil {
		return nil, err
	}
	s.contents = parsed
	out := make([]string, len(s.contents))
	copy(out, s.contents)
	return out, nil
}

// Count returns the number of content items.
func (s *ContentService) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.contents)
}
