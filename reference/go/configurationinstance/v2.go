package configurationinstance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"unicode/utf8"

	"seme.local/reference/canonicaleval"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/wire"
)

type BoundInput struct {
	Contracts contractcatalog.ProjectContractSetV7
	Base      Input
	Model     BoundModel
	Artifact  []byte
	v3        *V3BaseInput
}
type V3Input struct {
	Contracts contractcatalog.ProjectContractSetV8
	Base      V3BaseInput
	Model     BoundModel
	Artifact  []byte
}

func EmitV3(in V3Input) ([]byte, error) {
	in.Base.Contracts = in.Contracts
	return EmitBound(BoundInput{Model: in.Model, v3: &in.Base})
}
func ValidateV3(in V3Input) error {
	in.Base.Contracts = in.Contracts
	return ValidateBound(BoundInput{Model: in.Model, Artifact: in.Artifact, v3: &in.Base})
}

type BoundModel struct {
	RuntimeInputs []RuntimeInput
	Initializers  []BoundInitializer
}

type RuntimeInput struct {
	Identity   string
	Type       wire.ID
	Capability *wire.ID
}

type ArgumentSourceKind uint64

const (
	ResolvedConfigurationField ArgumentSourceKind = iota
	RuntimeInputSource
	PredecessorOKPayload
	StaticCanonical
	RecordConstruction
)

type ArgumentSource struct {
	Kind            ArgumentSourceKind
	DerivedType     wire.ID
	FieldKey        string
	RuntimeIdentity string
	Predecessor     wire.ID // canonical callable identity
	StaticValue     wire.ID
	Record          *RecordAssembly
}

type RecordAssembly struct {
	RecordType wire.ID
	Members    []RecordMemberBinding
}

type RecordMemberBinding struct {
	Field  wire.ID
	Source ArgumentSource
}

type InitializerArgument struct {
	Parameter wire.ID
	Index     uint64
	Source    ArgumentSource
}

type BoundInitializer struct {
	Callable  wire.ID
	Arguments []InitializerArgument
}

var (
	sRuntimeInput = id("4018")
	sSourceKind   = id("4019")
	sSource       = id("401a")
	sBoundInit    = id("401b")
	sAssembly     = id("401c")
	sMemberBind   = id("401d")
	sArgument     = id("401e")
	sBoundGraph   = id("401f")
)

