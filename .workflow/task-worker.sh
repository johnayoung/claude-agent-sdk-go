#!/usr/bin/env bash
# task-worker.sh - Autonomous worker loop that picks up individual tasks from
# .workflow/tasks/active/ phase files and runs /task-execute on each until killed.
#
# Pattern: infinite loop, commit-keyed logs, graceful signal handling.
# Based on: https://www.anthropic.com/engineering/building-c-compiler
#
# Usage:
#   ./task-worker.sh [--log-dir <path>]

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
ACTIVE_DIR="$REPO_ROOT/.workflow/tasks/active"
LOG_DIR="$REPO_ROOT/logs/worker"
SHUTDOWN=0

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --log-dir)
      LOG_DIR="$2"
      shift 2
      ;;
    --log-dir=*)
      LOG_DIR="${1#*=}"
      shift
      ;;
    -h|--help)
      echo "Usage: task-worker.sh [--log-dir <path>]"
      echo ""
      echo "Autonomous worker loop that processes individual tasks from"
      echo ".workflow/tasks/active/ phase files using /task-execute."
      echo ""
      echo "Options:"
      echo "  --log-dir <path>  Log directory (default: logs/worker/)"
      echo "  -h, --help        Show this help"
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

# ---------------------------------------------------------------------------
# Prerequisite checks
# ---------------------------------------------------------------------------
if ! command -v git &>/dev/null; then
  echo "ERROR: git not found in PATH." >&2
  exit 1
fi

if ! command -v claude &>/dev/null; then
  echo "ERROR: claude CLI not found in PATH." >&2
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo "ERROR: jq not found in PATH." >&2
  exit 1
fi

if [[ ! -d "$ACTIVE_DIR" ]]; then
  echo "ERROR: Active tasks directory not found: $ACTIVE_DIR" >&2
  exit 1
fi

if ! mkdir -p "$LOG_DIR"; then
  echo "ERROR: Could not create log directory: $LOG_DIR" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Signal handling
# ---------------------------------------------------------------------------
trap 'SHUTDOWN=1; echo "[worker] Shutdown requested, waiting for current session..." >&2' SIGINT SIGTERM

# ---------------------------------------------------------------------------
# phase_is_complete <phase-file>
# Returns 0 if all tasks in the phase have passing:true, 1 otherwise.
# ---------------------------------------------------------------------------
phase_is_complete() {
  local phase_file="$1"
  local incomplete
  incomplete=$(jq '[.tasks[] | select(.passing == false)] | length' "$phase_file" 2>/dev/null) || return 1
  [[ "$incomplete" -eq 0 ]]
}

