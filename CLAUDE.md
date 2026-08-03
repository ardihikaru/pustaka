# CLAUDE.md

Guidance for AI assistants working in this repository.

## What this is

Pustaka is an HTML-first documentation framework: pages are plain HTML (never
Markdown), and a single Go binary serves, indexes, scaffolds, and validates the
site. Stdlib only — `go.mod` declares `module pustaka`, `go 1.24`, zero
dependencies. Never add one.

**One repo = one product.** `docs/` is that product's documentation,
`docs/changelog.html` its release history, and `product.semver` in
`docs/assets/toc.js` its current version (currently `0.7.0`).

## Layout

```
main.go                  The whole engine (~850 lines): serve / check / index / new
main_test.go             Go tests for check() and version-bound parsing
ai/AUTHORING_SPEC.md     The page contract — the source of truth for authoring
ai/SYSTEM_PROMPT.md      Drop-in system prompt that turns a model into a Pustaka author
docs/                    The site itself
  index.html             Landing page (data-layout="landing")
  changelog.html         Release history (.clog component)
  _template.html         Canonical page skeleton (leading _ = ignored by the engine)
  guide/ concept/        Pages, nestable to any depth (guide/deploy/, guide/sisflow/)
  assets/
    toc.js               Parent registry: site + product meta, parts list
    toc/*.js             Nav parts: overview, guide, guide-samples, concept, project
    site.js              Runtime (~610 lines): shell, search, filters, HTMX swaps
    site.css             Design system (~660 lines): light/dark, 380 px → desktop
    fonts.css + fonts/   Self-hosted woff2
    vendor/              ECharts + offline Sisflow/kanban demo iframes
    img/                 Real device screenshots for the landing page
```

## Commands

```bash
go build -o pustaka .            # ~8 MB static binary
go test ./...                    # Go unit tests
./pustaka serve ./docs           # http://localhost:8080, dev mode (live rebuild)
./pustaka serve ./docs --addr :3000 --prod   # cache instead of per-request rebuild
./pustaka check ./docs           # THE GATE — validates every page against the spec
./pustaka index ./docs           # print the generated search index (JSON)
./pustaka new ./docs guide/deploy/production.html \
    --title "Production deployment" --part guide --group "Guide"
```

Current baseline: `check` reports `✓ 21 pages valid, registry consistent`;
`index` emits 137 records. If your change moves those numbers, that is expected —
if it makes `check` fail, the change is not done.

Pages also work with zero install: opening `docs/index.html` over `file://`
gives search, filters, charts, and theming from the static registry. Do not
break static mode.

## The workflow that matters

Any docs change is a loop, not a one-shot:

1. Write or scaffold the page (`pustaka new` computes the asset prefix and
   registers the entry for you).
2. Register it in the right `docs/assets/toc/<part>.js`, above the
   `/* pustaka:insert */` marker.
3. `./pustaka check ./docs` — fix every reported problem, re-run until `✓`.
4. `./pustaka serve ./docs` and look at it.

`check` is the same gate the AI authoring loop uses and the one to wire into CI.
Treat its output as authoritative; do not work around it by loosening a rule in
`main.go` unless the rule itself is the bug.

## Git workflow — never commit directly to `main`

**Always open a pull request. `main` is only ever updated by merging a PR.**

```bash
git checkout -b <type>/<short-slug>     # feat/ fix/ docs/ chore/ rebrand/
# …work…
go test ./... && ./pustaka check ./docs # both must pass before you push
git push -u origin <branch>
gh pr create --base main --title "…" --body "…"
```

- Never `git commit` while `main` is checked out, and never push to `main`.
  If you find yourself on `main` with changes, branch first, then commit.
- `go test ./...` and `pustaka check ./docs` must both pass before a PR is
  opened — `check` is the CI gate, so a red PR is a wasted round trip.
- Only commit or push when the user asks. Opening a PR is outward-facing:
  confirm before creating one unless the user already asked for it.
- Merge via the PR (`gh pr merge`), not a local fast-forward.

## Authoring rules you will trip over

`ai/AUTHORING_SPEC.md` is the full contract — read it before writing any page.
The rules that break things silently:

- **Asset prefix by depth.** `../` once per folder level: root pages reference
  `assets/site.css`, `guide/…` uses `../assets/site.css`, `guide/deploy/…` uses
  `../../assets/site.css`. Applies to `site.css`, `toc.js`, and `site.js`.
- **Registry key order is mandatory** — `id, file, title, desc, tags, sections`
  for page leaves, `id, title, children` for menu nodes. The Go engine parses
  these files with regexes (`rePageEntry`, `reMenuEntry` in `main.go`), so a
  reordered key makes the page invisible to the engine.
- `file:` is the full path from the docs root, forward slashes, always.
- `<body data-page="ID">` must equal the registered `id`.
- Exactly one `<main class="doc">` holds all authored content. Optional extra
  classes: `wide`, `no-pn`. Never author headers, sidebars, search UI,
  prev/next, footers, or back-to-top — the runtime injects them.
- Every `h2` needs a unique kebab-case `id` (it becomes a search record and a
  page-toc entry). `h3` ids are optional but indexed when present.
- Last two elements of `<body>`: `assets/toc.js` then `assets/site.js`.
- Registry files stay under 200 lines; `check` warns past that. Split the group
  into a new part instead of letting one grow.
- Never delete a `/* pustaka:insert */` marker — `pustaka new` needs it.
- Never duplicate a page leaf at the root and inside a menu.
- Only use `var()` tokens defined in `:root` (spec §9). An undefined custom
  property voids the whole declaration silently.
