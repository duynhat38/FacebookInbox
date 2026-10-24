package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"facebook-inbox/internal/utils"
)

// LinkStore persists the link list as a newline-separated text file.
type LinkStore struct {
	path string
}

func NewLinkStore(p Paths) *LinkStore {
	return &LinkStore{path: p.Links}
}

// Load returns the links. A missing file yields an empty slice and no error.
func (s *LinkStore) Load() ([]string, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read links: %w", err)
	}
	return utils.CleanLines(string(raw)), nil
}

// Save writes the links atomically, one per line.
func (s *LinkStore) Save(links []string) error {
	if err := utils.AtomicWrite(s.path, []byte(utils.JoinLines(links)), 0o644); err != nil {
		return fmt.Errorf("save links: %w", err)
	}
	return nil
}
