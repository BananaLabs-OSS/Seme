#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
fixture="$root/fixtures/go-upb03-dependency-v1"
proxy="$root/fixtures/go-upb03-offline-proxy"
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb03.XXXXXX")
trap 'chmod -R u+w "$work" 2>/dev/null || true; rm -rf "$work"' EXIT

export GOPROXY="file://$proxy"
export GOSUMDB=off
export GONOSUMDB='*'
export GOPATH="$work/gopath"
export GOMODCACHE="$work/modcache"
export GOCACHE="$work/buildcache"

validate_manifest() {
  local file=$1
  [[ $(grep -Ec '^require example\.test/seme/checksum v1\.2\.3$' "$file") -eq 1 ]] || return 1
  if grep -Eq '^[[:space:]]*(replace|exclude)[[:space:]]' "$file"; then return 1; fi
  [[ $(grep -Ec '^[[:space:]]*require ' "$file") -eq 1 ]] || return 1
}

validate_manifest "$fixture/go.mod"
printf '%s  %s\n' b8c1fa6b91ffd1c51b765506ac14b80210a4019d75892e51782596d0f994910e "$proxy/example.test/seme/checksum/@v/v1.2.3.zip" | sha256sum -c - >/dev/null
(cd "$fixture" && go mod download all && go test ./...)
(cd "$fixture" && go list -m -f '{{.Path}} {{.Version}} {{.Sum}} {{.GoModSum}}' all) > "$work/first.txt"
(cd "$fixture" && go list -m -f '{{.Path}} {{.Version}} {{.Sum}} {{.GoModSum}}' all) > "$work/second.txt"
cmp "$work/first.txt" "$work/second.txt"
[[ $(wc -l < "$work/first.txt") -eq 2 ]]
grep -F 'example.test/seme/checksum v1.2.3 h1:iqNHCyOnAuyQDdyru3Tt6tsFgwUzMObOH5iQ9+5SMXA=' "$work/first.txt" >/dev/null

cp -R "$fixture" "$work/substituted"
printf '\nreplace example.test/seme/checksum => %s\n' "$root/fixtures/go-upb03-external-checksum-v1.2.3" >> "$work/substituted/go.mod"
if validate_manifest "$work/substituted/go.mod"; then echo substituted dependency accepted >&2; exit 1; fi

cp -R "$fixture" "$work/undeclared"
sed -i '/^require /d' "$work/undeclared/go.mod"
if validate_manifest "$work/undeclared/go.mod"; then echo undeclared dependency accepted >&2; exit 1; fi

cp -R "$fixture" "$work/floating"
sed -i 's/v1\.2\.3/v1.2.x/' "$work/floating/go.mod"
if validate_manifest "$work/floating/go.mod" || (cd "$work/floating" && go list -m all >/dev/null 2>&1); then echo floating dependency accepted >&2; exit 1; fi

cp -R "$fixture" "$work/bad-integrity"
sed -i 's/h1:iqNHC/h1:AqNHC/' "$work/bad-integrity/go.sum"
chmod -R u+w "$GOMODCACHE" 2>/dev/null || true
rm -rf "$GOMODCACHE"
if (cd "$work/bad-integrity" && go mod download all >/dev/null 2>&1); then echo integrity mismatch accepted >&2; exit 1; fi

echo 'go UPB-03 offline fixture: pass'
