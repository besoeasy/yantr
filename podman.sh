#!/usr/bin/env bash
#
# podman.sh — local dev loop for Yantr.
#
#   ./podman.sh              remove old build, rebuild image, run
#   ./podman.sh --skip-build reuse the existing image, just recreate the container
#   ./podman.sh --no-cache   rebuild ignoring cached layers
#   ./podman.sh logs         tail yantr logs
#   ./podman.sh stop         stop and remove the container (image + volumes kept)
#   ./podman.sh shell        interactive shell inside the running container
#
# Overridable via environment:
#   YANTR_DEV_IMAGE     image name           (default yantr:dev)
#   YANTR_DEV_CONTAINER container name       (default yantr)
#   YANTR_DEV_VOLUME    /data volume         (default yantr_data, shared with prod)
#   YANTR_DEV_PORT      host port            (default 5252)

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

IMAGE="${YANTR_DEV_IMAGE:-yantr:dev}"
CONTAINER="${YANTR_DEV_CONTAINER:-yantr}"
DATA_VOLUME="${YANTR_DEV_VOLUME:-yantr_data}"
PORT="${YANTR_DEV_PORT:-5252}"

SOCKET="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/podman/podman.sock"

SKIP_BUILD=0
NO_CACHE=0

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
  R=$'\033[31m'; G=$'\033[32m'; Y=$'\033[33m'; B=$'\033[34m'; D=$'\033[2m'; N=$'\033[0m'
else
  R=''; G=''; Y=''; B=''; D=''; N=''
fi

log()  { printf '%s==>%s %s\n' "$B" "$N" "$*"; }
ok()   { printf '%s  ok%s %s\n' "$G" "$N" "$*"; }
warn() { printf '%swarn%s %s\n' "$Y" "$N" "$*" >&2; }
die()  { printf '%sfail%s %s\n' "$R" "$N" "$*" >&2; exit 1; }

# ── arg parsing ────────────────────────────────────────────────────────────────
case "${1:-}" in
  --skip-build) SKIP_BUILD=1; shift ;;
  --no-cache)   NO_CACHE=1; shift ;;
  logs|stop|shell|"") ;;
  -h|--help)    sed -n '3,16p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
  *)            die "unknown argument: $1  (try --help)" ;;
esac

# ── preflight ──────────────────────────────────────────────────────────────────
command -v podman >/dev/null 2>&1 || die "podman not found in PATH"

if [ ! -S "$SOCKET" ]; then
  die "podman socket not found at $SOCKET
       Start rootless podman first:  systemctl --user start podman.socket"
fi

# ── subcommands ────────────────────────────────────────────────────────────────
case "${1:-}" in
  logs)  exec podman logs -f --tail 200 "$CONTAINER" ;;
  shell) exec podman exec -it "$CONTAINER" /bin/sh ;;
  stop)
    podman rm -f "$CONTAINER" >/dev/null 2>&1 && ok "container removed" || true
    exit 0
    ;;
esac

if [ "$SKIP_BUILD" -eq 0 ]; then
  # ── 1. remove old build ──────────────────────────────────────────────────────
  log "Removing previous build"
  if podman container exists "$CONTAINER"; then
    podman rm -f "$CONTAINER" >/dev/null && ok "stopped and removed container '$CONTAINER'"
  else
    printf '%s  ..%s no existing container\n' "$D" "$N"
  fi
  if podman image exists "$IMAGE"; then
    podman rmi -f "$IMAGE" >/dev/null && ok "removed previous image '$IMAGE'"
  fi
  # Drop the previous image's dangling layers so disk does not creep per rebuild.
  podman image prune -f >/dev/null 2>&1 && ok "pruned dangling image layers" || true

  # ── 2. rebuild ───────────────────────────────────────────────────────────────
  log "Building $IMAGE  ${D}(multi-stage: vite -> go -> alpine)${N}"
  if [ "$NO_CACHE" -eq 1 ]; then
    podman build --no-cache -t "$IMAGE" "$ROOT"
  else
    podman build -t "$IMAGE" "$ROOT"
  fi
  ok "image built"
else
  log "Skipping build (--skip-build)"
  podman container exists "$CONTAINER" && podman rm -f "$CONTAINER" >/dev/null
fi

# ── 3. run ─────────────────────────────────────────────────────────────────────
# --network host mirrors the Quadlet unit and install.sh: the managed stacks get
# their ports published by the *host* podman, so only Yantr's own UI needs to be
# reachable, and the mounted socket is a host path.
log "Starting container"
podman run -d \
  --name "$CONTAINER" \
  --network host \
  --security-opt label=disable \
  -e CONTAINER_HOST=unix:///run/podman/podman.sock \
  -e YANTR_TELEMETRY=false \
  -v "$SOCKET:/run/podman/podman.sock" \
  -v "$DATA_VOLUME:/data:z" \
  -v "$ROOT/apps:/app/apps:z" \
  "$IMAGE" >/dev/null

ok "container '$CONTAINER' up"

# Mounting the repo's apps/ makes catalog edits visible without a rebuild, and
# keeps the generated .compose.<project>.yml / .env.<project> across recreates.
# Those generated files are gitignored.

# ── 4. wait for health ─────────────────────────────────────────────────────────
printf '%s  ..%s waiting for /api/health' "$D" "$N"
for _ in $(seq 1 40); do
  if curl -fsS "http://127.0.0.1:${PORT}/api/health" >/dev/null 2>&1; then
    printf '\r%s  ok%s healthy                                   \n' "$G" "$N"
    printf '\n  %shttp://localhost:%s%s\n\n' "$G" "$PORT" "$N"
    exit 0
  fi
  if ! podman container exists "$CONTAINER"; then
    printf '\r'
    warn "container exited during startup — last log lines:"
    podman logs --tail 30 "$CONTAINER" >&2 || true
    exit 1
  fi
  printf '.'
  sleep 0.5
done

printf '\r'
warn "did not become healthy in 20s — check: ./podman.sh logs"
exit 1
