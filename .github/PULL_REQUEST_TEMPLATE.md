## What

<!-- One or two sentences: what does this PR change? -->

## Why

<!-- The problem it solves or the issue it closes (Closes #123). -->

## Design

<!-- For larger changes: link the approved design in designs/ and its goldfish
review. Write "Not needed" for small fixes. -->

- Design doc:
- Goldfish review: <!-- resolved / not run / not needed -->

## Testing

- [ ] `go test ./...` passes
- [ ] `./pustaka check ./docs` prints ✓
- [ ] Looked at the change in `./pustaka serve ./docs`
- [ ] Static mode (`docs/index.html` over `file://`) still works, if the runtime changed

## Release

- [ ] Not user-visible, no release needed
- [ ] `docs/changelog.html` has a new `.rel` entry **and** `product.semver` / `site.version` in `docs/assets/toc.js` are bumped
- [ ] Docs pages describing the changed behaviour are updated
