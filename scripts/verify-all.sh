#!/usr/bin/env bash
# scripts/verify-all.sh
#
# Discover every Go module in the repo, build its CLI, run the CLI's
# auth-status command, and print an aligned table of results.
#
# BUILD failures make the script exit non-zero.
# AUTH failures/unconfigured state do not.

set -u

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
REPORT_FILE="${REPORT:-}"

TMP_DIR=$(mktemp -d /tmp/verify-all.XXXXXX)
if [ -z "$TMP_DIR" ] || [ ! -d "$TMP_DIR" ]; then
  echo "verify-all: failed to create a temporary directory" >&2
  exit 1
fi
trap 'rm -rf "$TMP_DIR"' EXIT

names=()
builds=()
auths=()
any_build_failed=0

__discover_auth_cmd() {
  local bin=$1 helpfile=$2
  local section auth_help line token desc

  section=$(sed -n '/^Available Commands:/,/^Flags:/p' "$helpfile" | sed '1d;$d')

  local has_auth=0
  local doctor_desc=""
  local health_desc=""

  while IFS= read -r line; do
    [ -z "$line" ] && continue
    token=$(printf '%s' "$line" | awk '{print $1}')
    [ -z "$token" ] && continue
    desc=$(printf '%s' "$line" | sed -E 's/^  [^ ]+ +//')

    case "$token" in
      auth) has_auth=1 ;;
      doctor) doctor_desc=$desc ;;
      health) health_desc=$desc ;;
    esac
  done <<< "$section"

  if [ "$has_auth" -eq 1 ]; then
    auth_help="$TMP_DIR/auth-help.$(basename "$bin")"
    "$bin" auth --help > "$auth_help" 2>&1 || true
    if grep -qE '^  status( |$)' "$auth_help"; then
      printf 'auth status\n'
      return
    fi
  fi

  if [ -n "$doctor_desc" ]; then
    if printf '%s' "$doctor_desc" | grep -qiE 'credential|auth|verify|configuration'; then
      printf 'doctor\n'
      return
    fi
  fi

  if [ -n "$health_desc" ]; then
    if printf '%s' "$health_desc" | grep -qiE 'credential|auth|verify|configuration'; then
      printf 'health\n'
      return
    fi
  fi

  printf '\n'
}

__auth_status_for() {
  local bin=$1 name=$2
  local helpfile outf errf has_agent agent_arg cmd combined

  helpfile="$TMP_DIR/$name-help.txt"
  "$bin" --help > "$helpfile" 2>&1 || true

  has_agent=0
  if grep -qE '^      --agent' "$helpfile"; then
    has_agent=1
  fi

  cmd=$(__discover_auth_cmd "$bin" "$helpfile")
  if [ -z "$cmd" ]; then
    printf 'no auth-status command\n'
    return
  fi

  agent_arg=""
  [ "$has_agent" -eq 1 ] && agent_arg="--agent"

  outf="$TMP_DIR/$name-auth.out"
  errf="$TMP_DIR/$name-auth.err"
  "$bin" $cmd $agent_arg > "$outf" 2> "$errf" || true

  local out err
  out=$(cat "$outf" 2>/dev/null || true)
  err=$(cat "$errf" 2>/dev/null || true)
  combined="${out}${err}"

  if printf '%s' "$out" | grep -q '"authenticated": true' || \
     printf '%s' "$out" | grep -q '"token_configured": true' || \
     printf '%s' "$out" | grep -q '"ok": true'; then
    if printf '%s' "$out" | grep -q '"verified": false'; then
      printf 'configured (unverified)\n'
    else
      printf 'configured\n'
    fi
    return
  fi

  if printf '%s' "$out" | grep -q '"authenticated": false' || \
     printf '%s' "$out" | grep -q '"token_configured": false' || \
     printf '%s' "$out" | grep -q '"ok": false'; then
    printf 'not configured\n'
    return
  fi

  if printf '%s' "$combined" | grep -qEi 'source: (config|environment|file)'; then
    printf 'configured\n'
    return
  fi

  if printf '%s' "$combined" | grep -qiE 'not authenticated|no credentials|missing credentials|creds_invalid|not configured'; then
    printf 'not configured\n'
    return
  fi

  printf 'unknown\n'
}

# --- main loop ----------------------------------------------------------

while IFS= read -r gomod; do
  moddir=$(dirname "$gomod")
  name=$(basename "$moddir")
  build_out="$TMP_DIR/$name-build.out"
  build_err="$TMP_DIR/$name-build.err"

  names+=("$name")

  # Find every main package, excluding MCP servers.
  main_dirs=$(cd "$moddir" && go list -f '{{if eq .Name "main"}}{{.Dir}}{{end}}' ./... 2>/dev/null | grep -v -E 'mcp$|-mcp$' | grep -v '^$' || true)

  if [ -z "$main_dirs" ]; then
    if (cd "$moddir" && go build ./... > "$build_out" 2> "$build_err"); then
      builds+=("ok")
      auths+=("no binary")
    else
      any_build_failed=1
      err=$(head -n 1 "$build_err" | sed -E 's/[[:space:]]+/ /g')
      builds+=("FAILED: $err")
      auths+=("-")
    fi
    continue
  fi

  # Pick the CLI main package (prefer the one ending in -pp-cli).
  main_dir=""
  found_pp=0
  while IFS= read -r d; do
    [ -z "$d" ] && continue
    if [ "$found_pp" -eq 0 ] && [[ "$d" == *-pp-cli ]]; then
      main_dir=$d
      found_pp=1
      break
    fi
    [ -z "$main_dir" ] && main_dir=$d
  done <<< "$main_dirs"

  rel=${main_dir#$moddir/}
  binname=$(basename "$main_dir")
  binpath="$TMP_DIR/bin/$binname"
  mkdir -p "$TMP_DIR/bin"

  if (cd "$moddir" && go build ./... > "$build_out" 2> "$build_err") && \
     (cd "$moddir" && go build -o "$binpath" "./$rel" >> "$build_out" 2>> "$build_err"); then
    builds+=("ok")
    auths+=("$(__auth_status_for "$binpath" "$name")")
  else
    any_build_failed=1
    err=$(head -n 1 "$build_err" | sed -E 's/[[:space:]]+/ /g')
    builds+=("FAILED: $err")
    auths+=("-")
  fi
done < <(find "$REPO_ROOT" -maxdepth 2 -name go.mod | sort)

# --- print aligned table ------------------------------------------------

max_name=3
max_build=5
max_auth=15

count=${#names[@]}
for ((i = 0; i < count; i++)); do
  n=${#names[$i]}
  b=${#builds[$i]}
  a=${#auths[$i]}
  [ "$n" -gt "$max_name" ] && max_name=$n
  [ "$b" -gt "$max_build" ] && max_build=$b
  [ "$a" -gt "$max_auth" ] && max_auth=$a
done

if [ -n "$REPORT_FILE" ]; then
  if ! mkdir -p "$(dirname "$REPORT_FILE")" 2>/dev/null || ! touch "$REPORT_FILE" 2>/dev/null; then
    echo "verify-all: cannot write report to $REPORT_FILE" >&2
    exit 1
  fi
  exec > >(tee "$REPORT_FILE")
fi

printf "%-${max_name}s   %-${max_build}s   %s\n" "CLI" "BUILD" "AUTH"

for ((i = 0; i < count; i++)); do
  printf "%-${max_name}s   %-${max_build}s   %s\n" "${names[$i]}" "${builds[$i]}" "${auths[$i]}"
done

if [ "$any_build_failed" -ne 0 ]; then
  exit 1
fi
exit 0
