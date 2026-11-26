# Pustaka Blueprint

## System Overview

Pustaka is an HTML-first documentation framework: one repo documents one
product, pages are plain HTML under `docs/`, and a single stdlib-only Go binary
(`main.go`, `package main`, `module pustaka`, `go 1.24`, zero dependencies)
serves, indexes, scaffolds and validates the site. A browser runtime
(`docs/assets/site.js`) wraps each page in a shared shell and works both from
`file://` (static mode) and against the Go server (server mode with partial
navigation).

```
 docs/assets/toc.js + toc/*.js  (registry)          docs/**/*.html  (pages)
        |  regex-parsed by Go                              |
        v                                                  v
 main.go: loadRegistry -> Site.rebuild -> records + inverted index
        |                                                  |
        +-- check  : validates pages + registry (exit 0/1, the CI gate)
        +-- index  : prints records as JSON
        +-- new    : stamps _template.html, inserts registry entry
        +-- serve  : newHandler -> gzipMiddleware -> [auth.middleware]
                       /__pustaka/{info,index.json,search,partial/…,auth/…}
                       /  (static files under safeJoin)
                              ^
                              | fetch / partial swaps
                    docs/assets/site.js (browser runtime)
```

## Local Interface & Invariants

`main.go` is one file sectioned by banner comments: model, parsing, index,
server, auth, check, new, main.

### Interface

- CLI (`[main.go:main]`): `pustaka <serve|check|index|new> [dir] [args]
  [flags]`, `dir` defaults to `./docs`. Flags: `--addr` (`:8080`), `--prod`,
  `--title`, `--part` (`guide`), `--group`.
- Model: `[main.go:Page]` (`id, file, title, desc, tags`),
  `[main.go:Record]` (`pageId, pageTitle, title, text, href, top`),
  `[main.go:Site]` (pages, records, inverted index `inv`, `built`, `mu`).
- Parsing: `[main.go:registryFiles]`, `[main.go:loadRegistry]`,
  `[main.go:siteMeta]`, `[main.go:mainInner]`, `[main.go:extractRecords]`,
  `[main.go:versionBound]`, `[main.go:versionAfter]`.
- Index: `[main.go:Site.rebuild]`, `[main.go:Site.search]` (AND of terms,
  prefix match, title hits +6 prefix / +4 contains, Top +1).
- Server: `[main.go:newHandler]`, `[main.go:serve]`,
  `[main.go:gzipMiddleware]`, `[main.go:safeJoin]`, `[main.go:jsonStatus]`,
  `[main.go:jsonOut]`. Routes under `[main.go:internalRoutePrefix]`
  (`/__pustaka`): `/info`, `/index.json`, `/search?q=` (limit 15),
  `/partial/<path>.html` → `{id, title, classes, layout, html}`; `/` serves
  static files (dirs → `index.html`). Served `.html` pages not starting with
  `_` get a pre-paint theme/font snippet before `</head>` via
  `[main.go:injectHead]` (marked by `[main.go:headMarker]`); files on disk are
  never modified.
- Auth (only when `PUSTAKA_AUTH` is truthy): `[main.go:loadAuthConfig]`,
  `[main.go:newAuth]`, `[main.go:auth]` with `middleware`, `handleLogin`,
  `handleLogout`, `sign`/`verify` (token `v1.<expiry>.<credFP>.<hmac>`),
  cookie `[main.go:authCookieName]` (`pustaka_session`); routes under
  `[main.go:authRoutePrefix]` (`/__pustaka/auth/login`, `/logout`);
  `[main.go:throttle]` brute-force limiter.
- Check: `[main.go:check]` returns the process exit code;
  `[main.go:prefixFor]` gives the `../` asset prefix per folder depth.
- New: `[main.go:newPage]`.

### Invariants

- Stdlib only; never add a module dependency.
- Registry entries are parsed by regex (`[main.go:rePageEntry]`,
  `[main.go:reMenuEntry]`), so key order in `docs/assets/toc*.js` is part of the
  contract (see [docs/assets](./docs/assets/EGM.md)).
- Every filesystem path derived from a request or CLI argument goes through
  `[main.go:safeJoin]` (static `/`, `/partial/`, `newPage`).
