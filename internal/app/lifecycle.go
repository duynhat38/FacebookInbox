package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"facebook-inbox/internal/domain"
	"facebook-inbox/internal/runner"
	"facebook-inbox/internal/service"
	"facebook-inbox/internal/storage"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// statsEvent is the Wails event name used to push aggregated stats to the UI.
const statsEvent = "runner:stats"

// Startup wires every layer together once the Wails runtime context exists.
// Any failure is captured in initErr so bound methods degrade gracefully and
// the user sees an error dialog instead of a crash.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	a.paths = storage.NewPaths(storage.ResolveRoot())
	if err := a.paths.EnsureAll(); err != nil {
		a.initErr = err
		a.reportFatal(err)
		return
	}

	if err := a.setupLogger(); err != nil {
		a.initErr = err
		a.reportFatal(err)
		return
	}

	if err := a.setupServices(); err != nil {
		a.initErr = err
		a.logger.Printf("startup failed: %v", err)
		a.reportFatal(err)
		return
	}

	a.logger.Printf("FacebookInbox started; root=%s", a.paths.RootDir)
}

// Shutdown performs graceful cleanup when the window closes.
func (a *App) Shutdown(ctx context.Context) {
	if a.runnerSvc != nil {
		a.runnerSvc.Stop()
	}
	if a.logger != nil {
		a.logger.Printf("shutting down")
	}
	if a.logFile != nil {
		_ = a.logFile.Close()
	}
}

func (a *App) setupLogger() error {
	name := fmt.Sprintf("mcare-%s.log", time.Now().Format("2006-01-02"))
	path := filepath.Join(a.paths.LogsDir, name)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	a.logFile = f
	a.logger = log.New(f, "", log.LstdFlags)
	return nil
}

func (a *App) setupServices() error {
	settingsStore := storage.NewSettingsStore(a.paths)
	pageStore := storage.NewPageStore(a.paths)
	proxyStore := storage.NewProxyStore(a.paths)
	accountStore := storage.NewAccountStore(a.paths)
	contentStore := storage.NewContentStore(a.paths)
	shareContentStore := storage.NewShareContentStore(a.paths)
	linkStore := storage.NewLinkStore(a.paths)

	var err error
	if a.settings, err = service.NewSettingsService(settingsStore); err != nil {
		return err
	}
	if a.pages, err = service.NewPageService(pageStore); err != nil {
		return err
	}
	if a.proxies, err = service.NewProxyService(proxyStore); err != nil {
		return err
	}
	if a.accounts, err = service.NewAccountService(accountStore); err != nil {
		return err
	}
	if a.contents, err = service.NewContentService(contentStore); err != nil {
		return err
	}
	if a.shareContents, err = service.NewContentService(shareContentStore); err != nil {
		return err
	}
	if a.links, err = service.NewLinkService(linkStore); err != nil {
		return err
	}
	a.images = service.NewImageService(a.paths)

	r := runner.New(a.emitStats, a.logger)
	a.runnerSvc = service.NewRunnerService(
		a.settings, a.pages, a.proxies, a.accounts, a.links, a.contents, a.shareContents, a.images, r, nil,
	)
	// Push the remaining page list to the frontend as pages are consumed.
	a.runnerSvc.SetPagesChangedHandler(a.emitPagesUpdated)
	return nil
}

// emitStats pushes an aggregated stats snapshot to the frontend. It is called
// only from the runner's single aggregator goroutine.
func (a *App) emitStats(stats domain.Stats) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, statsEvent, stats)
}

// pagesEvent is the Wails event carrying the remaining page list after pages
// are consumed during a run.
const pagesEvent = "pages:updated"

// emitPagesUpdated pushes the remaining page list to the frontend.
func (a *App) emitPagesUpdated(pages []string) {
	if a.ctx == nil {
		return
	}
	if pages == nil {
		pages = []string{}
	}
	runtime.EventsEmit(a.ctx, pagesEvent, pages)
}

func (a *App) reportFatal(err error) {
	if a.ctx == nil {
		return
	}
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.ErrorDialog,
		Title:   "FacebookInbox",
		Message: fmt.Sprintf("Initialization failed: %v", err),
	})
}
