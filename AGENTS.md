# AGENTS.md

@shared-context.md

---

## Agent Customization System (`.agents/`)

The repository includes pre-configured skills, rules, and hooks under `.agents/`:

### Dispatch & Skills Matrix
| Task | Target Layer | Relevant Skill | Relevant Rule |
|---|---|---|---|
| Create/Scaffold DB Migration | `backend/migrations/` | `.agents/skills/create-migration` | `.agents/rules/multi-tenancy.md` |
| New Backend Feature/Resource | `backend/internal/` | `.agents/skills/new-backend-feature` | `.agents/rules/clean-architecture.md` |
| New Frontend Feature/Page | `clients/*/src/features/` | `.agents/skills/new-frontend-feature` | `.agents/rules/frontend-conventions.md` |

Before using any of the above, check it against the active cycle's scope in
`plan/cycles/` — a skill/rule existing doesn't mean its target layer is in scope yet.

### Automated Lifecycle Hooks (`.agents/hooks.json`)
- **PreToolUse**:
  - `check-migration.sh`: Protects committed migration files from accidental modification.
  - `tenant-scope-reminder.sh`: Enforces `tenant_id` scoping reminders when editing repositories or handlers.
- **PostToolUse**:
  - `backend-check.sh`: Runs `gofmt` and `go vet` on Go edits.
  - `frontend-check.sh`: Runs type checks on modified TypeScript files.

---

## Docs

In addition to the shared documentation map above:
- `CLAUDE.md` — equivalent guidance for Claude Code sessions; imports the same
  `shared-context.md`, so only its tool-specific dispatch section can drift
  from this file.
