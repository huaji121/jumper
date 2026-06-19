#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

echo "==> Building Jumper..."
go build -o jumper.exe ./src/

echo "==> Running..."
exec ./jumper.exe
