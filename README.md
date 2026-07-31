# Pustaka — HTML-first documentation framework (v0.6.0)

Pustaka is an open-source, HTML-first documentation framework designed for
humans and AI agents. It enables interactive documentation that agents can
discover, understand, generate, update, and validate programmatically.

Pages are plain HTML (not Markdown), so every page can ship real interactivity:
live filters, Apache ECharts, tabs, and scroll reveals. A single Go binary
serves, indexes, scaffolds, and validates the site.

**Usage model: one repo = one product.** Fork this repo for each product you
document; `docs/` becomes that product's docs, `changelog.html` its release
history, and `product.semver` in `assets/toc.js` its current version.

```
pustaka/
├── main.go                    The engine: serve / check / index / new
├── go.mod                     Go 1.24+ (stdlib only — any newer Go works)
├── docs/                      The site — pages at any folder depth
│   ├── index.html             Landing page — distinct top-bar layout,
│   │                          hero, real device screenshots, live search
│   ├── changelog.html         Product release history (semver)
│   ├── guide/                 ← folders keep it tidy
│   │   ├── getting-started.html
│   │   ├── authoring.html
│   │   ├── markdown-to-html.html
│   │   ├── page-metadata.html
│   │   ├── nested-sidebar.html
│   │   ├── ai-tutorial.html   ← how to prompt a model to write pages
│   │   ├── charts.html
│   │   ├── sisflow/           ← offline Sisflow guides and derived samples
│   │   │   ├── erd.html
│   │   │   ├── system-architecture.html
│   │   │   ├── block-diagram.html
│   │   │   ├── class-diagram.html
│   │   │   ├── request-workflow.html
│   │   │   ├── org-chart-editor.html
│   │   │   └── flowchart.html
│   │   ├── kanban-board.html  ← DOM-first board sample
│   │   └── deploy/
│   │       └── production.html   ← two levels deep, made by `pustaka new`
│   ├── concept/
│   │   ├── architecture.html
│   │   ├── performance.html   ← measured numbers, reproducible
│   │   └── faq.html           ← authored via the AI loop
│   ├── _template.html         Canonical skeleton (underscore = ignored)
│   └── assets/
│       ├── fonts.css + fonts/ Self-hosted typefaces (woff2, latin)
│       ├── vendor/            ECharts + generic Sisflow workbench/viewer, all offline
│       ├── img/               Real device screenshots for the landing page
│       ├── toc.js             Parent registry: site + product meta, parts list
│       ├── toc/               Nav split into parts, each < 200 lines
│       │   ├── overview.js
│       │   ├── guide.js
│       │   └── concept.js
│       ├── site.css           Design system (light/dark, 380 px → desktop)
│       └── site.js            Runtime: shell, search, filters, HTMX swaps
├── ai/
│   ├── AUTHORING_SPEC.md      The standard every page must follow
│   └── SYSTEM_PROMPT.md       Paste into a model → it becomes a Pustaka author
└── README.md
```

## Run it

**Zero-install (static mode).** Open `docs/index.html` in a browser —
search, filtering, charts, theming all work from disk.

**Full engine (server mode).** Requires Go ≥ 1.24 — use the latest release
from https://go.dev/dl (the engine is stdlib-only, so every newer Go works
unchanged):

```bash
go build -o pustaka .                        # one ~8 MB static binary
./pustaka serve ./docs                       # http://localhost:8080
./pustaka serve ./docs --addr :3000 --prod   # cache instead of live rebuild
./pustaka check ./docs                       # validate against the spec
./pustaka index ./docs                       # print the generated search index
```

Served pages auto-upgrade: the search index is generated from the real page
HTML, and navigation becomes HTMX-style partial swaps of `<main class="doc">`
(View Transitions animation, gzip, graceful fallback to full loads). Dev mode
re-reads changed files per request — edit, refresh, done.

## Folders: how to organize pages

Pages can live at any depth. Two mechanical rules, both enforced by
`pustaka check`:

1. **Asset prefix** — `../` once per folder level:
   root → `assets/site.css` · `guide/…` → `../assets/site.css` ·
   `guide/deploy/…` → `../../assets/site.css`
2. **Registry path** — register with the full path from the docs root:
   `file: "guide/deploy/production.html"`.

Three ways to add a nested page:

```bash
# 1 — scaffold (recommended): computes the prefix, registers at the marker
./pustaka new ./docs guide/deploy/production.html \
    --title "Production deployment" --part guide --group "Guide"

# 2 — copy the template by hand, then fix the prefix for the depth
cp docs/_template.html docs/concept/design-tokens.html
#    …edit: data-page id, title, and assets/ → ../assets/ …
#    …add the entry to docs/assets/toc/concept.js above /* pustaka:insert */…

# 3 — let the AI do it: give it ai/SYSTEM_PROMPT.md + the spec + the ToC
#    and the task "add guide/troubleshooting.html covering X, Y, Z"
```

Then `./pustaka check ./docs` — it validates the prefix against the depth
and that every internal link between pages resolves to a real file.

## The ToC: parent + parts

`assets/toc.js` holds site/product metadata and a `parts` list; each part
(`assets/toc/*.js`) holds one nav group and ends with the
`/* pustaka:insert */` marker. This keeps every registry file **under 200
lines** — `check` warns when one grows past that, which is your signal to
split a group. The runtime loads parts in order; the Go engine parses the
same files, so there is exactly one source of truth.

