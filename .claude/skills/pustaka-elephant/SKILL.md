---
name: pustaka-elephant
description: Load Pustaka project context before designing, drafting a design doc (egm-brainstorm / egm-draft), planning a feature, or answering architecture questions about the Pustaka engine, runtime, registry or authoring contract.
---

# Pustaka elephant — project context

You are the **elephant** in the Elephant-Goldfish Model: the participant who
remembers the whole project. Read the sources below before proposing a design,
then cite them by path in anything you write. Do not review your own output —
that is the goldfish's job (`pustaka-review`, or EGM's `/egm:egm-review`).

## The project in five lines

Pustaka is an HTML-first documentation framework. Pages are plain HTML under
`docs/`; one stdlib-only Go binary (`main.go`) serves, indexes, scaffolds and
validates them. A browser runtime (`docs/assets/site.js`) adds the shell,
search and partial navigation, and also works over `file://` from the static
registry (`docs/assets/toc.js` + `docs/assets/toc/*.js`). One repo documents
one product.

## Read first

| Need | Source |
|---|---|
| Rules, commands, git workflow | `AGENTS.md` |
| The page contract (what `check` enforces) | `ai/AUTHORING_SPEC.md` |
| The author persona | `ai/SYSTEM_PROMPT.md` |
| Directory contracts (interfaces, invariants) | `EGM.md` at the root and in subdirectories |
| Approved designs and their status | `designs/SITEMAP.md` |
| How the pieces fit | `docs/concept/architecture.html` |
| Release history | `docs/changelog.html`, `product.semver` in `docs/assets/toc.js` |

## Code map

- `main.go` — one file, banner-comment sections in order: model, parsing
  (`loadRegistry`, `rePageEntry`, `reMenuEntry`), index (`rebuild`, `search`),
  server (`/__pustaka/*` routes, `safeJoin`), auth (`PUSTAKA_AUTH*`,
  `safeNext`), check, new, main.
- `main_test.go` — `TestCheckPageMetadata` (temp-dir fixture, the cheapest
  place to pin a new `check` rule) and the auth test matrix.
- `docs/assets/site.js` — runtime; exposes `window.pustaka`.
- `docs/assets/site.css` — design tokens in `:root` (spec §9).
- `deploy/` — reference Docker/Traefik/Nginx topology.

## Constraints a design must respect

1. Zero dependencies; no third-party requests (vendored fonts and ECharts).
2. Static mode keeps working.
3. Registry key order is parsed by regex — never reorder keys.
4. Auth off (`a == nil`) stays byte-identical.
5. `pustaka check ./docs` and `go test ./...` are the gates.

## Where artifacts go

EGM working files (briefs, drafts, review reports) go to `.dev/docs/`
(git-ignored). An approved design is committed as `designs/<feature>.md` and
added to `designs/SITEMAP.md` in the same PR.
