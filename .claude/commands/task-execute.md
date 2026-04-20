---
description: Execute a single task from a phase file
---

Execute this task immediately. Do not summarize - just do it.

You are a coding agent working on this project. You will implement exactly one task and exit.

## ARGUMENTS

$ARGUMENTS

Format: `<task-id> <phase-file-basename>`
Example: `ABC-0627 59-feature-verification`

## STEP 1: PARSE ARGUMENTS AND LOAD TASK

```bash
# Split arguments
TASK_ID="<first-argument>"
PHASE_BASENAME="<second-argument>"

# Load the phase file
cat .workflow/tasks/active/${PHASE_BASENAME}.json
```

Find the task matching `TASK_ID` in the `tasks` array. Handle errors:
- **Phase file not found** -- STOP and report error
- **Task not found** -- STOP and report error
- **Task already passing** -- Report "task already passing" and exit successfully

## STEP 2: VALIDATE PREREQUISITES

Check that all task `prerequisites` have `passing: true` in the same phase file.

- **Prerequisites not met** -- STOP and report which prerequisites are still failing

## STEP 3: IMPLEMENT TASK

### 3.1 Research (if needed)

| Research Needed | Scenario |
|-----------------|----------|
| Yes | Third-party API, new blockchain/protocol, unfamiliar library |
| No | Internal refactoring, tests, docs, established patterns |

### 3.2 Execute Steps

Follow the `steps` array from the task definition. Work in the appropriate directory.

### 3.3 Verify Acceptance Criteria

**Do not mark done until ALL acceptance criteria are verified.**

Auto-detect the project's verification method using this priority order:

1. **`.workflow/verify.sh`** -- If this script exists, run it. This is the project-defined verification override.
2. **Makefile with `test` target** -- Run `make test`
3. **package.json with `test` script** -- Run `npm test`
4. **go.mod present** -- Run `go test ./...`
5. **Cargo.toml present** -- Run `cargo test`
6. **pytest.ini or pyproject.toml with `[tool.pytest]`** -- Run `pytest`
7. **Nothing detected** -- Warn that no verification method was found and skip automated verification. Manually verify acceptance criteria instead.

If the project contains multiple apps (e.g., a monorepo with several subdirectories each having their own build system), run the appropriate verification for each app affected by the task's changes.

## STEP 4: COMMIT

Use the `commit` message from the task definition. Include task ID in the commit body.

## STEP 5: UPDATE PHASE FILE

After successful commit, update the task in the phase file:
- Set `passing` to `true`
- Set `implementation_notes` to a summary of what was implemented and the commit hash

Re-read the phase file, apply the update via jq or direct edit, and write it back.

## STEP 6: EXIT

Report the result and exit. Do not continue to other tasks.

```
## Task Complete: <task-id>

**Phase:** <phase-basename>
**Commit:** <commit-hash>
**Summary:** <1-2 sentence summary>
```

---

## SINGLE TASK RULE

This skill implements exactly one task and exits. Do not continue to other tasks. Do not loop. Do not check for the next task. Complete this task, update the phase file, and stop.

---

## CRITICAL RULES

1. **SINGLE TASK** - Implement exactly one task and exit. Do not continue to other tasks.
2. **RESPECT DEPENDENCIES** - Never skip prerequisites
3. **VERIFY ALL CRITERIA** - Each acceptance criterion must be explicitly checked
4. **USE TASK COMMIT MESSAGES** - From the task definition
5. **CLEAN STATE** - Leave codebase runnable after each task
6. **NO INCOMPLETE IMPLEMENTATIONS** - No stubs, TODOs, placeholders, mocks. Every line production-ready.

## IF BLOCKED

**STOP immediately. Do NOT commit incomplete code.** Exit with a clear error report.

Present:
1. **Problem Statement** - What specifically is blocking
2. **Task Context** - Which task, which step
3. **What Was Attempted** - Approaches tried
4. **Paths Forward** - 3-5 options with trade-offs
5. **Impact on Phase** - What else is blocked by this

Do not attempt workarounds that could leave partial or broken state. It is better to exit cleanly and report the blocker than to commit incomplete work.
