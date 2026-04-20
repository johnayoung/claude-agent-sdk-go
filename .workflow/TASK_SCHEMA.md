# Task Schema v2

Phase-based task management schema. Tasks live in phase files under `tasks/active/`.

## Directory Structure

```
.workflow/
  tasks/
    active/
      01-production-readiness.json    # Phases ordered by filename prefix
      02-next-phase.json
    archive/
      2025-01-legacy.json             # Archived tasks
```

## Phase File Structure

```json
{
  "phase": {
    "id": "production-readiness",
    "name": "Production Readiness",
    "depends_on_phases": [],
    "exit_criteria": "Observability and error handling complete"
  },
  "tasks": [...]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `phase.id` | `string` | Unique phase identifier |
| `phase.name` | `string` | Human-readable name |
| `phase.depends_on_phases` | `string[]` | Phase IDs that must complete first |
| `phase.exit_criteria` | `string` | What defines phase completion |

## Task Schema

```json
{
  "id": "PRJ-0108",
  "description": "...",
  "category": "chore",
  "commit": "chore(router): add observability",
  "steps": [...],
  "acceptance_criteria": [...],
  "passing": false,
  "implementation_notes": "",
  "github_item_id": null,
  "prerequisites": [],
  "parallel_group": null,
  "priority": 1
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | Yes | Unique ID: `<PREFIX>-NNNN` format |
| `description` | `string` | Yes | Detailed description |
| `category` | `string` | Yes | `feat`, `refactor`, `docs`, `test`, `chore` |
| `commit` | `string` | Yes | Conventional commit message |
| `steps` | `string[]` | Yes | Implementation steps (min 2) |
| `acceptance_criteria` | `string[]` | Yes | Verifiable criteria |
| `passing` | `boolean` | Yes | True when criteria met |
| `implementation_notes` | `string` | No | Post-implementation notes |
| `github_item_id` | `string?` | No | GitHub Project item ID |
| `prerequisites` | `string[]` | No | Task IDs that must be `passing:true` first |
| `parallel_group` | `string?` | No | Tasks in same group can run concurrently |
| `priority` | `int` | No | Lower = higher priority (1 is highest) |

## ID Prefixes

Task ID prefixes are auto-derived from the repository name:

1. Take the repo basename (e.g., from `git rev-parse --show-toplevel`)
2. Split on `-` or `_`
3. Take the first letter of each segment, uppercased
4. Example: `my-cool-api` -> `MCA-`, `backend` -> `B-`

For multi-app monorepos, define prefixes in `.workflow/apps.json`:

```json
{
  "apps": [
    { "name": "server", "prefix": "SRV", "path": "server/" },
    { "name": "client", "prefix": "CLT", "path": "client/" },
    { "name": "cross-cutting", "prefix": "XCT", "path": null }
  ]
}
```

When `apps.json` is absent, all tasks use the single auto-derived prefix.

## Categories

| Category | Description |
|----------|-------------|
| `feat` | New feature or capability |
| `refactor` | Code restructuring without behavior change |
| `docs` | Documentation updates |
| `test` | Adding or updating tests |
| `chore` | Maintenance tasks |

## Task Selection Logic

1. Scan `active/*.json` sorted by filename
2. For each phase, check `depends_on_phases` are complete
3. Within phase, find tasks where:
   - `passing: false`
   - All `prerequisites` are `passing: true`
4. Sort by `priority` (lower first)
5. Tasks in same `parallel_group` can run concurrently
