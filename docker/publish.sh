#!/usr/bin/env bash
# Simple publish script for TodoIng images
# Usage:
#   ./docker/publish.sh build              # build images (backend + frontend)
#   ./docker/publish.sh push               # push images with existing tags
#   ./docker/publish.sh release            # build then push
# Options:
#   VERSION=X.Y.Z                          # override version (default: frontend package.json version)
#   REGISTRY=axiu                          # override registry namespace
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

REGISTRY="${REGISTRY:-axiu}"
# Extract version from frontend/package.json if not provided
if [[ -z "${VERSION:-}" ]]; then
  if command -v jq >/dev/null 2>&1; then
    VERSION=$(jq -r '.version' frontend/package.json)
  else
    VERSION=$(grep '"version"' frontend/package.json | head -1 | sed -E 's/.*"version" *: *"([^"]+)".*/\1/')
  fi
fi

if [[ -z "$VERSION" ]]; then
  echo "Failed to determine version" >&2
  exit 1
fi

BACKEND_IMAGE="$REGISTRY/todoing-go:$VERSION"
FRONTEND_IMAGE="$REGISTRY/todoing-frontend:$VERSION"

DOCKER=${DOCKER_BIN:-docker}
if ! command -v "$DOCKER" >/dev/null 2>&1; then
  echo "docker CLI not found; aborting build" >&2
  exit 2
fi

do_build() {
  local pullFlag="--pull"
  if [[ "${NO_PULL:-}" == "1" ]]; then
    echo "NO_PULL=1 detected -> skipping --pull (using local base images)"
    pullFlag=""
  fi
  echo "==> Building backend image: $BACKEND_IMAGE"
  $DOCKER build \
    ${pullFlag} \
    -t "$BACKEND_IMAGE" \
    -t "$REGISTRY/todoing-go:latest" \
    -f backend-go/Dockerfile \
    --target production \
    backend-go

  echo "==> Building frontend image: $FRONTEND_IMAGE"
  $DOCKER build \
    ${pullFlag} \
    -t "$FRONTEND_IMAGE" \
    -t "$REGISTRY/todoing-frontend:latest" \
    -f frontend/Dockerfile \
    frontend
}

do_push() {
  echo "==> Pushing backend images"
  $DOCKER push "$BACKEND_IMAGE"
  $DOCKER push "$REGISTRY/todoing-go:latest"
  echo "==> Pushing frontend images"
  $DOCKER push "$FRONTEND_IMAGE"
  $DOCKER push "$REGISTRY/todoing-frontend:latest"
}

case "${1:-}" in
  build)
    do_build
    ;;
  push)
    do_push
    ;;
  release)
    do_build
    do_push
    ;;
  *)
    echo "Usage: $0 {build|push|release}" >&2
    exit 1
    ;;
 esac

echo "Done."
