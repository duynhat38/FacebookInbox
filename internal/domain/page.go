package domain

// A Page ID is represented as a plain string. The list of page IDs is stored,
// one per line, in data/page_ids.txt. A dedicated type keeps the intent clear
// across the service and runner layers without adding behaviour.
type PageID = string
