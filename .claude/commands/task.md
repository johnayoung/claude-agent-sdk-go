---
description: Break down a request into a structured task definition
---

Analyze the request, assess complexity, and generate an appropriate task structure following `.workflow/TASK_SCHEMA.md`.

## INPUT

$ARGUMENTS

Input formats:
- **Spec reference**: `NNNNN-FEATURE-<name>` or `FEATURE-<name>` (legacy) -> Load `.workflow/specs/NNNNN-FEATURE-<name>.md` and generate tasks from it
- **Free text**: A description of work to be done -> Generate tasks directly

When a spec reference is provided, read the spec file first to understand the full requirements before generating tasks.

## STEP 1: UNDERSTAND THE CODEBASE

Before generating tasks, explore what exists:

```bash
# Discover project structure
ls -la
cat README.md 2>/dev/null | head -100

# Discover apps/modules -- look for build system markers
for dir in */; do
  for marker in go.mod package.json Cargo.toml pyproject.toml setup.py build.gradle pom.xml mix.exs Gemfile; do
    if [ -f "$dir$marker" ]; then
      echo "App: $dir ($marker)"
    fi
  done
done
for marker in go.mod package.json Cargo.toml pyproject.toml; do
  [ -f "$marker" ] && echo "Root app: $marker"
done
```

- Identify which apps exist and their types
- Find relevant existing code patterns

## STEP 2: ASSESS REQUEST COMPLEXITY

Classify the request to determine task structure:

| Complexity | Indicators | Result |
|------------|------------|--------|
| **Simple** | Single endpoint, bug fix, small feature | 1 task |
| **Medium** | Feature with multiple components | 2-4 tasks, existing phase |
| **Complex** | New system, cross-cutting concern | Multiple tasks, likely new phase |

**Assessment checklist:**
1. How many distinct components need modification?
2. Are there multiple independent pieces of work?
3. Do any pieces depend on others being done first?
4. Does this fit an existing phase's scope?

## STEP 3: REVIEW EXISTING PHASES

Scan active phases to determine placement:

```bash
# List active phases
ls .workflow/tasks/active/*.json 2>/dev/null

# Find highest task ID across all phases
grep -r '"id":' .workflow/tasks/active/ 2>/dev/null | grep -oP '"[A-Z]+-\d+"' | sort | tail -5

# Search for related tasks
grep -ri "<relevant-keywords>" .workflow/tasks/active/ 2>/dev/null
```

**Phase placement decision:**
- **Add to existing phase** if:
  - Request relates to phase's exit criteria
  - Existing tasks cover similar domain
  - Work scope fits within phase goal
- **Create new phase** if:
  - Request introduces a new major capability
  - No existing phase covers this domain
  - Work is large enough to warrant its own phase

## STEP 4: GENERATE TASK BREAKDOWN

### Auto-derive Task ID Prefix

Determine the prefix for task IDs using this logic:

```bash
# 1. Check for explicit app prefix overrides
if [ -f .workflow/apps.json ]; then
  # apps.json format: {"apps": [{"name": "backend", "prefix": "BE"}, {"name": "frontend", "prefix": "FE"}]}
  # Use the appropriate prefix based on which app the task affects
  # For cross-cutting tasks spanning multiple apps, use "XCT"
  cat .workflow/apps.json
else
  # 2. Auto-derive from repo name
  # Take the repo basename, split on dashes/underscores, uppercase first letter of each segment
  # Examples:
  #   my-api         -> MA
  #   cool-web-app   -> CWA
  #   backend        -> B
  #   data-pipeline  -> DP
  REPO_NAME=$(basename "$(git rev-parse --show-toplevel)")
  PREFIX=$(echo "$REPO_NAME" | sed 's/[-_]/ /g' | awk '{for(i=1;i<=NF;i++) printf toupper(substr($i,1,1))}')
  echo "Auto-derived prefix: ${PREFIX}-"
fi
```

**`.workflow/apps.json` format (optional, for multi-app repos):**
```json
{
  "apps": [
    {"name": "backend", "prefix": "BE", "path": "backend/"},
    {"name": "frontend", "prefix": "FE", "path": "frontend/"},
    {"name": "docs", "prefix": "DOC", "path": "docs/"}
  ],
  "cross_cutting_prefix": "XCT"
}
```

