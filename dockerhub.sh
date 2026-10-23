#!/usr/bin/env bash
# ==============================================================================
#  Yantr — publish the container image to Docker Hub
#
#  Same image as .github/workflows/ghcr.yml, built with Podman instead of
#  Buildx and pushed to Docker Hub instead of GHCR. Tags match the workflow:
#
#    :latest            rolling, newest published build
#    :YYYY.MM.DD[.N]    release tag, the CalVer scheme the workflow uses
#
#  A multi-arch manifest list (linux/amd64 + linux/arm64) is produced with
#  `podman build --platform ... --manifest`, then pushed with
#  `podman manifest push --all`.
#
#  Usage:
#    ./dockerhub.sh                    # HEAD's tag if tagged, else today's CalVer
#    ./dockerhub.sh 2026.10.21         # publish an explicit tag
#    ./dockerhub.sh --no-latest        # skip the rolling :latest tag
#    ./dockerhub.sh --dry-run          # print what would run, push nothing
#
#  Environment:
#    DOCKERHUB_REPO   repo path without registry (default: besoeasy/yantr)
#    PLATFORMS        platforms to build (default: linux/amd64,linux/arm64)
#    SKIP_BINFMT_CHECK=1  build for foreign arches even without binfmt
#
#  Requirements:
#    * podman (4 or newer)
#    * qemu-user-static registered in binfmt_misc for the foreign architecture
#      (Fedora: `sudo dnf install qemu-user-static`, Debian/Ubuntu:
#      `sudo apt install qemu-user-static`, Arch: `sudo pacman -S qemu-user-static`)
#    * a prior `podman login docker.io` (rootless: as your user,
#      rootful: `sudo podman login docker.io`)
# ==============================================================================

set -euo pipefail

DOCKERHUB_REPO="${DOCKERHUB_REPO:-besoeasy/yantr}"
DOCKERHUB_REPO="${DOCKERHUB_REPO#docker.io/}"
DOCKERHUB_REPO="${DOCKERHUB_REPO#docker://}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"

PUSH_LATEST=1
DRY_RUN=0
REQUESTED_TAG=""

# --------------------------------------------------------------------------
# Logging — same conventions as install.sh: tput palette when available,
# plain output when colors are off, piped, or TERM is dumb.
# --------------------------------------------------------------------------
if [ -n "${NO_COLOR:-}" ] || [ ! -t 1 ] || [ "${TERM:-xterm}" = "dumb" ]; then
  USE_COLOR=0
else
  USE_COLOR=1
fi

if [ "$USE_COLOR" -eq 1 ]; then
  if command -v tput >/dev/null 2>&1 && tput setaf 1 >/dev/null 2>&1; then
    BOLD="$(tput bold)"; DIM="$(tput dim)"
    GREEN="$(tput setaf 2)"; YELLOW="$(tput setaf 3)"
    RED="$(tput setaf 1)"; CYAN="$(tput setaf 6)"; NC="$(tput sgr0)"
  else
    BOLD="\033[1m"; DIM="\033[2m"; GREEN="\033[0;32m"; YELLOW="\033[1;33m"
    RED="\033[0;31m"; CYAN="\033[0;36m"; NC="\033[0m"
  fi
else
  BOLD=""; DIM=""; GREEN=""; YELLOW=""; RED=""; CYAN=""; NC=""
fi

log_info()  { echo -e "${CYAN}->${NC} $*"; }
log_done()  { echo -e "${GREEN}OK${NC} $*"; }
log_warn()  { echo -e "${YELLOW}warning:${NC} $*" >&2; }
log_error() { echo -e "${RED}error:${NC} $*" >&2; }
die()       { log_error "$*"; exit 1; }

