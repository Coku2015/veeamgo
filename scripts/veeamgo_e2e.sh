#!/usr/bin/env bash
set -uo pipefail

LOG_FILE=${LOG_FILE:-"$(pwd)/veeamgo-test-$(date +'%Y%m%d-%H%M%S').log"}
RUN_MUTATING=${RUN_MUTATING_COMMANDS:-0}
OVERALL_STATUS=0
LAST_OUTPUT=""

mkdir -p "$(dirname "$LOG_FILE")"
: >"$LOG_FILE"

log() {
  local message="[$(date +'%Y-%m-%d %H:%M:%S')] $*"
  echo "$message"
  echo "$message" >>"$LOG_FILE"
}

append_output() {
  local desc=$1
  local payload=$2
  {
    echo "----- output begin: $desc -----"
    printf '%s\n' "$payload"
    echo "----- output end: $desc -----"
  } >>"$LOG_FILE"
}

run_cmd() {
  local desc=$1
  shift
  log "RUN: $desc"
  log "CMD: $*"
  local output
  if output=$("$@" 2>&1); then
    log "STATUS: success"
    append_output "$desc" "$output"
    LAST_OUTPUT="$output"
    return 0
  else
    local status=$?
    log "STATUS: failure (exit $status)"
    append_output "$desc" "$output"
    LAST_OUTPUT="$output"
    OVERALL_STATUS=1
    return $status
  fi
}

run_expect_fail() {
  local desc=$1
  shift
  log "RUN: $desc (expecting failure)"
  log "CMD: $*"
  local output
  if output=$("$@" 2>&1); then
    log "STATUS: unexpected success"
    append_output "$desc" "$output"
    LAST_OUTPUT="$output"
    OVERALL_STATUS=1
    return 1
  else
    local status=$?
    log "STATUS: expected failure (exit $status)"
    append_output "$desc" "$output"
    LAST_OUTPUT="$output"
    return 0
  fi
}

