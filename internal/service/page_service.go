package service

import (
	"sync"

	"facebook-inbox/internal/storage"
	"facebook-inbox/internal/utils"
)

// PageService manages the page ID list: cleaning input, deduping and persisting.
type PageService struct {
	mu    sync.RWMutex
	store *storage.PageStore
	pages []string
}

func NewPageService(store *storage.PageStore) (*PageService, error) {
	pages, err := store.Load()
	if err != nil {
		return nil, err
	}
	return &PageService{store: store, pages: pages}, nil
}

// Get returns the current cleaned page IDs.
func (s *PageService) Get() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.pages))
	copy(out, s.pages)
	return out
}

// SetFromText replaces the list with the cleaned, deduped contents of raw text
// (one page ID per line) and auto-saves. It returns the resulting list.
func (s *PageService) SetFromText(raw string) ([]string, error) {
	cleaned := utils.CleanLines(raw)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Save(cleaned); err != nil {
		return nil, err
	}
	s.pages = cleaned
	out := make([]string, len(s.pages))
	copy(out, s.pages)
	return out, nil
}

// Count returns the number of page IDs.
func (s *PageService) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pages)
}

// Remove deletes the given page IDs from the list and auto-saves, returning the
// remaining pages. It is used to consume pages as they are successfully
// processed, so a page is never shared twice and re-runs skip finished pages.
func (s *PageService) Remove(ids []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(ids) > 0 {
		toDelete := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			toDelete[id] = struct{}{}
		}
		kept := s.pages[:0:0]
		for _, p := range s.pages {
			if _, ok := toDelete[p]; ok {
				continue
			}
			kept = append(kept, p)
		}
		s.pages = kept
		if err := s.store.Save(s.pages); err != nil {
			return nil, err
		}
	}

	out := make([]string, len(s.pages))
	copy(out, s.pages)
	return out, nil
}
