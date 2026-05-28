#!/usr/bin/env bash
set -euo pipefail

ACTION="${1:-generate}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MODULE="github.com/kamalyes/grpc-runtime"

GOOGLEAPIS_VERSION="master"
GRPC_GATEWAY_VERSION="v2.28.0"

log() { printf '\n>>> %s\n' "$*"; }
log_ok() { printf '\n[OK] %s\n' "$*"; }
log_err() { printf '\n[ERROR] %s\n' "$*" >&2; }

run() { printf '+ %s\n' "$*"; "$@"; }

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'missing required command: %s\n' "$1" >&2
    exit 1
  fi
}

to_unix_path() {
  if command -v cygpath >/dev/null 2>&1; then cygpath -u "$1"; return 0; fi
  printf '%s\n' "$1"
}

go_bin_dir() {
  local gobin gopath
  gobin="$(go env GOBIN)"
  if [[ -n "$gobin" ]]; then to_unix_path "$gobin"; return 0; fi
  gopath="$(go env GOPATH)"
  to_unix_path "$gopath/bin"
}

export PATH="$(go_bin_dir):$PATH"

gopath_src_dir() { local g; g="$(go env GOPATH)"; to_unix_path "$g/src"; }
googleapis_dir() { printf '%s/github.com/googleapis\n' "$(gopath_src_dir)"; }
grpc_runtime_dir() { printf '%s/github.com/kamalyes/grpc-runtime\n' "$(gopath_src_dir)"; }

protoc_include_dir() {
  local protoc_path protoc_dir protoc_root include_dir
  protoc_path="$(command -v protoc)"
  protoc_path="$(to_unix_path "$protoc_path")"
  protoc_dir="$(cd "$(dirname "$protoc_path")" && pwd)"
  protoc_root="$(cd "$protoc_dir/.." && pwd)"
  for include_dir in "$protoc_root/include" "$(go_bin_dir)/include"; do
    if [[ -f "$include_dir/google/protobuf/descriptor.proto" ]]; then
      printf '%s\n' "$include_dir"
      return 0
    fi
  done
  printf 'could not find google/protobuf/*.proto include files\n' >&2
  exit 1
}

proto_includes() {
  local protoc_include googleapis grpc_runtime
  protoc_include="$(protoc_include_dir)"
  googleapis="$(googleapis_dir)"
  grpc_runtime="$(grpc_runtime_dir)"
  printf '%s\0' -I "$PROJECT_ROOT"
  if [[ -d "$googleapis" ]]; then printf '%s\0' -I "$googleapis"; fi
  if [[ -d "$grpc_runtime" ]]; then printf '%s\0' -I "$grpc_runtime"; fi
  printf '%s\0' -I "$protoc_include"
}

read_proto_includes() {
  local -n out="$1"
  mapfile -d '' -t out < <(proto_includes)
}

setup_dependencies() {
  local gopath_src googleapis grpc_runtime
  gopath_src="$(gopath_src_dir)"
  googleapis="$(googleapis_dir)"
  grpc_runtime="$(grpc_runtime_dir)"
  mkdir -p "$gopath_src/github.com"

  if [[ -d "$googleapis" ]]; then
    log "googleapis already exists, skipping"
  else
    log "Downloading googleapis..."
    require_cmd git
    git clone --depth=1 --branch="$GOOGLEAPIS_VERSION" https://github.com/googleapis/googleapis.git "$googleapis"
  fi

  if [[ -d "$grpc_runtime" ]]; then
    log "grpc-runtime already exists, skipping"
  else
    log "Downloading grpc-runtime..."
    require_cmd git
    git clone --depth=1 --branch="$GRPC_GATEWAY_VERSION" https://github.com/kamalyes/grpc-runtime.git "$grpc_runtime"
  fi

  if [[ -f "$googleapis/google/api/annotations.proto" ]]; then
    log_ok "googleapis dependency verified"
  else
    log_err "googleapis dependency verification failed"
    exit 1
  fi

  if [[ -f "$grpc_runtime/protocgen/openapiv2/options/annotations.proto" ]]; then
    log_ok "grpc-runtime dependency verified"
  else
    log_err "grpc-runtime dependency verification failed"
    exit 1
  fi
}

install_tools() {
  log "Installing protoc plugins..."
  run go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
  run go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
}

generate_testpb() {
  log "Generating testpb..."
  require_cmd protoc
  require_cmd protoc-gen-go
  require_cmd protoc-gen-go-grpc
  local includes=()
  read_proto_includes includes
  (
    cd "$PROJECT_ROOT"
    run protoc \
      "${includes[@]}" \
      --go_out=. --go_opt=module="$MODULE" \
      --go-grpc_out=. --go-grpc_opt=module="$MODULE" \
      testpb/example.proto \
      testpb/proto2.proto \
      testpb/proto3.proto \
      testpb/non_standard_names.proto
  )
}

generate_protocgen() {
  log "Generating protocgen descriptors..."
  require_cmd protoc
  require_cmd protoc-gen-go
  local includes=()
  read_proto_includes includes
  (
    cd "$PROJECT_ROOT"
    run protoc \
      "${includes[@]}" \
      --go_out=. --go_opt=module="$MODULE" \
      protocgen/openapiv2/options/openapiv2.proto \
      protocgen/openapiv2/options/annotations.proto \
      protocgen/descriptor/openapiconfig/openapiconfig.proto
  )
}

clean_generated() {
  log "Cleaning generated proto code..."
  rm -f "$PROJECT_ROOT"/testpb/*.pb.go
  rm -f "$PROJECT_ROOT"/testpb/*_grpc.pb.go
  rm -f "$PROJECT_ROOT"/testpb/*.swagger.json "$PROJECT_ROOT"/testpb/*.swagger.yaml
  rm -f "$PROJECT_ROOT"/protocgen/openapiv2/options/*.pb.go
  rm -f "$PROJECT_ROOT"/protocgen/descriptor/openapiconfig/*.pb.go
}

clean_unused() {
  rm -f "$PROJECT_ROOT"/testpb/*.swagger.json "$PROJECT_ROOT"/testpb/*.swagger.yaml
}

verify() {
  log "Running tests..."
  (cd "$PROJECT_ROOT" && run go test ./...)
}

usage() {
  cat <<'USAGE'
Usage: scripts/generate.sh [command]

Commands:
  generate     Generate testpb and protocgen descriptors
  testpb       Generate grpc-runtime/testpb only
  protocgen    Generate grpc-runtime/protocgen descriptors only
  setup        Download googleapis and grpc-runtime to GOPATH
  clean        Clean generated pb files
  verify       Run go test ./...
  test         Alias for verify
  tools        Install protoc-gen-go and protoc-gen-go-grpc
  help         Show this help
USAGE
}

case "$ACTION" in
  generate|proto)
    generate_testpb
    generate_protocgen
    clean_unused
    ;;
  testpb|generate-testpb)
    generate_testpb
    clean_unused
    ;;
  protocgen|generate-protocgen)
    generate_protocgen
    clean_unused
    ;;
  setup|setup-deps)
    setup_dependencies
    ;;
  clean)
    clean_generated
    ;;
  verify|test)
    verify
    ;;
  tools)
    install_tools
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
