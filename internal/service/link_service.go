package service

import (
	"sync"

	"facebook-inbox/internal/storage"
	"facebook-inbox/internal/utils"
)

// LinkService manages the link list: cleaning input, deduping and persisting.
type LinkService struct {
	mu    sync.RWMutex
	store *storage.LinkStore
	links []string
}

func NewLinkService(store *storage.LinkStore) (*LinkService, error) {
	links, err := store.Load()
	if err != nil {
		return nil, err
	}
	return &LinkService{store: store, links: links}, nil
}

// Get returns the current cleaned links.
func (s *LinkService) Get() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.links))
	copy(out, s.links)
	return out
}

// SetFromText replaces the list with the cleaned, deduped contents of raw text
// (one link per line) and auto-saves.
func (s *LinkService) SetFromText(raw string) ([]string, error) {
	cleaned := utils.CleanLines(raw)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Save(cleaned); err != nil {
		return nil, err
	}
	s.links = cleaned
	out := make([]string, len(s.links))
	copy(out, s.links)
	return out, nil
}

// Count returns the number of links.
func (s *LinkService) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.links)
}