If `apps.json` exists, use the prefix matching the app affected by the task. For tasks spanning multiple apps, use the `cross_cutting_prefix` (defaults to `"XCT"` if not specified).

### For Simple requests (1 task)
Generate a single task definition.

### For Medium/Complex requests (multiple tasks)
Break down following these rules:

**Sizing:**
- Keep tasks focused and independently completable
- If a task covers too many concerns, break it down further

**Dependencies:**
- Identify which tasks must complete before others
- Use `prerequisites` array to enforce order
- Tasks with no dependencies can run in parallel

**Parallel groups:**
- Assign same `parallel_group` to tasks that can run concurrently
- Only group tasks at the same dependency level

**Priority:**
- Lower number = higher priority
- Tasks with prerequisites get equal or lower priority than their dependencies

### Task JSON Format

```json
{
  "id": "<PREFIX>-NNNN",
  "description": "<detailed description>",
  "category": "<feat|refactor|docs|test|chore>",
  "commit": "<category>(<scope>): <short description>",
  "steps": [
    "Step 1: <actionable step>",
    "Step 2: <actionable step>"
  ],
  "acceptance_criteria": [
    "Criterion 1: <testable outcome>",
    "Criterion 2: <testable outcome>"
  ],
  "passing": false,
  "implementation_notes": "",
  "github_item_id": null,
  "prerequisites": [],
  "parallel_group": null,
  "priority": 1
}
```

### Categories

| Category | Commit Prefix | Use For |
|----------|---------------|---------|
| feat | feat: | New functionality |
| refactor | refactor: | Code restructuring |
| docs | docs: | Documentation |
| test | test: | Test additions |
| chore | chore: | Maintenance |

### Phase Numbering

When creating a new phase file, auto-detect the next available number:

```bash
# Find highest phase number across active and archived phases
HIGHEST=$(ls .workflow/tasks/active/[0-9]*.json .workflow/tasks/archive/[0-9]*.json 2>/dev/null \
  | sed 's/.*\/\([0-9]*\)-.*/\1/' | sort -n | tail -1)
NEXT=$((${HIGHEST:-0} + 1))
echo "Next phase number: $NEXT"
```

## STEP 5: PRESENT PROPOSAL

**Do not write anything yet.** Present the proposal for user approval:

```
## Proposal

**Complexity:** <Simple|Medium|Complex>
**Phase:** Adding to existing `NN-phase-name.json`
  OR
**Phase:** Creating new `NN-phase-name.json`
  - id: <phase-id>
  - name: <Phase Name>
  - exit_criteria: <what defines completion>

### Tasks (<count>):

1. **<ID>**: <title> (priority <N>)
   - Category: <category>
   - Steps: <count> steps
   - Criteria: <count> criteria

2. **<ID>**: <title> (priority <N>, requires <dependency-ID>)
   - Category: <category>
   - Steps: <count> steps
   - Criteria: <count> criteria

### Task Details

<Full JSON for each task>

---

**Proceed?** Reply to confirm and I'll write to the phase file.
```

## STEP 6: WRITE TASKS (after confirmation)

After user confirms:

### 6.1 Write Tasks to Phase File

**If adding to existing phase:**
1. Read the phase file
2. Append new tasks to the `tasks` array
3. Write the updated file

**If creating new phase:**
1. Create new file with `NN-phase-name.json` naming
2. Use next available prefix number
3. Include phase metadata and tasks array:

```json
{
  "phase": {
    "id": "<phase-id>",
    "name": "<Phase Name>",
    "depends_on_phases": [],
    "exit_criteria": "<what defines completion>"
  },
  "tasks": [...]
}
```

### 6.2 Report Success

Report:
- Tasks written to phase file
- Task IDs and their dependencies

## RULES

1. **Never create overly large tasks** - break them down
2. **Steps must be actionable** - concrete actions, not vague goals
3. **Acceptance criteria must be verifiable** - testable outcomes
4. **Use conventional commits** - `category(scope): description`
5. **IDs use 4-digit numbers** - e.g., `MA-0042`, not `MA-42`
6. **Always include github_item_id: null**
7. **Always present proposal before writing** - user must confirm
8. **Prerequisites reference task IDs** - e.g., `["MA-0109"]`
