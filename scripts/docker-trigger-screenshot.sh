#!/bin/bash

set -euo pipefail

IMAGE_NAME="cmux-shell"
CONTAINER_NAME="cmux-screenshot"
WORKER_PORT=39377
POLLING_BASE="http://localhost:${WORKER_PORT}/socket.io/?EIO=4&transport=polling"
PR_URL=""
EXEC_COMMAND=""
NON_INTERACTIVE=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --pr)
      if [[ $# -lt 2 ]]; then
        echo "Error: --pr requires an argument" >&2
        exit 1
      fi
      PR_URL="$2"
      shift 2
      ;;
    --pr=*)
      PR_URL="${1#*=}"
      shift 1
      ;;
    --exec)
      if [[ $# -lt 2 ]]; then
        echo "Error: --exec requires a command string" >&2
        exit 1
      fi
      EXEC_COMMAND="$2"
      NON_INTERACTIVE=true
      shift 2
      ;;
    --exec=*)
      EXEC_COMMAND="${1#*=}"
      NON_INTERACTIVE=true
      shift 1
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

container_started=false

cleanup() {
  if [ "$container_started" = true ]; then
    echo "Stopping container..."
    docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
    container_started=false
  fi
}

cleanup_and_exit() {
  cleanup
  exit 0
}

trap cleanup EXIT
trap cleanup_and_exit INT TERM

echo "Building Docker image..."
docker build -t "$IMAGE_NAME" .

echo "Starting container..."
if docker ps -a --format '{{.Names}}' | grep -Fxq "$CONTAINER_NAME"; then
  docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
fi

docker run -d \
  --rm \
  --privileged \
  --cgroupns=host \
  --tmpfs /run \
  --tmpfs /run/lock \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  -v docker-data:/var/lib/docker \
  -p 39375:39375 \
  -p 39376:39376 \
  -p 39377:39377 \
  -p 39378:39378 \
  -p 39379:39379 \
  -p 39380:39380 \
  -p 39381:39381 \
  -p 39382:39382 \
  -p 39383:39383 \
  --name "$CONTAINER_NAME" \
  "$IMAGE_NAME"
container_started=true

# Allow services to initialize
sleep 5

# Wait for worker health endpoint to be ready
printf "Waiting for worker health endpoint"
health_ready=false
for _ in {1..30}; do
  if curl -sSf "http://localhost:${WORKER_PORT}/health" >/dev/null 2>&1; then
    printf "\n"
    health_ready=true
    break
  fi
  printf "."
  sleep 1
done

if [ "$health_ready" = false ]; then
  echo ""
  echo "Worker health endpoint did not respond in time"
  exit 1
fi

if [ -n "$PR_URL" ]; then
  echo "Cloning PR $PR_URL into /root/workspace..."
  GH_TOKEN_VALUE=$(gh auth token 2>/dev/null || true)
  if [ -z "$GH_TOKEN_VALUE" ]; then
    echo "Error: GitHub auth token unavailable. Run 'gh auth login' on the host before using --pr." >&2
    cleanup
    exit 1
  fi
  docker exec \
    -e PR_URL="$PR_URL" \
    -e GH_TOKEN="$GH_TOKEN_VALUE" \
    "$CONTAINER_NAME" \
    bash -lc '
set -euo pipefail

if [[ -z "${GH_TOKEN:-}" ]]; then
  echo "GH_TOKEN not provided; cannot check out PR inside container" >&2
  exit 1
fi

if [[ -z "${PR_URL:-}" ]]; then
  echo "PR_URL environment variable is required when cloning" >&2
  exit 1
fi

if [[ "$PR_URL" != https://github.com/*/pull/* ]]; then
  echo "PR URL must match https://github.com/<owner>/<repo>/pull/<number>" >&2
  exit 1
fi

REPO_URL="${PR_URL%/pull/*}"
WORKTREE=/root/workspace

rm -rf "$WORKTREE"
echo "Cloning $REPO_URL into $WORKTREE"
git clone "$REPO_URL" "$WORKTREE"
cd "$WORKTREE"
echo "Fetching PR branch via gh pr checkout..."
export GIT_TERMINAL_PROMPT=0
gh pr checkout "$PR_URL" >/dev/null
echo "Repository ready at $WORKTREE"
' 
fi

echo "Performing Socket.IO polling handshake..."
HANDSHAKE_RESPONSE=$(curl -s "${POLLING_BASE}&t=$(date +%s%3N)")

if [ -z "$HANDSHAKE_RESPONSE" ]; then
  echo "Handshake with worker failed; empty response"
  exit 1
fi

SID=$(node -e "const raw = process.argv[1]; const start = raw.indexOf('{'); const end = raw.lastIndexOf('}') + 1; if (start === -1 || end === 0) { process.stderr.write('Unable to parse handshake response\n'); process.exit(1); } const payload = JSON.parse(raw.slice(start, end)); process.stdout.write(payload.sid);" "$HANDSHAKE_RESPONSE")

if [ -z "$SID" ]; then
  echo "Failed to parse session id from handshake response"
  exit 1
fi

echo "Connecting to /management namespace..."
curl -s \
  -X POST \
  -H 'Content-Type: text/plain;charset=UTF-8' \
  --data-binary '40/management' \
  "${POLLING_BASE}&sid=${SID}&t=$(date +%s%3N)" >/dev/null

echo "Triggering worker:start-screenshot-collection..."
curl -s \
  -X POST \
  -H 'Content-Type: text/plain;charset=UTF-8' \
  --data-binary '42/management,["worker:start-screenshot-collection"]' \
  "${POLLING_BASE}&sid=${SID}&t=$(date +%s%3N)" >/dev/null

echo "Screenshot collection trigger sent. View logs via http://localhost:39378/?folder=/var/log/cmux"

echo ""
echo "================================ URLs ================================="
printf "| %-18s | %s |\n" "Worker Logs" "http://localhost:39378/?folder=/var/log/cmux"
printf "| %-18s | %s |\n" "Workspace" "http://localhost:39378/?folder=/root/workspace"
printf "| %-18s | %s |\n" "VS Code" "http://localhost:39378/?folder=/root/workspace"
printf "| %-18s | %s |\n" "noVNC" "http://localhost:39380/vnc.html"
echo "========================================================================"

if [ -n "$EXEC_COMMAND" ]; then
  echo "Executing command inside container: $EXEC_COMMAND"
  docker exec "$CONTAINER_NAME" bash -lc "$EXEC_COMMAND"
fi

if [ "$NON_INTERACTIVE" = false ] && [ -t 0 ]; then
  echo ""
  echo "Leave this script running while you inspect the worker."
  echo "Press Enter (or Ctrl+C) when you're ready to stop the container."
  read -r _
elif [ "$NON_INTERACTIVE" = false ]; then
  echo ""
  echo "Non-interactive shell detected; keeping the container alive for 5 minutes before cleanup."
  sleep 300
fi

# Drain any pending poll so server can close cleanly once we're done
curl -s "${POLLING_BASE}&sid=${SID}&t=$(date +%s%3N)" >/dev/null || true

cleanup