# ---------------------------------------------------------------------------
# phase_deps_met <phase-file>
# Returns 0 if all depends_on_phases are satisfied (all tasks passing in
# those referenced phases). Returns 1 if any dependency is unmet.
# ---------------------------------------------------------------------------
phase_deps_met() {
  local phase_file="$1"
  local deps
  deps=$(jq -r '.phase.depends_on_phases[]?' "$phase_file" 2>/dev/null) || return 0

  if [[ -z "$deps" ]]; then
    return 0
  fi

  while IFS= read -r dep_id; do
    # Find the phase file matching this dependency ID in active or archive
    local found=0
    for candidate in "$ACTIVE_DIR"/*.json "$REPO_ROOT/.workflow/tasks/archive"/*.json; do
      [[ -f "$candidate" ]] || continue
      local cid
      cid=$(jq -r '.phase.id' "$candidate" 2>/dev/null) || continue
      if [[ "$cid" == "$dep_id" ]]; then
        found=1
        if ! phase_is_complete "$candidate"; then
          return 1
        fi
        break
      fi
    done
    # If dependency phase not found at all, consider it unmet
    if [[ "$found" -eq 0 ]]; then
      return 1
    fi
  done <<< "$deps"

  return 0
}

# ---------------------------------------------------------------------------
# next_task
# Scans active/*.json sorted by filename, checks phase dependencies, then
# finds the first task with passing:false whose prerequisites are all
# passing:true, sorted by priority (lowest first).
# Prints "<task-id> <phase-basename>" to stdout, or nothing if no eligible
# task is found.
# ---------------------------------------------------------------------------
next_task() {
  for phase_file in "$ACTIVE_DIR"/*.json; do
    [[ -f "$phase_file" ]] || continue

    # Validate JSON
    if ! jq empty "$phase_file" 2>/dev/null; then
      echo "[worker] WARNING: Invalid JSON in $(basename "$phase_file"), skipping" >&2
      continue
    fi

    # Skip phases where all tasks are already passing
    if phase_is_complete "$phase_file"; then
      continue
    fi

    # Skip phases whose dependencies are not met
    if ! phase_deps_met "$phase_file"; then
      continue
    fi

    # Find the first eligible task: passing==false, all prerequisites passing,
    # sorted by priority (lowest first).
    local result
    result=$(jq -r '
      # Build a map of task passing status for prerequisite lookup
      (.tasks | map({(.id): .passing}) | add // {}) as $status |
      # Filter to tasks that are not passing and have all prerequisites met
      [.tasks[] |
        select(.passing == false) |
        select(
          .prerequisites == null or
          .prerequisites == [] or
          ([ .prerequisites[] | $status[.] // true | . == true ] | all)
        )
      ] |
      # Sort by priority (lowest first)
      sort_by(.priority) |
      # Take the first one
      first |
      if . then .id else empty end
    ' "$phase_file" 2>/dev/null)

    if [[ -n "$result" ]]; then
      local basename
      basename=$(basename "$phase_file" .json)
      echo "$result $basename"
      return
    fi
  done
}

# ---------------------------------------------------------------------------
# log_path <task-id>
# Returns a unique log file path based on the task ID and timestamp.
# ---------------------------------------------------------------------------
log_path() {
  local task_id="$1"
  local commit
  commit=$(git -C "$REPO_ROOT" rev-parse --short=6 HEAD 2>/dev/null || echo "unknown")
  echo "${LOG_DIR}/${task_id}_${commit}_$(date +%Y%m%dT%H%M%S).log"
}

# ---------------------------------------------------------------------------
# task_is_passing <task-id> <phase-file>
# Returns 0 if the specified task has passing:true in the phase file.
# ---------------------------------------------------------------------------
task_is_passing() {
  local task_id="$1"
  local phase_file="$2"
  local passing
  passing=$(jq -r --arg id "$task_id" '.tasks[] | select(.id == $id) | .passing' "$phase_file" 2>/dev/null) || return 1
  [[ "$passing" == "true" ]]
}

# ---------------------------------------------------------------------------
# phase_has_running_agent <phase-basename>
# Returns 0 if a claude process is currently running task-execute for this
# phase (i.e. another worker instance is mid-session on a task in this phase).
# ---------------------------------------------------------------------------
phase_has_running_agent() {
  local phase_basename="$1"
  # Match claude processes whose arguments contain task-execute and the phase name.
  # This catches: claude ... -p "/task-execute <task-id> <phase-basename>"
  pgrep -f "claude.*task-execute.*${phase_basename}" &>/dev/null
}

# ---------------------------------------------------------------------------
# cleanup_completed_phases
# Scans active/*.json for phases where every task has passing:true. If no
# claude agent is currently running against that phase, moves the file to
# archive/. Safe to call repeatedly -- it's a no-op when nothing qualifies.
# ---------------------------------------------------------------------------
cleanup_completed_phases() {
  local archive_dir="$REPO_ROOT/.workflow/tasks/archive"
  mkdir -p "$archive_dir"

  for phase_file in "$ACTIVE_DIR"/*.json; do
    [[ -f "$phase_file" ]] || continue

    if ! phase_is_complete "$phase_file"; then
      continue
    fi

    local bn
    bn=$(basename "$phase_file" .json)

    if phase_has_running_agent "$bn"; then
      echo "[worker] Phase $bn complete but agent still running, skipping archive" >&2
      continue
    fi

    mv "$phase_file" "$archive_dir/"
    echo "[worker] Archived completed phase: $bn" >&2
  done
}

# ---------------------------------------------------------------------------
# Main loop
# ---------------------------------------------------------------------------
echo "[worker] Task worker started" >&2
echo "[worker] Active dir: $ACTIVE_DIR" >&2
echo "[worker] Log dir:    $LOG_DIR" >&2
echo "[worker] PID:        $$" >&2
echo "" >&2

while true; do
  if [[ "$SHUTDOWN" -eq 1 ]]; then
    echo "[worker] Shutting down." >&2
    exit 0
  fi

  RESULT=$(next_task)

  if [[ -z "$RESULT" ]]; then
    cleanup_completed_phases
    echo "[worker] No eligible tasks. Sleeping 60s..." >&2
    # Sleep in a loop so we can check SHUTDOWN flag more frequently
    for (( i=0; i<60; i++ )); do
      if [[ "$SHUTDOWN" -eq 1 ]]; then
        echo "[worker] Shutting down." >&2
        exit 0
      fi
      sleep 1
    done
    continue
  fi

  TASK_ID="${RESULT%% *}"
  PHASE_BASENAME="${RESULT#* }"
  LOGFILE=$(log_path "$TASK_ID")

  echo "[worker] Selected task:  $TASK_ID (phase: $PHASE_BASENAME)" >&2
  echo "[worker] Session log:    $LOGFILE" >&2
  echo "[worker] Session start:  $(date -Iseconds)" >&2

  # Run task-execute -- do not exit on non-zero (set +e)
  # --output-format stream-json emits every event (tool calls, results,
  # thinking, text) as newline-delimited JSON for full observability.
  set +e
  claude --dangerously-skip-permissions \
    -p "/task-execute $TASK_ID $PHASE_BASENAME" \
    --model sonnet \
    --output-format stream-json \
    --verbose \
    &> "$LOGFILE"
  EXIT_CODE=$?
  set -e

  echo "[worker] Session end:    $(date -Iseconds) (exit=$EXIT_CODE)" >&2

  if [[ "$SHUTDOWN" -eq 1 ]]; then
    echo "[worker] Shutting down." >&2
    exit 0
  fi

  # Re-read the phase file and check if the task is now passing
  PHASE_FILE="$ACTIVE_DIR/${PHASE_BASENAME}.json"
  if task_is_passing "$TASK_ID" "$PHASE_FILE"; then
    echo "[worker] Task $TASK_ID passed. Selecting next task..." >&2
    cleanup_completed_phases
    echo "" >&2
  else
    echo "[worker] Task $TASK_ID still failing. Retrying immediately..." >&2
    echo "" >&2
    # Loop continues, next_task will return the same task since it's still failing
  fi
done
