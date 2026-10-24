package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// Legacy files were named data.json and used a "dataPerLink" settings key. Both
// were renamed (account.json / "accountPerLink"); loading must migrate them so
// an upgrade never silently loses the user's accounts or their configured value.
func TestLegacyMigration(t *testing.T) {
	root := t.TempDir()
	p := NewPaths(root)
	if err := p.EnsureAll(); err != nil {
		t.Fatalf("EnsureAll: %v", err)
	}
	dataDir := filepath.Join(root, "data")

	// Legacy account list under the old filename.
	legacyAccounts := `{"items":[{"id":"a1","value":"token-1"}]}`
	if err := os.WriteFile(filepath.Join(dataDir, "data.json"), []byte(legacyAccounts), 0o600); err != nil {
		t.Fatal(err)
	}
	// Legacy settings using the old key.
	legacySettings := `{"threads":5,"dataPerLink":7}`
	if err := os.WriteFile(p.Settings, []byte(legacySettings), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := NewAccountStore(p).Load()
	if err != nil {
		t.Fatalf("account Load: %v", err)
	}
	if len(items) != 1 || items[0].Value != "token-1" {
		t.Fatalf("accounts not migrated from data.json: %+v", items)
	}
	if _, err := os.Stat(p.Account); err != nil {
		t.Fatalf("account.json not created by migration: %v", err)
	}

	s, err := NewSettingsStore(p).Load()
	if err != nil {
		t.Fatalf("settings Load: %v", err)
	}
	if s.AccountPerLink != 7 {
		t.Fatalf("dataPerLink not migrated to AccountPerLink: got %d", s.AccountPerLink)
	}
}