func EmitBound(in BoundInput) ([]byte, error) {
	base, baseGraph, err := boundComponents(in)
	if err != nil {
		return nil, err
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for x, q := range base.Entities {
		if q.Schema != sModule && q.Schema != sImport {
			e.Entities[x] = clone(q)
		}
	}
	seed := in.Base.Artifact
	if in.v3 != nil {
		seed = in.v3.Artifact
	}
	runtime := map[string]wire.ID{}
	runtimeRefs := []wire.Value{}
	items := append([]RuntimeInput(nil), in.Model.RuntimeInputs...)
	sort.Slice(items, func(i, j int) bool { return items[i].Identity < items[j].Identity })
	for i, v := range items {
		if v.Identity == "" || !utf8.ValidString(v.Identity) || i > 0 && items[i-1].Identity == v.Identity {
			return nil, fmt.Errorf("configuration_v2.runtime_identity")
		}
		x := stableV2(seed, "runtime", v.Identity)
		fs := map[wire.ID]wire.Value{id("4180"): blob([]byte(v.Identity)), id("4181"): ref(v.Type)}
		if v.Capability != nil {
			fs[id("4182")] = ref(*v.Capability)
		}
		if err := put(e.Entities, wire.Entity{ID: x, Schema: sRuntimeInput, Version: 1, Fields: fs}); err != nil {
			return nil, err
		}
		runtime[v.Identity] = x
		runtimeRefs = append(runtimeRefs, ref(x))
	}
	baseInit, fields := boundLookups(base, baseGraph)
	boundIDs := map[wire.ID]wire.ID{}
	initializers := append([]BoundInitializer(nil), in.Model.Initializers...)
	sort.Slice(initializers, func(i, j int) bool {
		return bytes.Compare(initializers[i].Callable[:], initializers[j].Callable[:]) < 0
	})
	for _, v := range initializers {
		b, ok := baseInit[v.Callable]
		if !ok {
			return nil, fmt.Errorf("configuration_v2.initializer")
		}
		boundIDs[v.Callable] = stableV2(seed, "bound-initializer", b.String())
	}
	boundRefs := []wire.Value{}
	var emitSource func(ArgumentSource, string, map[*RecordAssembly]bool) (wire.ID, error)
	emitSource = func(v ArgumentSource, path string, active map[*RecordAssembly]bool) (wire.ID, error) {
		x := stableV2(seed, "source", path)
		kind := stableV2(seed, "source-kind", fmt.Sprint(v.Kind))
		if _, ok := e.Entities[kind]; !ok {
			e.Entities[kind] = wire.Entity{ID: kind, Schema: sSourceKind, Version: 1, Fields: map[wire.ID]wire.Value{id("4190"): {Tag: 3, Unsigned: uint64(v.Kind)}}}
		}
		fs := map[wire.ID]wire.Value{id("41a0"): ref(kind), id("41a1"): ref(v.DerivedType)}
		switch v.Kind {
		case ResolvedConfigurationField:
			f, ok := fields[v.FieldKey]
			if !ok {
				return x, fmt.Errorf("configuration_v2.source_field")
			}
			fs[id("41a2")] = ref(f)
		case RuntimeInputSource:
			r, ok := runtime[v.RuntimeIdentity]
			if !ok {
				return x, fmt.Errorf("configuration_v2.source_runtime")
			}
			fs[id("41a3")] = ref(r)
		case PredecessorOKPayload:
			p, ok := boundIDs[v.Predecessor]
			if !ok {
				return x, fmt.Errorf("configuration_v2.source_predecessor")
			}
			fs[id("41a4")] = ref(p)
		case StaticCanonical:
			fs[id("41a5")] = ref(v.StaticValue)
		case RecordConstruction:
			if v.Record == nil || active[v.Record] {
				return x, fmt.Errorf("configuration_v2.record_cycle")
			}
			active[v.Record] = true
			aid := stableV2(seed, "assembly", path)
			members := []wire.Value{}
			membersModel := append([]RecordMemberBinding(nil), v.Record.Members...)
			sort.Slice(membersModel, func(i, j int) bool { return bytes.Compare(membersModel[i].Field[:], membersModel[j].Field[:]) < 0 })
			for i, m := range membersModel {
				sid, er := emitSource(m.Source, fmt.Sprintf("%s.member.%d", path, i), active)
				if er != nil {
					return x, er
				}
				mid := stableV2(seed, "member", path, m.Field.String())
				if er = put(e.Entities, wire.Entity{ID: mid, Schema: sMemberBind, Version: 1, Fields: map[wire.ID]wire.Value{id("41d0"): ref(m.Field), id("41d1"): ref(sid)}}); er != nil {
					return x, er
				}
				members = append(members, ref(mid))
			}
			delete(active, v.Record)
			sortRefs(members)
			if er := put(e.Entities, wire.Entity{ID: aid, Schema: sAssembly, Version: 1, Fields: map[wire.ID]wire.Value{id("41c0"): ref(v.Record.RecordType), id("41c1"): {Tag: 7, List: members}}}); er != nil {
				return x, er
			}
			fs[id("41a6")] = ref(aid)
		default:
			return x, fmt.Errorf("configuration_v2.source_kind")
		}
		if err := put(e.Entities, wire.Entity{ID: x, Schema: sSource, Version: 1, Fields: fs}); err != nil {
			return x, err
		}
		return x, nil
	}
	for _, v := range initializers {
		bid := boundIDs[v.Callable]
		args := []wire.Value{}
		arguments := append([]InitializerArgument(nil), v.Arguments...)
		sort.Slice(arguments, func(i, j int) bool { return arguments[i].Index < arguments[j].Index })
		for i, a := range arguments {
			sid, er := emitSource(a.Source, fmt.Sprintf("%s.arg.%d", bid.String(), i), map[*RecordAssembly]bool{})
			if er != nil {
				return nil, er
			}
			x := stableV2(seed, "argument", bid.String(), fmt.Sprint(a.Index))
			if er = put(e.Entities, wire.Entity{ID: x, Schema: sArgument, Version: 1, Fields: map[wire.ID]wire.Value{id("41e0"): ref(a.Parameter), id("41e1"): {Tag: 3, Unsigned: a.Index}, id("41e2"): ref(sid)}}); er != nil {
				return nil, er
			}
			args = append(args, ref(x))
		}
		sortRefs(args)
		if err = put(e.Entities, wire.Entity{ID: bid, Schema: sBoundInit, Version: 1, Fields: map[wire.ID]wire.Value{id("41b0"): ref(baseInit[v.Callable]), id("41b1"): {Tag: 7, List: args}}}); err != nil {
			return nil, err
		}
		boundRefs = append(boundRefs, ref(bid))
	}
	sortRefs(runtimeRefs)
	sortRefs(boundRefs)
	graph := stableV2(seed, "bound-graph")
	e.Entities[graph] = wire.Entity{ID: graph, Schema: sBoundGraph, Version: 1, Fields: map[wire.ID]wire.Value{id("41f0"): ref(baseGraph), id("41f1"): {Tag: 7, List: runtimeRefs}, id("41f2"): {Tag: 7, List: boundRefs}, id("41f3"): blob(make([]byte, 32))}}
	q := e.Entities[graph]
	q.Fields[id("41f3")] = blob(BoundGraphRevision(e, graph))
	e.Entities[graph] = q
	module := stableV2(seed, "module")
	imports := []wire.Value{}
	pins := []contractcatalog.Pin{{Module: id("3000"), Revision: id("3001")}, {Module: id("9000"), Revision: id("9023")}, {Module: id("b000"), Revision: id("b003")}, {Module: id("4000"), Revision: id("4005")}}
	label := "configuration-instance-v2"
	if in.v3 != nil {
		pins = []contractcatalog.Pin{{Module: id("3000"), Revision: id("3001")}, {Module: id("9000"), Revision: id("9024")}, {Module: id("b000"), Revision: id("b004")}, {Module: id("4000"), Revision: id("4006")}}
		label = "configuration-instance-v3"
	}
	for _, pin := range pins {
		x := stableV2(seed, "import", pin.Module.String())
		e.Entities[x] = wire.Entity{ID: x, Schema: sImport, Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(pin.Module), id("131"): blob(pin.Revision[:])}}
		imports = append(imports, ref(x))
	}
	sortRefs(imports)
	e.Module = module
	e.Entities[module] = wire.Entity{ID: module, Schema: sModule, Version: 1, Fields: map[wire.ID]wire.Value{id("120"): blob([]byte(label)), id("121"): {Tag: 7, List: imports}, id("122"): {Tag: 7, List: []wire.Value{ref(graph)}}}}
	e.Revision = BoundArtifactRevision(e)
	out, err := wire.Encode(e)
	if err != nil {
		return nil, err
	}
	in.Artifact = out
	if err = ValidateBound(in); err != nil {
		return nil, fmt.Errorf("configuration_v2.emit_validate:%w", err)
	}
	return out, nil
}

