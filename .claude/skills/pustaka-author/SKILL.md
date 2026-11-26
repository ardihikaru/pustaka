---
name: pustaka-author
description: Write or update Pustaka documentation pages (HTML under docs/) as the elephant author — scaffold, register in the ToC, author per AUTHORING_SPEC, loop on `pustaka check` until it passes, then hand off to pustaka-review. Use for any "document X", "add a page", "update the docs page" task in this repo.
---

# Pustaka author (elephant)

The author persona is defined once, in `ai/SYSTEM_PROMPT.md`. Read it and
`ai/AUTHORING_SPEC.md` now and follow them; this skill only adapts them to an
agent that can edit files and run commands instead of emitting code blocks.

## Loop

1. **Context.** Read `docs/assets/toc.js`, the part file that will own the
   page (`docs/assets/toc/<part>.js`), and one nearby page as a style
   reference. For design-level work, load the `pustaka-elephant` skill too.
2. **Scaffold** new pages instead of hand-writing boilerplate:
   ```bash
   go build -o pustaka .
   ./pustaka new ./docs <folder>/<page>.html --title "<Title>" --part <part> --group "<Group>"
   ```
   It computes the asset prefix and registers the entry above
   `/* pustaka:insert */`. Then edit `desc`, `tags` and `sections`.
3. **Write** inside `<main class="doc">` only, using the spec's component
   vocabulary. Real facts only — mark gaps `<!-- TODO: verify -->`.
4. **Gate.** `./pustaka check ./docs`. Fix every `✗` and re-run until `✓`.
   Never loosen a rule in `main.go` to get there.
5. **Release?** If the change ships product behaviour, prepend a `.rel`
   section to `docs/changelog.html` and bump `product.semver` and
   `site.version` in `docs/assets/toc.js` together.
6. **Hand off.** Run the `pustaka-review` skill on the changed pages. You do
   not approve your own pages.
