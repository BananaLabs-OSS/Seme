package canonicaleval

import (
	"fmt"

	"seme.local/reference/wire"
)

// EvaluateFunction invokes one structurally valid canonical Function directly.
// It does not synthesize or change a Program entry. Effectful functions fail
// closed; callers that intentionally authorize effects use
// EvaluateFunctionAuthorized.
func EvaluateFunction(g wire.Envelope, function wire.ID, arguments []Value) (Value, error) {
	value, _, err := EvaluateFunctionAuthorized(g, function, arguments, nil)
	return value, err
}

// EvaluateFunctionAuthorized invokes one canonical Function with exact ordered
// typed arguments and records only its reachable authorized effects.
func EvaluateFunctionAuthorized(g wire.Envelope, function wire.ID, arguments []Value, authorized map[string]bool) (Value, []EffectObservation, error) {
	fn, ok := g.Entities[function]
	if !ok || fn.Schema != id(0x9011) {
		return Value{}, nil, fmt.Errorf("canonicaleval.function")
	}
	parameters, pe := field(fn, 0x9111)
	resultType, re := field(fn, 0x9112)
	body, be := field(fn, 0x9113)
	if pe != nil || re != nil || be != nil || parameters.Tag != 7 || resultType.Tag != 6 || body.Tag != 6 || len(parameters.List) != len(arguments) {
		return Value{}, nil, fmt.Errorf("canonicaleval.function_shape")
	}
	if err := validateType(g, resultType.Reference, map[wire.ID]bool{}, 32); err != nil {
		return Value{}, nil, fmt.Errorf("canonicaleval.function_result_type:%w", err)
	}
	if err := authorizeReachable(g, body.Reference, authorized); err != nil {
		return Value{}, nil, err
	}
	env := map[wire.ID]Value{}
	runtime := &observedRuntime{}
	env[wire.ID{}] = Value{runtime: runtime}
	seen := map[wire.ID]bool{}
	for index, item := range parameters.List {
		if item.Tag != 6 || seen[item.Reference] {
			return Value{}, nil, fmt.Errorf("canonicaleval.function_parameter")
		}
		seen[item.Reference] = true
		parameter, exists := g.Entities[item.Reference]
		if !exists || parameter.Schema != id(0x9012) {
			return Value{}, nil, fmt.Errorf("canonicaleval.function_parameter")
		}
		typ, te := field(parameter, 0x9121)
		position, ie := field(parameter, 0x9122)
		if te != nil || ie != nil || typ.Tag != 6 || position.Tag != 3 || position.Unsigned != uint64(index) {
			return Value{}, nil, fmt.Errorf("canonicaleval.function_parameter_order")
		}
		if err := validateType(g, typ.Reference, map[wire.ID]bool{}, 32); err != nil {
			return Value{}, nil, fmt.Errorf("canonicaleval.function_parameter_type:%w", err)
		}
		if err := validateValue(g, typ.Reference, arguments[index], 32); err != nil {
			return Value{}, nil, fmt.Errorf("canonicaleval.function_argument.%d:%w", index, err)
		}
		env[item.Reference] = arguments[index]
	}
	value, err := evalBlock(g, body.Reference, env, 64)
	if err != nil {
		return Value{}, nil, err
	}
	if err := validateValue(g, resultType.Reference, value, 32); err != nil {
		return Value{}, nil, fmt.Errorf("canonicaleval.function_result:%w", err)
	}
	return value, append([]EffectObservation(nil), runtime.trace...), nil
}

// EvaluateExpression evaluates a closed, pure canonical expression. Parameter
// reads, effects, malformed references, and type-invalid results reject.
func EvaluateExpression(g wire.Envelope, expression wire.ID) (Value, error) {
	typ, ok := expressionType(g, expression)
	if !ok {
		return Value{}, fmt.Errorf("canonicaleval.expression_type")
	}
	if err := validateType(g, typ, map[wire.ID]bool{}, 32); err != nil {
		return Value{}, fmt.Errorf("canonicaleval.expression_type:%w", err)
	}
	if err := authorizeReachable(g, expression, nil); err != nil {
		return Value{}, err
	}
	value, err := eval(g, expression, map[wire.ID]Value{}, 64)
	if err != nil {
		return Value{}, err
	}
	if err := validateValue(g, typ, value, 32); err != nil {
		return Value{}, fmt.Errorf("canonicaleval.expression_result:%w", err)
	}
	return value, nil
}

func authorizeReachable(g wire.Envelope, root wire.ID, authorized map[string]bool) error {
	seen := map[wire.ID]bool{}
	todo := []wire.ID{root}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if seen[x] {
			continue
		}
		seen[x] = true
		if len(seen) > len(g.Entities) {
			return fmt.Errorf("canonicaleval.reachability_budget")
		}
		entity, exists := g.Entities[x]
		if !exists {
			return fmt.Errorf("canonicaleval.reachable_missing")
		}
		if entity.Schema == id(0x90f1) {
			capability, _, err := observedInvocation(g, entity)
			if err != nil {
				return err
			}
			if !authorized[capability] {
				return fmt.Errorf("canonicaleval.effect_denied")
			}
		}
		for _, value := range entity.Fields {
			collectReferences(value, &todo)
		}
	}
	return nil
}

func collectReferences(value wire.Value, out *[]wire.ID) {
	if value.Tag == 6 {
		*out = append(*out, value.Reference)
	}
	if value.Tag == 7 {
		for _, item := range value.List {
			collectReferences(item, out)
		}
	}
}
