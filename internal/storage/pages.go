package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"facebook-inbox/internal/utils"
)

// PageStore persists the page ID list as a newline-separated text file.
type PageStore struct {
	path string
}

func NewPageStore(p Paths) *PageStore {
	return &PageStore{path: p.PageIDs}
}

// Load returns the page IDs. A missing file yields an empty slice and no error.
func (s *PageStore) Load() ([]string, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read page ids: %w", err)
	}
	return utils.CleanLines(string(raw)), nil
}

// Save writes the page IDs atomically, one per line.
func (s *PageStore) Save(pages []string) error {
	if err := utils.AtomicWrite(s.path, []byte(utils.JoinLines(pages)), 0o644); err != nil {
		return fmt.Errorf("save page ids: %w", err)
	}
	return nil
}
