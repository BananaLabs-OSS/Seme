// Package configurationinstance emits and validates immutable Configuration v1 instances.
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
	"seme.local/reference/projectv5instance"
	"seme.local/reference/wire"
)

type Input struct {
	Contracts contractcatalog.ProjectContractSetV6
	ProjectV5 projectv5instance.Inputs
	Model     Model
	Artifact  []byte
}
type Model struct {
	Fields       []Field
	Values       []ResolvedValue
	Validations  []Validation
	Initializers []Initializer
	Transitions  []Transition
}
type Field struct {
	Owner, Type, Origin wire.ID
	Key                 string
	Default             *wire.ID
	Required            bool
}
type ResolutionOrigin uint64

const (
	Explicit ResolutionOrigin = iota
	Default
	CapabilityInput
)

type ResolvedValue struct {
	Key        string
	Origin     ResolutionOrigin
	Value      *wire.ID
	Capability *wire.ID
}
type Validation struct {
	Key       string
	Validator wire.ID
	Order     uint64
}
type Initializer struct {
	Owner, Callable wire.ID
	Dependencies    []wire.ID
	Order           uint64
}
type Transition struct {
	Initializer     wire.ID
	From, To, Order uint64
}

var (
	sModule           = id("12")
	sImport           = id("13")
	sPackage          = id("b010")
	sDetail           = id("b021")
	sMember           = id("b022")
	sOrigin           = id("b026")
	sFunction         = id("9011")
	sParameter        = id("9012")
	sResultType       = id("9042")
	sEffectInvoke     = id("90f1")
	sCapability       = id("16")
	sGraph            = id("4010")
	sField            = id("4011")
	sValue            = id("4012")
	sResolutionOrigin = id("4013")
	sValidation       = id("4014")
	sInitializer      = id("4015")
	sState            = id("4016")
	sTransition       = id("4017")
)

func Emit(in Input) ([]byte, error) {
	return emit(in, false)
}

// EmitBindableBase emits a Configuration v1 structural base that may contain
// parameterized initializers. It is not a valid standalone v1 execution plan;
// it is accepted only as input to Configuration v2, which must bind every
// parameter exactly.
func EmitBindableBase(in Input) ([]byte, error) {
	return emit(in, true)
}

