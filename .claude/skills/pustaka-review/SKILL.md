---
name: pustaka-review
description: Goldfish-review changed Pustaka docs pages — run `pustaka check`, then fresh-context reviewers (pustaka-goldfish, egm-comprehension, egm-style) in parallel, consolidate findings, resolve them and re-check. Use after writing or editing pages in docs/, or when asked to review docs.
---

# Pustaka review (goldfish)

Reviews must come from reviewers that never saw the drafting conversation.
You orchestrate; you do not review the pages yourself.

## Inputs

- `targets`: page paths under `docs/`. Default: the `.html` files changed on
  this branch — `git diff --name-only main...HEAD -- 'docs/*.html'` plus
  uncommitted changes.

## Stages (sequential)

1. **Mechanical goldfish.** `go build -o pustaka . && ./pustaka check ./docs`.
   Fix failures (or send them back to the author) before stage 2 — a reviewer
   should not spend tokens on what the checker already reports.
2. **Fresh-context reviewers, in parallel**, one Agent call each, all in one
   message. Pass absolute paths and nothing from this conversation:
   - `pustaka-goldfish` — the targets + `ai/AUTHORING_SPEC.md`.
   - `egm:egm-comprehension` — the targets as the document under review
     (naive reader: can someone new follow this page?).
   - `egm:egm-style` — the targets (technical writing).
   Skip the `egm:*` reviewers if the EGM plugin is not installed
   (`claude plugin install egm@egm`); say that you skipped them.
3. **Consolidate** into `.dev/docs/goldfish_review_report.md` (create the
   folder; it is git-ignored): deduplicate, keep each finding's reviewer,
   severity, `file:line`, and a proposed fix. Drop findings that contradict
   `ai/AUTHORING_SPEC.md`.
4. **Resolve.** For each finding: fix it, or record why it is rejected. Then
   re-run `./pustaka check ./docs`. Mark the report `Status: resolved` when
   every finding has an outcome.
5. **Report** a short summary to the user: fixed, rejected (with reason),
   and the final `check` line.
