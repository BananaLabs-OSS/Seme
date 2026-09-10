// Package goconfigurationadapter resolves declarative startup selections
// against one authenticated Go provider run. It emits no canonical artifact;
// its project-neutral Plan is input to the Configuration v2 instance layer.
package goconfigurationadapter

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sort"
	"unicode/utf8"

	"seme.local/reference/canonicaleval"
	"seme.local/reference/configurationinstance"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprovider"
	"seme.local/reference/packagev3instance"
	"seme.local/reference/wire"
)

type Compile func(context.Context, []byte) ([]byte, error)

type Input struct {
	Session     goprovider.SessionResult
	CanonicalG1 []byte
	Packages    []goprovider.PackageMetadata
	Compile     Compile
	Contracts   contractcatalog.ProjectContractSetV5
	PackageV2   []byte
	PackageV3   []byte
	Fields      []FieldSelection
	Runtime     []RuntimeInputSelection
	Units       []UnitSelection
}
type Selection struct {
	Fields  []FieldSelection
	Runtime []RuntimeInputSelection
	Units   []UnitSelection
}

type TypeSelection struct{ Package, Name, ID string }
type FunctionSelection struct{ Package, Name string }
type FieldSelection struct {
	Key, OwnerPackage string
	Type              TypeSelection
	Origin            TypeSelection
	Default           *wire.ID
	Required          bool
	Resolution        ResolutionSelection
	Validator         *FunctionSelection
	ValidationOrder   uint64
}
type ResolutionKind uint8

const (
	ExplicitValue ResolutionKind = iota
	DefaultValue
	CapabilityValue
)

type ResolutionSelection struct {
	Kind              ResolutionKind
	Value, Capability *wire.ID
}
type RuntimeInputSelection struct {
	Identity   string
	Type       TypeSelection
	Capability *wire.ID
}
type SourceKind uint8

const (
	ResolvedField SourceKind = iota
	RuntimeInputSource
	PredecessorOK
	StaticCanonical
	RecordConstruction
)

type SourceSelection struct {
	Kind                        SourceKind
	Field, Runtime, Predecessor string
	Static                      *wire.ID
	Record                      *RecordSelection
}
type RecordSelection struct {
	Type    TypeSelection
	Members []MemberSelection
}
type MemberSelection struct {
	Name   string
	Source SourceSelection
}
type UnitSelection struct {
	Key          string
	Callable     FunctionSelection
	Dependencies []string
	Arguments    []SourceSelection
}

type Plan struct {
	Fields        []Field
	RuntimeInputs []RuntimeInput
	Units         []Unit
}
type Field struct {
	Key                 string
	Owner, Type, Origin wire.ID
	Default             *wire.ID
	Required            bool
	Resolution          Resolution
	Validator           *wire.ID
	ValidationOrder     uint64
}
type Resolution struct {
	Kind              ResolutionKind
	Value, Capability *wire.ID
}
type RuntimeInput struct {
	Identity   string
	Type       wire.ID
	Capability *wire.ID
}
type Unit struct {
	Key             string
	Owner, Callable wire.ID
	Dependencies    []string
	Arguments       []Argument
}
type Argument struct {
	Parameter wire.ID
	Index     uint64
	Source    Source
}
type Source struct {
	Kind                             SourceKind
	DerivedType                      wire.ID
	Field, RuntimeInput, Predecessor string
	Static                           *wire.ID
	Record                           *Record
}
type Record struct {
	Type    wire.ID
	Members []Member
}
type Member struct {
	Field  wire.ID
	Source Source
}

type evidence struct {
	graph     wire.Envelope
	packages  wire.Envelope
	functions map[string]function
	types     map[string]ownedType
}
type function struct {
	id, owner  wire.ID
	parameters []wire.ID
	result     wire.ID
}
type ownedType struct{ id, owner, origin wire.ID }

func Resolve(ctx context.Context, in Input) (Plan, error) {
	ev, err := authenticate(ctx, in)
	if err != nil {
		return Plan{}, err
	}
	p, err := resolve(in, ev)
	if err == nil {
		Deterministic(&p)
	}
	return p, err
}

