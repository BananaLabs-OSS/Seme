// Package presentationinstance emits and validates neutral source-presentation
// manifests. Presentation aliases name an existing Execution type; they never
// create a second semantic type.
package presentationinstance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"unicode/utf8"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectv9instance"
	"seme.local/reference/wire"
)

const MaxAliases = 256

type Alias struct {
	Owner, Target, SourceUnit                              wire.ID
	Name                                                   string
	Visibility                                             uint64
	Path                                                   string
	Digest                                                 [32]byte
	Start, End, StartLine, StartColumn, EndLine, EndColumn uint64
	ImportBindings                                         []wire.ID
}
type Model struct{ Aliases []Alias }
type Inputs struct {
	Contracts contractcatalog.ProjectContractSetV10
	ProjectV9 projectv9instance.Inputs
	Model     Model
	Artifact  []byte
}

// ModelFromArtifact decodes only the declarative alias model. Callers obtain
// authenticity by passing the result back through Validate with exact inputs.
func ModelFromArtifact(data []byte) (Model, error) {
	e, err := wire.Decode(data)
	if err != nil {
		return Model{}, err
	}
	var out Model
	for _, q := range e.Entities {
		if q.Schema != id("1011") {
			continue
		}
		owner, target, source := q.Fields[id("1110")], q.Fields[id("1113")], q.Fields[id("1114")]
		namev, visref, originref, bindings := q.Fields[id("1111")], q.Fields[id("1112")], q.Fields[id("1115")], q.Fields[id("1116")]
		vis := e.Entities[visref.Reference]
		origin := e.Entities[originref.Reference]
		if owner.Tag != 6 || target.Tag != 6 || source.Tag != 6 || namev.Tag != 5 || visref.Tag != 6 || vis.Schema != id("1012") || originref.Tag != 6 || origin.Schema != id("b026") || bindings.Tag != 7 {
			return Model{}, fmt.Errorf("presentation_instance.decode")
		}
		digest := origin.Fields[id("b262")]
		if digest.Tag != 5 || len(digest.Bytes) != 32 {
			return Model{}, fmt.Errorf("presentation_instance.decode_origin")
		}
		var sum [32]byte
		copy(sum[:], digest.Bytes)
		a := Alias{Owner: owner.Reference, Target: target.Reference, SourceUnit: source.Reference, Name: string(namev.Bytes), Visibility: vis.Fields[id("1120")].Unsigned, Path: string(origin.Fields[id("b261")].Bytes), Digest: sum, Start: origin.Fields[id("b263")].Unsigned, End: origin.Fields[id("b264")].Unsigned, StartLine: origin.Fields[id("b265")].Unsigned, StartColumn: origin.Fields[id("b266")].Unsigned, EndLine: origin.Fields[id("b267")].Unsigned, EndColumn: origin.Fields[id("b268")].Unsigned}
		for _, v := range bindings.List {
			if v.Tag != 6 {
				return Model{}, fmt.Errorf("presentation_instance.decode_binding")
			}
			a.ImportBindings = append(a.ImportBindings, v.Reference)
		}
		out.Aliases = append(out.Aliases, a)
	}
	if _, err := one(e, "1010"); err != nil {
		return Model{}, fmt.Errorf("presentation_instance.decode_manifest")
	}
	return out, nil
}

func Emit(in Inputs) ([]byte, error) { return emit(in) }
func Validate(in Inputs) error {
	want, err := emit(Inputs{Contracts: in.Contracts, ProjectV9: in.ProjectV9, Model: in.Model})
	if err != nil {
		return err
	}
	if !bytes.Equal(want, in.Artifact) {
		return fmt.Errorf("presentation_instance.artifact")
	}
	return nil
}

