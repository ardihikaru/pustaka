# Security policy

## Supported versions

Only the latest release on `main` receives fixes. The current version is in
`docs/assets/toc.js` (`product.semver`) and on the changelog page.

## Reporting a vulnerability

Please report privately through GitHub:
**Security → Advisories → Report a vulnerability** on
<https://github.com/ardihikaru/pustaka/security/advisories/new>.
Do not open a public issue.

Include the version, how you ran Pustaka (`serve` flags, `PUSTAKA_AUTH_*`
settings with secrets removed), and a minimal reproduction. You should get an
acknowledgement within a week.

## Scope

In scope — the Go engine in `main.go`:

- the static file handler and its traversal guard (`safeJoin`),
- the `/__pustaka/*` endpoints,
- the optional login layer (`PUSTAKA_AUTH`): session signing, the allowlist,
  redirect handling (`safeNext`), lockout.

Out of scope:

- **Static mode.** Opening `docs/` from disk or serving it with another web
  server has no access control by design.
- The public demo credentials in `docker-compose.yml` and the README. They are
  documented as demo-only.
- Content you author into your own pages.