func authenticate(ctx context.Context, in Input) (evidence, error) {
	construction, packages := []byte(in.Session.CanonicalG1), in.Session.Packages
	if len(in.CanonicalG1) != 0 {
		if len(construction) != 0 || len(in.Packages) == 0 {
			return evidence{}, fmt.Errorf("go_configuration.run_ambiguous")
		}
		construction, packages = in.CanonicalG1, in.Packages
	} else if !in.Session.Accepted || !in.Session.Valid || in.Session.Disposition != "accepted-valid" || in.Session.Revision == 0 || in.Session.LastValidRevision != in.Session.Revision {
		return evidence{}, fmt.Errorf("go_configuration.run")
	}
	if len(construction) == 0 || in.Compile == nil {
		return evidence{}, fmt.Errorf("go_configuration.run")
	}
	compiled, err := in.Compile(ctx, construction)
	if err != nil {
		return evidence{}, fmt.Errorf("go_configuration.compile:%w", err)
	}
	g, err := wire.Decode(compiled)
	if err != nil {
		return evidence{}, fmt.Errorf("go_configuration.execution:%w", err)
	}
	canonical, err := wire.Encode(g)
	if err != nil || !bytes.Equal(canonical, compiled) {
		return evidence{}, fmt.Errorf("go_configuration.execution_noncanonical")
	}
	if err = packagev3instance.Validate(in.Contracts, in.PackageV2, in.PackageV3); err != nil {
		return evidence{}, fmt.Errorf("go_configuration.package_v3:%w", err)
	}
	p, err := wire.Decode(in.PackageV3)
	if err != nil {
		return evidence{}, err
	}
	ev := evidence{graph: g, packages: p, functions: map[string]function{}, types: map[string]ownedType{}}
	detailOwner := map[wire.ID]wire.ID{}
	for _, q := range p.Entities {
		if q.Schema != id("b021") {
			continue
		}
		for _, m := range q.Fields[id("b211")].List {
			member := p.Entities[m.Reference]
			detailOwner[member.Fields[id("b220")].Reference] = q.ID
		}
	}
	for _, pm := range packages {
		for _, fm := range pm.Functions {
			fid, er := wire.ParseID(fm.ID)
			if er != nil {
				return evidence{}, fmt.Errorf("go_configuration.function_id")
			}
			q, ok := g.Entities[fid]
			if !ok || q.Schema != id("9011") {
				return evidence{}, fmt.Errorf("go_configuration.function_missing")
			}
			if !reflect.DeepEqual(q, p.Entities[fid]) {
				return evidence{}, fmt.Errorf("go_configuration.function_run_mismatch")
			}
			params := refs(q, id("9111"))
			if q.Fields[id("9111")].Tag != 7 || len(params) != len(q.Fields[id("9111")].List) {
				return evidence{}, fmt.Errorf("go_configuration.function_parameters")
			}
			result, ok := reference(q, id("9112"))
			if !ok {
				return evidence{}, fmt.Errorf("go_configuration.function_shape")
			}
			owner := detailOwner[fid]
			if owner == (wire.ID{}) {
				return evidence{}, fmt.Errorf("go_configuration.function_owner")
			}
			key := selectKey(pm.Name, fm.Name)
			if _, exists := ev.functions[key]; exists {
				return evidence{}, fmt.Errorf("go_configuration.function_duplicate")
			}
			ev.functions[key] = function{fid, owner, params, result}
		}
	}
	for _, q := range p.Entities {
		if q.Schema != id("b028") {
			continue
		}
		decl, dok := reference(q, id("b280"))
		owner, ook := reference(q, id("b281"))
		origin, rok := reference(q, id("b286"))
		if !dok || !ook || !rok {
			return evidence{}, fmt.Errorf("go_configuration.ownership_shape")
		}
		d := g.Entities[decl]
		if d.Schema != id("9030") {
			continue
		}
		if !reflect.DeepEqual(d, p.Entities[decl]) {
			return evidence{}, fmt.Errorf("go_configuration.type_run_mismatch")
		}
		name := string(d.Fields[id("9300")].Bytes)
		pkg := packageName(p, owner)
		if pkg == "" {
			return evidence{}, fmt.Errorf("go_configuration.type_owner")
		}
		key := selectKey(pkg, name)
		if _, exists := ev.types[key]; exists {
			return evidence{}, fmt.Errorf("go_configuration.type_duplicate")
		}
		ev.types[key] = ownedType{decl, owner, origin}
	}
	return ev, nil
}

