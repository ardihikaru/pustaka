# docs/assets — browser runtime and nav registry

## Overview

The client half of Pustaka. `toc.js` plus the part files in `toc/` form the
**registry** (site/product metadata and the navigation tree); `site.js` is the
**runtime** that injects the shell (header, sidebar, search modal, page ToC,
prev/next, footer, back-to-top) around each page's authored
`<main class="doc">`. The same registry files are regex-parsed by the Go engine
(`/main.go`), so they are a cross-language contract, not just JavaScript.

Two modes, chosen at runtime:

- **Static mode** (`file://` or any static host): search runs over the
  `sections` arrays in the registry; links are plain navigations.
- **Server mode** (under `pustaka serve`): detected by a successful
  `GET __pustaka/info`; the search index is replaced by `__pustaka/index.json`
  and internal `.html` links become partial swaps via `__pustaka/partial/<file>`.

Out of scope here: `site.css`, `login.css`, `fonts.css` (styling), and the
vendored `vendor/`, `fonts/`, `img/` trees (third-party or binary assets).

## Interface

### Registry (`toc.js`, `toc/*.js`)

- `[toc.js:window.DOCS]` — `{ site: {name, tagline, version}, product: {name,
  semver, repo}, parts: [...], nav: [] }`. `parts` lists part files relative to
  the docs root, loaded in order.
- Part files (`toc/overview.js`, `toc/guide.js`, `toc/guide-samples.js`,
  `toc/concept.js`, `toc/project.js`) each call
  `window.DOCS.nav.push({ group, pages: [...] })`.
- Page leaf: `{ id, file, title, desc, tags, sections }`, where each section is
  `{ anchor, title, text }`.
- Menu node: `{ id, title, children: [...] }`, nestable to any depth.
- Insert marker `/* pustaka:insert … */` in each part — the anchor
  `[main.go:newPage]` writes new entries above.

### Runtime (`site.js`)

- `[site.js:boot]` loads every `DOCS.parts` file, then `[site.js:init]` builds
  the shell once and calls `[site.js:wirePage]` (re-run after every swap).
- `window.pustaka` (`[site.js:API]`): `version`, `product`, `pages`, `records`,
  `serverMode`, `auth`, `search(q, limit)`, `highlight(text, terms)`,
  `url(relPath)`, `onIndex(fn)`, `onAuth(fn)`, `echarts()`,
  `chartFailed(node, msg)`, `reduced`.
- DOM events: `pustaka:ready` (detail = API, fired once per full load),
  `pustaka:theme` (detail = `"light"|"dark"`, fired by `[site.js:applyTheme]`).
- Server endpoints consumed: `__pustaka/info`, `__pustaka/index.json`,
  `__pustaka/partial/<file>`, `__pustaka/auth/login`, `__pustaka/auth/logout`
  (see `[main.go:newHandler]`).
- Declarative hooks wired by `[site.js:wirePage]`: `.page-meta[data-*]`,
  `input[data-filter]` / `data-empty`, `.codeblock .copy`, `.ctabs`, `.rv`,
  `main.doc` classes `no-pn` / `no-toc`, `<body data-layout="landing">`.

## Invariants

- **Registry key order is load-bearing.** Page leaves must start
  `id, file, title, desc, tags` and menus `id, title, children`; otherwise
  `[main.go:rePageEntry]` / `[main.go:reMenuEntry]` do not match and the page is
  invisible to the engine (search, `check`, partials).
- `file:` is a forward-slash path from the docs root; `site.js` resolves it
  against `SITE_ROOT`, which is derived from the `src` of the `site.js` tag, so
  any folder depth works.
- `site: { … }` must precede `product: { … }` in `toc.js` and contain no nested
  braces: `[main.go:reSiteBlock]` matches it non-greedily to read
  `name`/`version` for the login page.
- `site.version` must equal `v` + `product.semver` (release policy).
- Every part file listed after `toc.js` keeps the `pustaka:insert` marker and
  stays at or under 200 lines (`[main.go:check]` warns otherwise). A menu's
  `children` may not be empty (`[main.go:reEmptyMenu]`, a failure).
- Page ids and menu ids share one namespace; duplicates fail `check`.
- Page scripts run before `site.js` on a full load and after it on a swap, so
  they must wait for `window.pustaka` or `pustaka:ready`, and be idempotent.
- Server mode is entered only when `/info` has no `auth` key or
  `auth.authenticated` is true; a signed-out visitor stays in static mode so no
  content endpoint is ever hit just to receive a 401.
- Any non-2xx or failed `/partial/` fetch falls back to a full navigation
  (`[site.js:swap]`), which the auth middleware then redirects to login.
- `/partial/` responses carry `layout` explicitly because `<body>` is outside
  the swapped region; `[site.js:swap]` applies or clears `data-layout`.

## Side Effects

- Injects `<script>` tags for each part file and, on demand, for ECharts
  (`[site.js:loadEcharts]`: vendored `assets/vendor/echarts.min.js` first, then
  a cdnjs URL as fallback — the only third-party request the runtime can make).
- `localStorage` keys `pustaka-theme` and `pustaka-nav-open` (swallowed if
  storage is unavailable). `_login.html` shares `pustaka-theme`.
- Sets `document.documentElement.dataset.theme`, `document.body.dataset.page` /
  `dataset.layout`, `document.title`.
- `history.pushState` on swaps and a `popstate` listener; a document-level
  click interceptor for same-origin `.html` links in server mode.
- Re-executes every `<script>` inside the swapped `main.doc`
  (`[site.js:execScripts]`).
- Clipboard writes from code-copy buttons.
- Sign-out is a real `POST` form to `__pustaka/auth/logout`.

## Verification

- Run: `go test ./... && go build -o pustaka . && ./pustaka check ./docs`
- Manual: open `docs/index.html` over `file://` (static mode) and
  `./pustaka serve ./docs` (server mode); both must render nav and search.