Page leaves may be wrapped in recursive menu nodes shaped as
`{ id, title, children }`. The active path opens automatically, manual branch
state persists locally, and search/previous-next continue to follow leaf order.

## Optional page metadata

Add one empty `.page-meta` element after a page lede to render tags, hashtags,
published/updated dates, and the system-version range. The component is
optional; when present, `pustaka check` validates every field. See
`guide/page-metadata.html` for the exact contract.

## Changelog & semver (one product per repo)

`docs/changelog.html` is the product's single release history, rendered with
the standardized `.clog` component (release sections, `major/minor/patch`
bump badges, `Added/Changed/Fixed/Removed` chips). The policy, also encoded
in the AI prompt: **a release = a changelog section + a `product.semver`
bump in `assets/toc.js`, always together.** Release headings carry ids like
`v0-3-0`, so individual releases are searchable and deep-linkable.

## No third-party requests

Fonts and ECharts are vendored into `docs/assets/`, so pages render identically
offline, in air-gapped deployments, and behind a strict CSP — verified in a real
browser (`performance.getEntriesByType('resource')` shows zero external hosts).
To refresh either dependency, drop a newer file in place; nothing else changes.

## The landing layout

The home page opts into a different shell with one attribute:

```html
<body data-page="home" data-layout="landing">
```

That turns the sidebar off, promotes the ToC groups into a top bar, drops the
page-toc rail and prev/next, and enables full-bleed `.lp-hero` / `.lp-full`
sections. The partial endpoint reports the layout per page, so navigating from
home to a docs page restores the docs chrome. Exactly one page should use it.

## Embedding the engine in a page

`site.js` publishes `window.pustaka` so pages can build their own search UI
instead of duplicating the engine (the landing page's live demo uses it):

```js
const start = (H) => {
  const { terms, results } = H.search("semver", 8);   // AND + prefix matching
  // H.highlight(text, terms) · H.url(relPath) · H.records · H.serverMode
  H.onIndex(() => {/* server-generated index replaced the static one */});
};
// page scripts run BEFORE site.js on a normal load, AFTER it on a swap:
if (window.pustaka) start(window.pustaka);
else document.addEventListener("pustaka:ready", e => start(e.detail), { once: true });
```

## Let an AI write the docs

1. System prompt: `ai/SYSTEM_PROMPT.md`. Context: `ai/AUTHORING_SPEC.md`,
   `docs/assets/toc.js`, and the `docs/assets/toc/*.js` parts.
2. Task: *"Document the rate-limits API with a chart per plan tier."*
3. Save the emitted files (complete pages + the complete changed ToC part).
4. `./pustaka check ./docs` — paste any errors back verbatim; the model
   fixes and re-emits. Repeat until `✓`.
5. `./pustaka serve ./docs` — nav, search, and partial navigation pick the
   page up automatically.

`docs/concept/faq.html` was produced exactly this way (first draft failed
`check` with three violations; second passed), and
`docs/guide/deploy/production.html` came from `pustaka new`.

## Fork checklist (per product)

1. Fork, then edit `docs/assets/toc.js`: `site.name`, `site.tagline`,
   `product.{name,semver,repo}`.
2. Reset `docs/changelog.html` to your product's `v0.1.0`.
3. Replace the example pages (keep `_template.html`, `assets/`, and the
   part-file structure), or keep them temporarily as living component docs.
4. Wire CI: `pustaka check ./docs` on every PR — it is the same gate the
   AI loop uses.

## Regenerating the device screenshots

The landing page shows real captures, not mock-ups. Any headless browser works;
the capture must scroll instantly (the site sets `scroll-behavior: smooth`,
which makes `scrollIntoView` animated and easy to cancel by accident):

```js
await page.setViewportSize({ width: 390, height: 780 });
await page.goto('http://localhost:8080/guide/charts.html');
await page.evaluate(() => {
  document.querySelectorAll('.rv').forEach(n => n.classList.add('in'));
  const t = document.querySelector('#demo-line');
  window.scrollTo({ top: t.getBoundingClientRect().top + scrollY - 74, behavior: 'instant' });
});
await page.waitForTimeout(2400);                 // lazy charts need to paint
await page.screenshot({ path: 'docs/assets/img/device-phone.png' });
```

## Tested

Verified in a **real browser** (headless Chromium via Playwright), not only
jsdom — jsdom has no layout engine, which is exactly why the filter, mockup and
chart bugs fixed in v0.5 went unnoticed earlier.

`check` (14 pages incl. depth-2, registry consistency, prefix + link
resolution) · generated index (76 records incl. per-release anchors) ·
server endpoints for nested paths, gzip integrity, traversal guard, live
dev rebuild · `new` scaffolding at depth 2 · jsdom smoke tests in static,
server, and reduced-motion modes: shell, search overlay, embedded search
demo (AND semantics, chips, empty state, server-index upgrade), AI terminal
replay, device mockups, live filters, cross-folder partial swaps + history.

Browser regression over all 14 pages asserts: zero third-party requests, the
brand font actually applied, correct chrome per layout, filters that really
hide (computed `display`) and restore, chart canvases present **and painted**
(sampled pixels), no console/page errors — plus `file://` static mode and
landing↔docs partial navigation.

Measured on this repo (loopback, warm): median partial-swap response
0.8 ms · 2.9 kB gzipped article per navigation · 31 ms full index rebuild.
Method and caveats on the performance page.
