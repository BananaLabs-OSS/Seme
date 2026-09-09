// Package projectemitter creates a neutral Project Contract instance from an
// already compiled Execution graph and explicit package metadata.
package projectemitter

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/executioninstance"
	"seme.local/reference/packageinstance"
	"seme.local/reference/projectinstance"
	"seme.local/reference/projectsnapshot"
	"seme.local/reference/wire"
)

type Input struct {
	Identity    string
	RootPackage string
	Packages    []Package
	Execution   wire.Envelope
}

type Package struct {
	Name         string
	Interfaces   []Interface
	Dependencies []Dependency
}

type Interface struct {
	Name       string
	Function   wire.ID
	Parameters []wire.ID
	Result     wire.ID
}

type Dependency struct{ Name, Package string }

var (
	moduleSchema      = id("00000000000000000000000000000012")
	importSchema      = id("00000000000000000000000000000013")
	programSchema     = id("00000000000000000000000000009015")
	functionSchema    = id("00000000000000000000000000009011")
	fProgramFunctions = id("00000000000000000000000000009150")
	fProgramEntry     = id("00000000000000000000000000009151")
	packageSchema     = id("0000000000000000000000000000b010")
	interfaceSchema   = id("0000000000000000000000000000b011")
	dependencySchema  = id("0000000000000000000000000000b012")
	identitySchema    = id("0000000000000000000000000000e010")
	snapshotSchema    = id("0000000000000000000000000000e011")
)

