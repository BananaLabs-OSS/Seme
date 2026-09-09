package projectsnapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"seme.local/reference/wire"
	"sort"
)

var (
	projectRevision = id("0000000000000000000000000000e001")
	projectModule   = id("0000000000000000000000000000e000")
	moduleSchema    = id("00000000000000000000000000000012")
	importSchema    = id("00000000000000000000000000000013")
	fImports        = id("00000000000000000000000000000121")
	fImportModule   = id("00000000000000000000000000000130")
	fImportRevision = id("00000000000000000000000000000131")
	identitySchema  = id("0000000000000000000000000000e010")
	snapshotSchema  = id("0000000000000000000000000000e011")
	packageSchema   = id("0000000000000000000000000000b010")
	programSchema   = id("00000000000000000000000000009015")
	fIdentity       = id("0000000000000000000000000000e110")
	fRevision       = id("0000000000000000000000000000e111")
	fPackages       = id("0000000000000000000000000000e112")
	fRoot           = id("0000000000000000000000000000e113")
	fProgram        = id("0000000000000000000000000000e114")
)

func id(s string) wire.ID {
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
func Validate(source []byte) error {
	e, x := wire.Decode(source)
	if x != nil {
		return x
	}
	return ValidateEnvelope(e)
}
func ValidateEnvelope(e wire.Envelope) error {
	module, ok := e.Entities[e.Module]
	if !ok || module.Schema != moduleSchema {
		return fmt.Errorf("project_snapshot.module_declaration")
	}
	imports, err := refs(module, fImports)
	if err != nil {
		return fmt.Errorf("project_snapshot.module_imports")
	}
	pinned := false
	for _, iid := range imports {
		imp, ok := e.Entities[iid]
		if !ok || imp.Schema != importSchema {
			continue
		}
		m, x := ref(imp, fImportModule)
		r, y := blob(imp, fImportRevision)
		if x == nil && y == nil && m == projectModule && bytes.Equal(r, projectRevision[:]) {
			pinned = true
		}
	}
	if !pinned {
		return fmt.Errorf("project_snapshot.project_import_unpinned")
	}
	count := 0
	for eid, item := range e.Entities {
		if item.Schema != snapshotSchema {
			continue
		}
		count++
		identity, x := ref(item, fIdentity)
		if x != nil {
			return at(eid, x)
		}
		revision, x := blob(item, fRevision)
		if x != nil {
			return at(eid, x)
		}
		packages, x := refs(item, fPackages)
		if x != nil {
			return at(eid, x)
		}
		root, x := ref(item, fRoot)
		if x != nil {
			return at(eid, x)
		}
		program, x := ref(item, fProgram)
		if x != nil {
			return at(eid, x)
		}
		if e.Entities[identity].Schema != identitySchema {
			return fmt.Errorf("project_snapshot.identity_schema:%s", eid)
		}
		if len(packages) == 0 {
			return fmt.Errorf("project_snapshot.packages_empty:%s", eid)
		}
		if !sort.SliceIsSorted(packages, func(i, j int) bool { return bytes.Compare(packages[i][:], packages[j][:]) < 0 }) {
			return fmt.Errorf("project_snapshot.packages_unsorted:%s", eid)
		}
		seen := map[wire.ID]bool{}
		roots := 0
		for _, p := range packages {
			if seen[p] {
				return fmt.Errorf("project_snapshot.package_duplicate:%s", eid)
			}
			seen[p] = true
			if e.Entities[p].Schema != packageSchema {
				return fmt.Errorf("project_snapshot.package_schema:%s", eid)
			}
			if p == root {
				roots++
			}
		}
		if roots != 1 {
			return fmt.Errorf("project_snapshot.root_membership:%s", eid)
		}
		if e.Entities[program].Schema != programSchema {
			return fmt.Errorf("project_snapshot.program_schema:%s", eid)
		}
		if len(revision) != sha256.Size {
			return fmt.Errorf("project_snapshot.revision_shape:%s", eid)
		}
		expected, x := Revision(e, identity, packages, root, program)
		if x != nil || !bytes.Equal(revision, expected) {
			return fmt.Errorf("project_snapshot.revision_mismatch:%s", eid)
		}
	}
	if count != 1 {
		return fmt.Errorf("project_snapshot.count:%d", count)
	}
	return nil
}
func Revision(e wire.Envelope, identity wire.ID, packages []wire.ID, root, program wire.ID) ([]byte, error) {
	h := sha256.New()
	h.Write([]byte("project-snapshot-v1\x00"))
	for _, x := range append(append([]wire.ID{identity}, packages...), root, program) {
		h.Write(x[:])
	}
	wanted := map[wire.ID]bool{}
	queue := append(append([]wire.ID{identity}, packages...), program)
	for len(queue) > 0 {
		x := queue[0]
		queue = queue[1:]
		if wanted[x] {
			continue
		}
		entity, ok := e.Entities[x]
		if !ok {
			return nil, fmt.Errorf("project_snapshot.digest_missing:%s", x)
		}
		wanted[x] = true
		for _, v := range entity.Fields {
			if err := collect(v, e.Entities, &queue); err != nil {
				return nil, err
			}
		}
	}
	ids := make([]wire.ID, 0, len(wanted))
	for x := range wanted {
		ids = append(ids, x)
	}
	sort.Slice(ids, func(i, j int) bool { return bytes.Compare(ids[i][:], ids[j][:]) < 0 })
	for _, x := range ids {
		entity := e.Entities[x]
		h.Write(x[:])
		h.Write(entity.Schema[:])
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], entity.Version)
		h.Write(n[:])
		keys := make([]wire.ID, 0, len(entity.Fields))
		for k := range entity.Fields {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i][:], keys[j][:]) < 0 })
		for _, k := range keys {
			h.Write(k[:])
			hash(h, entity.Fields[k])
		}
	}
	return h.Sum(nil), nil
}
func collect(v wire.Value, entities map[wire.ID]wire.Entity, q *[]wire.ID) error {
	if v.Tag == 6 {
		if _, ok := entities[v.Reference]; !ok {
			return fmt.Errorf("project_snapshot.reference_missing:%s", v.Reference)
		}
		*q = append(*q, v.Reference)
	}
	if v.Tag == 7 {
		for _, x := range v.List {
			if err := collect(x, entities, q); err != nil {
				return err
			}
		}
	}
	if v.Tag == 8 {
		for _, x := range v.Record {
			if err := collect(x, entities, q); err != nil {
				return err
			}
		}
	}
	return nil
}
func hash(h interface{ Write([]byte) (int, error) }, v wire.Value) {
	h.Write([]byte{v.Tag})
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], v.Unsigned)
	h.Write(n[:])
	binary.BigEndian.PutUint64(n[:], uint64(len(v.Bytes)))
	h.Write(n[:])
	h.Write(v.Bytes)
	if v.Tag == 6 {
		h.Write(v.Reference[:])
	}
	binary.BigEndian.PutUint64(n[:], uint64(len(v.List)))
	h.Write(n[:])
	for _, x := range v.List {
		hash(h, x)
	}
	if v.Tag == 8 {
		keys := make([]wire.ID, 0, len(v.Record))
		for k := range v.Record {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i][:], keys[j][:]) < 0 })
		binary.BigEndian.PutUint64(n[:], uint64(len(keys)))
		h.Write(n[:])
		for _, k := range keys {
			h.Write(k[:])
			hash(h, v.Record[k])
		}
	}
	if v.Tag == 9 {
		h.Write(v.Hole[:])
	}
}
func ref(e wire.Entity, f wire.ID) (wire.ID, error) {
	v, ok := e.Fields[f]
	if !ok || v.Tag != 6 {
		return wire.ID{}, fmt.Errorf("field_kind:%s", f)
	}
	return v.Reference, nil
}
func blob(e wire.Entity, f wire.ID) ([]byte, error) {
	v, ok := e.Fields[f]
	if !ok || v.Tag != 5 {
		return nil, fmt.Errorf("field_kind:%s", f)
	}
	return v.Bytes, nil
}
func refs(e wire.Entity, f wire.ID) ([]wire.ID, error) {
	v, ok := e.Fields[f]
	if !ok || v.Tag != 7 {
		return nil, fmt.Errorf("field_kind:%s", f)
	}
	r := make([]wire.ID, len(v.List))
	for i, x := range v.List {
		if x.Tag != 6 {
			return nil, fmt.Errorf("list_item_kind:%s", f)
		}
		r[i] = x.Reference
	}
	return r, nil
}
func at(x wire.ID, e error) error { return fmt.Errorf("project_snapshot.field:%s:%w", x, e) }
