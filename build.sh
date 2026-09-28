#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

case "$(uname -m)" in
  x86_64) GOARCH=amd64 ;;
  aarch64|arm64) GOARCH=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
esac

CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" go build   -trimpath   -ldflags="-s -w"   -o wecomchan   .

file wecomchan
