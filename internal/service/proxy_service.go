package service

import (
	"sync"

	"facebook-inbox/internal/storage"
	"facebook-inbox/internal/utils"
)

// ProxyService manages the proxy list: cleaning input, deduping and persisting.
type ProxyService struct {
	mu      sync.RWMutex
	store   *storage.ProxyStore
	proxies []string
}

func NewProxyService(store *storage.ProxyStore) (*ProxyService, error) {
	proxies, err := store.Load()
	if err != nil {
		return nil, err
	}
	return &ProxyService{store: store, proxies: proxies}, nil
}

// Get returns the current cleaned proxies.
func (s *ProxyService) Get() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.proxies))
	copy(out, s.proxies)
	return out
}

// SetFromText replaces the list with the cleaned, deduped contents of raw text
// (one proxy per line) and auto-saves. It returns the resulting list.
func (s *ProxyService) SetFromText(raw string) ([]string, error) {
	cleaned := utils.CleanLines(raw)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Save(cleaned); err != nil {
		return nil, err
	}
	s.proxies = cleaned
	out := make([]string, len(s.proxies))
	copy(out, s.proxies)
	return out, nil
}

// Count returns the number of proxies.
func (s *ProxyService) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.proxies)
}