func resolve(in Input, ev evidence) (Plan, error) {
	out := Plan{}
	fieldTypes := map[string]wire.ID{}
	for _, s := range in.Fields {
		if s.Key == "" || !utf8.ValidString(s.Key) || fieldTypes[s.Key] != (wire.ID{}) {
			return Plan{}, fmt.Errorf("go_configuration.field")
		}
		t, ok := resolveType(ev, s.Type)
		if !ok {
			return Plan{}, fmt.Errorf("go_configuration.field_type")
		}
		o, ok := ev.types[selectKey(s.Origin.Package, s.Origin.Name)]
		if !ok {
			return Plan{}, fmt.Errorf("go_configuration.field_origin")
		}
		owner := packageOwner(ev.packages, s.OwnerPackage)
		if owner == (wire.ID{}) || owner != o.owner {
			return Plan{}, fmt.Errorf("go_configuration.field_owner")
		}
		if s.Default != nil {
			if typ, ok := canonicaleval.ExpressionType(ev.graph, *s.Default); !ok || typ != t.id {
				return Plan{}, fmt.Errorf("go_configuration.default_type")
			}
		}
		resolution := Resolution{Kind: s.Resolution.Kind, Value: cloneID(s.Resolution.Value), Capability: cloneID(s.Resolution.Capability)}
		switch resolution.Kind {
		case ExplicitValue:
			if resolution.Value == nil || resolution.Capability != nil {
				return Plan{}, fmt.Errorf("go_configuration.resolution_explicit")
			}
		case DefaultValue:
			if s.Default == nil || resolution.Value == nil || *resolution.Value != *s.Default || resolution.Capability != nil {
				return Plan{}, fmt.Errorf("go_configuration.resolution_default")
			}
		case CapabilityValue:
			if resolution.Value != nil || resolution.Capability == nil || ev.graph.Entities[*resolution.Capability].Schema != id("16") {
				return Plan{}, fmt.Errorf("go_configuration.resolution_capability")
			}
		default:
			return Plan{}, fmt.Errorf("go_configuration.resolution_kind")
		}
		if resolution.Value != nil {
			if typ, ok := canonicaleval.ExpressionType(ev.graph, *resolution.Value); !ok || typ != t.id {
				return Plan{}, fmt.Errorf("go_configuration.resolution_type")
			}
		}
		var validator *wire.ID
		if s.Validator != nil {
			f, ok := ev.functions[selectKey(s.Validator.Package, s.Validator.Name)]
			if !ok || !validatorSignature(ev.graph, f, t.id) || !pure(ev.graph, f.id) {
				return Plan{}, fmt.Errorf("go_configuration.validator")
			}
			validator = cloneID(&f.id)
		}
		out.Fields = append(out.Fields, Field{s.Key, owner, t.id, o.origin, cloneID(s.Default), s.Required, resolution, validator, s.ValidationOrder})
		fieldTypes[s.Key] = t.id
	}
	runtimeTypes := map[string]wire.ID{}
	for _, s := range in.Runtime {
		if s.Identity == "" || !utf8.ValidString(s.Identity) || runtimeTypes[s.Identity] != (wire.ID{}) {
			return Plan{}, fmt.Errorf("go_configuration.runtime")
		}
		t, ok := resolveType(ev, s.Type)
		if !ok {
			return Plan{}, fmt.Errorf("go_configuration.runtime_type")
		}
		if s.Capability != nil {
			q := ev.graph.Entities[*s.Capability]
			if q.Schema != id("16") {
				return Plan{}, fmt.Errorf("go_configuration.runtime_capability")
			}
		}
		out.RuntimeInputs = append(out.RuntimeInputs, RuntimeInput{s.Identity, t.id, cloneID(s.Capability)})
		runtimeTypes[s.Identity] = t.id
	}
	units := map[string]function{}
	depSets := map[string]map[string]bool{}
	for _, s := range in.Units {
		if s.Key == "" || units[s.Key].id != (wire.ID{}) {
			return Plan{}, fmt.Errorf("go_configuration.unit")
		}
		f, ok := ev.functions[selectKey(s.Callable.Package, s.Callable.Name)]
		if !ok {
			return Plan{}, fmt.Errorf("go_configuration.callable")
		}
		units[s.Key] = f
		ds := map[string]bool{}
		for _, d := range s.Dependencies {
			if ds[d] {
				return Plan{}, fmt.Errorf("go_configuration.dependency_duplicate")
			}
			ds[d] = true
		}
		depSets[s.Key] = ds
	}
	for _, s := range in.Units {
		f := units[s.Key]
		if len(s.Arguments) != len(f.parameters) {
			return Plan{}, fmt.Errorf("go_configuration.argument_count")
		}
		u := Unit{Key: s.Key, Owner: f.owner, Callable: f.id, Dependencies: append([]string(nil), s.Dependencies...)}
		for i, ss := range s.Arguments {
			p := ev.graph.Entities[f.parameters[i]]
			if p.Schema != id("9012") || p.Fields[id("9122")].Unsigned != uint64(i) {
				return Plan{}, fmt.Errorf("go_configuration.parameter")
			}
			want, _ := reference(p, id("9121"))
			src, er := resolveSource(ss, want, s.Key, ev, fieldTypes, runtimeTypes, units, depSets, map[*RecordSelection]bool{}, 256)
			if er != nil {
				return Plan{}, er
			}
			u.Arguments = append(u.Arguments, Argument{f.parameters[i], uint64(i), src})
		}
		out.Units = append(out.Units, u)
	}
	if cycle(out.Units) {
		return Plan{}, fmt.Errorf("go_configuration.unit_cycle")
	}
	return out, nil
}

