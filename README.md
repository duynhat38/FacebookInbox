# FacebookInbox

Desktop application to manage lists of Facebook Page IDs, proxies and "Account",
and to run messaging tasks using a worker-pool model. Built with **Wails v2**
(Go backend) and **React 19 + TypeScript + Tailwind CSS 4** on the frontend.
No database — all persistent data lives in plain files inside the application
folder. UI is bilingual (Vietnamese default / English).

> **Scope & responsible use.** The messaging layer (`MessengerSender`) drives the
> Facebook Messenger web endpoints using an authenticated **session cookie** per
> Account — it resolves the recipient thread, then sends one message per step
> (text / photo / share link) and optionally deletes the message or thread
> afterwards. The live checker reports checkpoint / disabled states so dead
> cookies are skipped. Only use it with accounts you are authorized to operate and
> in compliance with Meta platform policy. "Account" values (cookies) are treated
> as sensitive: they are stored with owner-only permissions, never written to logs,
> and runtime stats are never persisted to disk.

---

## Tech stack

| Layer     | Technology                                   |
| --------- | -------------------------------------------- |
| Backend   | Go 1.25+, Wails v2.16                         |
| Frontend  | React 19, TypeScript, Vite 7, Tailwind CSS 4 |
| State     | Zustand                                       |
| Storage   | Local files (JSON / TXT), atomic writes      |
| Tests     | Storage migration test; else build / vet / typecheck / lint |
| Platform  | Windows (primary)                             |

## Folder structure

```
FacebookInbox/
├── main.go                 # Wails entry point (binds internal/app.App)
├── wails.json              # Wails project config (outputfilename: FacebookInbox)
├── go.mod / go.sum
├── build/                  # App icons & platform packaging
│   └── appicon.png         # Facebook icon source (icon.ico generated from it)
├── internal/
│   ├── app/                # Wails-bound facade + lifecycle wiring
│   ├── domain/             # Framework-free types (Settings, Stats, Worker…)
│   ├── service/            # Business logic, validation, auto-save
│   ├── storage/            # File I/O only (paths, atomic save)
│   ├── runner/             # Worker pool: runner, scheduler, queue, worker, stats
│   ├── facebook/           # Authorized Graph API sender + live checker
│   ├── httpclient/         # Per-worker HTTP client (own proxy + cookie jar)
│   └── utils/              # file / strings / validation helpers
├── frontend/
│   └── src/
│       ├── components/     # AccountTable, PageList, ContentList, ShareContentList, LinkList, ProxyList, ImagePicker, …
│       ├── pages/Home/     # Main screen
│       ├── stores/         # appStore, accountStore, runnerStore (Zustand)
│       ├── hooks/          # useT (i18n), useRunnerEvents
│       ├── i18n/           # vi.json, en.json
│       ├── types/ utils/   # shared types & helpers
│       └── App.tsx main.tsx
├── data/                   # settings.json, page_ids.txt, proxies.txt, account.json, content.txt, share_content.txt, links.txt
├── assets/images/          # imported images
└── logs/                   # daily log files
```

> **Layout note.** The spec suggested `cmd/mcare/main.go`. Wails v2 expects the
> entry point and bound struct at the project root, so `main.go` lives at the
> root and the bound `App` lives in `internal/app`. The rest of the separation
> (domain / service / storage / runner / facebook / frontend) is preserved.

## Prerequisites

- Go 1.25+
- Node.js 20+ and npm
- Wails CLI v2.16 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- A C compiler (MinGW-w64 / gcc) for cgo on Windows

## Development

```bash
# Hot-reloading dev app (Go + Vite)
wails dev
```

By default, data/assets/logs are resolved relative to the running executable.
During development you can point them at a fixed folder:

```bash
# PowerShell
$env:MCARE_ROOT = "D:\mcare-dev"; wails dev
```

Frontend-only commands (run inside `frontend/`):

```bash
npm install
npm run dev         # Vite dev server
npm run typecheck   # tsc --noEmit
npm run lint        # eslint
```

## Build

```bash
wails build
```

Produces `build/bin/FacebookInbox.exe` with the Facebook icon. The executable is
self-contained and creates its `data/`, `assets/images/` and `logs/` folders on
first run.

## Data files

All stored in the `data/` folder next to the executable:

| File            | Format                         | Contents                         |
| --------------- | ------------------------------ | -------------------------------- |
| `settings.json`      | JSON                      | threads, delay, maxErrors, pagesPerThread, accountPerLink, imagePath, language, deleteThreadAfterSend, deleteMessageAfterSend |
| `page_ids.txt`       | one per line              | Facebook Page IDs                |
| `proxies.txt`        | one per line              | proxy URLs                       |
| `content.txt`        | `|`-delimited (one line)  | message content items            |
| `share_content.txt`  | `|`-delimited (one line)  | text prepended to shared links   |
| `links.txt`          | one per line              | links to share                   |
| `account.json`       | JSON `{ "items": [...] }` | Account cookies (sensitive)      |

