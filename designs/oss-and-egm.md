# Open-source readiness and the Elephant-Goldfish workflow

*   **Author:** ardihikaru (drafted with Claude Code)
*   **Status:** Approved
*   **Date:** 2026-11-26

--------------------------------------------------------------------------------

## Overview

Pustaka's GitHub repository is public but was never packaged for outside
contributors. It had no license and no CI. Its deployment files named the
maintainer's private hosts, and its AI authoring loop let the same model write
and judge a page. This design makes the repository usable by strangers. It
also adopts the
[Elephant-Goldfish Model](https://github.com/google/elephant-goldfish-model)
(EGM) for both feature design and documentation authoring.

## The Problem

1.  **No license.** Without one, default copyright applies, and nobody may
    legally reuse the code.
2.  **No contributor path.** There was no CONTRIBUTING, SECURITY, PR
    template, or CI. `pustaka check` is described as "the CI gate", but no
    CI ran it.
3.  **Private infrastructure in Git.** `deploy/` hard-coded WireGuard
    addresses, Docker subnets, the trusted edge IP and real hostnames. These
    values only work on one installation. Each one is also a value that a
    fork must find and change by hand.
4.  **Self-review.** `ai/SYSTEM_PROMPT.md` ends at "loop on `pustaka check`
    until ✓". The checker tests structure. Nothing tested whether a page is
    understandable or truthful, and the author model was the only reader.
5.  **Shared login lockout behind a proxy.** `clientIP` ignores
    `X-Forwarded-For`, which is correct when the server faces clients
    directly. Behind the reference Traefik stack every request comes from
    Traefik, so the throttle (`authMaxAttempts`, `authLockout`) counts all
    visitors as one address. Five wrong passwords from anyone lock out
    everyone.
6.  **Stale facts.** The landing kicker said `v0.6.0` and `product.repo` was
    a placeholder. The commit that added auth hardening and the deploy files
    after v0.8.0 had no changelog entry. These are fixed in the same release
    and are listed here only so the release notes are complete.

## Goals

*   A stranger can clone, build, test and open a passing PR using only
    `README.md` and `CONTRIBUTING.md`.
*   CI runs `gofmt`, `go vet`, `go test ./...`, `go build` and
    `pustaka check ./docs` on every PR.
*   No committed file contains installation-specific addresses or hostnames
    (the public demo URL in the README is the one exception).
*   Every AI-authored page is reviewed by a reviewer in a fresh context
    before merge. That reviewer is the goldfish.
*   Behind a configured trusted proxy, the login throttle keys on the real
    client address.
*   Existing deployments keep working after one documented migration step:
    create the two `*.compose.env` files. Nothing else is required.

## Non-Goals

*   Vendoring EGM. It stays an optional plugin; the repo's own skills must
    work without it.
*   Retro-designs for existing subsystems. Their `EGM.md` contracts are
    enough until a drift report says otherwise.
*   User accounts or per-user throttling. Auth stays a single shared
    credential.
*   Publishing a release binary or Docker image. This may follow later.
*   Rate limiting IPv6 clients by prefix. The throttle keys on the full
    address, so a client holding a /64 can rotate addresses. That is
    acceptable for a shared-credential docs gate.

## Open Questions

*   None blocking. A GitHub Actions release workflow (binaries per OS) is a
    candidate follow-up.

## Critical User Journeys (CUJs)

### CUJ 1: First-time contributor

*   **Given**: a fresh clone and Go ≥ 1.24.
*   **When**: they follow `CONTRIBUTING.md`, change a page and open a PR.
*   **Then**: the PR template asks for tests, `check` and changelog. CI runs
    the same gate and is green iff the local run was.

### CUJ 2: AI-authored page

*   **Given**: Claude Code in this repo.
*   **When**: the user runs `/pustaka-author <task>` then `/pustaka-review`.
*   **Then**: `pustaka check` passes, and then `pustaka-goldfish` (plus
    `egm:egm-comprehension` and `egm:egm-style` if EGM is installed) review
    the changed pages in fresh contexts. Findings are written to
    `.dev/docs/goldfish_review_report.md` and each one is fixed or rejected
    with a reason.

### CUJ 3: Operator upgrading the reference deployment

*   **Given**: a running dev/prod stack from the previous `deploy/` layout.
*   **When**: they save the current render (`compose config`), `git pull`,
    and create `{dev,prod}.compose.env` from `compose.env.example` with
    their old bind address, subnet and edge IP. Their existing
    `*.docs.env` files are left untouched.
*   **Then**: the new render shows the same published ports, host IPs and
    subnets as the saved one. `up -d --build` replaces the containers, and
    an empty or missing variable fails the render instead of binding a
    default.

### CUJ 4: Login behind the proxy

*   **Given**: the reference stack, which sets
    `PUSTAKA_AUTH_TRUSTED_PROXIES` to the stack's Docker subnet and the
    edge's tunnel IP.
*   **When**: client A fails five logins.
*   **Then**: A is locked out. Client B, also arriving through Traefik, can
    still sign in.

## Technical Plan (Architecture)

### Repository and community files

`LICENSE` (MIT), `CONTRIBUTING.md`, `SECURITY.md` (private advisories),
`CODE_OF_CONDUCT.md` (Contributor Covenant 2.1 by reference),
`THIRD_PARTY_NOTICES.md` (ECharts Apache-2.0, fonts OFL-1.1),
`.github/PULL_REQUEST_TEMPLATE.md`, issue templates, and
`.github/workflows/ci.yml`. The README leads with the pitch, a quickstart and
a license section, and points deployment at `deploy/README.md`.

### Generic deployment

`deploy/stacks/compose.{dev,prod}.yaml` read `PUSTAKA_BIND`,
`PUSTAKA_SUBNET` and `PUSTAKA_EDGE_IP` with `${VAR:?message}`, so a missing
value is an error. These values come from untracked
`deploy/stacks/{dev,prod}.compose.env`, which is passed with `--env-file`.
The same two values also feed
`PUSTAKA_AUTH_TRUSTED_PROXIES` through the compose `environment:` block, so
the trusted list cannot drift from the network it describes. The example
file ships empty values, so an unedited copy fails the render.
Traefik's static configuration moves from `traefik.yml` to `command:` flags.
Traefik accepts exactly one static source, and the trusted edge IP must be
interpolated. Nginx templates use `docs.example.com` and `10.0.0.x`.

### Elephant-Goldfish roles

| Role | Artifact | Context it gets |
|---|---|---|
| Elephant (context) | `.claude/skills/pustaka-elephant` | Project brief, code map, links to the spec, contracts and designs |
| Elephant (author) | `.claude/skills/pustaka-author` → `ai/SYSTEM_PROMPT.md` | Spec, registry, task, conversation |
| Goldfish (mechanical) | `pustaka check` | The files on disk |
| Goldfish (model) | `.claude/agents/pustaka-goldfish` (`omitClaudeMd: true`, read-only) | Target pages + `ai/AUTHORING_SPEC.md` |
| Goldfish (EGM, optional) | `egm:egm-comprehension`, `egm:egm-style` | Target pages only |
| Orchestrator | `.claude/skills/pustaka-review` | Runs the goldfish stages, writes the report |

`omitClaudeMd` is the field EGM's own Claude Code adapters use. EGM
documents it as supported from Claude Code 2.1.271. On older versions the
field is ignored and the goldfish inherits `AGENTS.md`, so the isolation
holds only on supported versions. `pustaka-goldfish` overlaps with
`egm-comprehension` on reader clarity on purpose: the repo's review must
still work when EGM is not installed.

`ai/SYSTEM_PROMPT.md` stays the single definition of the author. The skill
references it instead of copying it, so non-Claude users keep the same
contract. `AGENTS.md` is the canonical instructions file and `CLAUDE.md`
imports it. EGM artifacts go to `.dev/docs` (git-ignored). Approved designs go
to `designs/`, not `docs/designs/`, because everything under `docs/` is
served and must stay HTML-only. Directory contracts are named `EGM.md` (via
`egm-bootstrap`'s `target_filename`) so hand-written READMEs stay
human-owned. `.dockerignore` keeps them out of the image.

### Trusted-proxy client IP

Add `PUSTAKA_AUTH_TRUSTED_PROXIES`, a comma- or space-separated list of IPs
or CIDRs. An invalid entry fails startup. The throttle key is then chosen
like this:

1.  If the list is empty, or the direct peer is not in it, return the peer.
    This is the old behaviour, unchanged.
2.  Otherwise, join all `X-Forwarded-For` header lines and walk them right to
    left. Skip trusted hops. The first untrusted hop is the client.
3.  If a hop is malformed, or every hop is trusted, return the peer. The
    function never guesses.

Entries left of the first untrusted hop come from the client and are never
read, so a client cannot spoof its way out of the throttle. Unset, the
variable changes nothing. That keeps the `a == nil` zero-diff contract and
the existing direct-exposure behaviour.

Trusting the whole stack subnet also trusts its bridge gateway, so a process
on the app host could set the header. That is acceptable, because Pustaka
publishes no port and the host is already trusted.

## Verification

*   `go test ./...`. `TestClientIP` covers no proxies, an untrusted peer, a
    trusted peer, spoofed left entries, multiple header lines, all hops
    trusted, a missing header, a malformed hop, an IPv6 client, and the
    rejection of bad list entries. `TestLoadAuthConfig` covers the
    variable's parsing and its startup error.
*   `./pustaka check ./docs` → ✓, including the new
    `guide/elephant-goldfish.html`.
*   With a filled-in env file and an existing `prod.docs.env`,
    `docker compose --env-file … -f deploy/stacks/compose.prod.yaml config`
    renders the bind address, subnet, Traefik flags and derived
    `PUSTAKA_AUTH_TRUSTED_PROXIES`. With the unedited example file, it fails
    and names the variable.
*   A search of tracked files for the previous installation's hostnames,
    tunnel addresses and subnets (the list is kept outside the repo) matches
    only the README demo link.
*   After deploying, `/__pustaka/info` on both stacks reports `0.10.0`.
    Five failed logins from one client do not lock out a second client.

## Rollout

One PR, one minor release (v0.10.0). The changes are kept together because
the generic deploy manifests are what make the trusted-proxy fix
configurable without duplicating values. Splitting them would mean shipping
the fix with hard-coded addresses first. Operators follow
`deploy/README.md` → *Upgrading from the hard-coded layout*: save the old
render, pull, create the two `*.compose.env` files, diff, then `up`.

## Rollback

`git checkout <previous tag or commit>` followed by `up -d --build` on each
stack. The old manifests restore `traefik.yml` and their hard-coded values.
The extra env files are ignored. Without `PUSTAKA_AUTH_TRUSTED_PROXIES` the
engine returns to keying the throttle on the direct peer.
