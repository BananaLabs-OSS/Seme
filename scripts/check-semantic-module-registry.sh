#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-module-registry.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
cd "$repo"

# Registry ownership is evaluated over checked authoritative artifacts. Exact
# semantic validation remains with each module's reproduction gate because an
# individual module may intentionally reference declarations from pinned
# imports that are unavailable when its artifact is inspected alone.
find modules -mindepth 3 -maxdepth 3 -name module.g1 -print | sort > "$work/modules.list"
while IFS= read -r module; do
  directory=$(dirname "$module")
  (cd "$directory" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256) >/dev/null
done < "$work/modules.list"

node scripts/semantic-module-registry.mjs --root modules --out "$work/first.json"
node scripts/semantic-module-registry.mjs --root modules --out "$work/second.json"
cmp "$work/first.json" "$work/second.json"
cmp conformance/semantic-module-registry-v1.json "$work/first.json"
node -e 'const r=require(process.argv[1]);if(r.contract!=="seme.semantic-module-registry/v1"||r.family_count<8||r.revision_count<42)process.exit(1);const execution=r.families.find(x=>x.family==="execution");if(!execution||execution.revisions.length<35||new Set(execution.revisions.map(x=>x.module)).size!==1)process.exit(1)' "$work/first.json"

copy_modules() { cp -R modules "$1"; }
expect_reject() {
  name=$1; code=$2
  if node scripts/semantic-module-registry.mjs --root "$work/$name" > "$work/$name.out" 2> "$work/$name.err"; then
    echo "semantic module registry accepted $name" >&2
    exit 1
  fi
  rg -q "$code" "$work/$name.err"
  test ! -s "$work/$name.out"
}

copy_modules "$work/collision"
node -e 'const fs=require("fs"),p=process.argv[1],s=fs.readFileSync(p,"utf8"),a="0000000000000000000000000000e010",b="0000000000000000000000000000b010";if(!s.includes(a))process.exit(2);fs.writeFileSync(p,s.replaceAll(a,b))' "$work/collision/project/v1/module.g1"
expect_reject collision registry.cross_family_owned_identity_collision

# Non-exported identities are local to a revision and may coincide across
# families. Add the same valid but unexported local entity to two artifacts.
copy_modules "$work/local-reuse"
for module in "$work/local-reuse/package/v1/module.g1" "$work/local-reuse/provider/v1/module.g1"; do
  node -e 'const fs=require("fs"),p=process.argv[1],s=fs.readFileSync(p,"utf8"),m=s.match(/^ec ([0-9]+)$/m);if(!m)process.exit(2);const out=s.replace(m[0],`ec ${Number(m[1])+1}`).replace(/\s*$/,"\n\n")+"en aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa 00000000000000000000000000000010 1 0\n";fs.writeFileSync(p,out)' "$module"
done
node scripts/semantic-module-registry.mjs --root "$work/local-reuse" > "$work/local-reuse.json"
test -s "$work/local-reuse.json"

copy_modules "$work/missing-export"
node -e 'const fs=require("fs"),p=process.argv[1],s=fs.readFileSync(p,"utf8"),a="fi 00000000000000000000000000000122 li 8",b="fi 00000000000000000000000000000122 li 9";if(!s.includes(a))process.exit(2);fs.writeFileSync(p,s.replace(a,b))' "$work/missing-export/project/v1/module.g1"
expect_reject missing-export registry.malformed_reference_list

copy_modules "$work/unknown-export"
node -e 'const fs=require("fs"),p=process.argv[1],s=fs.readFileSync(p,"utf8"),a="rf 0000000000000000000000000000e010",b="rf 0000000000000000000000000000efff";if(!s.includes(a))process.exit(2);fs.writeFileSync(p,s.replace(a,b))' "$work/unknown-export/project/v1/module.g1"
expect_reject unknown-export registry.export_entity_missing

copy_modules "$work/missing-parent"
node -e 'const fs=require("fs"),p=process.argv[1],s=fs.readFileSync(p,"utf8"),a="00000000000000000000000000009001",b="ffffffffffffffffffffffffffffffff";if(!s.includes(a))process.exit(2);fs.writeFileSync(p,s.replace(a,b))' "$work/missing-parent/execution/v35/module.g1"
expect_reject missing-parent registry.parent_pin_missing

copy_modules "$work/missing-import"
node -e 'const fs=require("fs"),p=process.argv[1],s=fs.readFileSync(p,"utf8"),a="fi 00000000000000000000000000000131 by 0000000000000000000000000000b001",b="fi 00000000000000000000000000000131 by ffffffffffffffffffffffffffffffff";if(!s.includes(a))process.exit(2);fs.writeFileSync(p,s.replace(a,b))' "$work/missing-import/project/v1/module.g1"
expect_reject missing-import registry.import_pin_missing

copy_modules "$work/duplicate-artifact"
mkdir "$work/duplicate-artifact/execution/copy"
cp "$work/duplicate-artifact/execution/v35/module.g1" "$work/duplicate-artifact/execution/copy/module.g1"
expect_reject duplicate-artifact registry.duplicate_module_revision_artifact

copy_modules "$work/duplicate-parent"
node -e 'const fs=require("fs"),p=process.argv[1],s=fs.readFileSync(p,"utf8"),pin="00000000000000000000000000009001";fs.writeFileSync(p,s.replace(`pc 1\n${pin}`,`pc 2\n${pin}\n${pin}`))' "$work/duplicate-parent/execution/v35/module.g1"
expect_reject duplicate-parent registry.duplicate_parent_pin

copy_modules "$work/unsorted-parent"
node -e 'const fs=require("fs"),p=process.argv[1],s=fs.readFileSync(p,"utf8"),a="00000000000000000000000000009001",b="00000000000000000000000000009002";fs.writeFileSync(p,s.replace(`pc 1\n${a}`,`pc 2\n${b}\n${a}`))' "$work/unsorted-parent/execution/v35/module.g1"
expect_reject unsorted-parent registry.unsorted_parent_pins

copy_modules "$work/parent-cycle"
node -e 'const fs=require("fs"),p=process.argv[1],s=fs.readFileSync(p,"utf8"),revision="00000000000000000000000000009002";fs.writeFileSync(p,s.replace("pc 0\nec ",`pc 1\n${revision}\nec `))' "$work/parent-cycle/execution/v1/module.g1"
expect_reject parent-cycle registry.parent_cycle

echo 'Semantic module registry: deterministic lineage/export/import ownership audit and adversarial rejection pass'
