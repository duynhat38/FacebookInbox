package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"facebook-inbox/internal/utils"
)

// ProxyStore persists the proxy list as a newline-separated text file.
type ProxyStore struct {
	path string
}

func NewProxyStore(p Paths) *ProxyStore {
	return &ProxyStore{path: p.Proxies}
}

// Load returns the proxies. A missing file yields an empty slice and no error.
func (s *ProxyStore) Load() ([]string, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read proxies: %w", err)
	}
	return utils.CleanLines(string(raw)), nil
}

// Save writes the proxies atomically, one per line.
func (s *ProxyStore) Save(proxies []string) error {
	if err := utils.AtomicWrite(s.path, []byte(utils.JoinLines(proxies)), 0o644); err != nil {
		return fmt.Errorf("save proxies: %w", err)
	}
	return nil
}
