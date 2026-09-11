#!/bin/sh
# Focused deterministic native projection/re-ingestion proof. Only the
# cumulative check-go-upb-11.sh gate may claim the UPB cell.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb11-reconcile.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"
export GOCACHE

(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/reconcile" ./cmd/go-upb11-reconcile)
run() {
  "$work/reconcile" \
    -project "$repo/fixtures/go-upb11-reconciliation" \
    -out "$1" \
    -module example.test/go-upb11-reconciliation \
    -package example.test/go-upb11-reconciliation/app \
    -entry Run -revision 1 \
    -execution-g1 "$repo/modules/execution/v36/module.g1" \
    -provider-g1 "$repo/modules/provider/v1/module.g1" \
    -target example.test/go-upb11-reconciliation/model.Value \
    -expected Value -replacement Reading
}
run "$work/a"
run "$work/b"
diff -ru "$work/a" "$work/b"
rg -q 'func Reading' "$work/a/model/value.go"
rg -q 'model.Reading' "$work/a/app/run.go"
rg -q 'Value in this comment' "$work/a/model/value.go"
rg -q 'seme-native-validation-v1' "$work/a/.seme-reconciliation-v1/native-validation.txt"
test ! -e "$repo/fixtures/go-upb11-reconciliation/.seme-reconciliation-v1"
printf 'Go UPB-11 deterministic atomic cross-package reconciliation passes\n'
