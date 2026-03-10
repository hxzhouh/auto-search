#!/usr/bin/env bash

set -euo pipefail

APP_DIR="${APP_DIR:-$(cd "$(dirname "$0")" && pwd)}"
GIT_BRANCH="${GIT_BRANCH:-main}"
CONFIG_PATH="${CONFIG_PATH:-configs/config.local.json}"
LISTEN_ADDR="${LISTEN_ADDR:-127.0.0.1:8080}"
BIN_DIR="${BIN_DIR:-bin}"
RUN_DIR="${RUN_DIR:-run}"
LOG_DIR="${LOG_DIR:-logs}"
APP_NAME="${APP_NAME:-auto-search}"
GO_BIN="${GO_BIN:-go}"
BUILD_OUTPUT="$BIN_DIR/$APP_NAME"
PID_FILE="$RUN_DIR/$APP_NAME.pid"
LOG_FILE="$LOG_DIR/$APP_NAME.log"

log() {
  printf '[deploy] %s\n' "$1"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf '[deploy] 缺少命令: %s\n' "$1" >&2
    exit 1
  fi
}

stop_old_process() {
  if [[ ! -f "$PID_FILE" ]]; then
    log "没有发现旧 PID 文件，跳过停服"
    return
  fi

  local pid
  pid="$(cat "$PID_FILE")"
  if [[ -z "$pid" ]]; then
    rm -f "$PID_FILE"
    log "PID 文件为空，已清理"
    return
  fi

  if ! kill -0 "$pid" >/dev/null 2>&1; then
    rm -f "$PID_FILE"
    log "旧进程不存在，已清理 PID 文件"
    return
  fi

  log "停止旧进程 PID=$pid"
  kill "$pid"

  for _ in $(seq 1 20); do
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      rm -f "$PID_FILE"
      log "旧进程已停止"
      return
    fi
    sleep 1
  done

  log "旧进程未在预期时间内退出，执行强制停止"
  kill -9 "$pid"
  rm -f "$PID_FILE"
}

start_new_process() {
  log "后台启动服务"
  nohup "$BUILD_OUTPUT" serve -config "$CONFIG_PATH" -addr "$LISTEN_ADDR" >>"$LOG_FILE" 2>&1 &
  echo $! >"$PID_FILE"
  sleep 2

  local pid
  pid="$(cat "$PID_FILE")"
  if ! kill -0 "$pid" >/dev/null 2>&1; then
    log "启动失败，最近日志如下："
    tail -n 50 "$LOG_FILE" || true
    exit 1
  fi

  log "启动成功 PID=$pid"
  log "访问地址: http://$LISTEN_ADDR"
}

main() {
  require_cmd git
  require_cmd "$GO_BIN"

  cd "$APP_DIR"

  mkdir -p "$BIN_DIR" "$RUN_DIR" "$LOG_DIR"

  if [[ ! -d .git ]]; then
    printf '[deploy] 当前目录不是 git 仓库: %s\n' "$APP_DIR" >&2
    exit 1
  fi

  log "更新代码分支: $GIT_BRANCH"
  git fetch origin "$GIT_BRANCH"
  git checkout "$GIT_BRANCH"
  git pull --ff-only origin "$GIT_BRANCH"

  log "下载依赖"
  "$GO_BIN" mod tidy

  log "执行测试"
  "$GO_BIN" test ./...

  log "编译二进制"
  "$GO_BIN" build -o "$BUILD_OUTPUT" ./cmd/auto-search

  stop_old_process
  start_new_process
}

main "$@"
