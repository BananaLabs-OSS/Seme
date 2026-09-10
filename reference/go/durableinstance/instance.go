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
	"seme.local/reference/projectv9instance"
	"seme.local/reference/wire"
)

const MaxPayloadBytes = 1 << 20
const MaxKeyBytes = 512
const CodecIdentity = "seme.durable-state.canonical.v1"
const LoadEffectIdentity = "seme.storage.load.v1"
const CompareExchangeEffectIdentity = "seme.storage.compare_exchange.v1"

type operationAuthority struct {
	loadEffect, compareExchangeEffect, loadCapability, compareExchangeCapability, codec, tokenPolicy string
	loadSequence, compareExchangeSequence                                                            uint64
}

type Model struct {
	Identity, PortIdentity                                                                          string
	StateOwner, PortOwner, Version1Type, Version2Type, ErrorType, Validator1, Validator2, Migration wire.ID
	KeyType                                                                                         wire.ID
	MaximumPayloadBytes, MaximumKeyBytes                                                            uint64
}
type Inputs struct {
	Contracts contractcatalog.ProjectContractSetV10
	ProjectV9 projectv9instance.Inputs
	Artifact  []byte
	Model     Model
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
	if err := projectv9instance.Validate(in.ProjectV9); err != nil {
		return nil, fmt.Errorf("durable_instance.project_v9:%w", err)
	}
	if !sameAuthorities(in.Contracts, in.ProjectV9.Contracts) {
		return nil, fmt.Errorf("durable_instance.project_authorities")
	}
	authority, err := authenticatedOperationAuthority(in.Contracts.DurableState())
	if err != nil {
		return nil, err
	}
	base, err := wire.Decode(in.ProjectV9.Composed)
	if err != nil {
		return nil, fmt.Errorf("durable_instance.project_v9:%w", err)
	}
	if err = baseAuthority(base); err != nil {
		return nil, err
	}
	m := in.Model
	if !name(m.Identity) || !name(m.PortIdentity) || m.MaximumPayloadBytes == 0 || m.MaximumPayloadBytes > MaxPayloadBytes || m.MaximumKeyBytes == 0 || m.MaximumKeyBytes > MaxKeyBytes {
		return nil, fmt.Errorf("durable_instance.metadata")
	}
	owners, _, functions := ownership(base)
	if !owners[m.StateOwner] || !owners[m.PortOwner] {
		return nil, fmt.Errorf("durable_instance.owner")
	}
	for _, x := range []wire.ID{m.Version1Type, m.Version2Type, m.ErrorType} {
		if !typeEntity(base.Entities[x]) {
			return nil, fmt.Errorf("durable_instance.type:%s", x)
		}
	}
	if base.Entities[m.KeyType].Schema != id("9040") {
		return nil, fmt.Errorf("durable_instance.key_type")
	}
	if m.Version1Type == m.Version2Type {
		return nil, fmt.Errorf("durable_instance.version_type_mix")
	}
	if m.Validator1 == m.Validator2 || m.Validator1 == m.Migration || m.Validator2 == m.Migration {
		return nil, fmt.Errorf("durable_instance.function_mix")
	}
	if !pureResultFunction(base, m.Validator1, m.Version1Type, m.Version1Type, m.ErrorType, functions[m.Validator1] == m.StateOwner) || !pureResultFunction(base, m.Validator2, m.Version2Type, m.Version2Type, m.ErrorType, functions[m.Validator2] == m.StateOwner) || !pureResultFunction(base, m.Migration, m.Version1Type, m.Version2Type, m.ErrorType, functions[m.Migration] == m.StateOwner) {
		return nil, fmt.Errorf("durable_instance.function")
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for x, q := range base.Entities {
		if q.Schema != id("12") && q.Schema != id("13") {
			e.Entities[x] = q
		}
	}
	baseBytes := in.ProjectV9.Composed
	fam := stable(baseBytes, "family", m.Identity)
	v1 := stable(baseBytes, "version", "1")
	v2 := stable(baseBytes, "version", "2")
	a := stable(baseBytes, "validator", "1")
	b := stable(baseBytes, "validator", "2")
	mig := stable(baseBytes, "migration")
	port := stable(baseBytes, "port", m.PortIdentity)
	plan := stable(baseBytes, "plan")
	loadCapability := stable(baseBytes, "capability", authority.loadCapability)
	casCapability := stable(baseBytes, "capability", authority.compareExchangeCapability)
	loadEffect := stable(baseBytes, "effect", authority.loadEffect)
	casEffect := stable(baseBytes, "effect", authority.compareExchangeEffect)
	e.Entities[loadCapability] = entity(loadCapability, "16", map[string]wire.Value{"160": blob(authority.loadCapability)})
	e.Entities[casCapability] = entity(casCapability, "16", map[string]wire.Value{"160": blob(authority.compareExchangeCapability)})
	e.Entities[loadEffect] = entity(loadEffect, "15", map[string]wire.Value{"150": blob(authority.loadEffect), "151": ref(loadCapability)})
	e.Entities[casEffect] = entity(casEffect, "15", map[string]wire.Value{"150": blob(authority.compareExchangeEffect), "151": ref(casCapability)})
	e.Entities[v1] = entity(v1, "8012", map[string]wire.Value{"8120": ref(fam), "8121": u(1), "8122": ref(m.Version1Type)})
	e.Entities[v2] = entity(v2, "8012", map[string]wire.Value{"8120": ref(fam), "8121": u(2), "8122": ref(m.Version2Type)})
	e.Entities[a] = entity(a, "8013", map[string]wire.Value{"8130": ref(v1), "8131": ref(m.Validator1)})
	e.Entities[b] = entity(b, "8013", map[string]wire.Value{"8130": ref(v2), "8131": ref(m.Validator2)})
	e.Entities[mig] = entity(mig, "8014", map[string]wire.Value{"8140": ref(v1), "8141": ref(v2), "8142": ref(m.Migration)})
	e.Entities[fam] = entity(fam, "8011", map[string]wire.Value{"8110": blob(m.Identity), "8111": ref(m.StateOwner), "8112": ref(v2), "8113": list(v1, v2), "8114": list(a, b), "8115": list(mig)})
	e.Entities[port] = entity(port, "8015", map[string]wire.Value{"8150": blob(m.PortIdentity), "8151": ref(m.PortOwner), "8152": ref(fam), "8153": ref(loadCapability), "8154": ref(casCapability), "8155": ref(loadEffect), "8156": ref(casEffect), "8157": ref(m.KeyType), "8158": u(m.MaximumPayloadBytes), "8159": blob(authority.codec), "815a": u(m.MaximumKeyBytes), "815b": ref(m.ErrorType), "815c": ref(id("8016"))})
	e.Entities[plan] = entity(plan, "8010", map[string]wire.Value{"8100": list(fam), "8101": list(port), "8102": blobBytes(make([]byte, 32))})
	q := e.Entities[plan]
	q.Fields[id("8102")] = blobBytes(contentRevision(e, plan))
	e.Entities[plan] = q
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("3000"), Revision: id("3001")}, {Module: id("8000"), Revision: id("8001")}, {Module: id("9000"), Revision: id("9024")}, {Module: id("b000"), Revision: id("b004")}, {Module: id("e000"), Revision: id("e00b")}} {
		x := stable(baseBytes, "import", pin.Module.String())
		e.Entities[x] = entity(x, "13", map[string]wire.Value{"130": ref(pin.Module), "131": blobBytes(pin.Revision[:])})
		imports = append(imports, ref(x))
	}
	sort.Slice(imports, func(i, j int) bool { return bytes.Compare(imports[i].Reference[:], imports[j].Reference[:]) < 0 })
	e.Module = stable(baseBytes, "module")
	e.Entities[e.Module] = entity(e.Module, "12", map[string]wire.Value{"120": blob("durable-state-plan-v1"), "121": {Tag: 7, List: imports}, "122": list(plan)})
	e.Revision = artifactRevision(e)
	return wire.Encode(e)
}