func ValidateBound(in BoundInput) error {
	base, baseGraph, err := boundComponents(in)
	if err != nil {
		return err
	}
	e, err := wire.Decode(in.Artifact)
	if err != nil {
		return fmt.Errorf("configuration_v2.wire:%w", err)
	}
	b, _ := wire.Encode(e)
	if !bytes.Equal(b, in.Artifact) {
		return fmt.Errorf("configuration_v2.noncanonical")
	}
	if e.Revision != BoundArtifactRevision(e) {
		return fmt.Errorf("configuration_v2.artifact_revision")
	}
	m := e.Entities[e.Module]
	label := "configuration-instance-v2"
	pins := map[wire.ID]wire.ID{id("3000"): id("3001"), id("9000"): id("9023"), id("b000"): id("b003"), id("4000"): id("4005")}
	if in.v3 != nil {
		label = "configuration-instance-v3"
		pins = map[wire.ID]wire.ID{id("3000"): id("3001"), id("9000"): id("9024"), id("b000"): id("b004"), id("4000"): id("4006")}
	}
	if m.Schema != sModule || m.Version != 1 || string(m.Fields[id("120")].Bytes) != label || !exactPins(e, m, pins) {
		return fmt.Errorf("configuration_v2.module")
	}
	for x, q := range base.Entities {
		if q.Schema != sModule && q.Schema != sImport {
			if got, ok := e.Entities[x]; !ok || !same(q, got) {
				return fmt.Errorf("configuration_v2.component")
			}
		}
	}
	graph, err := one(e, sBoundGraph)
	if err != nil {
		return err
	}
	g := e.Entities[graph]
	if !shape(g, map[wire.ID]byte{id("41f0"): 6, id("41f1"): 7, id("41f2"): 7, id("41f3"): 5}) || g.Fields[id("41f0")].Reference != baseGraph || !sortedUnique(g.Fields[id("41f1")].List) || !sortedUnique(g.Fields[id("41f2")].List) {
		return fmt.Errorf("configuration_v2.graph")
	}
	if !bytes.Equal(g.Fields[id("41f3")].Bytes, BoundGraphRevision(e, graph)) {
		return fmt.Errorf("configuration_v2.revision")
	}
	baseInit, fields := boundLookups(base, baseGraph)
	fieldSet := map[wire.ID]bool{}
	for _, fieldID := range fields {
		fieldSet[fieldID] = true
	}
	owners, _, _ := ownership(base)
	authorized := authorizedCapabilities(base)
	runtime := map[wire.ID]wire.Entity{}
	identities := map[string]bool{}
	for _, r := range g.Fields[id("41f1")].List {
		q := e.Entities[r.Reference]
		if q.Schema != sRuntimeInput || q.Version != 1 || !shapeOptional(q, map[wire.ID]byte{id("4180"): 5, id("4181"): 6}, map[wire.ID]byte{id("4182"): 6}) {
			return fmt.Errorf("configuration_v2.runtime")
		}
		name := string(q.Fields[id("4180")].Bytes)
		if name == "" || !utf8.ValidString(name) || identities[name] || !canonicalType(base, q.Fields[id("4181")].Reference) {
			return fmt.Errorf("configuration_v2.runtime")
		}
		identities[name] = true
		if c, ok := q.Fields[id("4182")]; ok && base.Entities[c.Reference].Schema != sCapability {
			return fmt.Errorf("configuration_v2.runtime_capability")
		}
		runtime[r.Reference] = q
	}
	bound := map[wire.ID]wire.Entity{}
	byBase := map[wire.ID]wire.ID{}
	for _, r := range g.Fields[id("41f2")].List {
		q := e.Entities[r.Reference]
		if q.Schema != sBoundInit || q.Version != 1 || !shape(q, map[wire.ID]byte{id("41b0"): 6, id("41b1"): 7}) || !sortedUnique(q.Fields[id("41b1")].List) {
			return fmt.Errorf("configuration_v2.initializer")
		}
		b := q.Fields[id("41b0")].Reference
		if base.Entities[b].Schema != sInitializer || byBase[b] != (wire.ID{}) {
			return fmt.Errorf("configuration_v2.initializer_base")
		}
		byBase[b] = r.Reference
		bound[r.Reference] = q
	}
	if len(bound) != len(baseInit) {
		return fmt.Errorf("configuration_v2.initializer_coverage")
	}
	var sourceType func(wire.ID, wire.ID, map[wire.ID]bool) (wire.ID, error)
	usedRuntime := map[wire.ID]bool{}
	sourceType = func(x, current wire.ID, active map[wire.ID]bool) (wire.ID, error) {
		if active[x] {
			return wire.ID{}, fmt.Errorf("configuration_v2.source_cycle")
		}
		active[x] = true
		defer delete(active, x)
		q := e.Entities[x]
		if q.Schema != sSource || q.Version != 1 || !shapeOptional(q, map[wire.ID]byte{id("41a0"): 6, id("41a1"): 6}, map[wire.ID]byte{id("41a2"): 6, id("41a3"): 6, id("41a4"): 6, id("41a5"): 6, id("41a6"): 6}) {
			return wire.ID{}, fmt.Errorf("configuration_v2.source")
		}
		kindEntity := e.Entities[q.Fields[id("41a0")].Reference]
		if kindEntity.Schema != sSourceKind || !shape(kindEntity, map[wire.ID]byte{id("4190"): 3}) {
			return wire.ID{}, fmt.Errorf("configuration_v2.source_kind")
		}
		typ := q.Fields[id("41a1")].Reference
		if !canonicalType(base, typ) {
			return wire.ID{}, fmt.Errorf("configuration_v2.source_type")
		}
		code := kindEntity.Fields[id("4190")].Unsigned
		optional := []wire.ID{id("41a2"), id("41a3"), id("41a4"), id("41a5"), id("41a6")}
		present := 0
		for _, f := range optional {
			if _, ok := q.Fields[f]; ok {
				present++
			}
		}
		if len(q.Fields) != 3 || present != 1 {
			return wire.ID{}, fmt.Errorf("configuration_v2.source_union")
		}
		switch code {
		case 0:
			fieldID := q.Fields[id("41a2")].Reference
			f := e.Entities[fieldID]
			if !fieldSet[fieldID] || f.Schema != sField || f.Fields[id("4112")].Reference != typ {
				return wire.ID{}, fmt.Errorf("configuration_v2.field_source")
			}
		case 1:
			runtimeID := q.Fields[id("41a3")].Reference
			r := runtime[runtimeID]
			if r.Schema != sRuntimeInput || r.Fields[id("4181")].Reference != typ {
				return wire.ID{}, fmt.Errorf("configuration_v2.runtime_source")
			}
			usedRuntime[runtimeID] = true
			if c, ok := r.Fields[id("4182")]; ok {
				owner := base.Entities[bound[current].Fields[id("41b0")].Reference].Fields[id("4150")].Reference
				if !owners[owner] || !authorized[owner][c.Reference] {
					return wire.ID{}, fmt.Errorf("configuration_v2.runtime_authorization")
				}
			}
		case 2:
			p := q.Fields[id("41a4")].Reference
			if _, ok := bound[p]; !ok {
				return wire.ID{}, fmt.Errorf("configuration_v2.predecessor")
			}
			baseCurrent := base.Entities[bound[current].Fields[id("41b0")].Reference]
			allowed := false
			for _, d := range baseCurrent.Fields[id("4152")].List {
				if byBase[d.Reference] == p {
					allowed = true
				}
			}
			if !allowed {
				return wire.ID{}, fmt.Errorf("configuration_v2.predecessor_dependency")
			}
			call := base.Entities[base.Entities[bound[p].Fields[id("41b0")].Reference].Fields[id("4151")].Reference]
			result := base.Entities[call.Fields[id("9112")].Reference]
			if result.Schema != sResultType || result.Fields[id("9400")].Reference != typ {
				return wire.ID{}, fmt.Errorf("configuration_v2.predecessor_type")
			}
		case 3:
			v := q.Fields[id("41a5")].Reference
			t, ok := expressionType(base, v)
			if !ok || t != typ || !pure(base, v) {
				return wire.ID{}, fmt.Errorf("configuration_v2.static_type")
			}
		case 4:
			a := e.Entities[q.Fields[id("41a6")].Reference]
			if a.Schema != sAssembly || a.Version != 1 || !shape(a, map[wire.ID]byte{id("41c0"): 6, id("41c1"): 7}) || a.Fields[id("41c0")].Reference != typ || base.Entities[typ].Schema != id("9030") || !sortedUnique(a.Fields[id("41c1")].List) {
				return wire.ID{}, fmt.Errorf("configuration_v2.record")
			}
			recordFields := base.Entities[typ].Fields[id("9301")].List
			if len(recordFields) != len(a.Fields[id("41c1")].List) {
				return wire.ID{}, fmt.Errorf("configuration_v2.record_coverage")
			}
			seen := map[wire.ID]bool{}
			for _, mr := range a.Fields[id("41c1")].List {
				mb := e.Entities[mr.Reference]
				if mb.Schema != sMemberBind || mb.Version != 1 || !shape(mb, map[wire.ID]byte{id("41d0"): 6, id("41d1"): 6}) {
					return wire.ID{}, fmt.Errorf("configuration_v2.record_member")
				}
				f := mb.Fields[id("41d0")].Reference
				if seen[f] {
					return wire.ID{}, fmt.Errorf("configuration_v2.record_member")
				}
				seen[f] = true
				ft := base.Entities[f].Fields[id("9311")].Reference
				st, er := sourceType(mb.Fields[id("41d1")].Reference, current, active)
				if er != nil || st != ft {
					return wire.ID{}, fmt.Errorf("configuration_v2.record_member_type")
				}
			}
			for _, f := range recordFields {
				if !seen[f.Reference] {
					return wire.ID{}, fmt.Errorf("configuration_v2.record_coverage")
				}
			}
		default:
			return wire.ID{}, fmt.Errorf("configuration_v2.source_kind")
		}
		return typ, nil
	}
	for bid, q := range bound {
		baseEntity := base.Entities[q.Fields[id("41b0")].Reference]
		call := base.Entities[baseEntity.Fields[id("4151")].Reference]
		params := call.Fields[id("9111")].List
		args := q.Fields[id("41b1")].List
		if len(params) != len(args) {
			return fmt.Errorf("configuration_v2.arguments")
		}
		seen := map[uint64]bool{}
		for _, ar := range args {
			a := e.Entities[ar.Reference]
			if a.Schema != sArgument || a.Version != 1 || !shape(a, map[wire.ID]byte{id("41e0"): 6, id("41e1"): 3, id("41e2"): 6}) {
				return fmt.Errorf("configuration_v2.argument")
			}
			i := a.Fields[id("41e1")].Unsigned
			if i >= uint64(len(params)) || seen[i] || a.Fields[id("41e0")].Reference != params[i].Reference {
				return fmt.Errorf("configuration_v2.argument_order")
			}
			seen[i] = true
			want := base.Entities[params[i].Reference].Fields[id("9121")].Reference
			got, er := sourceType(a.Fields[id("41e2")].Reference, bid, map[wire.ID]bool{})
			if er != nil {
				return er
			}
			if got != want {
				return fmt.Errorf("configuration_v2.argument_type")
			}
		}
	}
	if len(usedRuntime) != len(runtime) {
		return fmt.Errorf("configuration_v2.runtime_coverage")
	}
	used := map[wire.ID]bool{e.Module: true}
	for _, r := range m.Fields[id("121")].List {
		used[r.Reference] = true
	}
	for x, q := range base.Entities {
		if q.Schema != sModule && q.Schema != sImport {
			used[x] = true
		}
	}
	for x := range closure(e.Entities, graph) {
		used[x] = true
	}
	if len(used) != len(e.Entities) {
		return fmt.Errorf("configuration_v2.orphan")
	}
	return nil
}

