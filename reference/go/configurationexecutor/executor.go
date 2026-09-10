// Package configurationexecutor executes an authenticated Configuration v2
// bound plan without mutating its immutable project or configuration artifacts.
package configurationexecutor

import (
	"fmt"
	"sort"

	"seme.local/reference/canonicaleval"
	"seme.local/reference/configurationinstance"
	"seme.local/reference/wire"
)

type Input struct {
	Bound               configurationinstance.BoundInput
	ConfigurationInputs map[string]canonicaleval.Value
	RuntimeInputs       map[string]canonicaleval.Value
	CapabilityGrants    map[string]bool
}

type Result struct {
	Lifecycle []LifecycleObservation `json:"lifecycle"`
	Outputs   []Output               `json:"outputs,omitempty"`
}

type LifecycleObservation struct {
	Sequence    uint64 `json:"sequence"`
	Initializer string `json:"initializer"`
	From        string `json:"from"`
	To          string `json:"to"`
}

type Output struct {
	Order       uint64              `json:"order"`
	Initializer string              `json:"initializer"`
	Callable    string              `json:"callable"`
	Result      canonicaleval.Value `json:"result"`
}

type plan struct {
	graph        wire.Envelope
	fields       map[wire.ID]canonicaleval.Value
	runtime      map[wire.ID]canonicaleval.Value
	initializers []unit
	static       map[wire.ID]canonicaleval.Value
}
type unit struct {
	id, base, callable wire.ID
	order              uint64
	arguments          []wire.ID
}

func Execute(in Input) (Result, error) {
	p, err := preflight(in)
	if err != nil {
		return Result{}, err
	}
	observations := []LifecycleObservation{}
	sequence := uint64(0)
	observe := func(u unit, from, to string) {
		observations = append(observations, LifecycleObservation{Sequence: sequence, Initializer: u.id.String(), From: from, To: to})
		sequence++
	}
	payloads := map[wire.ID]canonicaleval.Value{}
	outputs := []Output{}
	for _, u := range p.initializers {
		observe(u, "validated", "initializing")
		args := make([]canonicaleval.Value, len(u.arguments))
		for i, source := range u.arguments {
			value, resolveErr := resolveSource(p, source, payloads, map[wire.ID]bool{})
			if resolveErr != nil {
				return Result{}, fmt.Errorf("configuration_executor.resolve:%w", resolveErr)
			}
			args[i] = value
		}
		value, effects, invokeErr := canonicaleval.EvaluateFunctionAuthorized(p.graph, u.callable, args, in.CapabilityGrants)
		if invokeErr != nil || len(effects) != 0 {
			return Result{Lifecycle: append(observations, LifecycleObservation{Sequence: sequence, Initializer: u.id.String(), From: "initializing", To: "failed"})}, fmt.Errorf("configuration_executor.invoke:%w", coalesce(invokeErr, fmt.Errorf("initializer_effect")))
		}
		if value.Kind != "result" || value.Payload == nil {
			return Result{Lifecycle: append(observations, LifecycleObservation{Sequence: sequence, Initializer: u.id.String(), From: "initializing", To: "failed"})}, fmt.Errorf("configuration_executor.result")
		}
		if value.Variant != "ok" {
			return Result{Lifecycle: append(observations, LifecycleObservation{Sequence: sequence, Initializer: u.id.String(), From: "initializing", To: "failed"})}, fmt.Errorf("configuration_executor.initializer_error")
		}
		payloads[u.id] = cloneValue(*value.Payload)
		outputs = append(outputs, Output{Order: u.order, Initializer: u.id.String(), Callable: u.callable.String(), Result: cloneValue(value)})
		observe(u, "initializing", "initialized")
	}
	return Result{Lifecycle: observations, Outputs: outputs}, nil
}