// authenticatedOperationAuthority reads the immutable operation profile from
// the already digest-authenticated Durable State contract. Runtime providers
// must realize this profile; this metadata alone does not certify a provider.
func authenticatedOperationAuthority(c contractcatalog.Contract) (operationAuthority, error) {
	if !c.Validated() || c.Pin() != (contractcatalog.Pin{Module: id("8000"), Revision: id("8001")}) {
		return operationAuthority{}, fmt.Errorf("durable_instance.operation_contract")
	}
	e := c.Envelope()
	q, ok := e.Entities[id("8016")]
	if !ok || q.Schema != id("10") || q.Version != 1 {
		return operationAuthority{}, fmt.Errorf("durable_instance.operation_authority")
	}
	fields := q.Fields[id("101")]
	if fields.Tag != 7 || len(fields.List) != 13 {
		return operationAuthority{}, fmt.Errorf("durable_instance.operation_authority")
	}
	names := map[wire.ID]string{}
	for _, v := range fields.List {
		f, exists := e.Entities[v.Reference]
		n := f.Fields[id("110")]
		if v.Tag != 6 || !exists || f.Schema != id("11") || n.Tag != 5 {
			return operationAuthority{}, fmt.Errorf("durable_instance.operation_authority")
		}
		names[v.Reference] = string(n.Bytes)
	}
	load, cas := names[id("8160")], names[id("8161")]
	loadCap, casCap := names[id("8162")], names[id("8163")]
	codec, policy := names[id("8166")], names[id("8167")]
	if load == "" || cas == "" || load == cas || loadCap == "" || loadCap == casCap || codec != CodecIdentity || policy != "opaque-thread-only" || names[id("8164")] != "seme.storage.sequence.0.load" || names[id("8165")] != "seme.storage.sequence.1.compare_exchange" {
		return operationAuthority{}, fmt.Errorf("durable_instance.operation_profile")
	}
	for field, schema := range map[string]string{"8168": "801a", "8169": "801c", "816a": "801d", "816b": "801e", "816c": "801f"} {
		f := e.Entities[id(field)]
		constraint := f.Fields[id("111")]
		if constraint.Tag != 8 || constraint.Record[id("2001")].Tag != 6 || constraint.Record[id("2001")].Reference != id("10") || e.Entities[id(schema)].Schema != id("10") {
			return operationAuthority{}, fmt.Errorf("durable_instance.operation_schema:%s", field)
		}
	}
	return operationAuthority{loadEffect: load, compareExchangeEffect: cas, loadCapability: loadCap, compareExchangeCapability: casCap, codec: codec, tokenPolicy: policy, loadSequence: 0, compareExchangeSequence: 1}, nil
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
func pureResultFunction(e wire.Envelope, x, arg, okType, errorType wire.ID, owned bool) bool {
	q, ok := e.Entities[x]
	if !ok || !owned || q.Schema != id("9011") {
		return false
	}
	ps := q.Fields[id("9111")]
	result := q.Fields[id("9112")]
	r, rok := e.Entities[result.Reference]
	if ps.Tag != 7 || len(ps.List) != 1 || result.Tag != 6 || !rok || r.Schema != id("9042") || r.Fields[id("9400")].Tag != 6 || r.Fields[id("9400")].Reference != okType || r.Fields[id("9401")].Tag != 6 || r.Fields[id("9401")].Reference != errorType {
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
func sameAuthorities(a contractcatalog.ProjectContractSetV10, b contractcatalog.ProjectContractSetV9) bool {
	return a.Foundation().Pin() == b.Foundation().Pin() && a.Execution().Pin() == b.Execution().Pin() && a.Package().Pin() == b.Package().Pin() && a.Dependency().Pin() == b.Dependency().Pin() && a.Configuration().Pin() == b.Configuration().Pin() && a.Resource().Pin() == b.Resource().Pin() && b.Project().Pin() == (contractcatalog.Pin{Module: id("e000"), Revision: id("e00b")})
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