- Do not reuse the reserved shell classes `.pn .foot .pagetoc .sidebar .hd` or
  redefine `.hide`.

## Page scripts and partial navigation

Server mode swaps `<main class="doc">` HTMX-style, so page scripts re-execute on
every visit and load order flips:

- Page `<script>` tags live **inside** `<main class="doc">`, as its last
  children. Scripts outside `main` never run after a swap.
- On a normal load a page script runs **before** `site.js`; on a swap, **after**.
  Always gate on both:
  ```js
  if (window.pustaka) start(window.pustaka);
  else document.addEventListener("pustaka:ready", e => start(e.detail), { once: true });
  ```
- Make scripts idempotent: dispose prior instances via a `window.__<page>…`
  slot; never assume first run.
- `window.pustaka` exposes `search(q, limit)`, `highlight(text, terms)`,
  `url(relPath)`, `echarts()`, `chartFailed(node, msg)`, `records`, `pages`,
  `version`, `product`, `serverMode`, `onIndex(fn)`. Embed this API rather than
  re-implementing search in a page.
- Charts: never hard-code an ECharts `<script src>`. Call `H.echarts()`, which
  loads the vendored `assets/vendor/echarts.min.js` first and falls back to CDN.
  Read colors from CSS tokens and re-render on the `pustaka:theme` event; guard
  `isDisposed()` after a swap.

## No third-party requests

Fonts and ECharts are vendored under `docs/assets/`, so pages render identically
offline and under a strict CSP. Do not introduce a CDN link, external font, or
tracking script. To update a vendored dependency, drop a newer file in place.

## Engine internals (`main.go`)

Single file, sectioned by banner comments: model → parsing → index → server →
check → new → main.

- `loadRegistry` reads `assets/toc.js`, follows its `parts` list, and regex-parses
  page entries. `rebuild` reads each registered page and builds `Record`s (one
  "Top" record per page plus one per `h2`/`h3`) and an inverted token index.
- `search` is AND-of-terms with prefix matching, ranked by title hits.
- Server routes live under `/__pustaka`: `/info`, `/index.json`, `/search`, and
  `/partial/<path>.html` (returns `{id, title, classes, layout, html}` — `layout`
  is carried explicitly because `<body>` is outside the swapped region). Static
  files are served by a hand-rolled handler; `http.FileServer`'s `/index.html`
  redirect fights SPA routing. `safeJoin` is the traversal guard — keep it on
  every path that touches the filesystem.
- Dev mode (`serve` without `--prod`) rebuilds when the index is >500 ms old.
- `check` returns the process exit code (0 or 1) and prints `✗` failures and `⚠`
  warnings; only failures affect the exit code.

When changing `check` behavior, add a case to `TestCheckPageMetadata` in
`main_test.go` — it drives `check()` over a temp-dir fixture, which is the
cheapest way to test a new rule.

## Auth (optional login layer)

Off unless `PUSTAKA_AUTH` is set. Lives in the `auth` banner section of
`main.go`; env vars are `PUSTAKA_AUTH`, `_USER`, `_PASS`, `_SECRET`, `_TTL`,
`_SECURE`. Sessions are a stateless HMAC-signed cookie whose payload includes a
fingerprint of the credential pair, so changing the user or password invalidates
outstanding cookies.

Rules to preserve when touching it:

- **`a == nil` must stay zero-diff.** When auth is off the routes are never
  registered and `/__pustaka/info` returns byte-identical JSON to before.
  `TestAuthDisabledPassthrough` pins this.
- The middleware wraps `gzipMiddleware`, not the other way round, and classifies
  on `normPath()` — a raw `r.URL.Path` comparison is an allowlist bypass because
  `net/http` has already percent-decoded it.
- Guarded requests answer 302-to-login for browser navigations and 401 JSON for
  everything else. Never redirect a non-GET. Denials use `jsonStatus`, not
  `jsonOut` — the latter always commits a 200.
- `safeNext` runs at all three points: link build, POST parse, final redirect.
- The design leans on `_`-prefixed files being invisible to the engine;
  `TestCheckIgnoresLoginPage` guards that contract.
- `docs/_login.html` is both the served page and the `go:embed` fallback, so
  `go build` depends on it existing. It must not load `toc.js`/`site.js` — that
  would leak the table of contents to a signed-out visitor.

## Changelog and versioning

A release is **one** change: prepend a `.rel` section to the top of `.clog` in
`docs/changelog.html` **and** bump `product.semver` in `docs/assets/toc.js`.
Never ship one without the other. Version h3 ids use the `v0-6-0` pattern so
releases are searchable and deep-linkable. SemVer rules: major = forks/pages must
change to keep working; minor = new backwards-compatible capability; patch =
fixes only.

Keep the `site.version` field in `toc.js` (`v0.7.0`) in sync with
`product.semver`, and update `ai/AUTHORING_SPEC.md`'s version header when the
contract itself changes.

## Docs about docs

The `docs/` tree doubles as living component documentation. When you change
runtime behavior, the CSS system, or the spec, update the page that documents it
(`guide/authoring.html`, `guide/page-metadata.html`, `guide/nested-sidebar.html`,
`concept/architecture.html`, …) in the same change. The README's page/record
counts drift — trust `pustaka check` and `pustaka index` output over prose.

## Style

- Sentence-case headings, short ledes, concrete verbs.
- Go code: stdlib only, banner-comment sections, comments explain *why* a
  non-obvious choice was made (see the `http.FileServer` and `data-layout` notes)
  rather than restating the code.
- Match the density and idiom of the surrounding file; this codebase is terse.
