// Package durableinstance emits and validates the neutral Durable-State-v1 plan.
package durableinstance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/wire"
)

const MaxPayloadBytes = 1 << 20

type Model struct {
	Identity, PortIdentity, CodecIdentity                                                 string
	Owner, Version1Type, Version2Type, Validator1, Validator2, Migration                  wire.ID
	LoadCapability, CompareExchangeCapability, LoadEffect, CompareExchangeEffect, KeyType wire.ID
	MaximumPayloadBytes                                                                   uint64
}
type Inputs struct {
	Contracts           contractcatalog.ProjectContractSetV10
	ProjectV9, Artifact []byte
	Model               Model
}

func Emit(in Inputs) ([]byte, error) { return emit(in) }
func Validate(in Inputs) error {
	want, err := emit(Inputs{Contracts: in.Contracts, ProjectV9: in.ProjectV9, Model: in.Model})
	if err != nil {
		return err
	}
	if !bytes.Equal(want, in.Artifact) {
		return fmt.Errorf("durable_instance.artifact")
	}
	return nil
}

func emit(in Inputs) ([]byte, error) {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e00e")}) || in.Contracts.DurableState().Pin() != (contractcatalog.Pin{Module: id("8000"), Revision: id("8001")}) {
		return nil, fmt.Errorf("durable_instance.contracts")
	}
	base, err := wire.Decode(in.ProjectV9)
	if err != nil {
		return nil, fmt.Errorf("durable_instance.project_v9:%w", err)
	}
	if err = baseAuthority(base); err != nil {
		return nil, err
	}
	m := in.Model
	if !name(m.Identity) || !name(m.PortIdentity) || !name(m.CodecIdentity) || m.MaximumPayloadBytes == 0 || m.MaximumPayloadBytes > MaxPayloadBytes {
		return nil, fmt.Errorf("durable_instance.metadata")
	}
	owners, effects, functions := ownership(base)
	if !owners[m.Owner] {
		return nil, fmt.Errorf("durable_instance.owner")
	}
	for _, x := range []wire.ID{m.Version1Type, m.Version2Type, m.KeyType} {
		if !typeEntity(base.Entities[x]) {
			return nil, fmt.Errorf("durable_instance.type:%s", x)
		}
	}
	if m.Version1Type == m.Version2Type {
		return nil, fmt.Errorf("durable_instance.version_type_mix")
	}
	if m.Validator1 == m.Validator2 || m.Validator1 == m.Migration || m.Validator2 == m.Migration {
		return nil, fmt.Errorf("durable_instance.function_mix")
	}
	if !pureFunction(base, m.Validator1, m.Version1Type, id("9020"), functions[m.Validator1] == m.Owner) || !pureFunction(base, m.Validator2, m.Version2Type, id("9020"), functions[m.Validator2] == m.Owner) || !pureFunction(base, m.Migration, m.Version1Type, m.Version2Type, functions[m.Migration] == m.Owner) {
		return nil, fmt.Errorf("durable_instance.function")
	}
	if m.LoadCapability == m.CompareExchangeCapability || m.LoadEffect == m.CompareExchangeEffect {
		return nil, fmt.Errorf("durable_instance.authorization_mix")
	}
	for _, p := range [][2]wire.ID{{m.LoadEffect, m.LoadCapability}, {m.CompareExchangeEffect, m.CompareExchangeCapability}} {
		q, ok := base.Entities[p[0]]
		if !ok || q.Schema != id("15") || q.Fields[id("151")].Tag != 6 || q.Fields[id("151")].Reference != p[1] || base.Entities[p[1]].Schema != id("16") || !effects[m.Owner][p[0]] {
			return nil, fmt.Errorf("durable_instance.effect")
		}
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for x, q := range base.Entities {
		if q.Schema != id("12") && q.Schema != id("13") {
			e.Entities[x] = q
		}
	}
	fam := stable(in.ProjectV9, "family", m.Identity)
	v1 := stable(in.ProjectV9, "version", "1")
	v2 := stable(in.ProjectV9, "version", "2")
	a := stable(in.ProjectV9, "validator", "1")
	b := stable(in.ProjectV9, "validator", "2")
	mig := stable(in.ProjectV9, "migration")
	port := stable(in.ProjectV9, "port", m.PortIdentity)
	plan := stable(in.ProjectV9, "plan")
	e.Entities[v1] = entity(v1, "8012", map[string]wire.Value{"8120": ref(fam), "8121": u(1), "8122": ref(m.Version1Type)})
	e.Entities[v2] = entity(v2, "8012", map[string]wire.Value{"8120": ref(fam), "8121": u(2), "8122": ref(m.Version2Type)})
	e.Entities[a] = entity(a, "8013", map[string]wire.Value{"8130": ref(v1), "8131": ref(m.Validator1)})
	e.Entities[b] = entity(b, "8013", map[string]wire.Value{"8130": ref(v2), "8131": ref(m.Validator2)})
	e.Entities[mig] = entity(mig, "8014", map[string]wire.Value{"8140": ref(v1), "8141": ref(v2), "8142": ref(m.Migration)})
	e.Entities[fam] = entity(fam, "8011", map[string]wire.Value{"8110": blob(m.Identity), "8111": ref(m.Owner), "8112": ref(v2), "8113": list(v1, v2), "8114": list(a, b), "8115": list(mig)})
	e.Entities[port] = entity(port, "8015", map[string]wire.Value{"8150": blob(m.PortIdentity), "8151": ref(m.Owner), "8152": ref(fam), "8153": ref(m.LoadCapability), "8154": ref(m.CompareExchangeCapability), "8155": ref(m.LoadEffect), "8156": ref(m.CompareExchangeEffect), "8157": ref(m.KeyType), "8158": u(m.MaximumPayloadBytes), "8159": blob(m.CodecIdentity)})
	e.Entities[plan] = entity(plan, "8010", map[string]wire.Value{"8100": list(fam), "8101": list(port), "8102": blobBytes(make([]byte, 32))})
	q := e.Entities[plan]
	q.Fields[id("8102")] = blobBytes(contentRevision(e, plan))
	e.Entities[plan] = q
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("3000"), Revision: id("3001")}, {Module: id("8000"), Revision: id("8001")}, {Module: id("9000"), Revision: id("9024")}, {Module: id("b000"), Revision: id("b004")}, {Module: id("e000"), Revision: id("e00b")}} {
		x := stable(in.ProjectV9, "import", pin.Module.String())
		e.Entities[x] = entity(x, "13", map[string]wire.Value{"130": ref(pin.Module), "131": blobBytes(pin.Revision[:])})
		imports = append(imports, ref(x))
	}
	sort.Slice(imports, func(i, j int) bool { return bytes.Compare(imports[i].Reference[:], imports[j].Reference[:]) < 0 })
	e.Module = stable(in.ProjectV9, "module")
	e.Entities[e.Module] = entity(e.Module, "12", map[string]wire.Value{"120": blob("durable-state-plan-v1"), "121": {Tag: 7, List: imports}, "122": list(plan)})
	e.Revision = artifactRevision(e)
	return wire.Encode(e)
}