func resolveSource(s SourceSelection, want wire.ID, current string, ev evidence, fields, runtime map[string]wire.ID, units map[string]function, deps map[string]map[string]bool, visiting map[*RecordSelection]bool, budget int) (Source, error) {
	if budget <= 0 {
		return Source{}, fmt.Errorf("go_configuration.source_depth")
	}
	set := 0
	if s.Field != "" {
		set++
	}
	if s.Runtime != "" {
		set++
	}
	if s.Predecessor != "" {
		set++
	}
	if s.Static != nil {
		set++
	}
	if s.Record != nil {
		set++
	}
	if set != 1 {
		return Source{}, fmt.Errorf("go_configuration.source_union")
	}
	out := Source{Kind: s.Kind, DerivedType: want}
	switch s.Kind {
	case ResolvedField:
		if fields[s.Field] != want {
			return Source{}, fmt.Errorf("go_configuration.source_field")
		}
		out.Field = s.Field
	case RuntimeInputSource:
		if runtime[s.Runtime] != want {
			return Source{}, fmt.Errorf("go_configuration.source_runtime")
		}
		out.RuntimeInput = s.Runtime
	case PredecessorOK:
		if !deps[current][s.Predecessor] {
			return Source{}, fmt.Errorf("go_configuration.source_predecessor")
		}
		f, ok := units[s.Predecessor]
		if !ok {
			return Source{}, fmt.Errorf("go_configuration.source_predecessor")
		}
		r := ev.graph.Entities[f.result]
		if r.Schema != id("9042") {
			return Source{}, fmt.Errorf("go_configuration.source_result")
		}
		okType, _ := reference(r, id("9400"))
		if okType != want {
			return Source{}, fmt.Errorf("go_configuration.source_result_type")
		}
		out.Predecessor = s.Predecessor
	case StaticCanonical:
		if s.Static == nil {
			return Source{}, fmt.Errorf("go_configuration.source_static")
		}
		typ, ok := canonicaleval.ExpressionType(ev.graph, *s.Static)
		if !ok || typ != want {
			return Source{}, fmt.Errorf("go_configuration.source_static_type")
		}
		out.Static = cloneID(s.Static)
	case RecordConstruction:
		if s.Record == nil {
			return Source{}, fmt.Errorf("go_configuration.source_record")
		}
		if visiting[s.Record] {
			return Source{}, fmt.Errorf("go_configuration.source_cycle")
		}
		visiting[s.Record] = true
		defer delete(visiting, s.Record)
		t, ok := resolveType(ev, s.Record.Type)
		if !ok || t.id != want {
			return Source{}, fmt.Errorf("go_configuration.source_record_type")
		}
		q := ev.graph.Entities[t.id]
		fieldIDs := refs(q, id("9301"))
		if len(fieldIDs) != len(s.Record.Members) {
			return Source{}, fmt.Errorf("go_configuration.source_record_complete")
		}
		rec := &Record{Type: t.id}
		for i, fid := range fieldIDs {
			fq := ev.graph.Entities[fid]
			name := string(fq.Fields[id("9310")].Bytes)
			if s.Record.Members[i].Name != name || fq.Fields[id("9312")].Unsigned != uint64(i) {
				return Source{}, fmt.Errorf("go_configuration.source_record_order")
			}
			ft, _ := reference(fq, id("9311"))
			child, err := resolveSource(s.Record.Members[i].Source, ft, current, ev, fields, runtime, units, deps, visiting, budget-1)
			if err != nil {
				return Source{}, err
			}
			rec.Members = append(rec.Members, Member{fid, child})
		}
		out.Record = rec
	default:
		return Source{}, fmt.Errorf("go_configuration.source_kind")
	}
	return out, nil
}

