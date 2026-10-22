# Reqbox

> **Naming note.** The Go module, binary and default data path use `reqbox`.
> The repository is currently named `SuperGet`, which does not match. A rename
> is deliberately deferred until the open questions in
> [docs/DECISIONS.md](docs/DECISIONS.md#naming) are settled.

A single-binary API workbench. Save HTTP requests in SQLite, run them from the
browser, inspect the response beside the definition, and republish any saved
request as a stable mock endpoint.

Written in Go + React. The web build is embedded in the Go binary, so shipping
it means copying one file.

Outstanding work and the reasoning behind non-obvious choices are kept in
[docs/](docs/README.md) — start there for the TODO list, decision records,
changelog and development notes.

---

## Why this exists

Most API clients are desktop apps backed by a cloud account. This one is a local
service: one binary, one SQLite file, no sign-in. That makes it useful on a test
box or an internal network where you want a shared set of request definitions
and a couple of mock endpoints, without standing up a whole platform.

## What it does

- **Saved requests** — method, URL, query params, headers, body (JSON / raw /
  XML / form), and auth (Bearer, Basic, API key in header or query).
- **Environments** — named variable sets. Anywhere you can type, `{{name}}` is
  replaced from the active environment. Built-ins: `{{$timestamp}}`,
  `{{$timestampMs}}`, `{{$isoTimestamp}}`, `{{$uuid}}`, `{{$randomInt}}`.
- **Live execution** — every send goes through the Go server, so there is no
  browser CORS wall and no proxy to configure.
- **History** — each run is stored with its status, timing, size, response
  headers, response body, and a snapshot of the request that produced it. Any
  entry can be reopened or replayed.
- **Mocks** — publish a saved request under a slug and it is reachable at
  `/mock/<slug>`. The stored URL, headers, and auth are applied server-side.
  Query parameters on the mock URL fill matching `{{placeholders}}`.
- **cURL export** — the current request rendered as a copy-pasteable command.
- **Export / import** — the whole workspace as one JSON file.

## Running it

```bash
go build -o reqbox ./cmd/reqbox
./reqbox
```

Then open <http://localhost:5178>.

Flags (each also has an environment variable):

| Flag        | Env              | Default                     | Meaning                              |
| ----------- | ---------------- | --------------------------- | ------------------------------------ |
| `-addr`     | `REQBOX_ADDR`   | `:5178`                     | Listen address                       |
| `-db`       | `REQBOX_DB`     | `./data/reqbox.db`         | SQLite file (created if missing)     |
| `-static`   | `REQBOX_STATIC` | —                           | Serve the UI from a directory        |
| `-verbose`  | —                | off                         | Log every request                    |
| `-version`  | —                | —                           | Print the version and exit           |

The database path defaults to `data/reqbox.db` next to the binary when that
directory is writable; otherwise it falls back to the OS config directory. The
schema migrates itself on start, so upgrades never need a manual step.

## Building from source

Prerequisites: Go 1.24+, Node 22+, pnpm.

```bash
# 1. build the web client
cd web
pnpm install
pnpm build          # writes web/dist

# 2. build the binary (the UI is embedded from web/dist)
cd ..
go build -ldflags "-s -w -X reqbox/internal/server.Version=0.1.0" -o reqbox ./cmd/reqbox
```

Cross-compiling works because the SQLite driver is pure Go:

```bash
GOOS=linux   GOARCH=amd64 go build -o reqbox-linux   ./cmd/reqbox
GOOS=darwin  GOARCH=arm64 go build -o reqbox-darwin  ./cmd/reqbox
GOOS=windows GOARCH=amd64 go build -o reqbox.exe     ./cmd/reqbox
```

`scripts/build.ps1` and `scripts/build.sh` do the whole sequence, including the
version stamp and the platform matrix.

### Frontend development

Two terminals. The Vite dev server proxies `/api` to the Go server, so hot
reload works against the real backend:

```bash
go run ./cmd/reqbox            # backend on :5178
cd web && pnpm dev              # UI on :5173, proxying to :5178
```

Without `web/dist` the binary still starts and serves a placeholder page
explaining that the UI was not embedded.

## Layout

```
cmd/reqbox/          entry point, flags, graceful shutdown
embed.go              //go:embed of web/dist
internal/store/       SQLite schema, migrations, and queries
internal/runner/      template resolution, HTTP execution, curl rendering
internal/server/      chi router, JSON API, mock gateway, SPA serving
web/src/
  components/         UI, one file per surface
  components/ui/      shadcn/ui primitives
  lib/                API client, types, formatting, theme, zustand store
docs/
  README.md           index: what lives here and the conventions for it
  TODO.md             open work, with rationale and acceptance criteria
  DECISIONS.md        why the non-obvious choices are the way they are
  CHANGELOG.md        what changed, and which defects were fixed
  DEVELOPMENT.md      local setup, build and verification
   LEGACY-WINDOWS.md   Windows Server 2008 R2 compatibility branch
```

## Naming

The Go module, binary and default data path use `reqbox`. The repository is
currently named `SuperGet`, which does not match. A rename is intentionally
deferred; see [docs/DECISIONS.md](docs/DECISIONS.md#naming) for the
reasoning and [docs/TODO.md](docs/TODO.md#p1) for the full list of what it touches.

## API

All JSON. Errors are `{"error": "..."}` with an appropriate status.

| Method | Path                        | Purpose                              |
| ------ | --------------------------- | ------------------------------------ |
| GET    | `/api/health`               | Liveness and version                 |
| GET    | `/api/stats`                | Row counts for the sidebar           |
| GET    | `/api/collections`          | List collections                     |
| POST   | `/api/collections`          | Create                               |
| PUT    | `/api/collections/{id}`     | Rename / reorder                     |
| DELETE | `/api/collections/{id}`     | Delete (requests are kept)           |
| GET    | `/api/requests`             | List saved requests                  |
| POST   | `/api/requests`             | Create                               |
| PUT    | `/api/requests/{id}`        | Replace                              |
| DELETE | `/api/requests/{id}`        | Delete                               |
| POST   | `/api/requests/{id}/duplicate` | Copy                              |
| PUT    | `/api/requests/reorder`     | Apply an id order                    |
| POST   | `/api/execute`              | Run a request (saved or draft)       |
| POST   | `/api/curl`                 | Render the curl equivalent           |
| GET    | `/api/history?q=&limit=`    | List runs, newest first              |
| GET    | `/api/history/{id}`         | One run with bodies                  |
| POST   | `/api/history/{id}/replay`  | Re-run from its snapshot             |
| GET    | `/api/history/{id}/body`    | Download the raw response body       |
| DELETE | `/api/history/{id}`         | Delete one entry                     |
| DELETE | `/api/history`              | Clear all history                    |
| GET    | `/api/environments`         | List plus the active id              |
| POST   | `/api/environments`         | Create                               |
| PUT    | `/api/environments/{id}`    | Replace name and variables           |
| PUT    | `/api/environments/active`  | Choose the active environment        |
| DELETE | `/api/environments/{id}`    | Delete                               |
| GET    | `/api/export`               | Download the workspace               |
| POST   | `/api/import`               | Replace the workspace                |
| ANY    | `/mock/{slug}`              | Public mock of a published request   |

### Example: run a request

```bash
curl -X POST http://localhost:5178/api/execute \
  -H 'Content-Type: application/json' \
  -d '{
    "request": {
      "method": "GET",
      "url": "{{baseUrl}}/users",
      "params": [{"id":"1","key":"page","value":"2","enabled":true}],
      "headers": [{"id":"2","key":"Accept","value":"application/json","enabled":true}]
    },
    "environmentId": "<id>"
  }'
```

The response carries the status, headers, body, timing, the resolved URL, and a
`historyId` you can use to fetch or replay the run.

## Design notes

- **One executor, pooled connections.** The server keeps one runner with clients
  keyed by the two transport choices that actually vary (redirect following, TLS
  verification), so repeated calls to a mock reuse connections. `.Run` returns a
  friendly error instead of a Go trace for timeouts, DNS failures, and refused
  connections.
- **Errors are results.** A failed outbound call returns HTTP 200 with an `error`
  field so the UI can render the failure inline, and the run is still recorded in
  history. Only malformed *inbound* requests return 4xx.
- **History is self-describing.** Each entry stores the request snapshot, so
  editing or deleting the original never breaks an old run.
- **Timestamps are forgiving.** `store.Timestamp` accepts RFC 3339, empty, or
  null. Drafts from the browser arrive without timestamps and must not 400.
- **Response bodies are capped** at 2 MiB per run to keep the database sane.
  Text is stored as text; anything else is base64 with an encoding flag.
- **The schema migrates by `PRAGMA user_version`**, one numbered statement block
  at a time. Adding a migration means appending to the slice, never editing an
  applied one.

## Tests

```bash
go test ./...
```

Covers template resolution, URL and body building, auth, timeouts, redirect
handling, binary responses, curl rendering, the store CRUD paths, history
pruning, export/import round-tripping, every API route, the mock gateway, and the
Spa fallback.

## Known limits

- No WebSocket or SSE client; HTTP only.
- No request chaining or scripting hooks.
- **No authentication.** Anyone who can reach the port can read and edit
  everything, including stored credentials, which are kept in plain text in the
  SQLite file and in history snapshots so requests can be replayed. Bind to
  localhost (`-addr 127.0.0.1:5178`) or keep it on a trusted network.
- **Published mocks are callable without credentials.** A mock applies the
  saved request's own URL, headers and auth on the caller's behalf. Query
  parameters on the mock URL fill `{{placeholders}}`; if they change the target
  host, the saved auth and headers are dropped so a caller cannot redirect the
  owner's credentials elsewhere. Publishing is still handing out a capability:
  only publish requests you are willing for others to run.
- Mock endpoints run live rather than serving canned responses. The caller's
  request body is not forwarded; the saved body is used.
- Cross-origin writes are rejected via an `Origin` check, but requests without
  an `Origin` header (curl, scripts) are allowed by design.

## Audit notes

Verified by test or by hand, and covered by the suites where possible:

- Cross-compiles and runs from a clean clone without a prior web build.
- Concurrent execution (20 parallel sends) and concurrent writes (15 parallel
  creates) complete without a SQLite busy error.
- SQL injection, path traversal and oversized/nested JSON payloads are rejected
  or harmlessly absorbed.
- Omitted `verifyTls` / `followRedirects` keep their secure defaults; only an
  explicit `false` disables them.

## License

MIT. See [LICENSE](LICENSE).