func boundComponents(in BoundInput) (wire.Envelope, wire.ID, error) {
	if in.v3 != nil {
		if err := ValidateV3Base(*in.v3); err != nil {
			return wire.Envelope{}, wire.ID{}, fmt.Errorf("configuration_v3.base:%w", err)
		}
		e, err := wire.Decode(in.v3.Artifact)
		if err != nil {
			return e, wire.ID{}, err
		}
		g, err := one(e, sGraph)
		return e, g, err
	}
	if !in.Contracts.Validated() || in.Contracts.Configuration().Pin() != (contractcatalog.Pin{Module: id("4000"), Revision: id("4005")}) || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e009")}) {
		return wire.Envelope{}, wire.ID{}, fmt.Errorf("configuration_v2.contracts")
	}
	if err := validate(in.Base, true); err != nil {
		return wire.Envelope{}, wire.ID{}, fmt.Errorf("configuration_v2.base:%w", err)
	}
	e, err := wire.Decode(in.Base.Artifact)
	if err != nil {
		return e, wire.ID{}, err
	}
	g, err := one(e, sGraph)
	return e, g, err
}
func boundLookups(e wire.Envelope, g wire.ID) (map[wire.ID]wire.ID, map[string]wire.ID) {
	init := map[wire.ID]wire.ID{}
	fields := map[string]wire.ID{}
	q := e.Entities[g]
	for _, r := range q.Fields[id("4103")].List {
		v := e.Entities[r.Reference]
		init[v.Fields[id("4151")].Reference] = r.Reference
	}
	for _, r := range q.Fields[id("4100")].List {
		v := e.Entities[r.Reference]
		fields[string(v.Fields[id("4111")].Bytes)] = r.Reference
	}
	return init, fields
}
func expressionType(e wire.Envelope, x wire.ID) (wire.ID, bool) {
	return canonicaleval.ExpressionType(e, x)
}
func BoundGraphRevision(e wire.Envelope, root wire.ID) []byte {
	q := clone(e.Entities[root])
	q.Fields[id("41f3")] = blob(make([]byte, 32))
	entities := map[wire.ID]wire.Entity{}
	for x, v := range e.Entities {
		entities[x] = v
	}
	entities[root] = q
	b, _ := wire.Encode(wire.Envelope{Entities: closure(entities, root)})
	h := sha256.Sum256(append([]byte("seme.configuration.bound-graph.v2\x00"), b...))
	return h[:]
}
func BoundArtifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.configuration.artifact.v2\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func stableV2(seed []byte, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.configuration-instance.identity.v2\x00"))
	s := sha256.Sum256(seed)
	h.Write(s[:])
	for _, p := range parts {
		_ = binary.Write(h, binary.BigEndian, uint64(len(p)))
		h.Write([]byte(p))
	}
	var x wire.ID
	copy(x[:], h.Sum(nil))
	return x
}