func emit(in Input, allowParameterized bool) ([]byte, error) {
	base, err := components(in)
	if err != nil {
		return nil, err
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for x, q := range base.Entities {
		if q.Schema != sModule && q.Schema != sImport {
			e.Entities[x] = clone(q)
		}
	}
	fields := append([]Field(nil), in.Model.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Key < fields[j].Key })
	fieldIDs := map[string]wire.ID{}
	fieldRefs := []wire.Value{}
	for i, f := range fields {
		if f.Key == "" || !utf8.ValidString(f.Key) || i > 0 && fields[i-1].Key == f.Key {
			return nil, fmt.Errorf("configuration.field_key")
		}
		x := stable(in.ProjectV5.Composed, "field", f.Key)
		fieldIDs[f.Key] = x
		required := byte(1)
		if f.Required {
			required = 2
		}
		fs := map[wire.ID]wire.Value{id("4110"): ref(f.Owner), id("4111"): blob([]byte(f.Key)), id("4112"): ref(f.Type), id("4114"): {Tag: required}, id("4115"): ref(f.Origin)}
		if f.Default != nil {
			fs[id("4113")] = ref(*f.Default)
		}
		if err = put(e.Entities, wire.Entity{ID: x, Schema: sField, Version: 1, Fields: fs}); err != nil {
			return nil, err
		}
		fieldRefs = append(fieldRefs, ref(x))
	}
	valueRefs := []wire.Value{}
	for _, v := range in.Model.Values {
		fid, ok := fieldIDs[v.Key]
		if !ok {
			return nil, fmt.Errorf("configuration.value_field")
		}
		x := stable(in.ProjectV5.Composed, "value", v.Key)
		oid := stable(in.ProjectV5.Composed, "resolution-origin", fmt.Sprint(v.Origin))
		if _, ok := e.Entities[oid]; !ok {
			e.Entities[oid] = wire.Entity{ID: oid, Schema: sResolutionOrigin, Version: 1, Fields: map[wire.ID]wire.Value{id("4130"): {Tag: 3, Unsigned: uint64(v.Origin)}}}
		}
		fs := map[wire.ID]wire.Value{id("4120"): ref(fid), id("4122"): ref(oid)}
		if v.Value != nil {
			fs[id("4121")] = ref(*v.Value)
		}
		if v.Capability != nil {
			fs[id("4123")] = ref(*v.Capability)
		}
		if err = put(e.Entities, wire.Entity{ID: x, Schema: sValue, Version: 1, Fields: fs}); err != nil {
			return nil, err
		}
		valueRefs = append(valueRefs, ref(x))
	}
	validationRefs := []wire.Value{}
	for _, v := range in.Model.Validations {
		x := stable(in.ProjectV5.Composed, "validation", v.Key, fmt.Sprint(v.Order), v.Validator.String())
		fs := map[wire.ID]wire.Value{id("4141"): ref(v.Validator), id("4142"): {Tag: 3, Unsigned: v.Order}}
		if v.Key != "" {
			fid, ok := fieldIDs[v.Key]
			if !ok {
				return nil, fmt.Errorf("configuration.validation_field")
			}
			fs[id("4140")] = ref(fid)
		}
		if err = put(e.Entities, wire.Entity{ID: x, Schema: sValidation, Version: 1, Fields: fs}); err != nil {
			return nil, err
		}
		validationRefs = append(validationRefs, ref(x))
	}
	initializerIDs := map[wire.ID]wire.ID{}
	for _, v := range in.Model.Initializers {
		initializerIDs[v.Callable] = stable(in.ProjectV5.Composed, "initializer", v.Owner.String(), v.Callable.String())
	}
	initializerRefs := []wire.Value{}
	for _, v := range in.Model.Initializers {
		deps := []wire.Value{}
		for _, call := range v.Dependencies {
			x, ok := initializerIDs[call]
			if !ok {
				return nil, fmt.Errorf("configuration.initializer_dependency")
			}
			deps = append(deps, ref(x))
		}
		sortRefs(deps)
		x := initializerIDs[v.Callable]
		if err = put(e.Entities, wire.Entity{ID: x, Schema: sInitializer, Version: 1, Fields: map[wire.ID]wire.Value{id("4150"): ref(v.Owner), id("4151"): ref(v.Callable), id("4152"): {Tag: 7, List: deps}, id("4153"): {Tag: 3, Unsigned: v.Order}}}); err != nil {
			return nil, err
		}
		initializerRefs = append(initializerRefs, ref(x))
	}
	stateIDs := map[uint64]wire.ID{}
	state := func(code uint64) wire.ID {
		if x, ok := stateIDs[code]; ok {
			return x
		}
		x := stable(in.ProjectV5.Composed, "lifecycle-state", fmt.Sprint(code))
		stateIDs[code] = x
		e.Entities[x] = wire.Entity{ID: x, Schema: sState, Version: 1, Fields: map[wire.ID]wire.Value{id("4160"): {Tag: 3, Unsigned: code}}}
		return x
	}
	transitionRefs := []wire.Value{}
	for _, v := range in.Model.Transitions {
		init, ok := initializerIDs[v.Initializer]
		if !ok {
			return nil, fmt.Errorf("configuration.transition_initializer")
		}
		if v.From > 4 || v.To > 4 {
			return nil, fmt.Errorf("configuration.transition_state")
		}
		from, to := state(v.From), state(v.To)
		x := stable(in.ProjectV5.Composed, "transition", init.String(), fmt.Sprint(v.Order))
		if err = put(e.Entities, wire.Entity{ID: x, Schema: sTransition, Version: 1, Fields: map[wire.ID]wire.Value{id("4170"): ref(init), id("4171"): ref(from), id("4172"): ref(to), id("4173"): {Tag: 3, Unsigned: v.Order}}}); err != nil {
			return nil, err
		}
		transitionRefs = append(transitionRefs, ref(x))
	}
	for _, xs := range [][]wire.Value{fieldRefs, valueRefs, validationRefs, initializerRefs, transitionRefs} {
		sortRefs(xs)
	}
	graph := stable(in.ProjectV5.Composed, "configuration-graph")
	e.Entities[graph] = wire.Entity{ID: graph, Schema: sGraph, Version: 1, Fields: map[wire.ID]wire.Value{id("4100"): {Tag: 7, List: fieldRefs}, id("4101"): {Tag: 7, List: valueRefs}, id("4102"): {Tag: 7, List: validationRefs}, id("4103"): {Tag: 7, List: initializerRefs}, id("4104"): {Tag: 7, List: transitionRefs}, id("4105"): blob(make([]byte, 32))}}
	q := e.Entities[graph]
	q.Fields[id("4105")] = blob(GraphRevision(e, graph))
	e.Entities[graph] = q
	module := stable(in.ProjectV5.Composed, "module")
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("3000"), Revision: id("3001")}, {Module: id("9000"), Revision: id("9023")}, {Module: id("b000"), Revision: id("b003")}, {Module: id("4000"), Revision: id("4001")}} {
		x := stable(in.ProjectV5.Composed, "import", pin.Module.String())
		e.Entities[x] = wire.Entity{ID: x, Schema: sImport, Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(pin.Module), id("131"): blob(pin.Revision[:])}}
		imports = append(imports, ref(x))
	}
	sortRefs(imports)
	e.Module = module
	e.Entities[module] = wire.Entity{ID: module, Schema: sModule, Version: 1, Fields: map[wire.ID]wire.Value{id("120"): blob([]byte("configuration-instance-v1")), id("121"): {Tag: 7, List: imports}, id("122"): {Tag: 7, List: []wire.Value{ref(graph)}}}}
	e.Revision = ArtifactRevision(e)
	out, err := wire.Encode(e)
	if err != nil {
		return nil, err
	}
	in.Artifact = out
	if err = validate(in, allowParameterized); err != nil {
		return nil, fmt.Errorf("configuration.emit_validate:%w", err)
	}
	return out, nil
}

