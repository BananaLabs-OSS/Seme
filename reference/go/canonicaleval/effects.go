package canonicaleval

import (
	"fmt"

	"seme.local/reference/wire"
)

type EffectObservation struct {
	Capability string `json:"capability"`
	Value      bool   `json:"value"`
}

// EvaluateObserved executes the bounded ordered-effect profile structurally.
// Authorization is explicit input: denial occurs before any observation runs.
func EvaluateObserved(g wire.Envelope, arguments []Value, authorized map[string]bool) (Value, []EffectObservation, error) {
	programs := schemaEntities(g, 0x9015)
	if len(programs) != 1 {
		return Value{}, nil, fmt.Errorf("canonicaleval.effect_program")
	}
	entry, err := field(programs[0], 0x9151)
	if err != nil || entry.Tag != 6 {
		return Value{}, nil, fmt.Errorf("canonicaleval.effect_entry")
	}
	fn := g.Entities[entry.Reference]
	params, pe := field(fn, 0x9111)
	body, be := field(fn, 0x9113)
	if fn.Schema != id(0x9011) || pe != nil || be != nil || params.Tag != 7 || body.Tag != 6 || len(params.List) != len(arguments) {
		return Value{}, nil, fmt.Errorf("canonicaleval.effect_arguments")
	}
	env := map[wire.ID]Value{}
	for i, p := range params.List {
		if p.Tag != 6 {
			return Value{}, nil, fmt.Errorf("canonicaleval.effect_parameter")
		}
		entity := g.Entities[p.Reference]
		typ, e := field(entity, 0x9121)
		if e != nil || typ.Tag != 6 || validateValue(g, typ.Reference, arguments[i], 16) != nil {
			return Value{}, nil, fmt.Errorf("canonicaleval.effect_argument")
		}
		env[p.Reference] = arguments[i]
	}
	block := g.Entities[body.Reference]
	statements, se := field(block, 0x9800)
	if block.Schema != id(0x9080) || se != nil || statements.Tag != 7 {
		return Value{}, nil, fmt.Errorf("canonicaleval.effect_block")
	}
	// Certify the entire ordered statement list before running any effect. A
	// malformed later statement must never leave a partial external trace.
	for index, item := range statements.List {
		if item.Tag != 6 {
			return Value{}, nil, fmt.Errorf("canonicaleval.effect_statement")
		}
		s := g.Entities[item.Reference]
		if index == len(statements.List)-1 {
			values, e := field(s, 0x9810)
			if s.Schema != id(0x9081) || e != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
				return Value{}, nil, fmt.Errorf("canonicaleval.effect_return")
			}
			continue
		}
		if _, _, e := observedInvocation(g, s); e != nil {
			return Value{}, nil, e
		}
	}
	trace := []EffectObservation{}
	for index, item := range statements.List {
		if item.Tag != 6 {
			return Value{}, nil, fmt.Errorf("canonicaleval.effect_statement")
		}
		s := g.Entities[item.Reference]
		if index == len(statements.List)-1 {
			values, e := field(s, 0x9810)
			if s.Schema != id(0x9081) || e != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
				return Value{}, nil, fmt.Errorf("canonicaleval.effect_return")
			}
			result, e := eval(g, values.List[0].Reference, env, 32)
			return result, trace, e
		}
		capability, argument, _ := observedInvocation(g, s)
		if !authorized[capability] {
			return Value{}, nil, fmt.Errorf("canonicaleval.effect_denied")
		}
		value, e := eval(g, argument, env, 32)
		if e != nil || value.Kind != "bool" {
			return Value{}, nil, fmt.Errorf("canonicaleval.effect_value")
		}
		trace = append(trace, EffectObservation{capability, value.Bool})
	}
	return Value{}, nil, fmt.Errorf("canonicaleval.effect_missing_return")
}

func observedInvocation(g wire.Envelope, s wire.Entity) (string, wire.ID, error) {
	effectRef, e := field(s, 0x9f10)
	args, a := field(s, 0x9f11)
	if s.Schema != id(0x90f1) || e != nil || a != nil || effectRef.Tag != 6 || args.Tag != 7 || len(args.List) != 1 || args.List[0].Tag != 6 {
		return "", wire.ID{}, fmt.Errorf("canonicaleval.effect_invoke")
	}
	effect := g.Entities[effectRef.Reference]
	name, ne := field(effect, 0x150)
	capRef, ce := field(effect, 0x151)
	cap := g.Entities[capRef.Reference]
	capName, cne := field(cap, 0x160)
	if effect.Schema != id(0x15) || cap.Schema != id(0x16) || ne != nil || ce != nil || cne != nil || name.Tag != 5 || capRef.Tag != 6 || capName.Tag != 5 || string(name.Bytes) != "observability.log" || string(capName.Bytes) != "observability.log" {
		return "", wire.ID{}, fmt.Errorf("canonicaleval.effect_authority")
	}
	return "observability.log", args.List[0].Reference, nil
}
