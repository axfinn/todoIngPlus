#!/usr/bin/env bash
# Simple publish script for TodoIng images
# Usage:
#   ./docker/publish.sh build              # build images (backend + frontend)
#   ./docker/publish.sh push               # push images with existing tags
#   ./docker/publish.sh release            # build then push
# Options:
#   VERSION=X.Y.Z                          # override version (default: frontend package.json version)
#   REGISTRY=axiu                          # override registry namespace
# Options (env vars):
#   PLATFORMS=linux/amd64,linux/arm64   # multi-arch build
#   BUILD_ARGS_FRONTEND="VITE_API_BASE_URL=/api VITE_APP_VERSION=auto"
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

PLATFORMS=${PLATFORMS:-}
BUILD_ARGS_FRONTEND=${BUILD_ARGS_FRONTEND:-}
PUSH_FLAG=${PUSH_FLAG:-false}

ensure_builder() {
  if [[ -n "$PLATFORMS" ]]; then
    if ! docker buildx inspect todoing-builder >/dev/null 2>&1; then
      echo "==> Creating buildx builder (todoing-builder)"
      docker buildx create --name todoing-builder --use --driver docker-container
    else
      docker buildx use todoing-builder
    fi
  fi
}

version_exists() {
  local img=$1
  if docker manifest inspect "$img" >/dev/null 2>&1; then
    return 0
  fi
  return 1
}

parse_frontend_build_args() {
  local args=()
  if [[ -n "$BUILD_ARGS_FRONTEND" ]]; then
    for kv in $BUILD_ARGS_FRONTEND; do
      key=${kv%%=*}
      val=${kv#*=}
      args+=(--build-arg "$key=$val")
    done
  fi
  echo "${args[@]}"
}

warn_if_overwrite() {
  for img in "$BACKEND_IMAGE" "$FRONTEND_IMAGE"; do
    if version_exists "$img"; then
      echo "[WARN] Image tag already exists in registry: $img (will rebuild / overwrite if pushed)" >&2
    fi
  done
}

do_build() {
  ensure_builder
  warn_if_overwrite
  local pullFlag="--pull"
  if [[ "${NO_PULL:-}" == "1" ]]; then
    echo "NO_PULL=1 detected -> skipping --pull (using local base images)"
    pullFlag=""
  fi
  local f_args
  f_args=$(parse_frontend_build_args)

  if [[ -n "$PLATFORMS" ]]; then
    echo "==> Multi-arch build platforms: $PLATFORMS"
  fi

  echo "==> Building backend image: $BACKEND_IMAGE"
  if [[ -n "$PLATFORMS" ]]; then
    docker buildx build \
      --platform "$PLATFORMS" \
      $pullFlag \
      -t "$BACKEND_IMAGE" \
      -t "$REGISTRY/todoing-go:latest" \
      -f backend-go/Dockerfile \
      --target production \
      backend-go ${PUSH_FLAG:+--push}
  else
    docker build $pullFlag \
      -t "$BACKEND_IMAGE" \
      -t "$REGISTRY/todoing-go:latest" \
      -f backend-go/Dockerfile \
      --target production \
      backend-go
  fi

  echo "==> Building frontend image: $FRONTEND_IMAGE"
  if [[ -n "$PLATFORMS" ]]; then
    docker buildx build \
      --platform "$PLATFORMS" \
      $pullFlag \
      -t "$FRONTEND_IMAGE" \
      -t "$REGISTRY/todoing-frontend:latest" \
      -f frontend/Dockerfile \
      $f_args \
      frontend ${PUSH_FLAG:+--push}
  else
    docker build $pullFlag \
      -t "$FRONTEND_IMAGE" \
      -t "$REGISTRY/todoing-frontend:latest" \
      -f frontend/Dockerfile \
      $f_args \
      frontend
  fi
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
    PUSH_FLAG=true do_build
    do_push
    ;;
  *)
    echo "Usage: $0 {build|push|release}" >&2
    exit 1
    ;;
 esac

echo "Done."