usage() {
  echo -e "${BOLD}Yantr — publish to Docker Hub${NC}"
  echo -e "${DIM}Podman multi-arch build · linux/amd64 + linux/arm64${NC}"
  echo ""
  echo -e "${BOLD}Usage:${NC}"
  echo "  ./dockerhub.sh                  # HEAD's tag if tagged, else today's CalVer"
  echo "  ./dockerhub.sh <tag>            # publish an explicit tag, e.g. 2026.10.21"
  echo ""
  echo -e "${BOLD}Options:${NC}"
  echo "  --no-latest       Skip the rolling :latest tag"
  echo "  --dry-run         Print the commands, build and push nothing"
  echo "  --help, -h        Show this help message"
  echo ""
  echo -e "${BOLD}Environment:${NC}"
  echo "  DOCKERHUB_REPO    Repo path without registry (default: besoeasy/yantr)"
  echo "  PLATFORMS         Platforms to build (default: linux/amd64,linux/arm64)"
  echo "  SKIP_BINFMT_CHECK=1  Build for foreign arches without binfmt emulation"
  echo ""
  echo -e "${DIM}Tags match .github/workflows/ghcr.yml: :latest and :YYYY.MM.DD[.N].${NC}"
  echo -e "${DIM}Requires 'podman login docker.io' and qemu-user-static for the foreign arch.${NC}"
}

# --------------------------------------------------------------------------
# Arguments
# --------------------------------------------------------------------------
for arg in "$@"; do
  case "$arg" in
    -h|--help)   usage; exit 0 ;;
    --no-latest) PUSH_LATEST=0 ;;
    --dry-run)   DRY_RUN=1 ;;
    -*)          die "unknown option '$arg' (try --help)" ;;
    *)
      [ -z "$REQUESTED_TAG" ] || die "only one tag may be given"
      REQUESTED_TAG="$arg"
      ;;
  esac
done

# --------------------------------------------------------------------------
# Tag resolution — keep the ghcr workflow's naming.
#
# The workflow tags the release with the CalVer date and bumps with .N when the
# git tag is already taken. Here an already-tagged HEAD keeps that tag (a
# dockerhub run right after a ghcr release therefore publishes the same tag),
# and anything newer falls back to the same date-and-bump rule.
# --------------------------------------------------------------------------
resolve_tag() {
  local tag

  if [ -n "$REQUESTED_TAG" ]; then
    printf '%s\n' "$REQUESTED_TAG"
    return 0
  fi

  # Tags created by the workflow may not be fetched yet.
  git fetch --tags --quiet origin >/dev/null 2>&1 || true

  if tag="$(git describe --tags --exact-match HEAD 2>/dev/null)"; then
    printf '%s\n' "$tag"
    return 0
  fi

  local base n
  base="$(date -u +%Y.%m.%d)"
  tag="$base"
  n=1
  while git rev-parse -q --verify "refs/tags/$tag" >/dev/null 2>&1; do
    n=$((n + 1))
    tag="${base}.${n}"
  done
  printf '%s\n' "$tag"
}

# --------------------------------------------------------------------------
# Preflight
# --------------------------------------------------------------------------
require_podman() {
  command -v podman >/dev/null 2>&1 || die "podman not found in PATH"
}

check_login() {
  local user
  if ! user="$(podman login --get-login docker.io 2>/dev/null)"; then
    die "not logged in to Docker Hub.
  Run: podman login docker.io
  If you logged in with 'sudo podman login', the credentials live in root's
  store; run this script the same way: sudo ./dockerhub.sh"
  fi
  log_info "Docker Hub user: ${BOLD}${user}${NC}"
}

# Map a platform arch to the binfmt_misc handler name.
binfmt_handler() {
  case "$1" in
    amd64) echo qemu-x86_64 ;;
    arm64) echo qemu-aarch64 ;;
    arm)   echo qemu-arm ;;
    *)     echo "" ;;
  esac
}

