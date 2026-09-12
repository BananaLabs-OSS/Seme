#!/bin/sh
# One cumulative acceptance gate for the single shared Go/JavaScript/Lua UPB12
# claim. Nothing below is a substitute language-specific claim.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-upb12-shared.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME

"$repo/scripts/build-upb12-seed.sh" "$work/prior-a" "$work/prior-export-a"
"$repo/scripts/build-upb12-seed.sh" "$work/prior-b" "$work/prior-export-b"
diff -ru "$work/prior-a" "$work/prior-b"
"$repo/scripts/build-upb12-revision.sh" "$work/result-a" "$work/result-export-a"
"$repo/scripts/build-upb12-revision.sh" "$work/result-b" "$work/result-export-b"
diff -ru "$work/result-a" "$work/result-b"

"$repo/scripts/check-upb12-runtime.sh" "$work/prior-a"
"$repo/scripts/check-upb12-revision-convergence.sh" "$work/prior-a" "$work/result-a" "$work/reconciliation"
"$repo/scripts/check-upb12-project-v14.sh" "$work/prior-export-a" "$work/result-export-a" "$work/reconciliation" "$work/project-v14"
"$repo/scripts/check-upb12-adversaries.sh" "$work/prior-export-a" "$work/result-export-a"

printf 'Shared UPB12 complete: two byte-identical publications; exact Go/JavaScript/Lua projection, native validation, re-lift, runtime parity, revision convergence, Project-v14, and atomic adversaries pass\n'
