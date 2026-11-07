package service

import (
	"facebook-inbox/internal/domain"
	"facebook-inbox/internal/facebook"
	"facebook-inbox/internal/runner"
)

// RunnerService ties the accounts services to the runner. It builds a run Config
// from current application state and starts/stops the runner.
type RunnerService struct {
	settings      *SettingsService
	pages         *PageService
	proxies       *ProxyService
	accounts      *AccountService
	links         *LinkService
	contents      *ContentService
	shareContents *ContentService
	images        *ImageService
	runner        *runner.Runner
	sender        facebook.Sender
	liveChecker   facebook.LiveChecker
	// onPagesChanged notifies the frontend with the remaining page list.
	onPagesChanged func([]string)
}

// SetPagesChangedHandler registers a callback invoked with the remaining pages
// whenever successfully-processed pages are removed during a run.
func (s *RunnerService) SetPagesChangedHandler(fn func([]string)) {
	s.onPagesChanged = fn
}

// handlePagesDone removes the given successfully-processed pages from the
// persistent list and notifies the frontend of the remaining pages.
func (s *RunnerService) handlePagesDone(ids []string) {
	remaining, err := s.pages.Remove(ids)
	if err != nil {
		return
	}
	if s.onPagesChanged != nil {
		s.onPagesChanged(remaining)
	}
}

func NewRunnerService(
	settings *SettingsService,
	pages *PageService,
	proxies *ProxyService,
	accounts *AccountService,
	links *LinkService,
	contents *ContentService,
	shareContents *ContentService,
	images *ImageService,
	r *runner.Runner,
	sender facebook.Sender,
) *RunnerService {
	if sender == nil {
		sender = facebook.NewMessengerSender()
	}
	return &RunnerService{
		settings:      settings,
		pages:         pages,
		proxies:       proxies,
		accounts:      accounts,
		links:         links,
		contents:      contents,
		shareContents: shareContents,
		images:        images,
		runner:        r,
		sender:        sender,
		liveChecker:   facebook.NewCookieLiveChecker(),
	}
}

// Start validates settings and inputs, then starts the runner.
func (s *RunnerService) Start() error {
	if err := s.settings.Validate(); err != nil {
		return err
	}

	settings := s.settings.Get()
	pages := s.pages.Get()
	if len(pages) == 0 {
		return runner.ErrNoPages
	}

	if settings.ImagePath != "" {
		if err := s.images.Validate(settings.ImagePath); err != nil {
			// Image path is stale (e.g. project was moved/renamed). Clear it
			// automatically so the run can proceed without the missing image.
			_, _ = s.settings.SetImagePath("")
			settings = s.settings.Get()
		}
	}

	cfg := runner.Config{
		Settings:      settings,
		Pages:         pages,
		Proxies:       s.proxies.Get(),
		Accounts:      s.accounts.Get(),
		Links:         s.links.Get(),
		Contents:      s.contents.Get(),
		ShareContents: s.shareContents.Get(),
		ImagePath:     settings.ImagePath,
		Language:      settings.Language,
		Sender:        s.sender,
		LiveChecker:   s.liveChecker,
		OnPagesDone:   s.handlePagesDone,
	}
	return s.runner.Start(cfg)
}

// Stop requests a graceful shutdown of the runner.
func (s *RunnerService) Stop() {
	s.runner.Stop()
}

// IsRunning reports whether a run is in progress.
func (s *RunnerService) IsRunning() bool {
	return s.runner.IsRunning()
}

// Stats returns the latest aggregated stats.
func (s *RunnerService) Stats() domain.Stats {
	return s.runner.LastStats()
}