func Validate(in Input) error {
	return validate(in, false)
}

// ValidateBindableBase validates the complete v1 structure while deferring
// only the zero-parameter initializer restriction to Configuration v2.
func ValidateBindableBase(in Input) error {
	return validate(in, true)
}

func validate(in Input, allowParameterized bool) error {
	base, err := components(in)
	if err != nil {
		return err
	}
	e, err := wire.Decode(in.Artifact)
	if err != nil {
		return fmt.Errorf("configuration.wire:%w", err)
	}
	canonical, _ := wire.Encode(e)
	if !bytes.Equal(canonical, in.Artifact) {
		return fmt.Errorf("configuration.noncanonical")
	}
	if e.Revision != ArtifactRevision(e) {
		return fmt.Errorf("configuration.artifact_revision")
	}
	m := e.Entities[e.Module]
	if m.Schema != sModule || m.Version != 1 || string(m.Fields[id("120")].Bytes) != "configuration-instance-v1" {
		return fmt.Errorf("configuration.module")
	}
	if !exactPins(e, m, map[wire.ID]wire.ID{id("3000"): id("3001"), id("9000"): id("9023"), id("b000"): id("b003"), id("4000"): id("4001")}) {
		return fmt.Errorf("configuration.pins")
	}
	graph, err := one(e, sGraph)
	if err != nil {
		return err
	}
	if m.Fields[id("122")].Tag != 7 || len(m.Fields[id("122")].List) != 1 || m.Fields[id("122")].List[0].Reference != graph {
		return fmt.Errorf("configuration.export")
	}
	for x, q := range base.Entities {
		if q.Schema == sModule || q.Schema == sImport {
			continue
		}
		if got, ok := e.Entities[x]; !ok || !same(q, got) {
			return fmt.Errorf("configuration.component:%s", x)
		}
	}
	if err = validateGraph(e, base, graph, allowParameterized); err != nil {
		return err
	}
	if !bytes.Equal(e.Entities[graph].Fields[id("4105")].Bytes, GraphRevision(e, graph)) {
		return fmt.Errorf("configuration.revision")
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
		return fmt.Errorf("configuration.orphan")
	}
	return nil
}

func components(in Input) (wire.Envelope, error) {
	if !in.Contracts.Validated() || in.Contracts.Configuration().Pin() != (contractcatalog.Pin{Module: id("4000"), Revision: id("4001")}) || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e008")}) {
		return wire.Envelope{}, fmt.Errorf("configuration.contracts")
	}
	if err := projectv5instance.Validate(in.ProjectV5); err != nil {
		return wire.Envelope{}, fmt.Errorf("configuration.project_v5:%w", err)
	}
	e, err := wire.Decode(in.ProjectV5.Composed)
	return e, err
}

