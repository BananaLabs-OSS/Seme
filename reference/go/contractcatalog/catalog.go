// Package contractcatalog resolves immutable semantic contracts from canonical
// wire envelopes. It deliberately does not compose or reinterpret contracts.
package contractcatalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"seme.local/reference/wire"
)

var (
	moduleSchema    = mustID("00000000000000000000000000000012")
	importSchema    = mustID("00000000000000000000000000000013")
	fImports        = mustID("00000000000000000000000000000121")
	fExports        = mustID("00000000000000000000000000000122")
	fImportMod      = mustID("00000000000000000000000000000130")
	fImportRev      = mustID("00000000000000000000000000000131")
	executionModule = mustID("00000000000000000000000000009000")
	executionRev    = mustID("00000000000000000000000000009023")
	packageModule   = mustID("0000000000000000000000000000b000")
	packageRev      = mustID("0000000000000000000000000000b001")
	projectModule   = mustID("0000000000000000000000000000e000")
	projectRev      = mustID("0000000000000000000000000000e001")
)

type Pin struct{ Module, Revision wire.ID }

type Expectation struct {
	Pin
	ModuleVersion   uint64
	RequiredExports []wire.ID
	Imports         []Pin
	Digest          [sha256.Size]byte
}

type Contract struct {
	envelope  wire.Envelope
	digest    [sha256.Size]byte
	exports   []wire.ID
	imports   []Pin
	validated bool
}

// Resolve accepts only the one canonical byte representation described by e.
func Resolve(source []byte, e Expectation) (Contract, error) {
	graph, err := wire.Decode(source)
	if err != nil {
		return Contract{}, fmt.Errorf("contract_catalog.wire:%w", err)
	}
	canonical, err := wire.Encode(graph)
	if err != nil || !bytes.Equal(canonical, source) {
		return Contract{}, fmt.Errorf("contract_catalog.noncanonical")
	}
	digest := sha256.Sum256(source)
	if e.Digest != ([sha256.Size]byte{}) && digest != e.Digest {
		return Contract{}, fmt.Errorf("contract_catalog.digest")
	}
	if graph.Module != e.Module || graph.Revision != e.Revision {
		return Contract{}, fmt.Errorf("contract_catalog.pin")
	}
	module, ok := graph.Entities[e.Module]
	if !ok || module.Schema != moduleSchema || module.Version != e.ModuleVersion {
		return Contract{}, fmt.Errorf("contract_catalog.module")
	}

	exports, err := typedRefs(module, fExports, false)
	if err != nil {
		return Contract{}, fmt.Errorf("contract_catalog.exports:%w", err)
	}
	exported := make(map[wire.ID]bool, len(exports))
	for _, id := range exports {
		item, ok := graph.Entities[id]
		if !ok || item.Schema == importSchema {
			return Contract{}, fmt.Errorf("contract_catalog.export_target:%s", id)
		}
		exported[id] = true
	}
	for _, id := range e.RequiredExports {
		if !exported[id] {
			return Contract{}, fmt.Errorf("contract_catalog.required_export:%s", id)
		}
	}

	importIDs, err := typedRefs(module, fImports, true)
	if err != nil {
		return Contract{}, fmt.Errorf("contract_catalog.imports:%w", err)
	}
	listed := make(map[wire.ID]bool, len(importIDs))
	imports := make([]Pin, 0, len(importIDs))
	for _, id := range importIDs {
		listed[id] = true
		item, ok := graph.Entities[id]
		if !ok || item.Schema != importSchema || item.Version != 1 {
			return Contract{}, fmt.Errorf("contract_catalog.import_target:%s", id)
		}
		moduleValue, mok := item.Fields[fImportMod]
		revisionValue, rok := item.Fields[fImportRev]
		if !mok || moduleValue.Tag != 6 || !rok || revisionValue.Tag != 5 || len(revisionValue.Bytes) != len(wire.ID{}) {
			return Contract{}, fmt.Errorf("contract_catalog.import_shape:%s", id)
		}
		var revision wire.ID
		copy(revision[:], revisionValue.Bytes)
		imports = append(imports, Pin{Module: moduleValue.Reference, Revision: revision})
	}
	for id, item := range graph.Entities {
		if item.Schema == importSchema && !listed[id] {
			return Contract{}, fmt.Errorf("contract_catalog.orphan_import:%s", id)
		}
	}
	if !samePins(imports, e.Imports) {
		return Contract{}, fmt.Errorf("contract_catalog.import_pins")
	}
	return Contract{envelope: graph, digest: digest, exports: append([]wire.ID(nil), exports...), imports: append([]Pin(nil), imports...), validated: true}, nil
}