All writes are **atomic** (write to a temp file, then rename) so a crash during
save cannot corrupt the existing file. Every change in the UI auto-saves — there
are no Save buttons.

## Configuration (Settings)

| Field            | Rule           | Meaning                                   |
| ---------------- | -------------- | ----------------------------------------- |
| Threads          | `>= 1`         | Max Account items running concurrently       |
| Delay (seconds)  | `>= 0`         | Context-aware pause between tasks per worker |
| Max Error        | `>= 1`         | Per-worker error limit; worker stops at it |
| Pages / Thread   | `>= 1`         | Batch size a worker pulls at a time       |
| Account / Link      | `>= 1`         | How many Accounts handle each link before the next |
| Image Path       | optional       | Image attached to tasks (copied to assets)|
| Delete thread    | checkbox       | After a page's send sequence, delete the whole thread |
| Delete message   | checkbox       | Delete each message right after it is sent |

The UI has two tabs — **Page & Content & Link** (Page list + Content list + Share
content list + Link list, four columns) and **Proxy** — while the **Account table
and the Config column stay visible on both tabs**. Content and Share-content items
are `|`-delimited (newlines also split). For each page a worker runs an ordered
send sequence, one message per step: **text → photo → share text → share link →
(optional) delete**. Share content is sent as a separate text message before the
link so Facebook renders the link preview (the URL message must be a bare URL).
`Account / Link` distributes links so each link is handled by up to N consecutive
Accounts (wrapping around so no Account is left out); it is independent of
`Pages / Thread`. One page counts as exactly one success, one error, or one skip,
regardless of how many steps it has.

Invalid settings block Start and show a localized validation message.

## Worker architecture

The UI is a single table whose rows are the **Account items**. Each Account item is a
work unit processed by its own `accountWorker` goroutine; a semaphore caps
concurrency at `Threads`, so at most `Threads` Accounts run at once and the rest
report **waiting** until a slot frees.

```
Runner
 ├── Scheduler ──► Queue (mutex-guarded cursor; no page handled twice)
 ├── accountWorker(Account 1)  ┐ own http.Client, own proxy, own cookie jar
 ├── accountWorker(Account 2)  ├─ at most `Threads` running concurrently (semaphore)
 └── accountWorker(Account N)  ┘ others "waiting"
        │  per-Account snapshots
        ▼
   Aggregator goroutine (single writer of stats)
        │  runtime.EventsEmit("runner:stats")
        ▼
   React runnerStore ──► Account table (no polling)
```

- **Proxy pool** — proxies are a shared pool, not bound to Accounts. At Start a
  `ProxyAllocator` (round-robin) assigns a proxy to each worker by index:
  `len==0 → direct`, else `proxies[i % len]`. Re-running reallocates from scratch.
- **Isolation** — each worker owns an independent `http.Client` with its own
  proxy and cookie jar. Two workers may share a proxy endpoint but never share a
  client or cookie jar.
- **Scheduling** — the shared `Queue` dispenses batches of `Pages/Thread` pages;
  a single cursor guarantees no page is processed twice. Running workers pull
  batches until the queue is empty.
- **Activity log** — the "Trạng thái" column is a live runtime log (not a status
  enum): `Check Live… → Live → Send Mess đến {PageID} → Send Mess thành công… →
  Error: … / Error limit reached`. Internal `WorkerStatus` stays in the backend.
- **Proxy column** — `—` before a row runs, `Direct` when running without a
  proxy, otherwise the proxy endpoint in use (kept after Stop).
- **Max error** — a worker that reaches its error limit stops independently (last
  log "Error limit reached"); other workers keep running.
- **Delay** — applied between tasks with a context-aware timer, so Stop is
  immediate and the UI thread is never blocked.
- **Graceful stop** — `context.WithCancel`; workers observe cancellation and
  finish cleanly (no force kill).
- **Stats** — a single aggregator goroutine owns all counters and is the only
  emitter, which keeps reporting race-free. Stats are pushed to the UI via Wails
  events.

`Start` requires at least one page **and** at least one Account item.

## Checks

The project ships with one Go test (`internal/storage`, legacy-file migration).
Otherwise validate changes with:

```bash
# Go
go build ./...
go vet ./...

# Frontend (inside frontend/)
npm run typecheck
npm run lint
```

## Release

1. `cd frontend && npm run typecheck && npm run lint`
2. `go build ./... && go vet ./...`
3. `wails build`
4. Verify `build/bin/FacebookInbox.exe` launches and shows the Facebook icon.

## Security notes

- Credentials / Account values are never logged, never shown in error messages, and
  are masked in the table. `account.json` is written with owner-only permissions.
- `.gitignore` excludes `data/`, `logs/` and imported images so sensitive data is
  never committed.
