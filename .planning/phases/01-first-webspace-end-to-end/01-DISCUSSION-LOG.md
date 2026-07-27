# Phase 1: First Webspace, End to End - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-27
**Phase:** 1-First Webspace, End to End
**Areas discussed:** Webspace config shape

---

## Webspace config shape

### Config file format

| Option | Description | Selected |
|--------|-------------|----------|
| TOML (Recommended) | Go-ecosystem favourite, forgiving to hand-edit, no YAML whitespace/quoting footguns; nested tables map cleanly to webspace × source structures | ✓ |
| YAML | Most familiar config idiom generally (docker-compose, Home Assistant); whitespace-sensitive | |
| JSON | Zero-dependency parsing, machine-writable; no comments | |
| You decide | Claude picks | |

**User's choice:** TOML

### Keyword → source mapping

| Option | Description | Selected |
|--------|-------------|----------|
| Keyword + overrides (Recommended) | One default keyword per webspace plus optional per-source overrides | |
| Explicit per-source always | Every webspace lists its match term for each source explicitly | |
| Single keyword only | One keyword, matched identically everywhere | |
| Other (free text) | Hybrid: a LIST of keywords per webspace, all matched in all plugins | ✓ |

**User's choice:** Free-text hybrid — each webspace has a list of keywords (example: `"house-move, House"`), and every plugin matches all of them against its native categorization (paperless tags for both, IMAP folders for both). No per-source syntax.

### Match strictness

| Option | Description | Selected |
|--------|-------------|----------|
| Exact, case-insensitive (Recommended) | 'house' matches 'House' but not 'Household'; variants added explicitly to the keyword list | ✓ |
| Exact, case-sensitive | 'House' matches only 'House' | |
| Substring/prefix match | 'house' matches 'Household' and 'house-move' | |

**User's choice:** Exact, case-insensitive

### Config location & secrets

| Option | Description | Selected |
|--------|-------------|----------|
| One file, XDG path (Recommended) | Everything incl. tokens in ~/.config/webspaces/config.toml | |
| Split: sources vs webspaces | config.toml for kernel + sources, webspaces.toml for the keyword map | |
| One file, env-var secrets | Single config.toml; tokens from environment variables / ${VAR} interpolation | ✓ |

**User's choice:** One file at `~/.config/webspaces/config.toml`, secrets via env vars / `${VAR}` interpolation

---

## Claude's Discretion

User selected only the config area for discussion; the following areas were offered and left to Claude during research/planning:
- Stream & detail presentation (item card contents, ordering timestamp, detail-pane rendering for paperless docs)
- Sync trigger for Phase 1 (startup vs interval vs manual; what is stored as the local preview)
- Running the service (command shape, plugins directory, listen port)

## Deferred Ideas

None — discussion stayed within phase scope.
