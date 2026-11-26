# CLAUDE.md

@AGENTS.md

EGM artifact directory: .dev/docs

## Claude Code specifics

- Project skills: `/pustaka-elephant` (load context before designing),
  `/pustaka-author` (write docs pages), `/pustaka-review` (goldfish review of
  changed pages). Agent: `pustaka-goldfish` (read-only, `omitClaudeMd`).
- EGM skills are invoked as `/egm:egm-<skill>`. They set
  `disable-model-invocation`, so the `Skill` tool refuses them: when one EGM
  skill needs another, read its `SKILL.md` and follow it in the same thread.
- Reviewers must start in a fresh context: spawn them with the Agent tool and
  pass absolute paths only — never a summary of this conversation.