func Emit(contracts contractcatalog.ProjectContractSet, in Input) ([]byte, error) {
	if !contracts.Validated() {
		return nil, fmt.Errorf("project_emitter.contracts_unvalidated")
	}
	if in.Identity == "" || in.RootPackage == "" || len(in.Packages) == 0 {
		return nil, fmt.Errorf("project_emitter.input")
	}
	if err := executioninstance.Validate(contracts.Execution(), in.Execution); err != nil {
		return nil, fmt.Errorf("project_emitter.execution:%w", err)
	}
	program, err := soleProgram(in.Execution)
	if err != nil {
		return nil, err
	}
	entities, err := closure(in.Execution.Entities, program)
	if err != nil {
		return nil, err
	}

	packages := append([]Package(nil), in.Packages...)
	sort.Slice(packages, func(i, j int) bool { return packages[i].Name < packages[j].Name })
	packageIDs := map[string]wire.ID{}
	for i, p := range packages {
		if p.Name == "" || (i > 0 && packages[i-1].Name == p.Name) {
			return nil, fmt.Errorf("project_emitter.package_identity")
		}
		packageIDs[p.Name] = stableID("package", p.Name)
	}
	root, ok := packageIDs[in.RootPackage]
	if !ok {
		return nil, fmt.Errorf("project_emitter.root")
	}
	packageList := make([]wire.ID, 0, len(packages))
	for _, p := range packages {
		packageList = append(packageList, packageIDs[p.Name])
	}
	orderedPackages, err := dependencyOrder(packages)
	if err != nil {
		return nil, err
	}
	functionOwners := map[wire.ID]string{}
	for _, p := range packages {
		for _, x := range p.Interfaces {
			if owner, found := functionOwners[x.Function]; found && owner != p.Name {
				return nil, fmt.Errorf("project_emitter.function_multiple_owners:%s", x.Function)
			}
			functionOwners[x.Function] = p.Name
		}
	}

	for _, p := range orderedPackages {
		pid := packageIDs[p.Name]
		interfaces := append([]Interface(nil), p.Interfaces...)
		sort.Slice(interfaces, func(i, j int) bool {
			if interfaces[i].Name != interfaces[j].Name {
				return interfaces[i].Name < interfaces[j].Name
			}
			return less(interfaces[i].Function, interfaces[j].Function)
		})
		interfaceRefs := []wire.Value{}
		for i, x := range interfaces {
			if x.Name == "" || (i > 0 && x.Name == interfaces[i-1].Name) {
				return nil, fmt.Errorf("project_emitter.interface_identity")
			}
			if _, ok := entities[x.Function]; !ok {
				return nil, fmt.Errorf("project_emitter.interface_function:%s", x.Function)
			}
			if err := validateInterfaceSignature(x, entities); err != nil {
				return nil, err
			}
			params := make([]wire.Value, len(x.Parameters))
			for i, typ := range x.Parameters {
				if _, ok := entities[typ]; !ok {
					return nil, fmt.Errorf("project_emitter.interface_parameter:%s", typ)
				}
				params[i] = ref(typ)
			}
			if _, ok := entities[x.Result]; !ok {
				return nil, fmt.Errorf("project_emitter.interface_result:%s", x.Result)
			}
			iid := stableID("interface", pid.String(), x.Name, x.Function.String())
			interfaceRefs = append(interfaceRefs, ref(iid))
			if err := put(entities, wire.Entity{ID: iid, Schema: interfaceSchema, Version: 1, Fields: map[wire.ID]wire.Value{
				id("0000000000000000000000000000b110"): blob([]byte(x.Name)), id("0000000000000000000000000000b111"): ref(x.Function),
				id("0000000000000000000000000000b112"): list(params), id("0000000000000000000000000000b113"): ref(x.Result),
			}}); err != nil {
				return nil, err
			}
		}
		deps := append([]Dependency(nil), p.Dependencies...)
		sort.Slice(deps, func(i, j int) bool {
			if deps[i].Name != deps[j].Name {
				return deps[i].Name < deps[j].Name
			}
			return deps[i].Package < deps[j].Package
		})
		dependencyRefs := []wire.Value{}
		for i, d := range deps {
			target, ok := packageIDs[d.Package]
			if !ok {
				return nil, fmt.Errorf("project_emitter.dependency_target:%s", d.Package)
			}
			if d.Name == "" || (i > 0 && d.Name == deps[i-1].Name) {
				return nil, fmt.Errorf("project_emitter.dependency_identity")
			}
			did := stableID("dependency", pid.String(), d.Name, target.String())
			dependencyRefs = append(dependencyRefs, ref(did))
			if err := put(entities, wire.Entity{ID: did, Schema: dependencySchema, Version: 1, Fields: map[wire.ID]wire.Value{
				id("0000000000000000000000000000b120"): blob([]byte(d.Name)), id("0000000000000000000000000000b121"): blob([]byte("local")), id("0000000000000000000000000000b122"): ref(target),
			}}); err != nil {
				return nil, err
			}
		}
		if err := put(entities, wire.Entity{ID: pid, Schema: packageSchema, Version: 1, Fields: map[wire.ID]wire.Value{
			id("0000000000000000000000000000b100"): blob([]byte(p.Name)), id("0000000000000000000000000000b101"): blob(nil),
			id("0000000000000000000000000000b102"): list(interfaceRefs), id("0000000000000000000000000000b103"): list(dependencyRefs),
			id("0000000000000000000000000000b104"): list(nil), id("0000000000000000000000000000b105"): list(nil), id("0000000000000000000000000000b106"): list(nil),
		}}); err != nil {
			return nil, err
		}
		version, err := packageinstance.Revision(wire.Envelope{Entities: entities}, pid)
		if err != nil {
			return nil, err
		}
		entity := entities[pid]
		entity.Fields[id("0000000000000000000000000000b101")] = blob(version)
		entities[pid] = entity
	}
	sortIDs(packageList)
	identity := stableID("project-identity", in.Identity)
	if err := put(entities, wire.Entity{ID: identity, Schema: identitySchema, Version: 1, Fields: map[wire.ID]wire.Value{id("0000000000000000000000000000e100"): blob([]byte(in.Identity))}}); err != nil {
		return nil, err
	}

	module := stableID("project-module", in.Identity)
	snapshot := stableID("project-snapshot", in.Identity)
	imports := []wire.Value{}
	for _, c := range []struct {
		name     string
		contract contractcatalog.Contract
	}{{"execution", contracts.Execution()}, {"package", contracts.Package()}, {"project", contracts.Project()}} {
		pin := c.contract.Pin()
		iid := stableID("import", module.String(), c.name)
		imports = append(imports, ref(iid))
		if err := put(entities, wire.Entity{ID: iid, Schema: importSchema, Version: 1, Fields: map[wire.ID]wire.Value{
			id("00000000000000000000000000000130"): ref(pin.Module), id("00000000000000000000000000000131"): blob(pin.Revision[:]),
		}}); err != nil {
			return nil, err
		}
	}
	sortValues(imports)
	exports := append([]wire.Value{ref(identity), ref(snapshot), ref(program)}, refs(packageList)...)
	sortValues(exports)
	if err := put(entities, wire.Entity{ID: module, Schema: moduleSchema, Version: 1, Fields: map[wire.ID]wire.Value{
		id("00000000000000000000000000000120"): blob([]byte(in.Identity)), id("00000000000000000000000000000121"): list(imports), id("00000000000000000000000000000122"): list(exports),
	}}); err != nil {
		return nil, err
	}
	envelope := wire.Envelope{Module: module, Entities: entities}
	revision, err := projectsnapshot.Revision(envelope, identity, packageList, root, program)
	if err != nil {
		return nil, err
	}
	if err := put(entities, wire.Entity{ID: snapshot, Schema: snapshotSchema, Version: 1, Fields: map[wire.ID]wire.Value{
		id("0000000000000000000000000000e110"): ref(identity), id("0000000000000000000000000000e111"): blob(revision), id("0000000000000000000000000000e112"): list(refs(packageList)), id("0000000000000000000000000000e113"): ref(root), id("0000000000000000000000000000e114"): ref(program),
	}}); err != nil {
		return nil, err
	}
	envelope.Revision, err = projectinstance.ArtifactRevision(envelope)
	if err != nil {
		return nil, err
	}
	out, err := wire.Encode(envelope)
	if err != nil {
		return nil, err
	}
	if err = projectinstance.Validate(out); err != nil {
		return nil, err
	}
	if err = packageinstance.Validate(out); err != nil {
		return nil, err
	}
	return out, nil
}

