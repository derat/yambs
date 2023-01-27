#!/bin/sh -e

set -x

if [ $# -ne 3 ]; then
  echo "Usage: $0 <os> <arch> <version>" >&2
  exit 2
fi

export GOOS=$1
export GOARCH=$2

deps=
if [ "$GOOS" = windows ]; then
  deps="zip"
fi

# Install dependencies here instead of in release.yaml since changes
# outside of the workspace don't persist across build steps.
if [ -n "$deps" ] && [ "$(id -u)" -eq 0 ]; then
  apt-get update && apt-get install -y $deps
fi

# Strip off the leading 'v' from tag names like 'v0.1'.
version=${3#v}

go build -ldflags "-X main.version=${version}" -tags nogcp ./cmd/yambs

archive=yambs-${version}-${GOOS}-${GOARCH}
files="README.md LICENSE"
if [ "$GOOS" = windows ]; then
  zip "${archive}.zip" yambs.exe $files
  rm yambs.exe
else
  tar -czvf "${archive}.tar.gz" yambs $files
  rm yambs
fi