func baseAuthority(e wire.Envelope) error {
	m, ok := e.Entities[e.Module]
	if !ok || m.Schema != id("12") {
		return fmt.Errorf("durable_instance.project_module")
	}
	pins := map[wire.ID]wire.ID{}
	listed := map[wire.ID]bool{}
	for _, v := range m.Fields[id("121")].List {
		if v.Tag != 6 || listed[v.Reference] {
			return fmt.Errorf("durable_instance.project_import")
		}
		listed[v.Reference] = true
		q, ok := e.Entities[v.Reference]
		if !ok || q.Schema != id("13") || q.Fields[id("130")].Tag != 6 || q.Fields[id("131")].Tag != 5 || len(q.Fields[id("131")].Bytes) != 16 {
			return fmt.Errorf("durable_instance.project_import")
		}
		var r wire.ID
		copy(r[:], q.Fields[id("131")].Bytes)
		pins[q.Fields[id("130")].Reference] = r
	}
	want := map[wire.ID]wire.ID{id("3000"): id("3001"), id("4000"): id("4006"), id("6000"): id("6001"), id("9000"): id("9024"), id("b000"): id("b004"), id("e000"): id("e00b"), id("f000"): id("f001")}
	if len(pins) != len(want) {
		return fmt.Errorf("durable_instance.project_pin")
	}
	for mod, rev := range want {
		if pins[mod] != rev {
			return fmt.Errorf("durable_instance.project_pin")
		}
	}
	for x, q := range e.Entities {
		if q.Schema == id("13") && !listed[x] {
			return fmt.Errorf("durable_instance.project_orphan_import")
		}
	}
	n := 0
	for _, q := range e.Entities {
		if q.Schema == id("e024") {
			n++
		}
	}
	if n != 1 {
		return fmt.Errorf("durable_instance.project_root")
	}
	return nil
}
func ownership(e wire.Envelope) (map[wire.ID]bool, map[wire.ID]map[wire.ID]bool, map[wire.ID]wire.ID) {
	owners := map[wire.ID]bool{}
	eff := map[wire.ID]map[wire.ID]bool{}
	fn := map[wire.ID]wire.ID{}
	for x, q := range e.Entities {
		if q.Schema != id("b010") {
			continue
		}
		owners[x] = true
		eff[x] = map[wire.ID]bool{}
		for _, v := range q.Fields[id("b104")].List {
			eff[x][v.Reference] = true
		}
		for _, v := range q.Fields[id("b102")].List {
			if z, ok := e.Entities[v.Reference]; ok && z.Schema == id("b011") {
				fn[z.Fields[id("b111")].Reference] = x
			}
		}
	}
	return owners, eff, fn
}
func pureFunction(e wire.Envelope, x, arg, result wire.ID, owned bool) bool {
	q, ok := e.Entities[x]
	if !ok || !owned || q.Schema != id("9011") {
		return false
	}
	ps := q.Fields[id("9111")]
	if ps.Tag != 7 || len(ps.List) != 1 || q.Fields[id("9112")].Reference != result {
		return false
	}
	p, ok := e.Entities[ps.List[0].Reference]
	if !ok || p.Schema != id("9012") || p.Fields[id("9121")].Reference != arg {
		return false
	}
	seen := map[wire.ID]bool{}
	todo := []wire.ID{q.Fields[id("9113")].Reference}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if seen[x] {
			continue
		}
		seen[x] = true
		z, ok := e.Entities[x]
		if !ok {
			return false
		}
		if z.Schema == id("90f1") {
			return false
		}
		for _, v := range z.Fields {
			collect(v, &todo)
		}
	}
	return true
}
func typeEntity(q wire.Entity) bool {
	switch q.Schema.String() {
	case id("9010").String(), id("9020").String(), id("9030").String(), id("9040").String(), id("9041").String(), id("9042").String(), id("90f2").String(), id("90f8").String(), id("a010").String(), id("a020").String(), id("a040").String(), id("a050").String():
		return true
	}
	return false
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
func entity(x wire.ID, s string, fs map[string]wire.Value) wire.Entity {
	f := map[wire.ID]wire.Value{}
	for k, v := range fs {
		f[id(k)] = v
	}
	return wire.Entity{ID: x, Schema: id(s), Version: 1, Fields: f}
}
func stable(base []byte, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.durable-state.identity.v1\x00"))
	x := sha256.Sum256(base)
	h.Write(x[:])
	for _, p := range parts {
		_ = binary.Write(h, binary.BigEndian, uint64(len(p)))
		h.Write([]byte(p))
	}
	var out wire.ID
	copy(out[:], h.Sum(nil))
	return out
}
func contentRevision(e wire.Envelope, root wire.ID) []byte {
	q := e.Entities[root]
	q.Fields = clone(q.Fields)
	q.Fields[id("8102")] = blobBytes(make([]byte, 32))
	e.Entities[root] = q
	b, _ := wire.Encode(wire.Envelope{Entities: closure(e.Entities, root)})
	h := sha256.Sum256(append([]byte("seme.durable-state.plan.v1\x00"), b...))
	return h[:]
}
func artifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.durable-state.artifact.v1\x00"), b...))
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
func clone(x map[wire.ID]wire.Value) map[wire.ID]wire.Value {
	o := map[wire.ID]wire.Value{}
	for k, v := range x {
		o[k] = v
	}
	return o
}
func name(s string) bool {
	if len(s) == 0 || len(s) > 128 || !utf8.ValidString(s) {
		return false
	}
	for _, part := range strings.Split(s, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-' || r == '/') {
			return false
		}
	}
	return true
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
