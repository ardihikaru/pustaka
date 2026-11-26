# Contributing to Pustaka

Thanks for helping. Pustaka is small on purpose: one Go file, zero
dependencies, and pages that are plain HTML. Contributions that keep it that
way are the easiest to merge.

## Ground rules

- **Stdlib only.** `go.mod` has no `require` block and must keep it that way.
- **No third-party requests.** Fonts and ECharts are vendored under
  `docs/assets/`. Do not add CDN links, external fonts, or trackers.
- **Static mode must keep working.** `docs/index.html` opened over `file://`
  has search, filters, charts and theming. Test it.
- **Pages follow the contract** in [`ai/AUTHORING_SPEC.md`](ai/AUTHORING_SPEC.md).
  `pustaka check` enforces it; treat its output as authoritative.

## Set up

```bash
git clone https://github.com/ardihikaru/pustaka.git && cd pustaka
go build -o pustaka .
./pustaka serve ./docs        # http://localhost:8080, live rebuild
```

Go 1.24 or newer is all you need.

## The change loop

```bash
git checkout -b feat/my-change          # feat/ fix/ docs/ chore/
# …edit…
gofmt -l . && go vet ./...
go test ./...
./pustaka check ./docs                  # must print ✓
git push -u origin feat/my-change
gh pr create --base main
```

`main` only moves through merged pull requests. CI runs the same commands
(`.github/workflows/ci.yml`), so a green local run means a green PR.

- New `check` rule? Add a case to `TestCheckPageMetadata` in `main_test.go`.
- New page? `./pustaka new ./docs guide/x.html --title "…" --part guide --group "Guide"`
  scaffolds it with the right asset prefix and registers it.
- Changed runtime, CSS or the spec? Update the docs page that documents it
  (`guide/authoring.html`, `guide/page-metadata.html`, `concept/architecture.html`, …)
  in the same PR.
- Release? Prepend a `.rel` section to `docs/changelog.html` **and** bump
  `product.semver` + `site.version` in `docs/assets/toc.js` together.

## Larger changes: design first (Elephant-Goldfish)

For anything bigger than a bug fix — a new engine command, a runtime
behaviour, a contract change — write a short design before the code. We use
the [Elephant-Goldfish Model](https://github.com/google/elephant-goldfish-model):

1. **Elephant drafts.** With full project context (the `pustaka-elephant`
   skill), draft `designs/<feature>.md`.
2. **Goldfish reviews.** Reviewers that see *only* the document — no chat
   history, no `CLAUDE.md` — look for gaps. Resolve or reject each finding.
3. **Humans approve** the design in a PR, then implementation follows it.

Approved designs are indexed in [`designs/SITEMAP.md`](designs/SITEMAP.md).
Working drafts and review reports go to `.dev/docs/` (git-ignored). You don't
need any AI tooling to contribute — a hand-written design in the same
template is just as welcome. See `docs/guide/elephant-goldfish.html` for the
full workflow.

## Reporting bugs and security issues

Open an issue with the template. For anything security-related (the login
layer, path traversal, open redirects) follow [SECURITY.md](SECURITY.md)
instead — please don't file a public issue.

By contributing you agree that your contribution is licensed under the
[MIT License](LICENSE) and that you follow the [Code of Conduct](CODE_OF_CONDUCT.md).
