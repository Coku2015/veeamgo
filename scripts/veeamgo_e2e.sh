#!/usr/bin/env bash
set -uo pipefail

LOG_FILE=${LOG_FILE:-"$(pwd)/veeamgo-test-$(date +'%Y%m%d-%H%M%S').log"}
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

run_help_command() {
  local desc=$1
  shift
  run_cmd "$desc" "$@" --help
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

fetch_default() {
  local var_name=$1
  local primary=$2
  local fallback=$3
  shift 3
  local output value
  if output=$("$@" 2>/dev/null); then
    value=$(first_json_field "$output" "$primary" "$fallback")
    if [[ -n $value ]]; then
      eval "$var_name=\"\$value\""
    fi
  fi
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
log "veeamgo binary: $(command -v veeamgo)"
log "jq binary: $(command -v jq)"
log "Host: $(uname -a)"

run_cmd "veeamgo version" veeamgo --version
run_cmd "veeamgo help" veeamgo --help

##################################
# Discover defaults via JSON    #
##################################
DEFAULT_MANAGED_SERVER_ID=""
DEFAULT_MANAGED_SERVER_NAME=""
DEFAULT_REPO_NAME=""
DEFAULT_SCALEOUT_REPO_NAME=""
DEFAULT_OBJECT_REPO_NAME=""
DEFAULT_PROXY_NAME=""
DEFAULT_JOB_NAME=""
DEFAULT_REPLICA_JOB_NAME=""
DEFAULT_BACKUP_NAME=""
DEFAULT_EXCLUSION_ID=""
DEFAULT_SESSION_ID=""
DEFAULT_TASK_ID=""
DEFAULT_INVENTORY_VI_NAME=""
DEFAULT_WAN_ACCELERATOR_NAME=""
DEFAULT_INSTANT_RECOVERY_ID=""
DEFAULT_PUBLISHED_DISK_ID=""

fetch_default DEFAULT_MANAGED_SERVER_ID "id" "Id" veeamgo get managedserver --limit 5 --output json
fetch_default DEFAULT_MANAGED_SERVER_NAME "name" "Name" veeamgo get managedserver --limit 5 --output json
fetch_default DEFAULT_REPO_NAME "name" "Name" veeamgo get repository --limit 5 --output json
fetch_default DEFAULT_SCALEOUT_REPO_NAME "name" "Name" veeamgo get sobr --limit 5 --output json
fetch_default DEFAULT_OBJECT_REPO_NAME "name" "Name" veeamgo get objectrepository --limit 5 --output json
fetch_default DEFAULT_PROXY_NAME "name" "Name" veeamgo get proxy --limit 5 --output json
fetch_default DEFAULT_JOB_NAME "name" "Name" veeamgo get job --limit 20 --output json
fetch_default DEFAULT_REPLICA_JOB_NAME "name" "Name" veeamgo get job --limit 20 --output json
fetch_default DEFAULT_BACKUP_NAME "name" "Name" veeamgo get backup --limit 5 --output json
fetch_default DEFAULT_EXCLUSION_ID "id" "Id" veeamgo get exclusionvm --limit 5 --output json
fetch_default DEFAULT_SESSION_ID "id" "Id" veeamgo get session --limit 5 --output json
fetch_default DEFAULT_TASK_ID "id" "Id" veeamgo get task --limit 5 --output json
fetch_default DEFAULT_INVENTORY_VI_NAME "name" "Name" veeamgo get inventory virtualinfra --limit 5 --output json
fetch_default DEFAULT_WAN_ACCELERATOR_NAME "name" "Name" veeamgo get wanaccelerator --limit 5 --output json
fetch_default DEFAULT_PUBLISHED_DISK_ID "id" "Id" veeamgo get publisheddisk --all --limit 5 --output json

##################################
# Phase 1: Table commands        #
##################################
section "Server"
run_cmd "get server info (table)" veeamgo get server info
run_cmd "get server time (table)" veeamgo get server time

section "Managed Servers"
run_cmd "get managedserver (table)" veeamgo get managedserver --limit 20
if [[ -z "$DEFAULT_MANAGED_SERVER_ID" ]]; then
  DEFAULT_MANAGED_SERVER_ID=$(extract_tab_field "$LAST_OUTPUT" 5)
fi
if [[ -n "$DEFAULT_MANAGED_SERVER_ID" ]]; then
  run_cmd "describe managedserver $DEFAULT_MANAGED_SERVER_ID (table)" veeamgo describe managedserver --id "$DEFAULT_MANAGED_SERVER_ID"
fi

section "Repositories"
run_cmd "get repository (table)" veeamgo get repository --limit 20
if [[ -z "$DEFAULT_REPO_NAME" ]]; then
  DEFAULT_REPO_NAME=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_REPO_NAME" ]]; then
  run_cmd "describe repository $DEFAULT_REPO_NAME (table)" veeamgo describe repository --name "$DEFAULT_REPO_NAME"
fi
run_cmd "get objectrepository (table)" veeamgo get objectrepository --limit 20
if [[ -z "$DEFAULT_OBJECT_REPO_NAME" ]]; then
  DEFAULT_OBJECT_REPO_NAME=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_OBJECT_REPO_NAME" ]]; then
  run_cmd "describe objectrepository $DEFAULT_OBJECT_REPO_NAME (table)" veeamgo describe objectrepository --name "$DEFAULT_OBJECT_REPO_NAME"
fi
run_cmd "get sobr (table)" veeamgo get sobr --limit 20
if [[ -z "$DEFAULT_SCALEOUT_REPO_NAME" ]]; then
  DEFAULT_SCALEOUT_REPO_NAME=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_SCALEOUT_REPO_NAME" ]]; then
  run_cmd "describe sobr $DEFAULT_SCALEOUT_REPO_NAME (table)" veeamgo describe sobr --name "$DEFAULT_SCALEOUT_REPO_NAME"
fi

section "WAN Accelerator"
run_cmd "get wanaccelerator (table)" veeamgo get wanaccelerator --limit 20
if [[ -n "$DEFAULT_WAN_ACCELERATOR_NAME" ]]; then
  run_cmd "describe wanaccelerator $DEFAULT_WAN_ACCELERATOR_NAME (table)" veeamgo describe wanaccelerator --name "$DEFAULT_WAN_ACCELERATOR_NAME"
fi

section "General Options"
run_cmd "get generaloption (table)" veeamgo get generaloption

section "Jobs & Backups"
run_cmd "get job (table)" veeamgo get job --limit 20
if [[ -z "$DEFAULT_JOB_NAME" ]]; then
  DEFAULT_JOB_NAME=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_JOB_NAME" ]]; then
  run_cmd "describe job $DEFAULT_JOB_NAME (table)" veeamgo describe job --name "$DEFAULT_JOB_NAME"
fi
run_cmd "get backup (table)" veeamgo get backup --limit 20
if [[ -z "$DEFAULT_BACKUP_NAME" ]]; then
  DEFAULT_BACKUP_NAME=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_JOB_NAME" ]]; then
  run_cmd "get restorepoint for $DEFAULT_JOB_NAME (table)" veeamgo get restorepoint --name "$DEFAULT_JOB_NAME" --limit 20
fi

section "Replicas"
if [[ -n "$DEFAULT_REPLICA_JOB_NAME" ]]; then
  run_cmd "get replica for $DEFAULT_REPLICA_JOB_NAME (table)" veeamgo get replica --name "$DEFAULT_REPLICA_JOB_NAME" --limit 20
fi

section "Proxies"
run_cmd "get proxy (table)" veeamgo get proxy --limit 20
if [[ -z "$DEFAULT_PROXY_NAME" ]]; then
  DEFAULT_PROXY_NAME=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_PROXY_NAME" ]]; then
  run_cmd "describe proxy $DEFAULT_PROXY_NAME (table)" veeamgo describe proxy --name "$DEFAULT_PROXY_NAME"
fi

section "Traffic Rules & Exclusions"
run_cmd "get trafficrule (table)" veeamgo get trafficrule
run_cmd "get exclusionvm (table)" veeamgo get exclusionvm --limit 20
if [[ -z "$DEFAULT_EXCLUSION_ID" ]]; then
  DEFAULT_EXCLUSION_ID=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_EXCLUSION_ID" ]]; then
  run_cmd "describe exclusionvm $DEFAULT_EXCLUSION_ID (table)" veeamgo describe exclusionvm --id "$DEFAULT_EXCLUSION_ID"
fi

section "Inventory"
run_cmd "get inventory virtualinfra (table)" veeamgo get inventory virtualinfra --limit 20
if [[ -z "$DEFAULT_INVENTORY_VI_NAME" ]]; then
  DEFAULT_INVENTORY_VI_NAME=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_INVENTORY_VI_NAME" ]]; then
  run_cmd "describe inventory virtualinfra $DEFAULT_INVENTORY_VI_NAME (table)" veeamgo describe inventory virtualinfra --name "$DEFAULT_INVENTORY_VI_NAME"
  run_cmd "get inventory virtualinfra objects for $DEFAULT_INVENTORY_VI_NAME (table)" veeamgo get inventory virtualinfra objects --name "$DEFAULT_INVENTORY_VI_NAME" --limit 20
fi
run_cmd "get inventory unstructured (table)" veeamgo get inventory unstructured --limit 20

section "Sessions & Tasks"
run_cmd "get session (table)" veeamgo get session --limit 20
if [[ -z "$DEFAULT_SESSION_ID" ]]; then
  DEFAULT_SESSION_ID=$(extract_tab_field "$LAST_OUTPUT" 8)
fi
if [[ -n "$DEFAULT_SESSION_ID" ]]; then
  run_cmd "describe session $DEFAULT_SESSION_ID (table)" veeamgo describe session --id "$DEFAULT_SESSION_ID"
  run_cmd "get session logs $DEFAULT_SESSION_ID (table)" veeamgo get session logs --id "$DEFAULT_SESSION_ID"
fi
run_cmd "get task (table)" veeamgo get task --limit 20
if [[ -z "$DEFAULT_TASK_ID" ]]; then
  DEFAULT_TASK_ID=$(extract_tab_field "$LAST_OUTPUT" 1)
fi
if [[ -n "$DEFAULT_TASK_ID" ]]; then
  run_cmd "describe task $DEFAULT_TASK_ID (table)" veeamgo describe task --id "$DEFAULT_TASK_ID"
  run_cmd "get task logs $DEFAULT_TASK_ID (table)" veeamgo get task logs --id "$DEFAULT_TASK_ID"
fi

section "Malware Detection"
run_cmd "get malwaredetectionevent (table)" veeamgo get malwaredetectionevent --limit 20
run_cmd "get yararule (table)" veeamgo get yararule

section "Published Disks"
run_cmd "get publisheddisk (table)" veeamgo get publisheddisk --all --limit 20
if [[ -z "$DEFAULT_PUBLISHED_DISK_ID" ]]; then
  DEFAULT_PUBLISHED_DISK_ID=$(extract_tab_field "$LAST_OUTPUT" 1)
fi

section "License"
run_cmd "get license summary (table)" veeamgo get license summary
run_cmd "get license sockets (table)" veeamgo get license sockets --limit 20
run_cmd "get license instances (table)" veeamgo get license instances --limit 20
run_cmd "get license capacity (table)" veeamgo get license capacity --limit 20

section "Configuration Backup"
run_cmd "describe configurationbackup (table)" veeamgo describe configurationbackup

##################################
# Phase 2: JSON spot checks      #
##################################
section "JSON Phase"
run_cmd "get repository (json)" veeamgo get repository --limit 5 --output json
run_cmd "get objectrepository (json)" veeamgo get objectrepository --limit 5 --output json
run_cmd "get job (json)" veeamgo get job --limit 5 --output json
run_cmd "get backup (json)" veeamgo get backup --limit 5 --output json
run_cmd "get session (json)" veeamgo get session --limit 5 --output json
run_cmd "get task (json)" veeamgo get task --limit 5 --output json
run_cmd "get inventory virtualinfra (json)" veeamgo get inventory virtualinfra --limit 5 --output json
run_cmd "get publisheddisk (json)" veeamgo get publisheddisk --all --limit 5 --output json

##################################
# Phase 3: Help text validation  #
##################################
section "Help Phase"

# top-level verbs
run_help_command "help: veeamgo get" veeamgo get
run_help_command "help: veeamgo describe" veeamgo describe
run_help_command "help: veeamgo rescan" veeamgo rescan
run_help_command "help: veeamgo template" veeamgo template
run_help_command "help: veeamgo add" veeamgo add
run_help_command "help: veeamgo edit" veeamgo edit
run_help_command "help: veeamgo enable" veeamgo enable
run_help_command "help: veeamgo disable" veeamgo disable
run_help_command "help: veeamgo clone" veeamgo clone
run_help_command "help: veeamgo delete" veeamgo delete
run_help_command "help: veeamgo start" veeamgo start
run_help_command "help: veeamgo stop" veeamgo stop
run_help_command "help: veeamgo retry" veeamgo retry
run_help_command "help: veeamgo publish" veeamgo publish
run_help_command "help: veeamgo migrate" veeamgo migrate

help_get_commands=(
  "server"
  "managedserver"
  "repository"
  "objectrepository"
  "sobr"
  "wanaccelerator"
  "generaloption"
  "job"
  "backup"
  "restorepoint"
  "replica"
  "proxy"
  "trafficrule"
  "exclusionvm"
  "inventory"
  "inventory virtualinfra"
  "inventory virtualinfra objects"
  "inventory unstructured"
  "session"
  "session logs"
  "task"
  "task logs"
  "security"
  "securityanalyzer"
  "securityanalyzer results"
  "malwaredetectionevent"
  "yararule"
  "publisheddisk"
  "license"
  "license summary"
  "license sockets"
  "license instances"
  "license capacity"
)

for entry in "${help_get_commands[@]}"; do
  IFS=' ' read -r -a parts <<< "$entry"
  run_help_command "help: veeamgo get $entry" veeamgo get "${parts[@]}"
done

help_describe_commands=(
  "managedserver"
  "repository"
  "objectrepository"
  "sobr"
  "wanaccelerator"
  "job"
  "inventory"
  "inventory virtualinfra"
  "security"
  "securityanalyzer schedule"
  "session"
  "task"
  "configurationbackup"
)

for entry in "${help_describe_commands[@]}"; do
  IFS=' ' read -r -a parts <<< "$entry"
  run_help_command "help: veeamgo describe $entry" veeamgo describe "${parts[@]}"
done

run_help_command "help: veeamgo start publishdisk" veeamgo start publishdisk
run_help_command "help: veeamgo stop publishdisk" veeamgo stop publishdisk
run_help_command "help: veeamgo template job" veeamgo template job

log ""
if (( OVERALL_STATUS == 0 )); then
  log "RESULT: all commands completed successfully"
else
  log "RESULT: one or more commands failed"
fi

exit $OVERALL_STATUS
