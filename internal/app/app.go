package app

import (
	"context"
	"errors"
	"log"
	"os"

	"facebook-inbox/internal/domain"
	"facebook-inbox/internal/service"
	"facebook-inbox/internal/storage"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the single struct bound to the Wails frontend. It is a thin facade:
// every method validates readiness and delegates to a service. No business
// logic lives here.
type App struct {
	ctx     context.Context
	paths   storage.Paths
	logFile *os.File
	logger  *log.Logger

	settings      *service.SettingsService
	pages         *service.PageService
	proxies       *service.ProxyService
	accounts      *service.AccountService
	links         *service.LinkService
	contents      *service.ContentService
	shareContents *service.ContentService
	images        *service.ImageService
	runnerSvc     *service.RunnerService

	initErr error
}

// New creates an uninitialised App. Wiring happens in Startup, once the Wails
// runtime context is available.
func New() *App {
	return &App{}
}

var errNotReady = errors.New("app.notReady")

// ready reports the initialisation error (if any) so bound methods can fail
// gracefully instead of panicking on nil services.
func (a *App) ready() error {
	if a.initErr != nil {
		return a.initErr
	}
	if a.settings == nil {
		return errNotReady
	}
	return nil
}

// --- Settings ---------------------------------------------------------------

func (a *App) GetSettings() domain.Settings {
	if a.ready() != nil {
		return domain.DefaultSettings()
	}
	return a.settings.Get()
}

func (a *App) SaveSettings(s domain.Settings) (domain.Settings, error) {
	if err := a.ready(); err != nil {
		return domain.Settings{}, err
	}
	if err := a.settings.Save(s); err != nil {
		return domain.Settings{}, err
	}
	return a.settings.Get(), nil
}

// --- Pages ------------------------------------------------------------------

func (a *App) GetPages() []string {
	if a.ready() != nil {
		return []string{}
	}
	return a.pages.Get()
}

func (a *App) SavePages(raw string) ([]string, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.pages.SetFromText(raw)
}

// --- Proxies ----------------------------------------------------------------

func (a *App) GetProxies() []string {
	if a.ready() != nil {
		return []string{}
	}
	return a.proxies.Get()
}

func (a *App) SaveProxies(raw string) ([]string, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.proxies.SetFromText(raw)
}

// --- Content ----------------------------------------------------------------

func (a *App) GetContent() []string {
	if a.ready() != nil {
		return []string{}
	}
	return a.contents.Get()
}

func (a *App) SaveContent(raw string) ([]string, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.contents.SetFromText(raw)
}

// --- ShareContent -----------------------------------------------------------

func (a *App) GetShareContent() []string {
	if a.ready() != nil {
		return []string{}
	}
	return a.shareContents.Get()
}

func (a *App) SaveShareContent(raw string) ([]string, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.shareContents.SetFromText(raw)
}

// --- Links ------------------------------------------------------------------

func (a *App) GetLinks() []string {
	if a.ready() != nil {
		return []string{}
	}
	return a.links.Get()
}

func (a *App) SaveLinks(raw string) ([]string, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.links.SetFromText(raw)
}

// --- Accounts -------------------------------------------------------------------

func (a *App) GetAccounts() []domain.AccountItem {
	if a.ready() != nil {
		return []domain.AccountItem{}
	}
	return a.accounts.Get()
}

func (a *App) AddAccounts(raw string) ([]domain.AccountItem, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.accounts.AddFromText(raw)
}

func (a *App) DeleteAccounts(ids []string) ([]domain.AccountItem, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.accounts.DeleteByIDs(ids)
}

// --- Image ------------------------------------------------------------------

// PickImage opens a native file dialog restricted to images, copies the chosen
// file into assets/images and persists the path in settings. It returns the
// stored path (empty string if the user cancelled).
func (a *App) PickImage() (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	selected, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select image",
		Filters: []runtime.FileFilter{
			{DisplayName: "Images & PDF", Pattern: "*.jpg;*.jpeg;*.png;*.webp;*.pdf"},
		},
	})
	if err != nil {
		return "", err
	}
	if selected == "" {
		return a.settings.Get().ImagePath, nil
	}
	stored, err := a.images.ImportToAssets(selected)
	if err != nil {
		return "", err
	}
	updated, err := a.settings.SetImagePath(stored)
	if err != nil {
		return "", err
	}
	return updated.ImagePath, nil
}

// ClearImage removes the configured image path.
func (a *App) ClearImage() error {
	if err := a.ready(); err != nil {
		return err
	}
	_, err := a.settings.SetImagePath("")
	return err
}

// --- Runner -----------------------------------------------------------------

func (a *App) Start() error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.runnerSvc.Start()
}

func (a *App) Stop() {
	if a.ready() != nil {
		return
	}
	a.runnerSvc.Stop()
}

func (a *App) IsRunning() bool {
	if a.ready() != nil {
		return false
	}
	return a.runnerSvc.IsRunning()
}

func (a *App) GetStats() domain.Stats {
	if a.ready() != nil {
		return domain.Stats{State: domain.RunnerStopped, Accounts: []domain.AccountSnapshot{}}
	}
	return a.runnerSvc.Stats()
}
