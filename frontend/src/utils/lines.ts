// cleanLines mirrors the backend's line cleaning: trim, drop empties, dedupe
// while preserving first-seen order. Used for realtime counts in the UI.
export function cleanLines(raw: string): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const line of raw.split('\n')) {
    const v = line.trim()
    if (v === '') continue
    if (seen.has(v)) continue
    seen.add(v)
    out.push(v)
  }
  return out
}

export function countLines(raw: string): number {
  return cleanLines(raw).length
}

// parseContent mirrors the backend: "|" delimits items, newlines within an item are preserved.
export function parseContent(raw: string): string[] {
  return raw
    .split('|')
    .map((s) => s.trim())
    .filter((s) => s !== '')
}

export function countContent(raw: string): number {
  return parseContent(raw).length
}
