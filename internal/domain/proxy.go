package domain

// A Proxy is represented as a plain URL string (e.g. http://user:pass@host:port).
// The list of proxies is stored, one per line, in data/proxies.txt.
type Proxy = string
