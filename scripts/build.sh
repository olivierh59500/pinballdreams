#!/bin/sh
# Build the standalone desktop application with the bundled game assets.
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
export GOCACHE="$PWD/.cache/go-build"
mkdir -p bin
go build -o bin/pinballdreams .
