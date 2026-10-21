# Reqbox web client

The React + TypeScript client for [Reqbox](../README.md). Vite builds it into
`dist/`, which the Go binary embeds — this folder is not deployed separately.

## Development

Start the Go backend first, then this dev server. `vite.config.ts` proxies
`/api` to `http://127.0.0.1:5178`, so the UI talks to the real server with hot
reload:

```bash
# terminal 1, repo root
go run ./cmd/reqbox

# terminal 2
pnpm install
pnpm dev          # http://localhost:5173
```

## Production

```bash
pnpm build        # writes dist/, picked up by `go build` at the repo root
```

## Structure

```
src/
  App.tsx                   shell: header, theme toggle, health poll
  components/
    workspace.tsx           layout and all request/response orchestration
    sidebar.tsx             saved requests, collections, and history
    request-bar.tsx         method, URL, send, save
    request-panel.tsx       params / headers / body / auth / publish
    response-panel.tsx      status strip, body, headers, raw
    kv-editor.tsx           key-value row editor
    code-editor.tsx         CodeMirror wrapper themed to the workspace
    environment-dialog.tsx  environment and variable editing
    curl-dialog.tsx         curl export
    run-settings-dialog.tsx timeout, redirects, TLS
    ui/                     shadcn/ui primitives (generated)
  lib/
    api.ts                  typed fetch client for every endpoint
    types.ts                shared shapes, mirroring the Go structs
    store.ts                zustand store: workspace, draft, run options
    format.ts               method tints, byte/time formatting, slugify
    theme.ts                dark/light with localStorage persistence
```

## Conventions

- **Components own their surface.** One file per pane; `workspace.tsx` only wires
  data and actions.
- **The server is the source of truth.** The Zustand store holds a draft copy for
  editing; anything persisted is re-read from the API after a mutation.
- **Colors mean something.** The only saturated colors are the HTTP method ramp
  and the status ramp, both defined as tokens in `src/index.css`. Chrome stays
  achromatic in both themes.
- **Keyboard:** `Ctrl/Cmd+Enter` sends, `Ctrl/Cmd+S` saves, `Ctrl/Cmd+N` starts a
  new request.