func validateGraph(e, base wire.Envelope, graph wire.ID, allowParameterized bool) error {
	g := e.Entities[graph]
	if g.Schema != sGraph || g.Version != 1 || !shape(g, map[wire.ID]byte{id("4100"): 7, id("4101"): 7, id("4102"): 7, id("4103"): 7, id("4104"): 7, id("4105"): 5}) {
		return fmt.Errorf("configuration.graph_shape")
	}
	for _, field := range []wire.ID{id("4100"), id("4101"), id("4102"), id("4103"), id("4104")} {
		if !sortedUnique(g.Fields[field].List) {
			return fmt.Errorf("configuration.graph_order")
		}
	}
	owners, origins, members := ownership(base)
	authorized := authorizedCapabilities(base)
	fields := map[wire.ID]wire.Entity{}
	keys := map[string]wire.ID{}
	for _, r := range g.Fields[id("4100")].List {
		q, ok := e.Entities[r.Reference]
		if r.Tag != 6 || !ok || q.Schema != sField || q.Version != 1 || !shapeOptional(q, map[wire.ID]byte{id("4110"): 6, id("4111"): 5, id("4112"): 6, id("4114"): 255, id("4115"): 6}, map[wire.ID]byte{id("4113"): 6}) {
			return fmt.Errorf("configuration.field_shape")
		}
		key := string(q.Fields[id("4111")].Bytes)
		owner := q.Fields[id("4110")].Reference
		if key == "" || !utf8.ValidString(key) || keys[key] != (wire.ID{}) || !owners[owner] || origins[q.Fields[id("4115")].Reference] != owner {
			return fmt.Errorf("configuration.field")
		}
		typ := q.Fields[id("4112")].Reference
		if !canonicalType(base, typ) {
			return fmt.Errorf("configuration.field_type")
		}
		if d, ok := q.Fields[id("4113")]; ok {
			if t, yes := canonicaleval.ExpressionType(base, d.Reference); !yes || t != typ || !pure(base, d.Reference) {
				return fmt.Errorf("configuration.default_type")
			}
		}
		keys[key] = r.Reference
		fields[r.Reference] = q
	}
	values := map[wire.ID]bool{}
	for _, r := range g.Fields[id("4101")].List {
		q := e.Entities[r.Reference]
		if r.Tag != 6 || q.Schema != sValue || q.Version != 1 || !shapeOptional(q, map[wire.ID]byte{id("4120"): 6, id("4122"): 6}, map[wire.ID]byte{id("4121"): 6, id("4123"): 6}) {
			return fmt.Errorf("configuration.value_shape")
		}
		fid := q.Fields[id("4120")].Reference
		f, ok := fields[fid]
		if !ok || values[fid] {
			return fmt.Errorf("configuration.value_field")
		}
		values[fid] = true
		o := e.Entities[q.Fields[id("4122")].Reference]
		if o.Schema != sResolutionOrigin || o.Version != 1 || !shape(o, map[wire.ID]byte{id("4130"): 3}) {
			return fmt.Errorf("configuration.origin_shape")
		}
		code := o.Fields[id("4130")].Unsigned
		value, vok := q.Fields[id("4121")]
		cap, cok := q.Fields[id("4123")]
		switch code {
		case 0:
			if !vok || cok {
				return fmt.Errorf("configuration.explicit")
			}
		case 1:
			d, dok := f.Fields[id("4113")]
			if !dok || !vok || cok || value.Reference != d.Reference || f.Fields[id("4114")].Tag == 2 {
				return fmt.Errorf("configuration.default")
			}
		case 2:
			if vok || !cok || e.Entities[cap.Reference].Schema != sCapability || !authorized[f.Fields[id("4110")].Reference][cap.Reference] {
				return fmt.Errorf("configuration.capability")
			}
		default:
			return fmt.Errorf("configuration.origin")
		}
		if vok {
			if t, yes := canonicaleval.ExpressionType(base, value.Reference); !yes || t != f.Fields[id("4112")].Reference || !pure(base, value.Reference) {
				return fmt.Errorf("configuration.value_type")
			}
		}
	}
	if len(values) != len(fields) {
		return fmt.Errorf("configuration.coverage")
	}
	validationOrders := map[uint64]bool{}
	for _, r := range g.Fields[id("4102")].List {
		q := e.Entities[r.Reference]
		if r.Tag != 6 || q.Schema != sValidation || q.Version != 1 || !shapeOptional(q, map[wire.ID]byte{id("4141"): 6, id("4142"): 3}, map[wire.ID]byte{id("4140"): 6}) {
			return fmt.Errorf("configuration.validation")
		}
		fv, ok := q.Fields[id("4140")]
		if !ok {
			return fmt.Errorf("configuration.whole_validation_unsupported")
		}
		f, ok := fields[fv.Reference]
		if !ok {
			return fmt.Errorf("configuration.validation_field")
		}
		fn := q.Fields[id("4141")].Reference
		if !validatorSignature(base, fn, f.Fields[id("4112")].Reference) || !pure(base, fn) {
			return fmt.Errorf("configuration.validator_signature")
		}
		order := q.Fields[id("4142")].Unsigned
		if validationOrders[order] {
			return fmt.Errorf("configuration.validation_order")
		}
		validationOrders[order] = true
	}
	if !contiguous(validationOrders) {
		return fmt.Errorf("configuration.validation_order")
	}
	initializers := map[wire.ID]wire.Entity{}
	callableInit := map[wire.ID]wire.ID{}
	initOrder := map[wire.ID]uint64{}
	initializerOrders := map[uint64]bool{}
	for _, r := range g.Fields[id("4103")].List {
		q := e.Entities[r.Reference]
		if r.Tag != 6 || q.Schema != sInitializer || q.Version != 1 || !shape(q, map[wire.ID]byte{id("4150"): 6, id("4151"): 6, id("4152"): 7, id("4153"): 3}) || !sortedUnique(q.Fields[id("4152")].List) {
			return fmt.Errorf("configuration.initializer")
		}
		owner, call := q.Fields[id("4150")].Reference, q.Fields[id("4151")].Reference
		member, owned := members[call]
		if !owners[owner] || !owned || member.owner != owner || member.visibility == 0 || callableInit[call] != (wire.ID{}) || !pure(base, call) || !initializerSignature(base, call, allowParameterized) {
			return fmt.Errorf("configuration.initializer_owner")
		}
		order := q.Fields[id("4153")].Unsigned
		if initializerOrders[order] {
			return fmt.Errorf("configuration.initializer_order")
		}
		initializerOrders[order] = true
		initializers[r.Reference] = q
		callableInit[call] = r.Reference
		initOrder[r.Reference] = order
	}
	if !contiguous(initializerOrders) {
		return fmt.Errorf("configuration.initializer_order")
	}
	for iid, q := range initializers {
		seen := map[wire.ID]bool{}
		for _, d := range q.Fields[id("4152")].List {
			_, exists := initializers[d.Reference]
			if d.Tag != 6 || !exists || seen[d.Reference] || initOrder[d.Reference] >= initOrder[iid] {
				return fmt.Errorf("configuration.initializer_dag")
			}
			seen[d.Reference] = true
		}
	}
	transitionOrder := map[uint64]bool{}
	transitionKinds := map[wire.ID]map[[2]uint64]bool{}
	for _, r := range g.Fields[id("4104")].List {
		q := e.Entities[r.Reference]
		if r.Tag != 6 || q.Schema != sTransition || q.Version != 1 || !shape(q, map[wire.ID]byte{id("4170"): 6, id("4171"): 6, id("4172"): 6, id("4173"): 3}) {
			return fmt.Errorf("configuration.transition")
		}
		init := q.Fields[id("4170")].Reference
		if _, exists := initializers[init]; !exists {
			return fmt.Errorf("configuration.transition_initializer")
		}
		fromEntity, toEntity := e.Entities[q.Fields[id("4171")].Reference], e.Entities[q.Fields[id("4172")].Reference]
		if fromEntity.Schema != sState || toEntity.Schema != sState || !shape(fromEntity, map[wire.ID]byte{id("4160"): 3}) || !shape(toEntity, map[wire.ID]byte{id("4160"): 3}) {
			return fmt.Errorf("configuration.transition_state")
		}
		from, to := fromEntity.Fields[id("4160")].Unsigned, toEntity.Fields[id("4160")].Unsigned
		if !legal(from, to) {
			return fmt.Errorf("configuration.transition_legal")
		}
		order := q.Fields[id("4173")].Unsigned
		if transitionOrder[order] {
			return fmt.Errorf("configuration.transition_order")
		}
		transitionOrder[order] = true
		if transitionKinds[init] == nil {
			transitionKinds[init] = map[[2]uint64]bool{}
		}
		pair := [2]uint64{from, to}
		if transitionKinds[init][pair] {
			return fmt.Errorf("configuration.transition_duplicate")
		}
		transitionKinds[init][pair] = true
	}
	if !contiguous(transitionOrder) {
		return fmt.Errorf("configuration.transition_order")
	}
	for init := range initializers {
		if !transitionKinds[init][[2]uint64{1, 2}] || !transitionKinds[init][[2]uint64{2, 3}] {
			return fmt.Errorf("configuration.transition_coverage")
		}
	}
	return nil
}

