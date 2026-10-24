package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"facebook-inbox/internal/utils"
)

// ContentStore persists the content list as a single pipe-delimited line.
type ContentStore struct {
	path string
}

func NewContentStore(p Paths) *ContentStore      { return &ContentStore{path: p.Content} }
func NewShareContentStore(p Paths) *ContentStore { return &ContentStore{path: p.ShareContent} }

// Load returns the content items. A missing file yields an empty slice.
func (s *ContentStore) Load() ([]string, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read content: %w", err)
	}
	return utils.ParseContent(string(raw)), nil
}

// Save writes the content items atomically as a pipe-delimited line.
func (s *ContentStore) Save(items []string) error {
	if err := utils.AtomicWrite(s.path, []byte(utils.JoinContent(items)), 0o644); err != nil {
		return fmt.Errorf("save content: %w", err)
	}
	return nil
}
