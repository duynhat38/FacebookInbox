package runner

import "facebook-inbox/internal/domain"

// Scheduler owns the shared page Queue, the Accounts work list, and the allocation
// of Links and Content to Accounts. The shared Queue guarantees that no page is
// handed to two workers, and batching by PagesPerThread is enforced by the
// Queue. Proxy allocation is handled separately by a ProxyAllocator.
type Scheduler struct {
	queue          *Queue
	accounts       []domain.AccountItem
	links          []string
	contents       []string
	shareContents  []string
	batchSize      int
	accountPerLink int
}

func NewScheduler(pages []string, accounts []domain.AccountItem, links, contents, shareContents []string, batchSize, accountPerLink int) *Scheduler {
	if batchSize < 1 {
		batchSize = 1
	}
	if accountPerLink < 1 {
		accountPerLink = 1
	}
	return &Scheduler{
		queue:          NewQueue(pages),
		accounts:       accounts,
		links:          links,
		contents:       contents,
		shareContents:  shareContents,
		batchSize:      batchSize,
		accountPerLink: accountPerLink,
	}
}

// NextPages returns the next batch of pages (up to PagesPerThread), or nil when
// the page queue is exhausted.
func (s *Scheduler) NextPages() []string {
	return s.queue.Next(s.batchSize)
}

// NextPage returns the next single page from the queue, or "" when exhausted.
func (s *Scheduler) NextPage() string {
	batch := s.queue.Next(1)
	if len(batch) == 0 {
		return ""
	}
	return batch[0]
}

// LinkForAccount returns the link assigned to the Accounts at index i. Links are
// distributed so that each link is handled by up to AccountPerLink consecutive
// Accounts; once every link has its quota the assignment wraps around, so no Accounts
// is left without a link. Returns "" when there are no links.
func (s *Scheduler) LinkForAccount(i int) string {
	if len(s.links) == 0 {
		return ""
	}
	if i < 0 {
		i = 0
	}
	group := i / s.accountPerLink
	return s.links[group%len(s.links)]
}

// ContentForTask returns the content for the n-th task (round-robin), or ""
// when there is no content.
func (s *Scheduler) ContentForTask(n int) string {
	if len(s.contents) == 0 {
		return ""
	}
	if n < 0 {
		n = 0
	}
	return s.contents[n%len(s.contents)]
}

// ShareContentForTask returns the share text for the n-th task (round-robin), or ""
// when there is no share content.
func (s *Scheduler) ShareContentForTask(n int) string {
	if len(s.shareContents) == 0 {
		return ""
	}
	if n < 0 {
		n = 0
	}
	return s.shareContents[n%len(s.shareContents)]
}

// Accounts returns the Accounts work items.
func (s *Scheduler) Accounts() []domain.AccountItem { return s.accounts }

// TotalPages returns the total page count.
func (s *Scheduler) TotalPages() int { return s.queue.Total() }

// Remaining returns the number of undispatched pages.
func (s *Scheduler) Remaining() int { return s.queue.Remaining() }
