# Pustaka — HTML-first documentation framework (v0.8.0)

Pustaka is an open-source, HTML-first documentation framework designed for
humans and AI agents. It enables interactive documentation that agents can
discover, understand, generate, update, and validate programmatically.

Demo: [pustaka.mfardiansyah.id](https://pustaka.mfardiansyah.id) — sign in with
`admin` / `pa55wd`. Those are public demo credentials; change them anywhere the
site is not a demo.

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
│   ├── _login.html            Optional login page — engine-served, editable
│   └── assets/
│       ├── fonts.css + fonts/ Self-hosted typefaces (woff2, latin)
│       ├── login.css          Styles for the login page only
│       ├── vendor/            ECharts + generic Sisflow workbench/viewer, all offline
│       ├── img/               Real device screenshots for the landing page
│       ├── toc.js             Parent registry: site + product meta, parts list
│       ├── toc/               Nav split into parts, each < 200 lines
│       │   ├── overview.js    Home
│       │   ├── guide.js       Guide, with nested Authoring/Components/
│       │   │                  Navigation/Diagrams/Sisflow/Deployment menus
│       │   ├── guide-samples.js  Samples group (Sisflow workbenches, kanban)
│       │   ├── concept.js     Architecture, performance, FAQ
│       │   └── project.js     Changelog
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
PUSTAKA_AUTH=1 PUSTAKA_AUTH_USER=admin \
  PUSTAKA_AUTH_PASS=secret ./pustaka serve ./docs   # gate it behind a login
```

Served pages auto-upgrade: the search index is generated from the real page
HTML, and navigation becomes HTMX-style partial swaps of `<main class="doc">`
(View Transitions animation, gzip, graceful fallback to full loads). Dev mode
re-reads changed files per request — edit, refresh, done.

### Docker (local)

The local container defaults to the demo login (`admin` / `pa55wd`) on port
8080. Copy the template before using it beyond a throwaway demo.

```bash
cp .env.example .env
# edit .env: at least set a fresh PUSTAKA_AUTH_SECRET
docker compose up --build
curl http://localhost:8080/__pustaka/info
```

Set `PUSTAKA_PORT=3000` for another host port. Local HTTP uses
`PUSTAKA_AUTH_SECURE=0`; deployed stacks set it to `1` because browsers reach
them over HTTPS at the edge.

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

## Login (optional)

A Pustaka site is public by default. Set `PUSTAKA_AUTH=1` and `serve` puts one
shared credential in front of the entire site, including its homepage:

```bash
PUSTAKA_AUTH=1 \
PUSTAKA_AUTH_USER=admin \
PUSTAKA_AUTH_PASS='a-long-passphrase' \
PUSTAKA_AUTH_SECRET="$(openssl rand -hex 32)" \
  ./pustaka serve ./docs
```

Unset `PUSTAKA_AUTH` (or set it to `0`) and the login layer disappears
completely — the routes are not even registered and `/__pustaka/info` returns
exactly what it returned before. Turning it on and off again is a restart, not
an edit.

| Mode | Environment | Signed-out behaviour |
|---|---|---|
| Private (default when enabled) | `PUSTAKA_AUTH=1` | All pages, homepage, ToC, vendor, and runtime assets require login |
| Public homepage | `PUSTAKA_AUTH=1 PUSTAKA_AUTH_PUBLIC_HOME=1` | Homepage and all assets are public; other pages require login |
| Auth off | `PUSTAKA_AUTH=0` or unset | No login routes or gate; entire site is public |

### Environment variables

| Variable | Default | Meaning |
|---|---|---|
| `PUSTAKA_AUTH` | *unset* | `1`, `true`, `yes` or `on` enables the login page. Anything else disables it |
| `PUSTAKA_AUTH_USER` | — | The username. Required when enabled |
| `PUSTAKA_AUTH_PASS` | — | The password. Required when enabled |
| `PUSTAKA_AUTH_SECRET` | random | Key that signs the session cookie. Leave it unset and a random one is generated, which logs everyone out on restart |
| `PUSTAKA_AUTH_TTL` | `12h` | How long a session lasts, as a Go duration (`30m`, `12h`, `168h`) |
| `PUSTAKA_AUTH_SECURE` | `auto` | `1` marks the cookie Secure, `0` never does, `auto` follows the connection. **Set it to `1` in production** — the engine speaks plain HTTP, so behind a TLS-terminating proxy `auto` cannot tell |
| `PUSTAKA_AUTH_PUBLIC_HOME` | *unset* | `1` keeps the homepage and all `/assets/` readable signed out; empty or `0` keeps them private. Invalid values are startup errors |

Enabling auth without a username or password is a startup error, not a silent
open door. Five failed attempts from one address lock it out for a minute.

### What is public and what is not

With `PUSTAKA_AUTH=1` (the default auth mode), public routes are only the login
routes, `/__pustaka/info`, `/robots.txt`, `/favicon.ico`, `/sw.js`, and the
login page's own assets: `assets/site.css`, `assets/login.css`,
`assets/fonts.css`, and `assets/fonts/*`.

Guarded: the homepage, every other page and asset (including ToC, runtime and
vendor files), `/__pustaka/partial/…`, `/index.json`, and `/search`. Because
the check runs before the file is looked up, a page that does not exist and a
page that does look identical to a signed-out visitor.

Set `PUSTAKA_AUTH_PUBLIC_HOME=1` to make `/`, `/index.html`, and all of
`/assets/` public. This deliberately exposes runtime and ToC data so the
anonymous homepage can work. In that mode, a signed-out navigation to a private
page redirects to login with `?next=`; successful login returns there.

### The login page

`docs/_login.html` is a real file you can edit — the underscore keeps it
invisible to the ToC, to `pustaka check`, and to the partial endpoint. It is
served at `/__pustaka/auth/login` with `{{SITE_NAME}}`, `{{SITE_VERSION}}`,
`{{BASE}}`, `{{ACTION}}`, `{{NEXT}}`, `{{ERROR}}`, `{{RETRY_AFTER}}` and `{{HOME_HIDDEN}}`
substituted. In dev mode it is re-read per request, so restyling it needs only
a refresh; `--prod` caches it. A copy is compiled into the binary, so a docs
root without the file still gets a working login page.

`{{HOME_HIDDEN}}` hides the back-to-home link while the home is gated, avoiding
a signed-out redirect loop. It is visible only in public-home mode.

It deliberately does not load `toc.js` or `site.js`: a locked site should not
hand its table of contents to someone who has not signed in. It borrows the
landing page's atmosphere instead — the constellation canvas, layered
gradients, engraved grid and pointer spotlight — plus a password reveal, Caps
Lock warning, a submit button that shows progress, and a lockout countdown.
Without JavaScript the plain form still works.

### Sign-in affordances

In public-home mode, when the site is gated and the visitor is signed out, two controls appear:
a sign-in button in the header next to the theme toggle, and a **Sign in**
button in the landing hero beside *Get started*. Both vanish once signed in,
and the header shows sign-out instead. With the login layer off, neither is
rendered at all.

The hero button is not hard-coded — `index.html` asks the runtime:

```js
H.onAuth(auth => {
  if (!auth || !auth.enabled || auth.authenticated) return;
  /* … append the button … */
});
```

`onAuth(fn)` resolves after the server answers (later than `pustaka:ready`) and
replays for late callers, so a script re-running after a partial swap still gets
its answer. It reports `null` when the site is not gated, including static
`file://` mode. Use the same hook to gate anything else you want signed-out
visitors to see.

