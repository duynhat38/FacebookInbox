package utils

import "strings"

// CleanLines splits raw text into trimmed, non-empty lines with duplicates
// removed while preserving the first-seen order. It is used for page IDs and
// proxies, which are line-oriented and must not contain blank or repeated
// entries.
func CleanLines(raw string) []string {
	return DedupeNonEmpty(strings.Split(raw, "\n"))
}

// DedupeNonEmpty trims each element, drops empties and removes duplicates while
// preserving first-seen order.
func DedupeNonEmpty(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, line := range in {
		v := strings.TrimSpace(line)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// JoinLines renders a slice as newline-separated text with a trailing newline,
// which is the on-disk format for page_ids.txt, proxies.txt and links.txt.
func JoinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// ParseContent splits content text into items using "|" as the delimiter.
// Newlines within an item are preserved — enabling multi-line messages.
// Duplicates are preserved.
func ParseContent(raw string) []string {
	parts := strings.Split(raw, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

// JoinContent renders content items as the on-disk format (pipe-delimited).
func JoinContent(items []string) string {
	return strings.Join(items, "|")
}