func cycle(units []Unit) bool {
	m := map[string][]string{}
	for _, u := range units {
		m[u.Key] = u.Dependencies
	}
	active, done := map[string]bool{}, map[string]bool{}
	var visit func(string) bool
	visit = func(x string) bool {
		if active[x] {
			return true
		}
		if done[x] {
			return false
		}
		active[x] = true
		for _, d := range m[x] {
			if _, ok := m[d]; !ok || visit(d) {
				return true
			}
		}
		delete(active, x)
		done[x] = true
		return false
	}
	for x := range m {
		if visit(x) {
			return true
		}
	}
	return false
}
func selectKey(p, n string) string { return p + "\x00" + n }
func resolveType(ev evidence, s TypeSelection) (ownedType, bool) {
	if s.ID != "" {
		x, err := wire.ParseID(s.ID)
		if err != nil || !canonicalType(ev.graph, x) {
			return ownedType{}, false
		}
		return ownedType{id: x}, true
	}
	x, ok := ev.types[selectKey(s.Package, s.Name)]
	return x, ok
}

func canonicalType(e wire.Envelope, x wire.ID) bool {
	switch e.Entities[x].Schema {
	case id("9010"), id("9020"), id("9030"), id("9040"), id("9041"), id("9042"), id("90f2"), id("90f8"), id("a004"), id("a010"), id("a020"), id("a040"), id("a050"):
		return true
	}
	return false
}
func validatorSignature(e wire.Envelope, f function, typ wire.ID) bool {
	if len(f.parameters) != 1 || e.Entities[f.parameters[0]].Fields[id("9121")].Reference != typ {
		return false
	}
	r := e.Entities[f.result]
	return r.Schema == id("9042") && r.Fields[id("9400")].Tag == 6 && r.Fields[id("9400")].Reference == typ
}
func pure(e wire.Envelope, root wire.ID) bool {
	seen, todo := map[wire.ID]bool{}, []wire.ID{root}
	for len(todo) > 0 {
		if len(seen) > 100000 {
			return false
		}
		x := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if seen[x] {
			continue
		}
		seen[x] = true
		q, ok := e.Entities[x]
		if !ok || q.Schema == id("90f1") {
			return false
		}
		for _, v := range q.Fields {
			todo = appendValueRefs(todo, v)
		}
	}
	return true
}
func appendValueRefs(out []wire.ID, v wire.Value) []wire.ID {
	if v.Tag == 6 {
		return append(out, v.Reference)
	}
	for _, x := range v.List {
		out = appendValueRefs(out, x)
	}
	for _, x := range v.Record {
		out = appendValueRefs(out, x)
	}
	return out
}
func cloneID(x *wire.ID) *wire.ID {
	if x == nil {
		return nil
	}
	v := *x
	return &v
}
func id(x string) wire.ID {
	for len(x) < 32 {
		x = "0" + x
	}
	v, _ := wire.ParseID(x)
	return v
}
func reference(e wire.Entity, f wire.ID) (wire.ID, bool) {
	v, ok := e.Fields[f]
	return v.Reference, ok && v.Tag == 6
}
func refs(e wire.Entity, f wire.ID) []wire.ID {
	v := e.Fields[f]
	out := []wire.ID{}
	if v.Tag != 7 {
		return out
	}
	for _, x := range v.List {
		if x.Tag == 6 {
			out = append(out, x.Reference)
		}
	}
	return out
}
func packageOwner(e wire.Envelope, name string) wire.ID {
	for _, q := range e.Entities {
		if q.Schema != id("b021") {
			continue
		}
		p := e.Entities[q.Fields[id("b210")].Reference]
		if string(p.Fields[id("b100")].Bytes) == name {
			return q.ID
		}
	}
	return wire.ID{}
}
func packageName(e wire.Envelope, detail wire.ID) string {
	q := e.Entities[detail]
	p := e.Entities[q.Fields[id("b210")].Reference]
	return string(p.Fields[id("b100")].Bytes)
}

// Deterministic sorts only selection-independent lookup output; caller order is
// semantically significant for record members and initializer arguments.
func Deterministic(p *Plan) {
	sort.Slice(p.Fields, func(i, j int) bool { return p.Fields[i].Key < p.Fields[j].Key })
	sort.Slice(p.RuntimeInputs, func(i, j int) bool { return p.RuntimeInputs[i].Identity < p.RuntimeInputs[j].Identity })
}

