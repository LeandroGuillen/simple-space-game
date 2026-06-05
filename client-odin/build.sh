#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"
odin build . -out:ssg-odin "$@"
echo "built: $(pwd)/ssg-odin"