type memberOwnership struct {
	owner      wire.ID
	visibility uint64
}

func ownership(e wire.Envelope) (map[wire.ID]bool, map[wire.ID]wire.ID, map[wire.ID]memberOwnership) {
	packages := map[wire.ID]bool{}
	origins := map[wire.ID]wire.ID{}
	members := map[wire.ID]memberOwnership{}
	for x, q := range e.Entities {
		if q.Schema == sPackage {
			packages[x] = true
		}
	}
	for _, q := range e.Entities {
		if q.Schema != sDetail {
			continue
		}
		owner := q.Fields[id("b210")].Reference
		for _, r := range q.Fields[id("b213")].List {
			origins[r.Reference] = owner
		}
		for _, r := range q.Fields[id("b211")].List {
			m := e.Entities[r.Reference]
			visibility := e.Entities[m.Fields[id("b222")].Reference].Fields[id("b230")].Unsigned
			members[m.Fields[id("b220")].Reference] = memberOwnership{owner: owner, visibility: visibility}
			origins[m.Fields[id("b224")].Reference] = owner
		}
	}
	for _, q := range e.Entities {
		if q.Schema == id("b028") {
			origins[q.Fields[id("b286")].Reference] = e.Entities[q.Fields[id("b281")].Reference].Fields[id("b210")].Reference
		}
	}
	return packages, origins, members
}
func authorizedCapabilities(e wire.Envelope) map[wire.ID]map[wire.ID]bool {
	out := map[wire.ID]map[wire.ID]bool{}
	for pid, p := range e.Entities {
		if p.Schema != sPackage {
			continue
		}
		out[pid] = map[wire.ID]bool{}
		for _, r := range p.Fields[id("b104")].List {
			effect := e.Entities[r.Reference]
			if cap, ok := effect.Fields[id("151")]; ok && cap.Tag == 6 {
				out[pid][cap.Reference] = true
			}
		}
	}
	return out
}
func validatorSignature(e wire.Envelope, fn, typ wire.ID) bool {
	q := e.Entities[fn]
	if q.Schema != sFunction {
		return false
	}
	ps := q.Fields[id("9111")].List
	if len(ps) != 1 || e.Entities[ps[0].Reference].Fields[id("9121")].Reference != typ {
		return false
	}
	r := e.Entities[q.Fields[id("9112")].Reference]
	return r.Schema == sResultType && r.Fields[id("9400")].Reference == typ
}
func canonicalType(e wire.Envelope, x wire.ID) bool {
	switch e.Entities[x].Schema {
	case id("9010"), id("9020"), id("9030"), id("9040"), id("9041"), id("9042"), id("90f2"), id("90f8"), id("a004"), id("a010"), id("a020"), id("a040"), id("a050"):
		return true
	}
	return false
}
func initializerSignature(e wire.Envelope, fn wire.ID, allowParameterized bool) bool {
	q := e.Entities[fn]
	if q.Schema != sFunction || (!allowParameterized && len(q.Fields[id("9111")].List) != 0) {
		return false
	}
	r, ok := q.Fields[id("9112")]
	return ok && r.Tag == 6 && e.Entities[r.Reference].Schema == sResultType
}
func pure(e wire.Envelope, root wire.ID) bool {
	seen := map[wire.ID]bool{}
	todo := []wire.ID{root}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if seen[x] {
			continue
		}
		seen[x] = true
		q, ok := e.Entities[x]
		if !ok || q.Schema == sEffectInvoke {
			return false
		}
		for _, v := range q.Fields {
			collect(v, &todo)
		}
	}
	return true
}
func legal(a, b uint64) bool {
	return a == 0 && b == 1 || a == 1 && b == 2 || a == 2 && (b == 3 || b == 4)
}
func contiguous(in map[uint64]bool) bool {
	for i := uint64(0); i < uint64(len(in)); i++ {
		if !in[i] {
			return false
		}
	}
	return true
}
func sortedUnique(in []wire.Value) bool {
	var prior wire.ID
	for i, v := range in {
		if v.Tag != 6 || i > 0 && bytes.Compare(prior[:], v.Reference[:]) >= 0 {
			return false
		}
		prior = v.Reference
	}
	return true
}
func GraphRevision(e wire.Envelope, root wire.ID) []byte {
	q := e.Entities[root]
	fields := make(map[wire.ID]wire.Value, len(q.Fields))
	for key, value := range q.Fields {
		fields[key] = value
	}
	q.Fields = fields
	q.Fields[id("4105")] = blob(make([]byte, 32))
	entities := make(map[wire.ID]wire.Entity, len(e.Entities))
	for entityID, entity := range e.Entities {
		entities[entityID] = entity
	}
	entities[root] = q
	b, _ := wire.Encode(wire.Envelope{Entities: closure(entities, root)})
	h := sha256.Sum256(append([]byte("seme.configuration.graph.v1\x00"), b...))
	return h[:]
}
func ArtifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.configuration.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func stable(seed []byte, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.configuration-instance.identity.v1\x00"))
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
func exactPins(e wire.Envelope, m wire.Entity, want map[wire.ID]wire.ID) bool {
	v := m.Fields[id("121")]
	if v.Tag != 7 || len(v.List) != len(want) || !sortedUnique(v.List) {
		return false
	}
	got := map[wire.ID]wire.ID{}
	for _, r := range v.List {
		q := e.Entities[r.Reference]
		if q.Schema != sImport || q.Version != 1 || !shape(q, map[wire.ID]byte{id("130"): 6, id("131"): 5}) || len(q.Fields[id("131")].Bytes) != 16 {
			return false
		}
		var rev wire.ID
		copy(rev[:], q.Fields[id("131")].Bytes)
		got[q.Fields[id("130")].Reference] = rev
	}
	for x, r := range want {
		if got[x] != r {
			return false
		}
	}
	return len(got) == len(want)
}
func one(e wire.Envelope, s wire.ID) (wire.ID, error) {
	var x wire.ID
	for id, q := range e.Entities {
		if q.Schema == s {
			if x != (wire.ID{}) {
				return x, fmt.Errorf("configuration.schema_count")
			}
			x = id
		}
	}
	if x == (wire.ID{}) {
		return x, fmt.Errorf("configuration.schema_count")
	}
	return x, nil
}
func shape(q wire.Entity, req map[wire.ID]byte) bool { return shapeOptional(q, req, nil) }
func shapeOptional(q wire.Entity, req, opt map[wire.ID]byte) bool {
	if len(q.Fields) < len(req) || len(q.Fields) > len(req)+len(opt) {
		return false
	}
	for f, t := range req {
		if t == 255 {
			if q.Fields[f].Tag != 1 && q.Fields[f].Tag != 2 {
				return false
			}
		} else if q.Fields[f].Tag != t {
			return false
		}
	}
	for f, v := range q.Fields {
		if _, ok := req[f]; ok {
			continue
		}
		if t, ok := opt[f]; !ok || v.Tag != t {
			return false
		}
	}
	return true
}
func closure(all map[wire.ID]wire.Entity, root wire.ID) map[wire.ID]wire.Entity {
	out := map[wire.ID]wire.Entity{}
	todo := []wire.ID{root}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if _, ok := out[x]; ok {
			continue
		}
		q, ok := all[x]
		if !ok {
			continue
		}
		out[x] = q
		for _, v := range q.Fields {
			collect(v, &todo)
		}
	}
	return out
}
func collect(v wire.Value, out *[]wire.ID) {
	if v.Tag == 6 {
		*out = append(*out, v.Reference)
	}
	for _, x := range v.List {
		collect(x, out)
	}
	for _, x := range v.Record {
		collect(x, out)
	}
}
func clone(q wire.Entity) wire.Entity {
	f := map[wire.ID]wire.Value{}
	for x, v := range q.Fields {
		f[x] = v
	}
	q.Fields = f
	return q
}
func same(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func put(m map[wire.ID]wire.Entity, q wire.Entity) error {
	if _, ok := m[q.ID]; ok {
		return fmt.Errorf("configuration.identity_collision")
	}
	m[q.ID] = q
	return nil
}
func ref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
func blob(x []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func sortRefs(x []wire.Value) {
	sort.Slice(x, func(i, j int) bool { return bytes.Compare(x[i].Reference[:], x[j].Reference[:]) < 0 })
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