// BoundModel converts a resolved plan without reinterpreting any identity.
// The Configuration v2 emitter remains the authority for canonical encoding.
func BoundModel(p Plan) (configurationinstance.BoundModel, error) {
	_, bound, err := Models(p)
	return bound, err
}

// Models produces the paired v1 declaration/lifecycle graph and v2 binding
// graph required by configurationinstance.EmitBound.
func Models(p Plan) (configurationinstance.Model, configurationinstance.BoundModel, error) {
	callables := map[string]wire.ID{}
	for _, u := range p.Units {
		if u.Key == "" || callables[u.Key] != (wire.ID{}) {
			return configurationinstance.Model{}, configurationinstance.BoundModel{}, fmt.Errorf("go_configuration.plan_unit")
		}
		callables[u.Key] = u.Callable
	}
	base := configurationinstance.Model{}
	out := configurationinstance.BoundModel{}
	for _, f := range p.Fields {
		base.Fields = append(base.Fields, configurationinstance.Field{Owner: f.Owner, Type: f.Type, Origin: f.Origin, Key: f.Key, Default: cloneID(f.Default), Required: f.Required})
		v := configurationinstance.ResolvedValue{Key: f.Key, Origin: configurationinstance.ResolutionOrigin(f.Resolution.Kind), Value: cloneID(f.Resolution.Value), Capability: cloneID(f.Resolution.Capability)}
		base.Values = append(base.Values, v)
		if f.Validator != nil {
			base.Validations = append(base.Validations, configurationinstance.Validation{Key: f.Key, Validator: *f.Validator, Order: f.ValidationOrder})
		}
	}
	for _, r := range p.RuntimeInputs {
		out.RuntimeInputs = append(out.RuntimeInputs, configurationinstance.RuntimeInput{Identity: r.Identity, Type: r.Type, Capability: cloneID(r.Capability)})
	}
	var convert func(Source) (configurationinstance.ArgumentSource, error)
	convert = func(s Source) (configurationinstance.ArgumentSource, error) {
		x := configurationinstance.ArgumentSource{Kind: configurationinstance.ArgumentSourceKind(s.Kind), DerivedType: s.DerivedType, FieldKey: s.Field, RuntimeIdentity: s.RuntimeInput}
		if s.Predecessor != "" {
			var ok bool
			x.Predecessor, ok = callables[s.Predecessor]
			if !ok {
				return x, fmt.Errorf("go_configuration.plan_predecessor")
			}
		}
		if s.Static != nil {
			x.StaticValue = *s.Static
		}
		if s.Record != nil {
			r := &configurationinstance.RecordAssembly{RecordType: s.Record.Type}
			for _, m := range s.Record.Members {
				child, err := convert(m.Source)
				if err != nil {
					return x, err
				}
				r.Members = append(r.Members, configurationinstance.RecordMemberBinding{Field: m.Field, Source: child})
			}
			x.Record = r
		}
		return x, nil
	}
	for _, u := range p.Units {
		deps := []wire.ID{}
		for _, key := range u.Dependencies {
			x, ok := callables[key]
			if !ok {
				return configurationinstance.Model{}, configurationinstance.BoundModel{}, fmt.Errorf("go_configuration.plan_dependency")
			}
			deps = append(deps, x)
		}
		base.Initializers = append(base.Initializers, configurationinstance.Initializer{Owner: u.Owner, Callable: u.Callable, Dependencies: deps, Order: uint64(len(base.Initializers))})
		base.Transitions = append(base.Transitions, configurationinstance.Transition{Initializer: u.Callable, From: 1, To: 2, Order: uint64(len(base.Transitions))}, configurationinstance.Transition{Initializer: u.Callable, From: 2, To: 3, Order: uint64(len(base.Transitions) + 1)})
		b := configurationinstance.BoundInitializer{Callable: u.Callable}
		for _, a := range u.Arguments {
			s, err := convert(a.Source)
			if err != nil {
				return configurationinstance.Model{}, configurationinstance.BoundModel{}, err
			}
			b.Arguments = append(b.Arguments, configurationinstance.InitializerArgument{Parameter: a.Parameter, Index: a.Index, Source: s})
		}
		out.Initializers = append(out.Initializers, b)
	}
	return base, out, nil
}
