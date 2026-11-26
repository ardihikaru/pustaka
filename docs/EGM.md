# docs — the docs root

## Overview

The site Pustaka serves and validates, and the default `dir` argument for every
`pustaka` command. Authored pages are plain HTML (`index.html`,
`changelog.html`, `guide/`, `concept/`); those are content, governed by
`ai/AUTHORING_SPEC.md` and enforced by `pustaka check`, and are not documented
here. This file covers the code and engine-facing contracts that live directly
in this folder.

## Interface

- `sw.js` — service-worker kill switch: `install` calls `skipWaiting()`,
  `activate` calls `registration.unregister()`. Nothing in the site registers a
  worker; it exists so a browser holding an obsolete one replaces and drops it.
- `_template.html` — skeleton read by `[main.go:newPage]`, which substitutes the
  literal placeholders `PAGE TITLE`, `PAGE-ID`, `GROUP NAME` and rewrites
  `href="assets/` / `src="assets/` with the depth prefix from
  `[main.go:prefixFor]`.
- `_login.html` — login page served by `[main.go:auth.renderLogin]` and embedded
  at build time as `[main.go:embeddedLogin]` (`//go:embed docs/_login.html`).
  Sentinels replaced at render time: `{{SITE_NAME}}`, `{{SITE_VERSION}}`,
  `{{BASE}}`, `{{ACTION}}`, `{{NEXT}}`, `{{ERROR}}`, `{{RETRY_AFTER}}`,
  `{{HOME_HIDDEN}}`.
- [assets/](./assets/EGM.md) — browser runtime (`site.js`) and the navigation
  registry (`toc.js`, `toc/*.js`).

## Invariants

- A leading `_` makes a file invisible to the engine: `[main.go:check]` skips
  `_`-prefixed files and directories, and the `/partial/` route rejects them.
  `_template.html` and `_login.html` depend on this
  (`TestCheckIgnoresLoginPage` in `main_test.go`).
- `_login.html` must exist for `go build` to succeed (go:embed), and must not
  load `assets/toc.js` or `assets/site.js` — that would expose the ToC to a
  signed-out visitor. Its only public assets are those in
  `[main.go:publicAsset]`: `assets/fonts.css`, `assets/site.css`,
  `assets/login.css`, `assets/fonts/*`.
- `_template.html` is written for a root-level page (`assets/…` with no
  prefix); `newPage` relies on that to compute nested prefixes.
- `sw.js` is on the auth allowlist (`[main.go:auth.isPublic]`) so the kill
  switch still reaches browsers after a site becomes private.
- Every non-`_` `.html` file outside `assets/` must be registered in the ToC,
  or `check` fails.

## Side Effects

- `sw.js` unregisters the active service worker in the visitor's browser.
- `_login.html` reads/writes `localStorage["pustaka-theme"]` (shared with
  `site.js`).

## Verification

- Run: `go test ./... && go build -o pustaka . && ./pustaka check ./docs`
