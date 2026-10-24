package service

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"

	"facebook-inbox/internal/domain"
	"facebook-inbox/internal/storage"
)

// AccountService manages the Accounts table. Values are treated as sensitive and are
// never logged. Adding is deduped by value; deleting is by ID.
type AccountService struct {
	mu    sync.RWMutex
	store *storage.AccountStore
	items []domain.AccountItem
}

func NewAccountService(store *storage.AccountStore) (*AccountService, error) {
	items, err := store.Load()
	if err != nil {
		return nil, err
	}
	return &AccountService{store: store, items: items}, nil
}

// Get returns a copy of the current accounts items.
func (s *AccountService) Get() []domain.AccountItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshotLocked()
}

// AddFromText parses raw multi-line text, appends new non-duplicate values
// (duplicates against existing items and within the input are skipped) and
// auto-saves. It returns the full updated list.
func (s *AccountService) AddFromText(raw string) ([]domain.AccountItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing := make(map[string]struct{}, len(s.items))
	for _, it := range s.items {
		existing[it.Value] = struct{}{}
	}

	for _, line := range strings.Split(raw, "\n") {
		v := strings.TrimSpace(line)
		if v == "" {
			continue
		}
		if _, ok := existing[v]; ok {
			continue
		}
		existing[v] = struct{}{}
		s.items = append(s.items, domain.AccountItem{ID: newID(), Value: v})
	}

	if err := s.store.Save(s.items); err != nil {
		return nil, err
	}
	return s.snapshotLocked(), nil
}

// DeleteByIDs removes the items whose IDs are in the set and auto-saves.
func (s *AccountService) DeleteByIDs(ids []string) ([]domain.AccountItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	toDelete := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		toDelete[id] = struct{}{}
	}

	kept := s.items[:0:0]
	for _, it := range s.items {
		if _, ok := toDelete[it.ID]; ok {
			continue
		}
		kept = append(kept, it)
	}
	s.items = kept

	if err := s.store.Save(s.items); err != nil {
		return nil, err
	}
	return s.snapshotLocked(), nil
}

// Values returns just the raw accounts values, used by the runner.
func (s *AccountService) Values() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.items))
	for i, it := range s.items {
		out[i] = it.Value
	}
	return out
}

func (s *AccountService) snapshotLocked() []domain.AccountItem {
	out := make([]domain.AccountItem, len(s.items))
	copy(out, s.items)
	return out
}

func newID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		// rand.Read essentially never fails; fall back to a zero-ish id.
		return "id-fallback"
	}
	return hex.EncodeToString(b[:])
}
