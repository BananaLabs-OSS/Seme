// Package projectv14instance emits a neutral accepted project-reconciliation
// record. Source editing and native validation remain provider evidence.
package projectv14instance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectv13instance"
	"seme.local/reference/wire"
)

type Inputs struct {
	Contracts                  contractcatalog.ProjectContractSetV14
	Prior, Result              projectv13instance.Inputs
	Patch                      []byte
	ClientRevision             uint64
	NativeValidationTranscript []byte
	Artifact                   []byte
}

func Emit(in Inputs) ([]byte, error) { return emit(in) }
func Validate(in Inputs) error {
	want, err := emit(Inputs{Contracts: in.Contracts, Prior: in.Prior, Result: in.Result, Patch: in.Patch, ClientRevision: in.ClientRevision, NativeValidationTranscript: in.NativeValidationTranscript})
	if err != nil {
		return err
	}
	if !bytes.Equal(want, in.Artifact) {
		return fmt.Errorf("project_v14.artifact")
	}
	return nil
}

func emit(in Inputs) ([]byte, error) {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e035")}) || in.ClientRevision == 0 || len(in.NativeValidationTranscript) == 0 || len(in.NativeValidationTranscript) > 1<<20 {
		return nil, fmt.Errorf("project_v14.authority")
	}
	if err := projectv13instance.Validate(in.Prior); err != nil {
		return nil, fmt.Errorf("project_v14.prior:%w", err)
	}
	if err := projectv13instance.Validate(in.Result); err != nil {
		return nil, fmt.Errorf("project_v14.result:%w", err)
	}
	if bytes.Equal(in.Prior.Composed, in.Result.Composed) {
		return nil, fmt.Errorf("project_v14.unchanged")
	}
	prior, err := wire.Decode(in.Prior.Composed)
	if err != nil {
		return nil, err
	}
	result, err := wire.Decode(in.Result.Composed)
	if err != nil {
		return nil, err
	}
	priorContent, err := contentRevision(prior)
	if err != nil {
		return nil, err
	}
	resultContent, err := contentRevision(result)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(priorContent, resultContent) {
		return nil, fmt.Errorf("project_v14.content_unchanged")
	}
	patch, patchID, err := validatePatch(in.Contracts.Patch(), in.Patch, prior, result)
	if err != nil {
		return nil, err
	}
	out := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for _, graph := range []wire.Envelope{in.Contracts.Project().Envelope(), in.Contracts.Patch().Envelope(), patch} {
		for key, value := range graph.Entities {
			if old, exists := out.Entities[key]; exists && !same(old, value) {
				return nil, fmt.Errorf("project_v14.collision:%s", key)
			}
			out.Entities[key] = value
		}
	}
	seed := sha256.New()
	seed.Write([]byte("seme.project-v14.identity.v1\x00"))
	for _, v := range [][]byte{in.Prior.Composed, in.Result.Composed, in.Patch, in.NativeValidationTranscript} {
		h := sha256.Sum256(v)
		seed.Write(h[:])
	}
	_ = binary.Write(seed, binary.BigEndian, in.ClientRevision)
	stable := func(label string) wire.ID {
		h := sha256.New()
		h.Write(seed.Sum(nil))
		h.Write([]byte(label))
		var x wire.ID
		copy(x[:], h.Sum(nil))
		return x
	}
	record := stable("record")
	priorDigest, resultDigest, validationDigest := sha256.Sum256(in.Prior.Composed), sha256.Sum256(in.Result.Composed), sha256.Sum256(in.NativeValidationTranscript)
	out.Entities[record] = entity(record, "e036", map[string]wire.Value{"e360": blob(priorDigest[:]), "e361": blob(resultDigest[:]), "e362": ref(patchID), "e363": unsigned(in.ClientRevision), "e364": blob(priorContent), "e365": blob(resultContent), "e366": blob(validationDigest[:])})
	importID, moduleID := stable("project-v14-import"), stable("module")
	pin := in.Contracts.Project().Pin()
	out.Entities[importID] = entity(importID, "13", map[string]wire.Value{"130": ref(pin.Module), "131": blob(pin.Revision[:])})
	out.Module = moduleID
	out.Entities[moduleID] = entity(moduleID, "12", map[string]wire.Value{"120": blob([]byte("reconciled-project-revision-v1")), "121": refs(importID), "122": refs(record)})
	out.Revision = revision(out)
	return wire.Encode(out)
}

