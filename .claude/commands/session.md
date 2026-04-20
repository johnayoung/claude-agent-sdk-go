---
description: Session workflow for implementing tasks from a phase file
---

Execute this workflow immediately. Do not summarize - just do it.

You are a coding agent working on this project. Work through a phase, implementing tasks in dependency order until the phase is complete or you're blocked.

## SESSION INPUT

$ARGUMENTS

Input formats:
- **Phase ID**: `simulator-foundation` → Work through all tasks in that phase
- **Task ID**: `ABC-0160` → Start with that specific task, continue through phase
- **Phase file**: `02-simulator-foundation.json` → Work through that phase file

## STEP 1: LOAD PHASE

```bash
# List available phases
ls -la .workflow/tasks/active/*.json

# Read the target phase file
cat .workflow/tasks/active/<NN>-<phase-name>.json
```

## STEP 2: CHECK PHASE PREREQUISITES

If `depends_on_phases` is non-empty, verify those phases are complete:
- All tasks in prerequisite phases must have `passing: true`
- If not, STOP and report which prerequisite phase/tasks are incomplete

## STEP 3: SELECT NEXT TASK

Find the next actionable task using this priority:
1. **Not passing** (`passing: false`)
2. **Prerequisites met** (all tasks in `prerequisites` array have `passing: true`)
3. **Lowest priority number** (priority 1 before priority 2)

If multiple tasks qualify and share a `parallel_group`, you may work on them concurrently.

**If no tasks are actionable:**
- All tasks passing → Phase complete, report success
- Tasks remain but prerequisites unmet → Report blocked tasks and what they need

## STEP 4: IMPLEMENT TASK

Delegate the selected task to `/task-execute`, which handles the full single-task lifecycle (research, steps, verification, commit, phase file update).

```
/task-execute <task-id> <phase-file-basename>
```

Example: `/task-execute ABC-0160 02-simulator-foundation`

## STEP 5: VERIFY AND CONTINUE

After `/task-execute` returns:

1. **Re-read the phase file** to confirm the task now has `passing: true`
2. **If task is not passing** - Report the failure and decide whether to retry or move on
3. **Check phase exit criteria** - If all tasks passing and exit criteria met, phase is complete
4. **Find next task** - Return to Step 3 to select the next actionable task
5. **Report progress** - Summarize what's done, what's next

Continue until:
- Phase exit criteria met
- No more actionable tasks (all done or blocked)
- Session limit reached

## STEP 6: ARCHIVE ON PHASE COMPLETION

**When all tasks in a phase are passing and exit criteria are met**, archive both the spec and task files:

### 6.1 Archive the Task Phase File

```bash
# Move the phase file from active to archive
mv .workflow/tasks/active/<NN>-<phase-name>.json .workflow/tasks/archive/
```

### 6.2 Archive the Feature Spec

Identify the associated feature spec from the phase context and move it:

```bash
# Move the spec file from active to archive
mv .workflow/specs/<NNNNN>-FEATURE-<name>.md .workflow/specs/archive/
# Also handles legacy non-numbered specs:
# mv .workflow/specs/FEATURE-<name>.md .workflow/specs/archive/
```

**Rules:**
- Only archive when ALL tasks in the phase have `passing: true`
- If the spec file is shared across multiple phases, only archive it when ALL related phases are complete
- Verify archive directories exist before moving (`mkdir -p .workflow/specs/archive .workflow/tasks/archive`)

## SESSION COMPLETION REPORT

At session end, report phase status, completed tasks with commit hashes, remaining/blocked tasks, and next actions.

---

## CRITICAL RULES

1. **PHASE-ORIENTED** - Goal is phase completion, not just one task