func emit(in Inputs) ([]byte, error) {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e00e")}) || in.Contracts.Presentation().Pin() != (contractcatalog.Pin{Module: id("1000"), Revision: id("1001")}) {
		return nil, fmt.Errorf("presentation_instance.contracts")
	}
	if err := projectv9instance.Validate(in.ProjectV9); err != nil {
		return nil, fmt.Errorf("presentation_instance.project_v9:%w", err)
	}
	base, err := wire.Decode(in.ProjectV9.Composed)
	if err != nil {
		return nil, err
	}
	aliases := append([]Alias(nil), in.Model.Aliases...)
	for i := range aliases {
		aliases[i].ImportBindings = append([]wire.ID(nil), aliases[i].ImportBindings...)
		sort.Slice(aliases[i].ImportBindings, func(a, b int) bool {
			return bytes.Compare(aliases[i].ImportBindings[a][:], aliases[i].ImportBindings[b][:]) < 0
		})
	}
	sort.Slice(aliases, func(i, j int) bool {
		if aliases[i].Owner == aliases[j].Owner {
			return aliases[i].Name < aliases[j].Name
		}
		return bytes.Compare(aliases[i].Owner[:], aliases[j].Owner[:]) < 0
	})
	if len(aliases) > MaxAliases {
		return nil, fmt.Errorf("presentation_instance.count")
	}
	root, err := one(base, "e024")
	if err != nil {
		return nil, err
	}
	ownerBindings := map[wire.ID]map[wire.ID]bool{}
	for _, detail := range base.Entities {
		if detail.Schema != id("b021") {
			continue
		}
		owner := detail.Fields[id("b210")].Reference
		if ownerBindings[owner] == nil {
			ownerBindings[owner] = map[wire.ID]bool{}
		}
		for _, binding := range detail.Fields[id("b212")].List {
			ownerBindings[owner][binding.Reference] = true
		}
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for x, q := range base.Entities {
		if q.Schema != id("12") && q.Schema != id("13") {
			e.Entities[x] = q
		}
	}
	refs := []wire.Value{}
	priorOwner := wire.ID{}
	priorName := ""
	for _, a := range aliases {
		if !name(a.Name) || a.Visibility > 1 || (a.Visibility == 1) != (a.Name[0] >= 'A' && a.Name[0] <= 'Z') || a.End <= a.Start || a.StartLine == 0 || a.StartColumn == 0 || a.EndLine == 0 || a.EndColumn == 0 {
			return nil, fmt.Errorf("presentation_instance.alias")
		}
		if a.Owner == priorOwner && a.Name == priorName {
			return nil, fmt.Errorf("presentation_instance.duplicate")
		}
		priorOwner, priorName = a.Owner, a.Name
		if base.Entities[a.Owner].Schema != id("b010") || !executionType(base.Entities[a.Target]) {
			return nil, fmt.Errorf("presentation_instance.authority")
		}
		sourceUnit := base.Entities[a.SourceUnit]
		if sourceUnit.Schema != id("e015") || string(sourceUnit.Fields[id("e150")].Bytes) != a.Path || !bytes.Equal(sourceUnit.Fields[id("e151")].Bytes, a.Digest[:]) || a.End > sourceUnit.Fields[id("e152")].Unsigned {
			return nil, fmt.Errorf("presentation_instance.source")
		}
		seen := map[wire.ID]bool{}
		bindingRefs := []wire.Value{}
		for _, b := range a.ImportBindings {
			if seen[b] || base.Entities[b].Schema != id("b024") || !ownerBindings[a.Owner][b] {
				return nil, fmt.Errorf("presentation_instance.binding")
			}
			seen[b] = true
			bindingRefs = append(bindingRefs, ref(b))
		}
		origin := stable(in.ProjectV9.Composed, "origin", a.Owner.String(), a.Name)
		e.Entities[origin] = entity(origin, "b026", map[string]wire.Value{"b260": ref(a.SourceUnit), "b261": blob(a.Path), "b262": blobBytes(a.Digest[:]), "b263": u(a.Start), "b264": u(a.End), "b265": u(a.StartLine), "b266": u(a.StartColumn), "b267": u(a.EndLine), "b268": u(a.EndColumn)})
		visibility := stable(in.ProjectV9.Composed, "visibility", a.Owner.String(), a.Name)
		e.Entities[visibility] = entity(visibility, "1012", map[string]wire.Value{"1120": u(a.Visibility)})
		x := stable(in.ProjectV9.Composed, "alias", a.Owner.String(), a.Name)
		e.Entities[x] = entity(x, "1011", map[string]wire.Value{"1110": ref(a.Owner), "1111": blob(a.Name), "1112": ref(visibility), "1113": ref(a.Target), "1114": ref(a.SourceUnit), "1115": ref(origin), "1116": {Tag: 7, List: bindingRefs}})
		refs = append(refs, ref(x))
	}
	manifest := stable(in.ProjectV9.Composed, "manifest")
	e.Entities[manifest] = entity(manifest, "1010", map[string]wire.Value{"1100": ref(root), "1101": {Tag: 7, List: refs}, "1102": blobBytes(make([]byte, 32))})
	q := e.Entities[manifest]
	q.Fields[id("1102")] = blobBytes(contentRevision(e, manifest))
	e.Entities[manifest] = q
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("1000"), Revision: id("1001")}, {Module: id("9000"), Revision: id("9024")}, {Module: id("b000"), Revision: id("b004")}, {Module: id("e000"), Revision: id("e00b")}} {
		x := stable(in.ProjectV9.Composed, "import", pin.Module.String())
		e.Entities[x] = entity(x, "13", map[string]wire.Value{"130": ref(pin.Module), "131": blobBytes(pin.Revision[:])})
		imports = append(imports, ref(x))
	}
	sort.Slice(imports, func(i, j int) bool { return bytes.Compare(imports[i].Reference[:], imports[j].Reference[:]) < 0 })
	e.Module = stable(in.ProjectV9.Composed, "module")
	e.Entities[e.Module] = entity(e.Module, "12", map[string]wire.Value{"120": blob("source-presentation-manifest-v1"), "121": {Tag: 7, List: imports}, "122": list(manifest)})
	e.Revision = artifactRevision(e)
	return wire.Encode(e)
}

