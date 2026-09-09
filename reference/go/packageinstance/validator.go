// Package packageinstance validates language-neutral Package Contract v1 instances.
package packageinstance

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"

	"seme.local/reference/wire"
)

var (
	moduleSchema     = id("00000000000000000000000000000012")
	importSchema     = id("00000000000000000000000000000013")
	packageModule    = id("0000000000000000000000000000b000")
	packageRevision  = id("0000000000000000000000000000b001")
	packageSchema    = id("0000000000000000000000000000b010")
	interfaceSchema  = id("0000000000000000000000000000b011")
	dependencySchema = id("0000000000000000000000000000b012")
	runtimeSchema    = id("0000000000000000000000000000b013")
	fidelitySchema   = id("0000000000000000000000000000b014")
	functionSchema   = id("00000000000000000000000000009011")
	parameterSchema  = id("00000000000000000000000000009012")
	fImports         = id("00000000000000000000000000000121")
	fImportModule    = id("00000000000000000000000000000130")
	fImportRevision  = id("00000000000000000000000000000131")
)

func id(s string) wire.ID {
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
func fid(n uint64) wire.ID { return id(fmt.Sprintf("%032x", n)) }

// Validate checks all Package v1 instances in canonical wire bytes.
func Validate(source []byte) error {
	e, err := wire.Decode(source)
	if err != nil {
		return err
	}
	return ValidateEnvelope(e)
}

func ValidateEnvelope(e wire.Envelope) error {
	m, ok := e.Entities[e.Module]
	if !ok || m.Schema != moduleSchema {
		return fmt.Errorf("package_instance.module_declaration")
	}
	imports, err := refs(m, fImports)
	if err != nil {
		return fmt.Errorf("package_instance.module_imports")
	}
	pinned := false
	for _, x := range imports {
		q := e.Entities[x]
		if q.Schema == importSchema {
			a, ae := ref(q, fImportModule)
			b, be := blob(q, fImportRevision)
			if ae == nil && be == nil && a == packageModule && bytes.Equal(b, packageRevision[:]) {
				pinned = true
			}
		}
	}
	if !pinned {
		return fmt.Errorf("package_instance.package_import_unpinned")
	}
	if err := validateDependencyDAG(e); err != nil {
		return err
	}
	owners := map[wire.ID]wire.ID{}
	count := 0
	for pid, p := range e.Entities {
		if p.Schema != packageSchema {
			continue
		}
		count++
		if p.Version != 1 {
			return fmt.Errorf("package_instance.version:%s", pid)
		}
		if err := exact(p, 0xb100, 0xb101, 0xb102, 0xb103, 0xb104, 0xb105, 0xb106); err != nil {
			return at(pid, err)
		}
		if name, ne := blob(p, fid(0xb100)); ne != nil || len(name) == 0 {
			if ne != nil {
				return at(pid, ne)
			}
			return fmt.Errorf("package_instance.name_empty:%s", pid)
		}
		version, er := blob(p, fid(0xb101))
		if er != nil {
			return at(pid, er)
		}
		groups := []struct {
			field  uint64
			schema wire.ID
			any    bool
		}{{0xb102, interfaceSchema, false}, {0xb103, dependencySchema, false}, {0xb104, wire.ID{}, true}, {0xb105, runtimeSchema, false}, {0xb106, fidelitySchema, false}}
		for _, g := range groups {
			xs, xer := refs(p, fid(g.field))
			if xer != nil {
				return at(pid, xer)
			}
			if !sortedUnique(xs) {
				return fmt.Errorf("package_instance.refs_unsorted:%s:%x", pid, g.field)
			}
			for _, x := range xs {
				target, exists := e.Entities[x]
				if !exists {
					return fmt.Errorf("package_instance.target_missing:%s", x)
				}
				if !g.any && target.Schema != g.schema {
					return fmt.Errorf("package_instance.target_schema:%s", x)
				}
				if g.schema == interfaceSchema {
					fn, ve := validateInterface(e, target)
					if ve != nil {
						return at(x, ve)
					}
					if prior, yes := owners[fn]; yes && prior != pid {
						return fmt.Errorf("package_instance.function_owned_twice:%s", fn)
					}
					owners[fn] = pid
				}
				if g.schema == dependencySchema {
					if err := validateDependency(target); err != nil {
						return at(x, err)
					}
				}
				if g.schema == runtimeSchema {
					if err := shape(target, runtimeSchema, []uint64{0xb130, 0xb131}, []byte{5, 5}); err != nil {
						return at(x, err)
					}
				}
				if g.schema == fidelitySchema {
					if err := validateFidelity(e, target); err != nil {
						return at(x, err)
					}
				}
			}
		}
		want, xer := Revision(e, pid)
		if xer != nil {
			return at(pid, xer)
		}
		if !bytes.Equal(version, want) {
			return fmt.Errorf("package_instance.revision_mismatch:%s", pid)
		}
	}
	if count == 0 {
		return fmt.Errorf("package_instance.package_missing")
	}
	return nil
}

func validateInterface(e wire.Envelope, x wire.Entity) (wire.ID, error) {
	if x.Version != 1 {
		return wire.ID{}, fmt.Errorf("version")
	}
	if err := exact(x, 0xb110, 0xb111, 0xb112, 0xb113); err != nil {
		return wire.ID{}, err
	}
	if _, err := blob(x, fid(0xb110)); err != nil {
		return wire.ID{}, err
	}
	fn, err := ref(x, fid(0xb111))
	if err != nil {
		return wire.ID{}, err
	}
	decl, ok := e.Entities[fn]
	if !ok || decl.Schema != functionSchema || decl.Version != 1 {
		return wire.ID{}, fmt.Errorf("interface.function_schema")
	}
	if err := shape(decl, functionSchema, []uint64{0x9110, 0x9111, 0x9112, 0x9113}, []byte{5, 7, 6, 6}); err != nil {
		return wire.ID{}, fmt.Errorf("interface.function_shape:%w", err)
	}
	advertised, err := refs(x, fid(0xb112))
	if err != nil {
		return wire.ID{}, err
	}
	params, err := refs(decl, fid(0x9111))
	if err != nil {
		return wire.ID{}, fmt.Errorf("interface.function_parameters")
	}
	actual := make([]wire.ID, len(params))
	for i, p := range params {
		q, yes := e.Entities[p]
		if !yes || q.Schema != parameterSchema || q.Version != 1 {
			return wire.ID{}, fmt.Errorf("interface.parameter_schema")
		}
		if err := shape(q, parameterSchema, []uint64{0x9120, 0x9121, 0x9122}, []byte{5, 6, 3}); err != nil {
			return wire.ID{}, fmt.Errorf("interface.parameter_shape:%w", err)
		}
		actual[i], err = ref(q, fid(0x9121))
		if err != nil {
			return wire.ID{}, fmt.Errorf("interface.parameter_type")
		}
	}
	if !equalIDs(advertised, actual) {
		return wire.ID{}, fmt.Errorf("interface.parameter_mismatch")
	}
	a, err := ref(x, fid(0xb113))
	if err != nil {
		return wire.ID{}, err
	}
	b, err := ref(decl, fid(0x9112))
	if err != nil || a != b {
		return wire.ID{}, fmt.Errorf("interface.result_mismatch")
	}
	return fn, nil
}
func validateDependency(x wire.Entity) error {
	if x.Version != 1 {
		return fmt.Errorf("version")
	}
	return shape(x, dependencySchema, []uint64{0xb120, 0xb121, 0xb122}, []byte{5, 5, 6})
}
func validateFidelity(e wire.Envelope, x wire.Entity) error {
	if x.Version != 1 {
		return fmt.Errorf("version")
	}
	if err := shape(x, fidelitySchema, []uint64{0xb140, 0xb141, 0xb142, 0xb143}, []byte{6, 6, 3, 5}); err != nil {
		return err
	}
	for _, f := range []uint64{0xb140, 0xb141} {
		r, _ := ref(x, fid(f))
		if _, ok := e.Entities[r]; !ok {
			return fmt.Errorf("fidelity.target_missing")
		}
	}
	return nil
}

// Revision returns SHA-256 over the package and its transitively referenced
// semantic content, excluding package.version itself to avoid self-reference.
func Revision(e wire.Envelope, root wire.ID) ([]byte, error) {
	if e.Entities[root].Schema != packageSchema {
		return nil, fmt.Errorf("package_instance.not_package")
	}
	if err := visitDependency(e, root, map[wire.ID]bool{}, nil); err != nil {
		return nil, err
	}
	closure := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	seen := map[wire.ID]bool{}
	queue := []wire.ID{root}
	for len(queue) > 0 {
		x := queue[0]
		queue = queue[1:]
		if seen[x] {
			continue
		}
		q, ok := e.Entities[x]
		if !ok {
			return nil, fmt.Errorf("package_instance.digest_missing:%s", x)
		}
		seen[x] = true
		fields := make(map[wire.ID]wire.Value, len(q.Fields))
		for field, value := range q.Fields {
			if x == root && field == fid(0xb101) {
				continue
			}
			fields[field] = value
			collectReferences(value, &queue)
		}
		q.Fields = fields
		closure.Entities[x] = q
	}
	canonical, err := wire.Encode(closure)
	if err != nil {
		return nil, fmt.Errorf("package_instance.digest_wire:%w", err)
	}
	h := sha256.New()
	h.Write([]byte("seme-package-instance-v1\x00"))
	h.Write(canonical)
	return h.Sum(nil), nil
}

func validateDependencyDAG(e wire.Envelope) error {
	packages := make([]wire.ID, 0)
	for x, q := range e.Entities {
		if q.Schema == packageSchema {
			packages = append(packages, x)
		}
	}
	sort.Slice(packages, func(i, j int) bool { return bytes.Compare(packages[i][:], packages[j][:]) < 0 })
	done := map[wire.ID]bool{}
	for _, x := range packages {
		if err := visitDependency(e, x, map[wire.ID]bool{}, done); err != nil {
			return err
		}
	}
	return nil
}

func visitDependency(e wire.Envelope, x wire.ID, active, done map[wire.ID]bool) error {
	if done != nil && done[x] {
		return nil
	}
	if active[x] {
		return fmt.Errorf("package_instance.dependency_cycle:%s", x)
	}
	p, ok := e.Entities[x]
	if !ok || p.Schema != packageSchema {
		return fmt.Errorf("package_instance.dependency_package_schema:%s", x)
	}
	deps, err := refs(p, fid(0xb103))
	if err != nil {
		return err
	}
	if !sortedUnique(deps) {
		return fmt.Errorf("package_instance.refs_unsorted:%s:%x", x, 0xb103)
	}
	next := make(map[wire.ID]bool, len(active)+1)
	for k, v := range active {
		next[k] = v
	}
	next[x] = true
	for _, d := range deps {
		q, yes := e.Entities[d]
		if !yes || q.Schema != dependencySchema {
			return fmt.Errorf("package_instance.dependency_schema:%s", d)
		}
		target, er := ref(q, fid(0xb122))
		if er != nil {
			return er
		}
		if er = visitDependency(e, target, next, done); er != nil {
			return er
		}
	}
	if done != nil {
		done[x] = true
	}
	return nil
}
func collectReferences(v wire.Value, q *[]wire.ID) {
	if v.Tag == 6 {
		*q = append(*q, v.Reference)
	}
	for _, x := range v.List {
		collectReferences(x, q)
	}
	for _, x := range v.Record {
		collectReferences(x, q)
	}
}
func shape(x wire.Entity, s wire.ID, fs []uint64, tags []byte) error {
	if x.Schema != s {
		return fmt.Errorf("schema")
	}
	if x.Version != 1 {
		return fmt.Errorf("version")
	}
	ids := make([]wire.ID, len(fs))
	for i, n := range fs {
		ids[i] = fid(n)
	}
	if err := exactIDs(x, ids); err != nil {
		return err
	}
	for i, f := range ids {
		if x.Fields[f].Tag != tags[i] {
			return fmt.Errorf("field_kind:%s", f)
		}
	}
	return nil
}
func exact(x wire.Entity, ns ...uint64) error {
	ids := make([]wire.ID, len(ns))
	for i, n := range ns {
		ids[i] = fid(n)
	}
	return exactIDs(x, ids)
}
func exactIDs(x wire.Entity, fs []wire.ID) error {
	if len(x.Fields) != len(fs) {
		return fmt.Errorf("field_count")
	}
	for _, f := range fs {
		if _, ok := x.Fields[f]; !ok {
			return fmt.Errorf("field_missing:%s", f)
		}
	}
	return nil
}
func refs(x wire.Entity, f wire.ID) ([]wire.ID, error) {
	v, ok := x.Fields[f]
	if !ok || v.Tag != 7 {
		return nil, fmt.Errorf("field_kind:%s", f)
	}
	out := make([]wire.ID, len(v.List))
	for i, q := range v.List {
		if q.Tag != 6 {
			return nil, fmt.Errorf("list_kind:%s", f)
		}
		out[i] = q.Reference
	}
	return out, nil
}
func ref(x wire.Entity, f wire.ID) (wire.ID, error) {
	v, ok := x.Fields[f]
	if !ok || v.Tag != 6 {
		return wire.ID{}, fmt.Errorf("field_kind:%s", f)
	}
	return v.Reference, nil
}
func blob(x wire.Entity, f wire.ID) ([]byte, error) {
	v, ok := x.Fields[f]
	if !ok || v.Tag != 5 {
		return nil, fmt.Errorf("field_kind:%s", f)
	}
	return v.Bytes, nil
}
func sortedUnique(xs []wire.ID) bool {
	for i := 1; i < len(xs); i++ {
		if bytes.Compare(xs[i-1][:], xs[i][:]) >= 0 {
			return false
		}
	}
	return true
}
func equalIDs(a, b []wire.ID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func at(x wire.ID, e error) error { return fmt.Errorf("package_instance.field:%s:%w", x, e) }
