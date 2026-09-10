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
	moduleSchema        = mustID("00000000000000000000000000000012")
	importSchema        = mustID("00000000000000000000000000000013")
	fImports            = mustID("00000000000000000000000000000121")
	fExports            = mustID("00000000000000000000000000000122")
	fImportMod          = mustID("00000000000000000000000000000130")
	fImportRev          = mustID("00000000000000000000000000000131")
	executionModule     = mustID("00000000000000000000000000009000")
	executionRev        = mustID("00000000000000000000000000009023")
	packageModule       = mustID("0000000000000000000000000000b000")
	packageRev          = mustID("0000000000000000000000000000b001")
	packageRevV2        = mustID("0000000000000000000000000000b002")
	packageRevV3        = mustID("0000000000000000000000000000b003")
	projectModule       = mustID("0000000000000000000000000000e000")
	projectRev          = mustID("0000000000000000000000000000e001")
	projectRevV2        = mustID("0000000000000000000000000000e002")
	projectRevV3        = mustID("0000000000000000000000000000e003")
	projectRevV4        = mustID("0000000000000000000000000000e004")
	projectRevV5        = mustID("0000000000000000000000000000e005")
	projectRevV6        = mustID("0000000000000000000000000000e008")
	projectRevV7        = mustID("0000000000000000000000000000e009")
	dependencyModule    = mustID("0000000000000000000000000000f000")
	dependencyRev       = mustID("0000000000000000000000000000f001")
	configurationModule = mustID("00000000000000000000000000004000")
	configurationRev    = mustID("00000000000000000000000000004001")
	configurationRevV2  = mustID("00000000000000000000000000004005")
	foundationModule    = mustID("00000000000000000000000000003000")
	foundationRev       = mustID("00000000000000000000000000003001")
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

