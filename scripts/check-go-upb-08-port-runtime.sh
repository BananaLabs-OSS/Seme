#!/bin/sh
# Authenticated selected host-port and pinned Pulp placement evidence for UPB08.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb08-port.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE
(cd "$repo/reference/go" && go test -count=1 ./orderedtransportruntime ./orderedtransportplacement ./goupb08portruntime)
printf 'Go UPB-08 authenticated selected host-port framing, external exact-byte evidence, retry, and Pulp TransportPort rejection pass\n'