func preflight(in Input) (plan, error) {
	if err := configurationinstance.ValidateBound(in.Bound); err != nil {
		return plan{}, fmt.Errorf("configuration_executor.bound:%w", err)
	}
	g, err := wire.Decode(in.Bound.Artifact)
	if err != nil {
		return plan{}, err
	}
	baseGraph, err := one(g, id("4010"))
	if err != nil {
		return plan{}, err
	}
	boundGraph, err := one(g, id("401f"))
	if err != nil {
		return plan{}, err
	}
	p := plan{graph: g, fields: map[wire.ID]canonicaleval.Value{}, runtime: map[wire.ID]canonicaleval.Value{}, static: map[wire.ID]canonicaleval.Value{}}
	fieldKeys := map[wire.ID]string{}
	for _, r := range g.Entities[baseGraph].Fields[id("4100")].List {
		q := g.Entities[r.Reference]
		fieldKeys[r.Reference] = string(q.Fields[id("4111")].Bytes)
	}
	usedConfig := map[string]bool{}
	for _, r := range g.Entities[baseGraph].Fields[id("4101")].List {
		q := g.Entities[r.Reference]
		field := q.Fields[id("4120")].Reference
		declaration := g.Entities[field]
		key := fieldKeys[field]
		typ := declaration.Fields[id("4112")].Reference
		var value canonicaleval.Value
		if embedded, ok := q.Fields[id("4121")]; ok {
			value, err = canonicaleval.EvaluateExpression(g, embedded.Reference)
		} else {
			capRef, ok := q.Fields[id("4123")]
			if !ok {
				return plan{}, fmt.Errorf("configuration_executor.configuration_source")
			}
			capability, capErr := capabilityName(g, capRef.Reference)
			if capErr != nil || !in.CapabilityGrants[capability] {
				return plan{}, fmt.Errorf("configuration_executor.configuration_capability")
			}
			var exists bool
			value, exists = in.ConfigurationInputs[key]
			if !exists {
				return plan{}, fmt.Errorf("configuration_executor.configuration_input:%s", key)
			}
			usedConfig[key] = true
		}
		if err != nil {
			return plan{}, fmt.Errorf("configuration_executor.configuration_expression:%w", err)
		}
		if err = canonicaleval.ValidateTypedValue(g, typ, value); err != nil {
			return plan{}, fmt.Errorf("configuration_executor.configuration_type:%s:%w", key, err)
		}
		p.fields[field] = cloneValue(value)
	}
	if !exactKeys(in.ConfigurationInputs, usedConfig) {
		return plan{}, fmt.Errorf("configuration_executor.configuration_inputs")
	}
	validations := append([]wire.Value(nil), g.Entities[baseGraph].Fields[id("4102")].List...)
	sort.Slice(validations, func(i, j int) bool {
		return g.Entities[validations[i].Reference].Fields[id("4142")].Unsigned < g.Entities[validations[j].Reference].Fields[id("4142")].Unsigned
	})
	for _, validationRef := range validations {
		validation := g.Entities[validationRef.Reference]
		field := validation.Fields[id("4140")].Reference
		value, effects, validationErr := canonicaleval.EvaluateFunctionAuthorized(g, validation.Fields[id("4141")].Reference, []canonicaleval.Value{p.fields[field]}, in.CapabilityGrants)
		if validationErr != nil || len(effects) != 0 || value.Kind != "result" || value.Variant != "ok" || value.Payload == nil {
			return plan{}, fmt.Errorf("configuration_executor.validation")
		}
		p.fields[field] = cloneValue(*value.Payload)
	}
	usedRuntime := map[string]bool{}
	for _, r := range g.Entities[boundGraph].Fields[id("41f1")].List {
		q := g.Entities[r.Reference]
		identity := string(q.Fields[id("4180")].Bytes)
		value, ok := in.RuntimeInputs[identity]
		if !ok {
			return plan{}, fmt.Errorf("configuration_executor.runtime_input:%s", identity)
		}
		if capRef, yes := q.Fields[id("4182")]; yes {
			capability, capErr := capabilityName(g, capRef.Reference)
			if capErr != nil || !in.CapabilityGrants[capability] {
				return plan{}, fmt.Errorf("configuration_executor.runtime_capability:%s", identity)
			}
		}
		if err := canonicaleval.ValidateTypedValue(g, q.Fields[id("4181")].Reference, value); err != nil {
			return plan{}, fmt.Errorf("configuration_executor.runtime_type:%s:%w", identity, err)
		}
		p.runtime[r.Reference] = cloneValue(value)
		usedRuntime[identity] = true
	}
	if !exactKeys(in.RuntimeInputs, usedRuntime) {
		return plan{}, fmt.Errorf("configuration_executor.runtime_inputs")
	}
	for _, r := range g.Entities[boundGraph].Fields[id("41f2")].List {
		q := g.Entities[r.Reference]
		base := q.Fields[id("41b0")].Reference
		b := g.Entities[base]
		u := unit{id: r.Reference, base: base, callable: b.Fields[id("4151")].Reference, order: b.Fields[id("4153")].Unsigned}
		args := q.Fields[id("41b1")].List
		sort.Slice(args, func(i, j int) bool {
			return g.Entities[args[i].Reference].Fields[id("41e1")].Unsigned < g.Entities[args[j].Reference].Fields[id("41e1")].Unsigned
		})
		for _, aRef := range args {
			u.arguments = append(u.arguments, g.Entities[aRef.Reference].Fields[id("41e2")].Reference)
		}
		p.initializers = append(p.initializers, u)
	}
	sort.Slice(p.initializers, func(i, j int) bool { return p.initializers[i].order < p.initializers[j].order })
	// Resolve every source that does not require a predecessor before invoking
	// anything. This evaluates all static/default expressions and validates all
	// record/runtime/config leaves atomically.
	for _, u := range p.initializers {
		for _, source := range u.arguments {
			if err := preflightSource(&p, source, map[wire.ID]bool{}); err != nil {
				return plan{}, err
			}
		}
	}
	return p, nil
}

