package storage

import (
	"os"
	"path/filepath"

	"facebook-inbox/internal/utils"
)

// Paths centralises every filesystem location the application uses. Nothing
// else in the codebase should build these paths by hand.
type Paths struct {
	RootDir      string
	DataDir      string
	Settings     string
	PageIDs      string
	Proxies      string
	Account      string
	Content      string
	ShareContent string
	Links        string
	AssetsDir    string
	ImagesDir    string
	LogsDir      string
}

// ResolveRoot determines the application root directory. It honours the
// MCARE_ROOT environment variable (useful for development and tests); otherwise
// it uses the directory containing the running executable, falling back to the
// current working directory.
func ResolveRoot() string {
	if env := os.Getenv("MCARE_ROOT"); env != "" {
		return env
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

// NewPaths builds the full path set rooted at root.
func NewPaths(root string) Paths {
	dataDir := filepath.Join(root, "data")
	assetsDir := filepath.Join(root, "assets")
	return Paths{
		RootDir:      root,
		DataDir:      dataDir,
		Settings:     filepath.Join(dataDir, "settings.json"),
		PageIDs:      filepath.Join(dataDir, "page_ids.txt"),
		Proxies:      filepath.Join(dataDir, "proxies.txt"),
		Account:      filepath.Join(dataDir, "account.json"),
		Content:      filepath.Join(dataDir, "content.txt"),
		ShareContent: filepath.Join(dataDir, "share_content.txt"),
		Links:        filepath.Join(dataDir, "links.txt"),
		AssetsDir:    assetsDir,
		ImagesDir:    filepath.Join(assetsDir, "images"),
		LogsDir:      filepath.Join(root, "logs"),
	}
}

// EnsureAll creates the data, assets/images and logs directories if they do not
// yet exist. It is safe to call repeatedly.
func (p Paths) EnsureAll() error {
	for _, dir := range []string{p.DataDir, p.AssetsDir, p.ImagesDir, p.LogsDir} {
		if err := utils.EnsureDir(dir); err != nil {
			return err
		}
	}
	return nil
}