### Caveats

- **Static mode has no protection at all.** Opening `docs/index.html` from disk,
  or serving `docs/` with any other web server, bypasses this entirely — the
  gate lives in the Go engine. Do not treat the folder itself as private.
- One shared credential, not user accounts. It keeps a documentation site off
  the open web; it is not an identity system.
- Put it behind HTTPS. A password over plain HTTP is readable in transit no
  matter how the cookie is signed.

## Development and production deployment

The deployed origins are `https://dev.pustaka.mfardiansyah.id` and
`https://pustaka.mfardiansyah.id`. Each has its own Traefik stack on `rnd`:
dev binds only `10.10.0.2:19401` (`172.30.15.0/24`) and production only
`10.10.0.2:18401` (`172.30.14.0/24`). Nginx on `clino` terminates TLS and
proxies each hostname through WireGuard; Pustaka is routed directly at `/`.

The repeatable setup, edge Nginx configuration, HTTP-01 certificate command,
secret handling, and post-rollout checks are in [deploy/README.md](deploy/README.md).
On `rnd`, first preserve host-local work with `git stash --include-untracked`,
then `git pull --ff-only`, create `dev.docs.env` and `prod.docs.env` from the
template, use different generated session secrets, validate both Compose files,
and build/up both stacks. Never commit those environment files.

## Contributing

**`main` is protected by convention: every change lands through a pull
request.** Branch, verify, push, open a PR — never commit or push to `main`
directly.

```bash
git checkout -b feat/my-change
go test ./... && ./pustaka check ./docs   # both must pass before pushing
git push -u origin feat/my-change
gh pr create --base main
```

`pustaka check ./docs` is the acceptance gate — the same one the AI authoring
loop uses — so run it locally before opening the PR.

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

`go test ./...` covers the checker's page-metadata contract, version-bound
parsing, and the whole login layer: config loading, session signing (tamper,
expiry, credential rotation), the redirect-vs-401 matrix, allowlist traversal,
open-redirect rejection, lockout, placeholder escaping, and a zero-diff
assertion for the disabled path. Beyond that: `check` (21 pages incl. depth-2,
registry consistency,
prefix + link resolution) · generated index (139 records incl. per-release anchors) ·
server endpoints for nested paths, gzip integrity, traversal guard, live
dev rebuild · `new` scaffolding at depth 2 · jsdom smoke tests in static,
server, and reduced-motion modes: shell, search overlay, embedded search
demo (AND semantics, chips, empty state, server-index upgrade), AI terminal
replay, device mockups, live filters, cross-folder partial swaps + history.

Browser regression over all 21 pages asserts: zero third-party requests, the
brand font actually applied, correct chrome per layout, filters that really
hide (computed `display`) and restore, chart canvases present **and painted**
(sampled pixels), no console/page errors — plus `file://` static mode and
landing↔docs partial navigation.

Measured on this repo (loopback, warm): median partial-swap response
0.8 ms · 2.9 kB gzipped article per navigation · 31 ms full index rebuild.
Method and caveats on the performance page.
