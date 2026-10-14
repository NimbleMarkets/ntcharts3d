#!/usr/bin/env bash
# Build reproducibly from committed sibling sources while nt3d is unpublished.
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p .deps
extract() {
  local name="$1" source="$2" revision="$3" staging
  if [[ -f ".deps/$name/.source-revision" ]] && [[ "$(cat ".deps/$name/.source-revision")" == "$revision" ]]; then
    return
  fi
  if [[ -e ".deps/$name" ]]; then
    echo ".deps/$name already exists; move it aside before changing the dependency pin." >&2
    exit 1
  fi
  git -C "$source" cat-file -e "$revision^{commit}"
  staging=$(mktemp -d ".deps/$name.XXXXXXXX")
  git -C "$source" archive "$revision" | tar -x -C "$staging"
  printf '%s\n' "$revision" > "$staging/.source-revision"
  mv "$staging" ".deps/$name"
}
extract nt3d "${NT3D_SOURCE:-../nt3d}" 972f2a4220a42f3bc148e6505ff992aaa6d6caa5
extract ntpicture "${NTPICTURE_SOURCE:-../ntpicture}" 1a8609901740ed0a22cc7f729a5aa0694d6578f1