// Validated reports whether this value was produced by Resolve rather than
// assembled as an unchecked struct literal.
func (c Contract) Validated() bool           { return c.validated }
func (c Contract) Pin() Pin                  { return Pin{Module: c.envelope.Module, Revision: c.envelope.Revision} }
func (c Contract) Digest() [sha256.Size]byte { return c.digest }
func (c Contract) Exports() []wire.ID        { return append([]wire.ID(nil), c.exports...) }
func (c Contract) Imports() []Pin            { return append([]Pin(nil), c.imports...) }
func (c Contract) Envelope() wire.Envelope   { return cloneEnvelope(c.envelope) }

func cloneEnvelope(in wire.Envelope) wire.Envelope {
	out := wire.Envelope{Module: in.Module, Revision: in.Revision, Parents: append([]wire.ID(nil), in.Parents...), Entities: make(map[wire.ID]wire.Entity, len(in.Entities))}
	for eid, e := range in.Entities {
		fields := make(map[wire.ID]wire.Value, len(e.Fields))
		for fid, v := range e.Fields {
			fields[fid] = cloneValue(v)
		}
		e.Fields = fields
		out.Entities[eid] = e
	}
	return out
}
func cloneValue(v wire.Value) wire.Value {
	v.Bytes = append([]byte(nil), v.Bytes...)
	v.List = append([]wire.Value(nil), v.List...)
	for i := range v.List {
		v.List[i] = cloneValue(v.List[i])
	}
	if v.Record != nil {
		r := make(map[wire.ID]wire.Value, len(v.Record))
		for k, x := range v.Record {
			r[k] = cloneValue(x)
		}
		v.Record = r
	}
	return v
}

func typedRefs(entity wire.Entity, field wire.ID, optional bool) ([]wire.ID, error) {
	v, ok := entity.Fields[field]
	if !ok {
		if optional {
			return nil, nil
		}
		return nil, fmt.Errorf("missing")
	}
	if v.Tag != 7 {
		return nil, fmt.Errorf("not_list")
	}
	seen := map[wire.ID]bool{}
	out := make([]wire.ID, 0, len(v.List))
	for _, item := range v.List {
		if item.Tag != 6 {
			return nil, fmt.Errorf("not_reference")
		}
		if seen[item.Reference] {
			return nil, fmt.Errorf("duplicate:%s", item.Reference)
		}
		seen[item.Reference] = true
		out = append(out, item.Reference)
	}
	return out, nil
}

func samePins(a, b []Pin) bool {
	if len(a) != len(b) {
		return false
	}
	used := make([]bool, len(b))
	for _, x := range a {
		found := false
		for i, y := range b {
			if !used[i] && x == y {
				used[i], found = true, true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

type ProjectContractSet struct {
	execution Contract
	packages  Contract
	project   Contract
	validated bool
}

func (s ProjectContractSet) Validated() bool     { return s.validated }
func (s ProjectContractSet) Execution() Contract { return s.execution }
func (s ProjectContractSet) Package() Contract   { return s.packages }
func (s ProjectContractSet) Project() Contract   { return s.project }

func ResolveProjectContractSet(execution, packages, project []byte) (ProjectContractSet, error) {
	execPin := Pin{executionModule, executionRev}
	packagePin := Pin{packageModule, packageRev}
	projectPin := Pin{projectModule, projectRev}
	x, err := Resolve(execution, Expectation{Pin: execPin, ModuleVersion: 35, RequiredExports: []wire.ID{mustID("00000000000000000000000000009015")}, Digest: mustDigest("54fdd39b5d78f7f12da37fd43505e0a7962bad9d9808b54a16c20e4cf95e736a")})
	if err != nil {
		return ProjectContractSet{}, fmt.Errorf("execution:%w", err)
	}
	p, err := Resolve(packages, Expectation{Pin: packagePin, ModuleVersion: 1, RequiredExports: []wire.ID{mustID("0000000000000000000000000000b010"), mustID("0000000000000000000000000000b011"), mustID("0000000000000000000000000000b012")}, Digest: mustDigest("9f9b6e7f72a00ab069c56e80a9021abc4c59e2d0bde579ab29bda7350c506bf6")})
	if err != nil {
		return ProjectContractSet{}, fmt.Errorf("package:%w", err)
	}
	r, err := Resolve(project, Expectation{Pin: projectPin, ModuleVersion: 1, RequiredExports: []wire.ID{mustID("0000000000000000000000000000e010"), mustID("0000000000000000000000000000e011")}, Imports: []Pin{packagePin, execPin}, Digest: mustDigest("5c8f22e9fee59c378f77ebdfd171d6220ccde8f3e1aafde0400830880e005dcc")})
	if err != nil {
		return ProjectContractSet{}, fmt.Errorf("project:%w", err)
	}
	return ProjectContractSet{execution: x, packages: p, project: r, validated: true}, nil
}

func mustID(s string) wire.ID {
	id, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return id
}
func mustDigest(s string) (out [sha256.Size]byte) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != len(out) {
		panic(s)
	}
	copy(out[:], b)
	return
}