func preflightSource(p *plan, x wire.ID, active map[wire.ID]bool) error {
	if active[x] {
		return fmt.Errorf("configuration_executor.source_cycle")
	}
	active[x] = true
	defer delete(active, x)
	q := p.graph.Entities[x]
	kind := p.graph.Entities[q.Fields[id("41a0")].Reference].Fields[id("4190")].Unsigned
	switch kind {
	case 0:
		if _, ok := p.fields[q.Fields[id("41a2")].Reference]; !ok {
			return fmt.Errorf("configuration_executor.field")
		}
	case 1:
		if _, ok := p.runtime[q.Fields[id("41a3")].Reference]; !ok {
			return fmt.Errorf("configuration_executor.runtime")
		}
	case 2:
		return nil
	case 3:
		ref := q.Fields[id("41a5")].Reference
		value, err := canonicaleval.EvaluateExpression(p.graph, ref)
		if err != nil {
			return fmt.Errorf("configuration_executor.static:%w", err)
		}
		p.static[x] = cloneValue(value)
	case 4:
		a := p.graph.Entities[q.Fields[id("41a6")].Reference]
		for _, mRef := range a.Fields[id("41c1")].List {
			if err := preflightSource(p, p.graph.Entities[mRef.Reference].Fields[id("41d1")].Reference, active); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("configuration_executor.source_kind")
	}
	return nil
}
func resolveSource(p plan, x wire.ID, payloads map[wire.ID]canonicaleval.Value, active map[wire.ID]bool) (canonicaleval.Value, error) {
	if active[x] {
		return canonicaleval.Value{}, fmt.Errorf("source_cycle")
	}
	active[x] = true
	defer delete(active, x)
	q := p.graph.Entities[x]
	kind := p.graph.Entities[q.Fields[id("41a0")].Reference].Fields[id("4190")].Unsigned
	switch kind {
	case 0:
		return cloneValue(p.fields[q.Fields[id("41a2")].Reference]), nil
	case 1:
		return cloneValue(p.runtime[q.Fields[id("41a3")].Reference]), nil
	case 2:
		v, ok := payloads[q.Fields[id("41a4")].Reference]
		if !ok {
			return canonicaleval.Value{}, fmt.Errorf("predecessor_missing")
		}
		return cloneValue(v), nil
	case 3:
		return cloneValue(p.static[x]), nil
	case 4:
		a := p.graph.Entities[q.Fields[id("41a6")].Reference]
		fields := map[string]canonicaleval.Value{}
		for _, mRef := range a.Fields[id("41c1")].List {
			m := p.graph.Entities[mRef.Reference]
			field := p.graph.Entities[m.Fields[id("41d0")].Reference]
			name := string(field.Fields[id("9310")].Bytes)
			v, err := resolveSource(p, m.Fields[id("41d1")].Reference, payloads, active)
			if err != nil {
				return canonicaleval.Value{}, err
			}
			fields[name] = v
		}
		value := canonicaleval.Value{Kind: "record", Fields: fields}
		if err := canonicaleval.ValidateTypedValue(p.graph, a.Fields[id("41c0")].Reference, value); err != nil {
			return canonicaleval.Value{}, err
		}
		return value, nil
	}
	return canonicaleval.Value{}, fmt.Errorf("source_kind")
}
func capabilityName(g wire.Envelope, x wire.ID) (string, error) {
	q := g.Entities[x]
	v, ok := q.Fields[id("160")]
	if q.Schema != id("16") || !ok || v.Tag != 5 || len(v.Bytes) == 0 {
		return "", fmt.Errorf("capability")
	}
	return string(v.Bytes), nil
}
func exactKeys(values map[string]canonicaleval.Value, used map[string]bool) bool {
	if len(values) != len(used) {
		return false
	}
	for k := range values {
		if !used[k] {
			return false
		}
	}
	return true
}
func cloneValue(v canonicaleval.Value) canonicaleval.Value {
	v.Items = append([]canonicaleval.Value(nil), v.Items...)
	for i := range v.Items {
		v.Items[i] = cloneValue(v.Items[i])
	}
	if v.Fields != nil {
		fields := map[string]canonicaleval.Value{}
		for k, x := range v.Fields {
			fields[k] = cloneValue(x)
		}
		v.Fields = fields
	}
	v.Entries = append([]canonicaleval.Entry(nil), v.Entries...)
	for i := range v.Entries {
		v.Entries[i].Key = cloneValue(v.Entries[i].Key)
		v.Entries[i].Value = cloneValue(v.Entries[i].Value)
	}
	if v.Payload != nil {
		x := cloneValue(*v.Payload)
		v.Payload = &x
	}
	if v.State != nil {
		x := cloneValue(*v.State)
		v.State = &x
	}
	if v.Result != nil {
		x := cloneValue(*v.Result)
		v.Result = &x
	}
	return v
}
func coalesce(primary, fallback error) error {
	if primary != nil {
		return primary
	}
	return fallback
}
func one(e wire.Envelope, s wire.ID) (wire.ID, error) {
	var out wire.ID
	for x, q := range e.Entities {
		if q.Schema == s {
			if out != (wire.ID{}) {
				return out, fmt.Errorf("configuration_executor.schema_count")
			}
			out = x
		}
	}
	if out == (wire.ID{}) {
		return out, fmt.Errorf("configuration_executor.schema_count")
	}
	return out, nil
}
func id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return x
}
