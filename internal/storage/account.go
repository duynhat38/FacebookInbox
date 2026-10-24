package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"facebook-inbox/internal/domain"
	"facebook-inbox/internal/utils"
)

// AccountStore persists the Accounts table as JSON using atomic writes.
type AccountStore struct {
	path   string
	legacy string // pre-rename "data.json" location, migrated on first load
}

func NewAccountStore(p Paths) *AccountStore {
	return &AccountStore{
		path:   p.Account,
		legacy: filepath.Join(filepath.Dir(p.Account), "data.json"),
	}
}

// Load returns the stored accounts items. A missing file yields an empty slice.
func (s *AccountStore) Load() ([]domain.AccountItem, error) {
	// One-time migration: the file was renamed data.json -> account.json. The
	// on-disk shape is unchanged, so a plain rename preserves existing accounts.
	if _, err := os.Stat(s.path); errors.Is(err, fs.ErrNotExist) {
		if _, lerr := os.Stat(s.legacy); lerr == nil {
			if rerr := os.Rename(s.legacy, s.path); rerr != nil {
				// Surface the failure instead of falling through to an empty
				// read, which would later strand data.json on the next Save.
				return nil, fmt.Errorf("migrate accounts: %w", rerr)
			}
		}
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []domain.AccountItem{}, nil
		}
		return nil, fmt.Errorf("read accounts: %w", err)
	}
	var file domain.AccountFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse accounts: %w", err)
	}
	if file.Items == nil {
		file.Items = []domain.AccountItem{}
	}
	return file.Items, nil
}

// Save writes the accounts items atomically.
func (s *AccountStore) Save(items []domain.AccountItem) error {
	if items == nil {
		items = []domain.AccountItem{}
	}
	raw, err := json.MarshalIndent(domain.AccountFile{Items: items}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode accounts: %w", err)
	}
	if err := utils.AtomicWrite(s.path, raw, 0o600); err != nil {
		return fmt.Errorf("save accounts: %w", err)
	}
	return nil
}
