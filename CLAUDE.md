# CLAUDE.md

@shared-context.md

## Dispatch

| Task | Agent | Skill |
|---|---|---|
| Go backend (`backend/`) | `backend-agent` | `create-migration`, `new-backend-feature` |
| React/TS (`clients/*`, `packages/*`) | `frontend-agent` | `new-frontend-feature`, `design-system` |

Rules: `.claude/agents/{backend,frontend}-agent.md`. Don't ad-hoc code in these areas —
dispatch instead. Cross-layer feature: `backend-agent` first (API), then `frontend-agent`.

## Docs

In addition to the shared documentation map above:
- `AGENTS.md` — equivalent guidance for other AI tools (`.agents/` skills/hooks/rules);
  imports the same `shared-context.md`, so only its tool-specific dispatch section
  can drift from this file.