func dependencyOrder(packages []Package) ([]Package, error) {
	byName := make(map[string]Package, len(packages))
	for _, p := range packages {
		byName[p.Name] = p
	}
	state := map[string]uint8{}
	ordered := make([]Package, 0, len(packages))
	var visit func(string) error
	visit = func(name string) error {
		switch state[name] {
		case 1:
			return fmt.Errorf("project_emitter.dependency_cycle:%s", name)
		case 2:
			return nil
		}
		p, ok := byName[name]
		if !ok {
			return fmt.Errorf("project_emitter.dependency_target:%s", name)
		}
		state[name] = 1
		deps := append([]Dependency(nil), p.Dependencies...)
		sort.Slice(deps, func(i, j int) bool {
			if deps[i].Package != deps[j].Package {
				return deps[i].Package < deps[j].Package
			}
			return deps[i].Name < deps[j].Name
		})
		for _, dependency := range deps {
			if err := visit(dependency.Package); err != nil {
				return err
			}
		}
		state[name] = 2
		ordered = append(ordered, p)
		return nil
	}
	for _, p := range packages {
		if err := visit(p.Name); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func soleProgram(e wire.Envelope) (wire.ID, error) {
	var out wire.ID
	for eid, x := range e.Entities {
		if x.Schema == programSchema {
			if out != (wire.ID{}) {
				return out, fmt.Errorf("project_emitter.program_count")
			}
			out = eid
		}
	}
	if out == (wire.ID{}) {
		return out, fmt.Errorf("project_emitter.program_count")
	}
	p := e.Entities[out]
	functions, fok := p.Fields[fProgramFunctions]
	entry, eok := p.Fields[fProgramEntry]
	if !fok || functions.Tag != 7 || !eok || entry.Tag != 6 || len(functions.List) == 0 {
		return out, fmt.Errorf("project_emitter.program_shape")
	}
	found := false
	for _, value := range functions.List {
		if value.Tag != 6 {
			return out, fmt.Errorf("project_emitter.program_function_shape")
		}
		function, ok := e.Entities[value.Reference]
		if !ok || function.Schema != functionSchema {
			return out, fmt.Errorf("project_emitter.program_function:%s", value.Reference)
		}
		if value.Reference == entry.Reference {
			found = true
		}
	}
	if !found {
		return out, fmt.Errorf("project_emitter.program_entry")
	}
	return out, nil
}
func closure(all map[wire.ID]wire.Entity, root wire.ID) (map[wire.ID]wire.Entity, error) {
	out := map[wire.ID]wire.Entity{}
	q := []wire.ID{root}
	for len(q) > 0 {
		x := q[0]
		q = q[1:]
		if _, ok := out[x]; ok {
			continue
		}
		e, ok := all[x]
		if !ok {
			return nil, fmt.Errorf("project_emitter.reference_missing:%s", x)
		}
		out[x] = e
		for _, v := range e.Fields {
			collect(v, &q)
		}
	}
	return out, nil
}
func collect(v wire.Value, q *[]wire.ID) {
	if v.Tag == 6 {
		*q = append(*q, v.Reference)
	}
	for _, x := range v.List {
		collect(x, q)
	}
	for _, x := range v.Record {
		collect(x, q)
	}
}
func put(entities map[wire.ID]wire.Entity, entity wire.Entity) error {
	if _, exists := entities[entity.ID]; exists {
		return fmt.Errorf("project_emitter.identity_collision:%s", entity.ID)
	}
	entities[entity.ID] = entity
	return nil
}
func validateInterfaceSignature(x Interface, entities map[wire.ID]wire.Entity) error {
	function := entities[x.Function]
	parameters, ok := function.Fields[id("00000000000000000000000000009111")]
	if !ok || parameters.Tag != 7 || len(parameters.List) != len(x.Parameters) {
		return fmt.Errorf("project_emitter.interface_signature_parameters:%s", x.Function)
	}
	for i, p := range parameters.List {
		if p.Tag != 6 {
			return fmt.Errorf("project_emitter.interface_signature_parameter:%s", x.Function)
		}
		entity, ok := entities[p.Reference]
		if !ok {
			return fmt.Errorf("project_emitter.interface_signature_parameter:%s", x.Function)
		}
		typ, ok := entity.Fields[id("00000000000000000000000000009121")]
		if !ok || typ.Tag != 6 || typ.Reference != x.Parameters[i] {
			return fmt.Errorf("project_emitter.interface_signature_parameter:%s", x.Function)
		}
	}
	result, ok := function.Fields[id("00000000000000000000000000009112")]
	if !ok || result.Tag != 6 || result.Reference != x.Result {
		return fmt.Errorf("project_emitter.interface_signature_result:%s", x.Function)
	}
	return nil
}
func writeBlob(h interface{ Write([]byte) (int, error) }, b []byte) {
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], uint64(len(b)))
	h.Write(n[:])
	h.Write(b)
}
func stableID(parts ...string) wire.ID {
	// This domain is intentionally role-specific to Project instance entities;
	// it does not claim equivalence with provider-owned declaration identities.
	h := sha256.New()
	h.Write([]byte("seme.project-emitter.identity.v1\x00"))
	for _, p := range parts {
		writeBlob(h, []byte(p))
	}
	sum := h.Sum(nil)
	var out wire.ID
	out[0] = 0x80
	copy(out[1:], sum[:15])
	return out
}
func id(s string) wire.ID {
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
func ref(x wire.ID) wire.Value       { return wire.Value{Tag: 6, Reference: x} }
func blob(x []byte) wire.Value       { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func list(x []wire.Value) wire.Value { return wire.Value{Tag: 7, List: x} }
func refs(x []wire.ID) []wire.Value {
	out := make([]wire.Value, len(x))
	for i, id := range x {
		out[i] = ref(id)
	}
	return out
}
func less(a, b wire.ID) bool { return bytes.Compare(a[:], b[:]) < 0 }
func sortIDs(x []wire.ID)    { sort.Slice(x, func(i, j int) bool { return less(x[i], x[j]) }) }
func sortValues(x []wire.Value) {
	sort.Slice(x, func(i, j int) bool { return less(x[i].Reference, x[j].Reference) })
}