// ResolveDependencyContract authenticates the exact Dependency Contract v1.
func ResolveDependencyContract(source []byte) (Contract, error) {
	return Resolve(source, Expectation{Pin: Pin{dependencyModule, dependencyRev}, ModuleVersion: 1, RequiredExports: ids("f010", "f011", "f012", "f013", "f014", "f015", "f016", "f017"), Digest: mustDigest("167cc9a93239db97075d064f0f008edae194bc79e8e9391e2e97958b345be024")})
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

// ProjectContractSetV4 adds the independently authenticated Dependency
// contract without weakening or changing the established v1-v3 set APIs.
type ProjectContractSetV4 struct {
	execution, packages, dependency, project Contract
	validated                                bool
}

// ProjectContractSetV5 authenticates the complete semantic declaration
// ownership graph without changing the established v1-v4 APIs.
type ProjectContractSetV5 struct {
	execution, packages, dependency, project Contract
	validated                                bool
}

type ProjectContractSetV6 struct {
	execution, packages, dependency, foundation, configuration, project Contract
	validated                                                           bool
}

type ProjectContractSetV7 struct {
	execution, packages, dependency, foundation, configuration, project Contract
	validated                                                           bool
}

func (s ProjectContractSetV7) Validated() bool         { return s.validated }
func (s ProjectContractSetV7) Execution() Contract     { return s.execution }
func (s ProjectContractSetV7) Package() Contract       { return s.packages }
func (s ProjectContractSetV7) Dependency() Contract    { return s.dependency }
func (s ProjectContractSetV7) Foundation() Contract    { return s.foundation }
func (s ProjectContractSetV7) Configuration() Contract { return s.configuration }
func (s ProjectContractSetV7) Project() Contract       { return s.project }

func ResolveProjectContractSetV7(foundation, execution, packages, dependency, configuration, project []byte) (ProjectContractSetV7, error) {
	foundationPin := Pin{foundationModule, foundationRev}
	execPin, packagePin := Pin{executionModule, executionRev}, Pin{packageModule, packageRevV3}
	dependencyPin, configurationPin, projectPin := Pin{dependencyModule, dependencyRev}, Pin{configurationModule, configurationRevV2}, Pin{projectModule, projectRevV7}
	f, err := Resolve(foundation, Expectation{Pin: foundationPin, ModuleVersion: 1, RequiredExports: ids("16"), Digest: mustDigest("bbb42f8d71f8c537713a478514f74f79cef60ba370b1a2e10525f631d922dd5d")})
	if err != nil {
		return ProjectContractSetV7{}, fmt.Errorf("foundation:%w", err)
	}
	x, err := Resolve(execution, Expectation{Pin: execPin, ModuleVersion: 35, RequiredExports: ids("9015"), Digest: mustDigest("54fdd39b5d78f7f12da37fd43505e0a7962bad9d9808b54a16c20e4cf95e736a")})
	if err != nil {
		return ProjectContractSetV7{}, fmt.Errorf("execution:%w", err)
	}
	p, err := Resolve(packages, Expectation{Pin: packagePin, ModuleVersion: 3, RequiredExports: ids("b010", "b011", "b012", "b013", "b014", "b020", "b021", "b022", "b023", "b024", "b025", "b026", "b027", "b028", "b029"), Digest: mustDigest("d05d708091bd5594f059d00ec51d083c2c694bc07015befb92cc0fc1a4f30f37")})
	if err != nil {
		return ProjectContractSetV7{}, fmt.Errorf("package:%w", err)
	}
	d, err := Resolve(dependency, Expectation{Pin: dependencyPin, ModuleVersion: 1, RequiredExports: ids("f010", "f011", "f012", "f013", "f014", "f015", "f016", "f017"), Digest: mustDigest("167cc9a93239db97075d064f0f008edae194bc79e8e9391e2e97958b345be024")})
	if err != nil {
		return ProjectContractSetV7{}, fmt.Errorf("dependency:%w", err)
	}
	c, err := Resolve(configuration, Expectation{Pin: configurationPin, ModuleVersion: 2, RequiredExports: ids("4010", "4011", "4012", "4013", "4014", "4015", "4016", "4017", "4018", "4019", "401a", "401b", "401c", "401d", "401e", "401f"), Imports: []Pin{packagePin, execPin, foundationPin}, Digest: mustDigest("8c10667d15d7ecb30f9567dc63af61a0fcd4e947993d2854eef27319c492cadf")})
	if err != nil {
		return ProjectContractSetV7{}, fmt.Errorf("configuration:%w", err)
	}
	r, err := Resolve(project, Expectation{Pin: projectPin, ModuleVersion: 7, RequiredExports: ids("e010", "e011", "e012", "e013", "e014", "e015", "e016", "e017", "e018", "e019", "e020", "e021", "e022"), Imports: []Pin{packagePin, execPin, dependencyPin, configurationPin}, Digest: mustDigest("68b5a7fa7412e88261938c701e3209c48041d8b1ccced46fb3ccd90f00e6885d")})
	if err != nil {
		return ProjectContractSetV7{}, fmt.Errorf("project:%w", err)
	}
	return ProjectContractSetV7{execution: x, packages: p, dependency: d, foundation: f, configuration: c, project: r, validated: true}, nil
}

func (s ProjectContractSetV6) Validated() bool         { return s.validated }
func (s ProjectContractSetV6) Execution() Contract     { return s.execution }
func (s ProjectContractSetV6) Package() Contract       { return s.packages }
func (s ProjectContractSetV6) Dependency() Contract    { return s.dependency }
func (s ProjectContractSetV6) Foundation() Contract    { return s.foundation }
func (s ProjectContractSetV6) Configuration() Contract { return s.configuration }
func (s ProjectContractSetV6) Project() Contract       { return s.project }

func ResolveProjectContractSetV6(foundation, execution, packages, dependency, configuration, project []byte) (ProjectContractSetV6, error) {
	foundationPin := Pin{foundationModule, foundationRev}
	execPin, packagePin := Pin{executionModule, executionRev}, Pin{packageModule, packageRevV3}
	dependencyPin, configurationPin, projectPin := Pin{dependencyModule, dependencyRev}, Pin{configurationModule, configurationRev}, Pin{projectModule, projectRevV6}
	f, err := Resolve(foundation, Expectation{Pin: foundationPin, ModuleVersion: 1, RequiredExports: ids("16"), Digest: mustDigest("bbb42f8d71f8c537713a478514f74f79cef60ba370b1a2e10525f631d922dd5d")})
	if err != nil {
		return ProjectContractSetV6{}, fmt.Errorf("foundation:%w", err)
	}
	x, err := Resolve(execution, Expectation{Pin: execPin, ModuleVersion: 35, RequiredExports: ids("9015"), Digest: mustDigest("54fdd39b5d78f7f12da37fd43505e0a7962bad9d9808b54a16c20e4cf95e736a")})
	if err != nil {
		return ProjectContractSetV6{}, fmt.Errorf("execution:%w", err)
	}
	p, err := Resolve(packages, Expectation{Pin: packagePin, ModuleVersion: 3, RequiredExports: ids("b010", "b011", "b012", "b013", "b014", "b020", "b021", "b022", "b023", "b024", "b025", "b026", "b027", "b028", "b029"), Digest: mustDigest("d05d708091bd5594f059d00ec51d083c2c694bc07015befb92cc0fc1a4f30f37")})
	if err != nil {
		return ProjectContractSetV6{}, fmt.Errorf("package:%w", err)
	}
	d, err := Resolve(dependency, Expectation{Pin: dependencyPin, ModuleVersion: 1, RequiredExports: ids("f010", "f011", "f012", "f013", "f014", "f015", "f016", "f017"), Digest: mustDigest("167cc9a93239db97075d064f0f008edae194bc79e8e9391e2e97958b345be024")})
	if err != nil {
		return ProjectContractSetV6{}, fmt.Errorf("dependency:%w", err)
	}
	c, err := Resolve(configuration, Expectation{Pin: configurationPin, ModuleVersion: 1, RequiredExports: ids("4010", "4011", "4012", "4013", "4014", "4015", "4016", "4017"), Imports: []Pin{packagePin, execPin, foundationPin}, Digest: mustDigest("141ffd47fa43744765fda76d1f11835dd71deb59ba7101fa925f6024adf4e397")})
	if err != nil {
		return ProjectContractSetV6{}, fmt.Errorf("configuration:%w", err)
	}
	r, err := Resolve(project, Expectation{Pin: projectPin, ModuleVersion: 6, RequiredExports: ids("e010", "e011", "e012", "e013", "e014", "e015", "e016", "e017", "e018", "e019", "e020", "e021"), Imports: []Pin{packagePin, execPin, dependencyPin, configurationPin}, Digest: mustDigest("d1552035f061e17c5e8fdaea5f0664b55549e0381fe2399c3909637f2627cd38")})
	if err != nil {
		return ProjectContractSetV6{}, fmt.Errorf("project:%w", err)
	}
	return ProjectContractSetV6{execution: x, packages: p, dependency: d, foundation: f, configuration: c, project: r, validated: true}, nil
}

func (s ProjectContractSetV5) Validated() bool      { return s.validated }
func (s ProjectContractSetV5) Execution() Contract  { return s.execution }
func (s ProjectContractSetV5) Package() Contract    { return s.packages }
func (s ProjectContractSetV5) Dependency() Contract { return s.dependency }
func (s ProjectContractSetV5) Project() Contract    { return s.project }

func ResolveProjectContractSetV5(execution, packages, dependency, project []byte) (ProjectContractSetV5, error) {
	execPin, packagePin := Pin{executionModule, executionRev}, Pin{packageModule, packageRevV3}
	dependencyPin, projectPin := Pin{dependencyModule, dependencyRev}, Pin{projectModule, projectRevV5}
	x, err := Resolve(execution, Expectation{Pin: execPin, ModuleVersion: 35, RequiredExports: []wire.ID{mustID("00000000000000000000000000009015")}, Digest: mustDigest("54fdd39b5d78f7f12da37fd43505e0a7962bad9d9808b54a16c20e4cf95e736a")})
	if err != nil {
		return ProjectContractSetV5{}, fmt.Errorf("execution:%w", err)
	}
	p, err := Resolve(packages, Expectation{Pin: packagePin, ModuleVersion: 3, RequiredExports: ids("b010", "b011", "b012", "b013", "b014", "b020", "b021", "b022", "b023", "b024", "b025", "b026", "b027", "b028", "b029"), Digest: mustDigest("d05d708091bd5594f059d00ec51d083c2c694bc07015befb92cc0fc1a4f30f37")})
	if err != nil {
		return ProjectContractSetV5{}, fmt.Errorf("package:%w", err)
	}
	d, err := Resolve(dependency, Expectation{Pin: dependencyPin, ModuleVersion: 1, RequiredExports: ids("f010", "f011", "f012", "f013", "f014", "f015", "f016", "f017"), Digest: mustDigest("167cc9a93239db97075d064f0f008edae194bc79e8e9391e2e97958b345be024")})
	if err != nil {
		return ProjectContractSetV5{}, fmt.Errorf("dependency:%w", err)
	}
	r, err := Resolve(project, Expectation{Pin: projectPin, ModuleVersion: 5, RequiredExports: ids("e010", "e011", "e012", "e013", "e014", "e015", "e016", "e017", "e018", "e019", "e020"), Imports: []Pin{packagePin, execPin, dependencyPin}, Digest: mustDigest("37279f2918a76abfb0eaeeb1ff62cb527dda40a27421a504cfa10003ff7e2641")})
	if err != nil {
		return ProjectContractSetV5{}, fmt.Errorf("project:%w", err)
	}
	return ProjectContractSetV5{execution: x, packages: p, dependency: d, project: r, validated: true}, nil
}

func (s ProjectContractSetV4) Validated() bool      { return s.validated }
func (s ProjectContractSetV4) Execution() Contract  { return s.execution }
func (s ProjectContractSetV4) Package() Contract    { return s.packages }
func (s ProjectContractSetV4) Dependency() Contract { return s.dependency }
func (s ProjectContractSetV4) Project() Contract    { return s.project }

// ResolveProjectContractSetV4 authenticates Project v4 and all three exact
// contracts it imports.
func ResolveProjectContractSetV4(execution, packages, dependency, project []byte) (ProjectContractSetV4, error) {
	execPin := Pin{executionModule, executionRev}
	packagePin := Pin{packageModule, packageRevV2}
	dependencyPin := Pin{dependencyModule, dependencyRev}
	projectPin := Pin{projectModule, projectRevV4}
	x, err := Resolve(execution, Expectation{Pin: execPin, ModuleVersion: 35, RequiredExports: []wire.ID{mustID("00000000000000000000000000009015")}, Digest: mustDigest("54fdd39b5d78f7f12da37fd43505e0a7962bad9d9808b54a16c20e4cf95e736a")})
	if err != nil {
		return ProjectContractSetV4{}, fmt.Errorf("execution:%w", err)
	}
	p, err := Resolve(packages, Expectation{Pin: packagePin, ModuleVersion: 2, RequiredExports: ids("b010", "b011", "b012", "b013", "b014", "b020", "b021", "b022", "b023", "b024", "b025", "b026"), Digest: mustDigest("f65c1ff583e3d7b2504e350d8b6a6dd3c00174e60de23a8dbcc73a4ae86f763b")})
	if err != nil {
		return ProjectContractSetV4{}, fmt.Errorf("package:%w", err)
	}
	d, err := Resolve(dependency, Expectation{Pin: dependencyPin, ModuleVersion: 1, RequiredExports: ids("f010", "f011", "f012", "f013", "f014", "f015", "f016", "f017"), Digest: mustDigest("167cc9a93239db97075d064f0f008edae194bc79e8e9391e2e97958b345be024")})
	if err != nil {
		return ProjectContractSetV4{}, fmt.Errorf("dependency:%w", err)
	}
	r, err := Resolve(project, Expectation{Pin: projectPin, ModuleVersion: 4, RequiredExports: ids("e010", "e011", "e012", "e013", "e014", "e015", "e016", "e017", "e018", "e019"), Imports: []Pin{packagePin, execPin, dependencyPin}, Digest: mustDigest("2bdb19577c73610337db0c60c3346f1bc9fe0fe8f6651b411e0522601fc0c886")})
	if err != nil {
		return ProjectContractSetV4{}, fmt.Errorf("project:%w", err)
	}
	return ProjectContractSetV4{execution: x, packages: p, dependency: d, project: r, validated: true}, nil
}

func (s ProjectContractSet) Validated() bool     { return s.validated }
func (s ProjectContractSet) Execution() Contract { return s.execution }
func (s ProjectContractSet) Package() Contract   { return s.packages }
func (s ProjectContractSet) Project() Contract   { return s.project }

func ResolveProjectContractSet(execution, packages, project []byte) (ProjectContractSet, error) {
	return resolveProjectContractSet(execution, packages, project, false)
}

// ResolveProjectContractSetV2 resolves the source-inventory revision without
// changing the v1 resolver used by executable Project artifacts.
func ResolveProjectContractSetV2(execution, packages, project []byte) (ProjectContractSet, error) {
	return resolveProjectContractSet(execution, packages, project, true)
}

// ResolveProjectContractSetV3 resolves the Project v3 graph-binding contract
// and its exact Package v2 and Execution v35 dependencies.
func ResolveProjectContractSetV3(execution, packages, project []byte) (ProjectContractSet, error) {
	execPin := Pin{executionModule, executionRev}
	packagePin := Pin{packageModule, packageRevV2}
	x, err := Resolve(execution, Expectation{Pin: execPin, ModuleVersion: 35, RequiredExports: []wire.ID{mustID("00000000000000000000000000009015")}, Digest: mustDigest("54fdd39b5d78f7f12da37fd43505e0a7962bad9d9808b54a16c20e4cf95e736a")})
	if err != nil {
		return ProjectContractSet{}, fmt.Errorf("execution:%w", err)
	}
	p, err := Resolve(packages, Expectation{Pin: packagePin, ModuleVersion: 2, RequiredExports: ids("b010", "b011", "b012", "b013", "b014", "b020", "b021", "b022", "b023", "b024", "b025", "b026"), Digest: mustDigest("f65c1ff583e3d7b2504e350d8b6a6dd3c00174e60de23a8dbcc73a4ae86f763b")})
	if err != nil {
		return ProjectContractSet{}, fmt.Errorf("package:%w", err)
	}
	r, err := Resolve(project, Expectation{Pin: Pin{projectModule, projectRevV3}, ModuleVersion: 3, RequiredExports: ids("e010", "e011", "e012", "e013", "e014", "e015", "e016", "e017", "e018"), Imports: []Pin{packagePin, execPin}, Digest: mustDigest("68e5b5fce56065bc94deac4ff24d26dadfe496b69db7f1aefa8a82327c87c2f2")})
	if err != nil {
		return ProjectContractSet{}, fmt.Errorf("project:%w", err)
	}
	return ProjectContractSet{execution: x, packages: p, project: r, validated: true}, nil
}

func resolveProjectContractSet(execution, packages, project []byte, sourceInventory bool) (ProjectContractSet, error) {
	execPin := Pin{executionModule, executionRev}
	packagePin := Pin{packageModule, packageRev}
	projectPin := Pin{projectModule, projectRev}
	projectVersion := uint64(1)
	requiredProjectExports := []wire.ID{mustID("0000000000000000000000000000e010"), mustID("0000000000000000000000000000e011")}
	projectDigest := mustDigest("5c8f22e9fee59c378f77ebdfd171d6220ccde8f3e1aafde0400830880e005dcc")
	if sourceInventory {
		projectPin.Revision = projectRevV2
		projectVersion = 2
		requiredProjectExports = append(requiredProjectExports,
			mustID("0000000000000000000000000000e012"),
			mustID("0000000000000000000000000000e013"),
			mustID("0000000000000000000000000000e014"),
			mustID("0000000000000000000000000000e015"),
			mustID("0000000000000000000000000000e016"),
		)
		projectDigest = mustDigest("19d499668b244b49a5dba2abc1441bdab025846ff5a21ef13739176d372ba000")
	}
	x, err := Resolve(execution, Expectation{Pin: execPin, ModuleVersion: 35, RequiredExports: []wire.ID{mustID("00000000000000000000000000009015")}, Digest: mustDigest("54fdd39b5d78f7f12da37fd43505e0a7962bad9d9808b54a16c20e4cf95e736a")})
	if err != nil {
		return ProjectContractSet{}, fmt.Errorf("execution:%w", err)
	}
	p, err := Resolve(packages, Expectation{Pin: packagePin, ModuleVersion: 1, RequiredExports: []wire.ID{mustID("0000000000000000000000000000b010"), mustID("0000000000000000000000000000b011"), mustID("0000000000000000000000000000b012")}, Digest: mustDigest("9f9b6e7f72a00ab069c56e80a9021abc4c59e2d0bde579ab29bda7350c506bf6")})
	if err != nil {
		return ProjectContractSet{}, fmt.Errorf("package:%w", err)
	}
	r, err := Resolve(project, Expectation{Pin: projectPin, ModuleVersion: projectVersion, RequiredExports: requiredProjectExports, Imports: []Pin{packagePin, execPin}, Digest: projectDigest})
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
func ids(short ...string) []wire.ID {
	out := make([]wire.ID, len(short))
	for i, s := range short {
		for len(s) < 32 {
			s = "0" + s
		}
		out[i] = mustID(s)
	}
	return out
}
func mustDigest(s string) (out [sha256.Size]byte) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != len(out) {
		panic(s)
	}
	copy(out[:], b)
	return
}