func executionType(q wire.Entity) bool {
	switch q.Schema.String() {
	case id("9010").String(), id("9020").String(), id("9030").String(), id("9040").String(), id("9041").String(), id("9042").String(), id("90f2").String(), id("90f8").String(), id("a010").String(), id("a020").String(), id("a040").String(), id("a050").String():
		return true
	}
	return false
}
func name(s string) bool {
	if s == "" || len(s) > 128 || !utf8.ValidString(s) {
		return false
	}
	for i, r := range s {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
func one(e wire.Envelope, s string) (wire.ID, error) {
	var x wire.ID
	n := 0
	for k, q := range e.Entities {
		if q.Schema == id(s) {
			x = k
			n++
		}
	}
	if n != 1 {
		return x, fmt.Errorf("presentation_instance.root")
	}
	return x, nil
}
func stable(base []byte, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.source-presentation.identity.v1\x00"))
	x := sha256.Sum256(base)
	h.Write(x[:])
	for _, p := range parts {
		_ = binary.Write(h, binary.BigEndian, uint64(len(p)))
		h.Write([]byte(p))
	}
	var o wire.ID
	copy(o[:], h.Sum(nil))
	return o
}
func contentRevision(e wire.Envelope, root wire.ID) []byte {
	q := e.Entities[root]
	q.Fields = cloneFields(q.Fields)
	q.Fields[id("1102")] = blobBytes(make([]byte, 32))
	e.Entities[root] = q
	b, _ := wire.Encode(wire.Envelope{Entities: closure(e.Entities, root)})
	h := sha256.Sum256(append([]byte("seme.source-presentation.manifest.v1\x00"), b...))
	return h[:]
}
func artifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.source-presentation.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func closure(es map[wire.ID]wire.Entity, root wire.ID) map[wire.ID]wire.Entity {
	o := map[wire.ID]wire.Entity{}
	todo := []wire.ID{root}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if _, ok := o[x]; ok {
			continue
		}
		q, ok := es[x]
		if !ok {
			continue
		}
		o[x] = q
		for _, v := range q.Fields {
			collect(v, &todo)
		}
	}
	return o
}
func collect(v wire.Value, o *[]wire.ID) {
	if v.Tag == 6 {
		*o = append(*o, v.Reference)
	}
	if v.Tag == 7 {
		for _, x := range v.List {
			collect(x, o)
		}
	}
	if v.Tag == 8 {
		for _, x := range v.Record {
			collect(x, o)
		}
	}
}
func cloneFields(x map[wire.ID]wire.Value) map[wire.ID]wire.Value {
	o := map[wire.ID]wire.Value{}
	for k, v := range x {
		o[k] = v
	}
	return o
}
func entity(x wire.ID, s string, fs map[string]wire.Value) wire.Entity {
	f := map[wire.ID]wire.Value{}
	for k, v := range fs {
		f[id(k)] = v
	}
	return wire.Entity{ID: x, Schema: id(s), Version: 1, Fields: f}
}
func ref(x wire.ID) wire.Value      { return wire.Value{Tag: 6, Reference: x} }
func u(x uint64) wire.Value         { return wire.Value{Tag: 3, Unsigned: x} }
func blob(x string) wire.Value      { return blobBytes([]byte(x)) }
func blobBytes(x []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func list(xs ...wire.ID) wire.Value {
	v := wire.Value{Tag: 7}
	for _, x := range xs {
		v.List = append(v.List, ref(x))
	}
	return v
}
func id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