func validatePatch(contract contractcatalog.Contract, data []byte, prior, result wire.Envelope) (wire.Envelope, wire.ID, error) {
	g, err := wire.Decode(data)
	if err != nil {
		return g, wire.ID{}, fmt.Errorf("project_v14.patch_wire:%w", err)
	}
	canonical, err := wire.Encode(g)
	if err != nil || !bytes.Equal(canonical, data) {
		return g, wire.ID{}, fmt.Errorf("project_v14.patch_noncanonical")
	}
	for key, want := range contract.Envelope().Entities {
		got, ok := g.Entities[key]
		if !ok || !same(got, want) {
			return g, wire.ID{}, fmt.Errorf("project_v14.patch_contract")
		}
	}
	patches := bySchema(g, id("5010"))
	if len(patches) != 1 {
		return g, wire.ID{}, fmt.Errorf("project_v14.patch_count")
	}
	p := patches[0]
	base := p.Fields[id("5101")]
	ops := p.Fields[id("5102")]
	if base.Tag != 5 || !bytes.Equal(base.Bytes, prior.Revision[:]) || ops.Tag != 7 || len(ops.List) == 0 {
		return g, wire.ID{}, fmt.Errorf("project_v14.patch_base_or_operations")
	}
	seen := map[string]bool{}
	for _, opref := range ops.List {
		op, ok := g.Entities[opref.Reference]
		if opref.Tag != 6 || !ok || op.Schema != id("5011") {
			return g, wire.ID{}, fmt.Errorf("project_v14.patch_operation")
		}
		target, field, expected, replacement := op.Fields[id("5110")], op.Fields[id("5111")], op.Fields[id("5112")], op.Fields[id("5113")]
		if target.Tag != 5 || len(target.Bytes) != 16 || field.Tag != 5 || len(field.Bytes) != 16 || expected.Tag != 5 || len(expected.Bytes) == 0 || replacement.Tag != 5 || len(replacement.Bytes) == 0 {
			return g, wire.ID{}, fmt.Errorf("project_v14.patch_shape")
		}
		var tid, fid wire.ID
		copy(tid[:], target.Bytes)
		copy(fid[:], field.Bytes)
		key := tid.String() + fid.String()
		if seen[key] {
			return g, wire.ID{}, fmt.Errorf("project_v14.patch_duplicate")
		}
		seen[key] = true
		before, bok := prior.Entities[tid]
		after, aok := result.Entities[tid]
		bv, bvok := before.Fields[fid]
		av, avok := after.Fields[fid]
		if !bok || !aok || !bvok || !avok || bv.Tag != 5 || av.Tag != 5 || !bytes.Equal(bv.Bytes, expected.Bytes) || !bytes.Equal(av.Bytes, replacement.Bytes) {
			return g, wire.ID{}, fmt.Errorf("project_v14.patch_transition")
		}
	}
	return g, p.ID, nil
}

func contentRevision(g wire.Envelope) ([]byte, error) {
	xs := bySchema(g, id("e02c"))
	if len(xs) != 1 {
		return nil, fmt.Errorf("project_v14.project_snapshot")
	}
	v := xs[0].Fields[id("e2c2")]
	if v.Tag != 5 || len(v.Bytes) != 32 {
		return nil, fmt.Errorf("project_v14.content_revision")
	}
	return bytes.Clone(v.Bytes), nil
}
func bySchema(g wire.Envelope, s wire.ID) []wire.Entity {
	var out []wire.Entity
	for _, e := range g.Entities {
		if e.Schema == s {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i].ID[:], out[j].ID[:]) < 0 })
	return out
}
func revision(g wire.Envelope) wire.ID {
	g.Revision = wire.ID{}
	b, _ := wire.Encode(g)
	h := sha256.Sum256(append([]byte("seme.project-v14.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func same(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func entity(x wire.ID, s string, fs map[string]wire.Value) wire.Entity {
	m := map[wire.ID]wire.Value{}
	for k, v := range fs {
		m[id(k)] = v
	}
	return wire.Entity{ID: x, Schema: id(s), Version: 1, Fields: m}
}
func blob(v []byte) wire.Value { return wire.Value{Tag: 5, Bytes: bytes.Clone(v)} }
func ref(v wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: v} }
func refs(v ...wire.ID) wire.Value {
	x := wire.Value{Tag: 7}
	for _, i := range v {
		x.List = append(x.List, ref(i))
	}
	return x
}
func unsigned(v uint64) wire.Value { return wire.Value{Tag: 3, Unsigned: v} }
func id(v string) wire.ID {
	for len(v) < 32 {
		v = "0" + v
	}
	x, err := wire.ParseID(v)
	if err != nil {
		panic(err)
	}
	return x
}
