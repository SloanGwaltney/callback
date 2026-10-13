#!/usr/bin/env bash
# Starts a development run for an agent to drive with scripts/drive/cdp.mjs: the app window stays
# hidden (CALLBACK_START_HIDDEN) and the data lives in temp/drive/appdata (CALLBACK_DATA_DIR), so
# nothing opens on the user's screen and their real data isn't touched. The app is then served at
# http://localhost:34115. -s skips the frontend install and build, and -m skips `go mod tidy`,
# neither of which an agent may run.
#
# Usage: bash scripts/drive/dev.sh        (run it in the background)
#        bash scripts/drive/dev.sh wait   (returns once the app is served, or fails if it stopped
#                                          or took longer than WAIT seconds, 180 by default)
#        bash scripts/drive/dev.sh stop   (when done)
#        bash scripts/drive/dev.sh reset  (while stopped: delete temp/drive/appdata, so the next
#                                          run starts with no jobs)
#
# To run as if it were another day, set CALLBACK_TODAY: CALLBACK_TODAY=2026-10-01 bash
# scripts/drive/dev.sh. Changes made in the run, like seeded jobs and moves, are dated that day.
# Stop and start again with a later date to see how they look days on; the data is kept.
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
  # Removed first, so the run sees it was stopped on purpose when its app exits.
  rm -f "$pidfile"
  if command -v taskkill >/dev/null; then
    taskkill //PID "$pid" //T //F
  else
    pkill -P "$pid"
    kill "$pid"
  fi
  exit 0
fi

if [ "$1" = "wait" ]; then
  for ((i = 0; i < ${WAIT:-180}; i++)); do
    # The run writes the pidfile just after it starts, so allow it a few seconds to appear.
    if [ ! -f "$pidfile" ] && [ "$i" -ge 5 ]; then
      echo "dev.sh: the app isn't running; start it in the background with: bash scripts/drive/dev.sh" >&2
      exit 1
    fi
    if curl -s -o /dev/null http://localhost:34115; then
      echo "dev.sh: serving at http://localhost:34115"
      exit 0
    fi
    sleep 1
  done
  echo "dev.sh: the app didn't start within ${WAIT:-180} seconds" >&2
  exit 1
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
status=$?
# `stop` removes the pidfile before killing the app, so a missing pidfile means it was stopped on
# purpose, not that it failed.
if [ ! -f "$pidfile" ]; then
  exit 0
fi
rm -f "$pidfile"
exit "$status"