first_json_field() {
  local json=$1
  local primary=$2
  local fallback=${3:-}
  jq -r --arg primary "$primary" --arg fallback "$fallback" '
    def first_entry:
      if type == "array" then (.[0] // empty)
      elif type == "object" and has("data") then (.data | first_entry)
      elif type == "object" and has("records") then (.records | first_entry)
      else .
      end;
    first_entry
    | if type == "object" then
        (.[$primary] // (if $fallback == "" then empty else .[$fallback] end) // empty)
      else empty end
  ' <<<"$json" 2>/dev/null
}

trim() {
  sed -e 's/^\s*//' -e 's/\s*$//' <<<"$1"
}

extract_tab_field() {
  local text=$1
  local field_index=${2:-1}
  local line
  line=$(printf '%s\n' "$text" | sed -n '2p')
  if [[ -z $line ]]; then
    echo ""
    return
  fi
  printf '%s\n' "$line" | awk -F"[[:space:]]{2,}" -v idx="$field_index" '{
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", $0);
    if (NF >= idx) {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", $idx);
      print $idx;
    }
  }'
}

section() {
  log ""
  log "===== $* ====="
}

for bin in veeamgo jq; do
  if ! command -v "$bin" >/dev/null 2>&1; then
    log "ERROR: required dependency '$bin' not found in PATH"
    exit 2
  fi
done

log "Log file: $LOG_FILE"
log "Mutating operations enabled: $RUN_MUTATING"
log "veeamgo binary: $(command -v veeamgo)"
log "jq binary: $(command -v jq)"
log "Host: $(uname -a)"

run_cmd "veeamgo version" veeamgo --version
run_cmd "veeamgo help" veeamgo --help

#############################
# Discover default targets #
#############################
DEFAULT_REPO_NAME=""
DEFAULT_PROXY_NAME=""
DEFAULT_JOB_NAME=""
DEFAULT_REPLICA_JOB_NAME=""
DEFAULT_VIRTUAL_HOST=""
DEFAULT_PROTECTION_GROUP=""
DEFAULT_PROTECTION_AGENT=""
DEFAULT_SCALEOUT_REPO_NAME=""
DEFAULT_WAN_ACCELERATOR_NAME=""
PROTECTION_AGENT_TABLE_DONE=0

if repo_json=$(veeamgo get repository --limit 5 --output json 2>/dev/null || true); then
  DEFAULT_REPO_NAME=$(first_json_field "$repo_json" "name" "Name")
fi

if proxy_json=$(veeamgo get proxy --limit 5 --output json 2>/dev/null || true); then
  DEFAULT_PROXY_NAME=$(first_json_field "$proxy_json" "name" "Name")
fi

if job_json=$(veeamgo get job --limit 50 --output json 2>/dev/null || true); then
  DEFAULT_JOB_NAME=$(jq -r 'def rows: if type=="array" then . else (.data? // []) end; rows | .[0]?.name // empty' <<<"$job_json")
  DEFAULT_REPLICA_JOB_NAME=$(jq -r 'def rows: if type=="array" then . else (.data? // []) end; rows | map(select((.type // "") | test("Replica"; "i")))[0]?.name // empty' <<<"$job_json")
fi

if sobr_json=$(veeamgo get sobr --limit 5 --output json 2>/dev/null || true); then
  DEFAULT_SCALEOUT_REPO_NAME=$(first_json_field "$sobr_json" "name" "Name")
fi

if wan_json=$(veeamgo get wanaccelerator --limit 5 --output json 2>/dev/null || true); then
  DEFAULT_WAN_ACCELERATOR_NAME=$(first_json_field "$wan_json" "name" "Name")
fi

##################################
# Phase 1: Table outputs
##################################
section "Table Phase"

run_cmd "get session (table)" veeamgo get session --limit 20
SESSION_ID_TABLE=$(extract_tab_field "$LAST_OUTPUT" 8)
if [[ -n "$SESSION_ID_TABLE" ]]; then
  run_cmd "describe session $SESSION_ID_TABLE (table)" veeamgo describe session --id "$SESSION_ID_TABLE"
  run_cmd "get session logs $SESSION_ID_TABLE (table)" veeamgo get session logs --id "$SESSION_ID_TABLE"
fi
run_cmd "get server info (table)" veeamgo get server info
run_cmd "get server time (table)" veeamgo get server time
run_cmd "get managedserver (table)" veeamgo get managedserver --limit 20

run_cmd "get repository (table)" veeamgo get repository --limit 20
if [[ -z "$DEFAULT_REPO_NAME" ]]; then
  DEFAULT_REPO_NAME=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_REPO_NAME" ]]; then
  run_cmd "describe repository $DEFAULT_REPO_NAME (table)" veeamgo describe repository --name "$DEFAULT_REPO_NAME"
fi

run_cmd "get sobr (table)" veeamgo get sobr --limit 20
if [[ -z "$DEFAULT_SCALEOUT_REPO_NAME" ]]; then
  DEFAULT_SCALEOUT_REPO_NAME=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_SCALEOUT_REPO_NAME" ]]; then
  run_cmd "describe sobr $DEFAULT_SCALEOUT_REPO_NAME (table)" veeamgo describe sobr --name "$DEFAULT_SCALEOUT_REPO_NAME"
fi

run_cmd "get wanaccelerator (table)" veeamgo get wanaccelerator --limit 20
if [[ -z "$DEFAULT_WAN_ACCELERATOR_NAME" ]]; then
  DEFAULT_WAN_ACCELERATOR_NAME=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_WAN_ACCELERATOR_NAME" ]]; then
  run_cmd "describe wanaccelerator $DEFAULT_WAN_ACCELERATOR_NAME (table)" veeamgo describe wanaccelerator --name "$DEFAULT_WAN_ACCELERATOR_NAME"
fi

run_cmd "get proxy (table)" veeamgo get proxy --limit 50
if [[ -z "$DEFAULT_PROXY_NAME" ]]; then
  DEFAULT_PROXY_NAME=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_PROXY_NAME" ]]; then
  run_cmd "describe proxy $DEFAULT_PROXY_NAME (table)" veeamgo describe proxy --name "$DEFAULT_PROXY_NAME"
fi

run_cmd "get job (table)" veeamgo get job --limit 50
if [[ -z "$DEFAULT_JOB_NAME" ]]; then
  DEFAULT_JOB_NAME=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_JOB_NAME" ]]; then
  run_cmd "describe job $DEFAULT_JOB_NAME (table)" veeamgo describe job "$DEFAULT_JOB_NAME"
fi

if [[ -n "$DEFAULT_JOB_NAME" ]]; then
  run_cmd "get job history for $DEFAULT_JOB_NAME (table)" veeamgo get job history --name "$DEFAULT_JOB_NAME" --limit 20
  run_cmd "get restorepoint for $DEFAULT_JOB_NAME (table)" veeamgo get restorepoint --name "$DEFAULT_JOB_NAME" --limit 10
  run_cmd "get backup files for $DEFAULT_JOB_NAME (table)" veeamgo get backup files --name "$DEFAULT_JOB_NAME" --limit 10
  run_cmd "get backup objects for $DEFAULT_JOB_NAME (table)" veeamgo get objectsinbackup --name "$DEFAULT_JOB_NAME"
fi

run_cmd "get task (table)" veeamgo get task --limit 20
TASK_ID_TABLE=$(extract_tab_field "$LAST_OUTPUT" 10)
if [[ -n "$TASK_ID_TABLE" ]]; then
  run_cmd "describe task $TASK_ID_TABLE (table)" veeamgo describe task --id "$TASK_ID_TABLE"
  run_cmd "get task logs $TASK_ID_TABLE (table)" veeamgo get task logs --id "$TASK_ID_TABLE"
fi

if [[ -z "$DEFAULT_REPLICA_JOB_NAME" ]]; then
  # Attempt to derive from the last job listing (type column is 2nd field)
  DEFAULT_REPLICA_JOB_NAME=$(printf '%s\n' "$LAST_OUTPUT" | awk 'NR>1 && tolower($2) ~ /replica/ {gsub(/^\s+|\s+$/, "", $1); print $1; exit}')
fi
if [[ -n "$DEFAULT_REPLICA_JOB_NAME" ]]; then
  run_cmd "get replica restore points for $DEFAULT_REPLICA_JOB_NAME (table)" veeamgo get replica --name "$DEFAULT_REPLICA_JOB_NAME" --limit 10
fi

run_cmd "get inventory virtualinfra (table)" veeamgo get inventory virtualinfra
if [[ -z "$DEFAULT_VIRTUAL_HOST" ]]; then
  DEFAULT_VIRTUAL_HOST=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_VIRTUAL_HOST" ]]; then
  run_cmd "describe inventory virtualinfra $DEFAULT_VIRTUAL_HOST (table)" veeamgo describe inventory virtualinfra --name "$DEFAULT_VIRTUAL_HOST"
  run_cmd "get inventory virtualinfra objects for $DEFAULT_VIRTUAL_HOST (table)" veeamgo get inventory virtualinfra objects --name "$DEFAULT_VIRTUAL_HOST" --limit 10
fi

run_cmd "get trafficrule (table)" veeamgo get trafficrule
run_cmd "get exclusionvm (table)" veeamgo get exclusionvm --limit 10
run_cmd "get license summary (table)" veeamgo get license
run_cmd "describe configurationbackup (table)" veeamgo describe configurationbackup
run_cmd "get generaloption (table)" veeamgo get generaloption
run_cmd "get malwaredetectionevent (table)" veeamgo get malwaredetectionevent --limit 20
run_cmd "get yararule (table)" veeamgo get yararule
run_cmd "describe security analyzer schedule (table)" veeamgo describe security analyzer schedule
run_cmd "get security analyzer send-results (table)" veeamgo get security analyzer send-results
run_cmd "get inventory protectiongroup (table)" veeamgo get inventory protectiongroup --limit 10
if [[ -z "$DEFAULT_PROTECTION_GROUP" ]]; then
  DEFAULT_PROTECTION_GROUP=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_PROTECTION_GROUP" ]]; then
  run_cmd "get inventory protectiongroup agents $DEFAULT_PROTECTION_GROUP (table)" veeamgo get inventory protectiongroup agents --name "$DEFAULT_PROTECTION_GROUP" --limit 10
  if [[ -z "$DEFAULT_PROTECTION_AGENT" ]]; then
    DEFAULT_PROTECTION_AGENT=$(extract_tab_field "$LAST_OUTPUT" 2)
  fi
  if [[ -n "$DEFAULT_PROTECTION_AGENT" ]]; then
    run_cmd "describe inventory protectiongroup agent $DEFAULT_PROTECTION_AGENT (table)" veeamgo describe inventory protectiongroup --name "$DEFAULT_PROTECTION_AGENT" --group "$DEFAULT_PROTECTION_GROUP"
    PROTECTION_AGENT_TABLE_DONE=1
  fi
fi

##################################
# Phase 2: JSON outputs
##################################
section "JSON Phase"
SESSION_ID=""
if run_cmd "get session (json)" veeamgo get session --limit 20 --output json; then
  SESSION_ID=$(first_json_field "$LAST_OUTPUT" "id" "ID")
  if [[ -n "$SESSION_ID" ]]; then
    run_cmd "describe session $SESSION_ID (json)" veeamgo describe session --id "$SESSION_ID" --output json
    run_cmd "get session logs $SESSION_ID (json)" veeamgo get session logs --id "$SESSION_ID" --output json
  fi
fi

TASK_ID=""
if run_cmd "get task (json)" veeamgo get task --limit 20 --output json; then
  TASK_ID=$(first_json_field "$LAST_OUTPUT" "id" "Task ID")
  if [[ -n "$TASK_ID" ]]; then
    run_cmd "describe task $TASK_ID (json)" veeamgo describe task --id "$TASK_ID" --output json
    run_cmd "get task logs $TASK_ID (json)" veeamgo get task logs --id "$TASK_ID" --output json
  fi
fi

run_cmd "get server info (json)" veeamgo get server info --output json
run_cmd "get server time (json)" veeamgo get server time --output json

if run_cmd "get repository (json)" veeamgo get repository --limit 20 --output json; then
  DEFAULT_REPO_NAME=$(first_json_field "$LAST_OUTPUT" "name" "Name")
  if [[ -n "$DEFAULT_REPO_NAME" ]]; then
    run_cmd "describe repository $DEFAULT_REPO_NAME (json)" veeamgo describe repository --name "$DEFAULT_REPO_NAME" --output json
  fi
fi

if run_cmd "get sobr (json)" veeamgo get sobr --limit 20 --output json; then
  DEFAULT_SCALEOUT_REPO_NAME=$(first_json_field "$LAST_OUTPUT" "name" "Name")
  if [[ -n "$DEFAULT_SCALEOUT_REPO_NAME" ]]; then
    run_cmd "describe sobr $DEFAULT_SCALEOUT_REPO_NAME (json)" veeamgo describe sobr --name "$DEFAULT_SCALEOUT_REPO_NAME" --output json
  fi
fi

if run_cmd "get wanaccelerator (json)" veeamgo get wanaccelerator --limit 20 --output json; then
  DEFAULT_WAN_ACCELERATOR_NAME=$(first_json_field "$LAST_OUTPUT" "name" "Name")
  if [[ -n "$DEFAULT_WAN_ACCELERATOR_NAME" ]]; then
    run_cmd "describe wanaccelerator $DEFAULT_WAN_ACCELERATOR_NAME (json)" veeamgo describe wanaccelerator --name "$DEFAULT_WAN_ACCELERATOR_NAME" --output json
  fi
fi

if run_cmd "get proxy (json)" veeamgo get proxy --limit 50 --output json; then
  DEFAULT_PROXY_NAME=$(first_json_field "$LAST_OUTPUT" "name" "Name")
  if [[ -n "$DEFAULT_PROXY_NAME" ]]; then
    run_cmd "describe proxy $DEFAULT_PROXY_NAME (json)" veeamgo describe proxy --name "$DEFAULT_PROXY_NAME" --output json
  fi
fi

if run_cmd "get job (json)" veeamgo get job --limit 50 --output json; then
  DEFAULT_JOB_NAME=$(first_json_field "$LAST_OUTPUT" "name" "Name")
  if [[ -n "$DEFAULT_JOB_NAME" ]]; then
    run_cmd "describe job $DEFAULT_JOB_NAME (json)" veeamgo describe job "$DEFAULT_JOB_NAME" --output json
    run_cmd "get job history for $DEFAULT_JOB_NAME (json)" veeamgo get job history --name "$DEFAULT_JOB_NAME" --limit 20 --output json
  fi
  DEFAULT_REPLICA_JOB_NAME=$(jq -r 'def rows: if type=="array" then . else (.data? // []) end; rows | map(select((.type // "") | test("Replica"; "i")))[0]?.name // empty' <<<"$LAST_OUTPUT")
fi

if [[ -n "$DEFAULT_JOB_NAME" ]]; then
  run_cmd "get restorepoint for $DEFAULT_JOB_NAME (json)" veeamgo get restorepoint --name "$DEFAULT_JOB_NAME" --limit 20 --output json
  run_cmd "get backup files for $DEFAULT_JOB_NAME (json)" veeamgo get backup files --name "$DEFAULT_JOB_NAME" --limit 10 --output json
  run_cmd "get backup objects for $DEFAULT_JOB_NAME (json)" veeamgo get objectsinbackup --name "$DEFAULT_JOB_NAME" --output json
fi

if [[ -n "$DEFAULT_REPLICA_JOB_NAME" ]]; then
  run_cmd "get replica restore points for $DEFAULT_REPLICA_JOB_NAME (json)" veeamgo get replica --name "$DEFAULT_REPLICA_JOB_NAME" --limit 20 --output json
fi

if run_cmd "get inventory virtualinfra (json)" veeamgo get inventory virtualinfra --output json; then
  DEFAULT_VIRTUAL_HOST=$(first_json_field "$LAST_OUTPUT" "name" "Name")
  if [[ -n "$DEFAULT_VIRTUAL_HOST" ]]; then
    run_cmd "describe inventory virtualinfra $DEFAULT_VIRTUAL_HOST (json)" veeamgo describe inventory virtualinfra --name "$DEFAULT_VIRTUAL_HOST" --output json
    run_cmd "get inventory virtualinfra objects for $DEFAULT_VIRTUAL_HOST (json)" veeamgo get inventory virtualinfra objects --name "$DEFAULT_VIRTUAL_HOST" --limit 10 --output json
  fi
fi

run_cmd "get license sockets (json)" veeamgo get license sockets --limit 50 --output json
run_cmd "get license instances (json)" veeamgo get license instances --limit 50 --output json
run_cmd "get license capacity (json)" veeamgo get license capacity --output json
run_cmd "describe configurationbackup (json)" veeamgo describe configurationbackup --output json
run_cmd "get generaloption (json)" veeamgo get generaloption --output json
run_cmd "get malwaredetectionevent (json)" veeamgo get malwaredetectionevent --limit 20 --output json
run_cmd "get yararule (json)" veeamgo get yararule --output json
run_cmd "get trafficrule (json)" veeamgo get trafficrule --output json
run_cmd "describe security analyzer schedule (json)" veeamgo describe security analyzer schedule --output json
run_cmd "get security analyzer send-results (json)" veeamgo get security analyzer send-results --output json
if run_cmd "get inventory protectiongroup (json)" veeamgo get inventory protectiongroup --limit 10 --output json; then
  DEFAULT_PROTECTION_GROUP=$(first_json_field "$LAST_OUTPUT" "Name" "name")
  if [[ -n "$DEFAULT_PROTECTION_GROUP" ]]; then
    if run_cmd "get inventory protectiongroup agents $DEFAULT_PROTECTION_GROUP (json)" veeamgo get inventory protectiongroup agents --name "$DEFAULT_PROTECTION_GROUP" --limit 10 --output json; then
      if [[ -z "$DEFAULT_PROTECTION_AGENT" ]]; then
        DEFAULT_PROTECTION_AGENT=$(first_json_field "$LAST_OUTPUT" "Server" "Server")
      fi
      if [[ -n "$DEFAULT_PROTECTION_AGENT" ]]; then
        run_cmd "describe inventory protectiongroup agent $DEFAULT_PROTECTION_AGENT (json)" veeamgo describe inventory protectiongroup --name "$DEFAULT_PROTECTION_AGENT" --group "$DEFAULT_PROTECTION_GROUP" --output json
        if [[ $PROTECTION_AGENT_TABLE_DONE -eq 0 ]]; then
          run_cmd "describe inventory protectiongroup agent $DEFAULT_PROTECTION_AGENT (table)" veeamgo describe inventory protectiongroup --name "$DEFAULT_PROTECTION_AGENT" --group "$DEFAULT_PROTECTION_GROUP"
          PROTECTION_AGENT_TABLE_DONE=1
        fi
      fi
    fi
  fi
fi

##################################
# Phase 3: Error validation
##################################
section "Error Phase"

run_expect_fail "missing restorepoint name" veeamgo get restorepoint
run_expect_fail "missing replica job name" veeamgo get replica
run_expect_fail "missing backup files job" veeamgo get backup files
run_expect_fail "missing backup objects job" veeamgo get objectsinbackup
run_expect_fail "missing repository name" veeamgo describe repository
run_expect_fail "missing wanaccelerator name" veeamgo describe wanaccelerator
run_expect_fail "missing sobr name" veeamgo describe sobr
run_expect_fail "missing proxy name" veeamgo describe proxy
run_expect_fail "missing job history name" veeamgo get job history
run_expect_fail "missing session describe id" veeamgo describe session
run_expect_fail "missing session logs id" veeamgo get session logs
run_expect_fail "missing task describe id" veeamgo describe task
run_expect_fail "missing task logs id" veeamgo get task logs

section "Summary"
if [[ "$OVERALL_STATUS" -eq 0 ]]; then
  log "All commands executed successfully."
else
  log "Completed with failures. Inspect $LOG_FILE for details."
fi

exit "$OVERALL_STATUS"
