#!/usr/bin/env bash
# Starts a development run for an agent to drive with scripts/drive/cdp.mjs: the app window stays
# hidden (CALLBACK_START_HIDDEN) and the data lives in temp/drive/appdata (CALLBACK_DATA_DIR), so
# nothing opens on the user's screen and their real data isn't touched. The app is then served at
# http://localhost:34115. -s skips the frontend install and build, and -m skips `go mod tidy`,
# neither of which an agent may run.
#
# Usage: bash scripts/drive/dev.sh        (run it in the background)
#        bash scripts/drive/dev.sh stop   (when done)
#        bash scripts/drive/dev.sh reset  (while stopped: delete temp/drive/appdata, so the next
#                                          run starts with no jobs)
#
# To empty or fill the board while the app runs, use clearBoard and seed in cdp.mjs instead.
#
# Stopping the background shell isn't enough: under Git Bash, the Windows processes it started
# keep running. So the run records wails's PID, and `stop` kills that process tree, and only it.
cd "$(dirname "$0")/../.." || exit 2

pidfile="temp/drive/wails.pid"

if [ "$1" = "stop" ]; then
  if [ ! -f "$pidfile" ]; then
    echo "dev.sh: not running"
    exit 0
  fi
  pid="$(cat "$pidfile")"
  if command -v taskkill >/dev/null; then
    taskkill //PID "$pid" //T //F
  else
    pkill -P "$pid"
    kill "$pid"
  fi
  rm -f "$pidfile"
  exit 0
fi

if [ "$1" = "reset" ]; then
  # The running app holds the database open, so its files can't be deleted under it.
  if [ -f "$pidfile" ]; then
    echo "dev.sh: the app is running; stop it first with: bash scripts/drive/dev.sh stop" >&2
    exit 1
  fi
  rm -rf temp/drive/appdata
  echo "dev.sh: deleted temp/drive/appdata"
  exit 0
fi

data="$PWD/temp/drive/appdata"
mkdir -p "$data"
# The app is a Windows program under Git Bash, so it needs a Windows path.
if command -v cygpath >/dev/null; then
  data="$(cygpath -w "$data")"
fi

CALLBACK_DATA_DIR="$data" CALLBACK_START_HIDDEN=1 wails dev -s -m &
pid=$!
# Git Bash's PID isn't the Windows one that taskkill needs.
if [ -f "/proc/$pid/winpid" ]; then
  cat "/proc/$pid/winpid" >"$pidfile"
else
  echo "$pid" >"$pidfile"
fi
wait "$pid"
