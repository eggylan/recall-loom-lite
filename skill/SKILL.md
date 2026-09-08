---
name: rll
description: Use when continuing a project, restoring project context, maintaining file-based project memory, recording milestone progress, or preparing a session handoff. Drives the rll CLI (RecallLoom Lite) for lightweight zero-state project memory.
---

# RecallLoom Lite (rll)

rll keeps project memory in plain Markdown files under `.rll/` so any session,
model, or tool can pick up where the last one stopped.

**Requirement:** the `rll` binary must be on PATH. If missing, tell the user to
install it from the project's GitHub Releases and stop — never hand-build sidecar files.

## Cold start

Run `rll resume` first. It prints the entire memory in one shot:
update protocol (if any), context brief, rolling summary, and the latest daily
log. Read it, then act. Do not read the files again — resume already gave you everything.

## Command cheatsheet

| Command | What it does |
|---|---|
| `rll init` | Create the `.rll/` skeleton (refuse if it exists) |
| `rll resume` | Print the full memory context for a cold start |
| `rll status` | One-screen overview (file sizes, latest log, entry counts) |
| `rll log append --title "..."` | Append a milestone entry to today's daily log (body via stdin) |
| `rll write brief / summary / protocol` | Overwrite a managed document (body via stdin) |
| `rll query <text...>` | Case-insensitive substring search over summary + active logs (`--all` includes archive) |
| `rll validate` | Check file formats only (5 structural checks) |
| `rll archive --before YYYY-MM-DD` | Preview moving old logs to `.rll/archive/` (`--apply` to move) |

All write commands accept `--dry-run` to preview rendered output and `--file PATH`
instead of stdin. Timestamps and dates default to local time; override with
`--date`/`--time` when backfilling.

## Deciding where a fact goes

Ask: what kind of change is this?

| Content | Target | Command |
|---|---|---|
| Durable rule, boundary, source of truth, project identity | `context_brief.md` | `rll write brief` |
| What is true right now: phase, active risks, next step | `rolling_summary.md` | `rll write summary` |
| Completed milestone, confirmed decision, validation result | daily log | `rll log append --title "..."` |
| Project-local read/write workflow override | `update_protocol.md` | `rll write protocol` |

One fact goes to exactly one place. When an event spans layers, split it:
decision → brief (if durable) or log (as evidence), current picture → summary.

## When NOT to write

Do not touch memory files when:

- you only ran `rll resume` or answered a question
- you explored without reaching a stable conclusion
- the change is wording-only with no new durable fact
- the discussion is still unstable — wait for it to settle

no_write is a successful outcome, not a failure.

## First attach

If `rll resume` fails with "no .rll directory found", ask the user whether to
initialize. If confirmed: run `rll init`, then interview the user (what is this
project, current phase, source of truth, constraints) and fill the skeleton:

```bash
cat <<'EOF' | rll write brief
# Context Brief

## What this project is
...
EOF
```

Body must be pure Markdown WITHOUT frontmatter — `rll write` adds it.

## Zero-state philosophy

There is no state file, no locking, no receipt. Files are the only truth.
Users may hand-edit any file at any time; the only guard is `rll validate`
checking format. If files look wrong, fix them with `rll write` or directly,
then run `rll validate`.
