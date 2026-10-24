package runner

// ProxyAllocator decides which proxy (if any) a worker uses. The proxy list is
// a shared pool; allocation happens only when a run starts, never persisted
// onto Account.
type ProxyAllocator interface {
	// Assign returns the proxy for the given worker index, or "" for a direct
	// (no-proxy) connection.
	Assign(workerIndex int, proxies []string) string
}

// RoundRobinAllocator distributes proxies to workers round-robin:
//
//	len(proxies) == 0 -> ""                    (direct)
//	len(proxies)  > 0 -> proxies[i % len(...)]  (wraps around)
type RoundRobinAllocator struct{}

func (RoundRobinAllocator) Assign(workerIndex int, proxies []string) string {
	if len(proxies) == 0 {
		return ""
	}
	if workerIndex < 0 {
		workerIndex = 0
	}
	return proxies[workerIndex%len(proxies)]
}