check_binfmt() {
  [ "${SKIP_BINFMT_CHECK:-0}" = "1" ] && return 0

  local host_arch platform arch handler missing=()
  host_arch="$(podman info --format '{{.Host.Arch}}' 2>/dev/null || uname -m)"

  local IFS=','
  for platform in $PLATFORMS; do
    arch="${platform##*/}"
    [ "$arch" = "$host_arch" ] && continue
    handler="$(binfmt_handler "$arch")"
    [ -n "$handler" ] || continue
    if [ ! -e "/proc/sys/fs/binfmt_misc/${handler}" ] && \
       [ ! -e "/proc/sys/fs/binfmt_misc/${handler}-static" ]; then
      missing+=("$arch")
    fi
  done

  if [ "${#missing[@]}" -gt 0 ]; then
    die "no binfmt emulation for ${missing[*]} on ${host_arch}.
  Install qemu-user-static so ${PLATFORMS} can be built on this host:
    Fedora:        sudo dnf install qemu-user-static
    Debian/Ubuntu: sudo apt install qemu-user-static
    Arch:          sudo pacman -S qemu-user-static
  Or build only the native platform: PLATFORMS=linux/${host_arch} $0
  (set SKIP_BINFMT_CHECK=1 to bypass this check)"
  fi
}

# Print the command in a copy-pasteable form instead of running it.
run() {
  if [ "$DRY_RUN" -eq 1 ]; then
    printf '  %s' "$(printf '%q ' "$@")"
    printf '\n'
  else
    "$@"
  fi
}

# --------------------------------------------------------------------------
# Main
# --------------------------------------------------------------------------
require_podman

if [ "$DRY_RUN" -eq 0 ]; then
  check_login
  check_binfmt
else
  log_warn "dry run: nothing will be built or pushed"
fi

# Uncommitted changes would publish code that ghcr never built.
if [ -n "$(git status --porcelain 2>/dev/null || true)" ]; then
  log_warn "working tree has uncommitted changes; the image will not match any ghcr release"
fi

TAG="$(resolve_tag)"
IMAGE="docker.io/${DOCKERHUB_REPO}"
MANIFEST="${IMAGE}:${TAG}"

if [ "$PUSH_LATEST" -eq 1 ]; then
  TAG_LABEL="${TAG}, latest"
else
  TAG_LABEL="${TAG}"
fi

log_info "Repository: ${BOLD}${IMAGE}${NC}"
log_info "Tag:        ${BOLD}${TAG_LABEL}${NC}"
log_info "Platforms:  ${BOLD}${PLATFORMS}${NC}"

if [ "$DRY_RUN" -eq 0 ]; then
  # A leftover local list of the same name would make the build append to it.
  podman manifest rm "$MANIFEST" >/dev/null 2>&1 || true
fi

log_info "Building image (this takes a while, both platforms build sequentially)"
run podman build \
  --platform "$PLATFORMS" \
  --manifest "$MANIFEST" \
  --format oci \
  .

# Push the manifest list, and the same list under :latest. Both platforms'
# layers are included, so consumers pull only their architecture.
push_manifest() {
  local dest="$1" digest_file=""
  if [ "$DRY_RUN" -eq 1 ]; then
    printf '  podman manifest push --all --format v2s2 %q %q\n' "$MANIFEST" "docker://$dest"
    return 0
  fi
  digest_file="$(mktemp "${TMPDIR:-/tmp}/dockerhub-digest.XXXXXX")"
  if ! podman manifest push --all --format v2s2 --digestfile "$digest_file" \
       "$MANIFEST" "docker://$dest"; then
    rm -f "$digest_file"
    die "push failed: $dest"
  fi
  log_done "pushed ${BOLD}${dest}${NC} ${DIM}sha256:$(cat "$digest_file")${NC}"
  rm -f "$digest_file"
}

push_manifest "$MANIFEST"
if [ "$PUSH_LATEST" -eq 1 ]; then
  push_manifest "${IMAGE}:latest"
fi

if [ "$DRY_RUN" -eq 0 ]; then
  echo
  if [ "$PUSH_LATEST" -eq 1 ]; then
    log_done "published ${IMAGE}:${TAG} and ${IMAGE}:latest"
  else
    log_done "published ${IMAGE}:${TAG}"
  fi
  log_info "verify with: podman manifest inspect ${IMAGE}:${TAG}"
fi