- Dev mode (no `--prod`) rebuilds the index when it is older than 500 ms;
  `--prod` builds once at startup and caches the login template.
- `/partial/` refuses non-`.html` and `_`-prefixed names; `_` files are
  invisible to the registry, `check`, and partials.
- `check` failures (`✗`) drive the exit code; warnings (`⚠`) do not. It
  requires per page: registration, `<!DOCTYPE html>`, viewport meta,
  `data-page` equal to the registry id, one `<main class="doc">`, depth-correct
  `assets/site.css` / `toc.js` / `site.js` references, resolvable internal
  `.html` links, unique heading ids, an id on every `h2`, and a valid
  `.page-meta` when present.
- `a == nil` (auth off) is zero-diff: auth routes are not registered and
  `/info` returns the original payload (`TestAuthDisabledPassthrough`).
- Auth wraps gzip (never the reverse); classification uses
  `[main.go:normPath]`; non-GET/HEAD or non-navigation requests get 401 JSON
  via `jsonStatus`, never a redirect; `[main.go:safeNext]` is applied when the
  login link is built, when the POST is parsed, and before redirecting.
- Changing `PUSTAKA_AUTH_USER` or `_PASS` changes `CredFP` and invalidates all
  outstanding cookies without server state.
- `[main.go:clientIP]` ignores `X-Forwarded-For` unless the direct peer is in
  `PUSTAKA_AUTH_TRUSTED_PROXIES` (`[main.go:parseTrustedProxies]`); then the
  rightmost untrusted hop is the client, so the login throttle is per visitor.

## Subdirectories & System Boundaries

- docs: [./docs/EGM.md](./docs/EGM.md) - the docs root; `sw.js` kill switch,
  `_template.html` (scaffold source for `new`), `_login.html` (served and
  go:embedded login page).
- docs/assets: [./docs/assets/EGM.md](./docs/assets/EGM.md) - browser runtime
  `site.js` (`window.pustaka` API, shell, search, partial swaps) and the
  registry `toc.js` + `toc/*.js` shared with the Go parser.
- deploy: [./deploy/EGM.md](./deploy/EGM.md) - reference nginx TLS edge,
  per-stack Traefik, dev/prod Compose stacks and env templates.
- ai (unlinked, no contract file): `ai/AUTHORING_SPEC.md` is the page-authoring
  contract that `check` enforces; `ai/SYSTEM_PROMPT.md` adapts it for a model.
- Root container files: `Dockerfile` (two-stage `golang:1.24-alpine` →
  `alpine:3.21`, non-root uid 10001, `ENTRYPOINT pustaka serve /docs --addr
  :8080 --prod`); `docker-compose.yml` (single-host demo with auth on and a
  `wget` health check on `/__pustaka/info`, overridable via `.env`).

## Global Invariants

- One product per repo: a release is a new `.rel` section in
  `docs/changelog.html` plus a `product.semver` bump in `docs/assets/toc.js`,
  with `site.version` kept equal to `v` + `product.semver`.
- No third-party requests from authored pages: fonts and ECharts are vendored
  under `docs/assets/`. The runtime's only external URL is the ECharts CDN
  fallback in `site.js`, used only when the vendored copy fails to load.
- Static mode must keep working: the registry alone must give navigation and
  search without the server.
- The Docker image ships `go.mod`, `main.go` and `docs/` only (`EGM.md` files
  are excluded by `.dockerignore`); `docs/_login.html` must
  exist at build time.

## Global Side Effects

- `serve` binds a TCP listener (`--addr`), reads `docs/` on every dev-mode
  request older than 500 ms, and sets/clears the `pustaka_session` cookie.
- `new` writes a new HTML page (creating folders) and rewrites a
  `docs/assets/toc/<part>.js` file.
- `check` and `index` are read-only and print to stdout.
- Auth keeps an in-memory per-IP failure map (swept past 1024 entries); a
  missing `PUSTAKA_AUTH_SECRET` generates a random per-process key.

## System Verification

- Run: `go test ./... && go build -o pustaka . && ./pustaka check ./docs`
