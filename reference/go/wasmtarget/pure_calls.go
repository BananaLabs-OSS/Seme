package wasmtarget

import (
	"crypto/sha256"
	"fmt"

	"seme.local/reference/wire"
)

// normalizePureCalls certifies a closed, acyclic pure call graph and expands
// calls for the first Wasm target. Calls remain explicit in canonical Seme;
// inlining is only this target's physical realization.
func normalizePureCalls(graph wire.Envelope, program wire.Entity) (wire.Envelope, error) {
	functionsValue, err := field(program, 0x9150)
	entryValue, entryErr := field(program, 0x9151)
	if err != nil || entryErr != nil || functionsValue.Tag != 7 || len(functionsValue.List) == 0 || len(functionsValue.List) > 64 || entryValue.Tag != 6 {
		return wire.Envelope{}, fmt.Errorf("wasm.pure_program_functions")
	}
	copyGraph := graph
	copyGraph.Entities = make(map[wire.ID]wire.Entity, len(graph.Entities))
	for id, entity := range graph.Entities {
		copyGraph.Entities[id] = cloneWireEntity(entity)
	}
	members := make(map[wire.ID]wire.Entity, len(functionsValue.List))
	for _, item := range functionsValue.List {
		if item.Tag != 6 || item.Reference == (wire.ID{}) {
			return wire.Envelope{}, fmt.Errorf("wasm.pure_function_membership")
		}
		function, exists := copyGraph.Entities[item.Reference]
		if !exists || function.Schema != identity(0x9011) || members[item.Reference].ID != (wire.ID{}) {
			return wire.Envelope{}, fmt.Errorf("wasm.pure_function_membership")
		}
		members[item.Reference] = function
	}
	entry, exists := members[entryValue.Reference]
	if !exists {
		return wire.Envelope{}, fmt.Errorf("wasm.pure_entry_membership")
	}
	for id, function := range members {
		parameterTypes, _, body, functionErr := pureFunctionShape(copyGraph, function)
		if functionErr != nil {
			return wire.Envelope{}, functionErr
		}
		copyGraph, functionErr = normalizePureLocals(copyGraph, body, parameterTypes)
		if functionErr != nil {
			return wire.Envelope{}, functionErr
		}
		members[id] = copyGraph.Entities[id]
	}
	// Validate every declared function, including members unreachable from the
	// selected entry, so invalid code cannot hide inside a certified program.
	for _, function := range members {
		_, parameters, body, shapeErr := pureFunctionShape(copyGraph, function)
		if shapeErr != nil {
			return wire.Envelope{}, shapeErr
		}
		probe := cloneWireGraph(copyGraph)
		probeBudget := 16384
		if err := expandCallBlock(&probe, body, members, parameters, map[wire.ID]bool{function.ID: true}, &probeBudget); err != nil {
			return wire.Envelope{}, err
		}
	}
	_, entryParameters, entryBody, err := pureFunctionShape(copyGraph, entry)
	if err != nil {
		return wire.Envelope{}, err
	}
	budget := 16384
	if err := expandCallBlock(&copyGraph, entryBody, members, entryParameters, map[wire.ID]bool{entry.ID: true}, &budget); err != nil {
		return wire.Envelope{}, err
	}
	program = cloneWireEntity(copyGraph.Entities[program.ID])
	program.Fields[identity(0x9150)] = wire.Value{Tag: 7, List: []wire.Value{{Tag: 6, Reference: entry.ID}}}
	copyGraph.Entities[program.ID] = program
	return copyGraph, nil
}

func pureFunctionShape(graph wire.Envelope, function wire.Entity) (map[wire.ID]string, []wire.ID, wire.ID, error) {
	parameters, a := field(function, 0x9111)
	result, b := field(function, 0x9112)
	body, c := field(function, 0x9113)
	if a != nil || b != nil || c != nil || parameters.Tag != 7 || len(parameters.List) > 32 || result.Tag != 6 || body.Tag != 6 {
		return nil, nil, wire.ID{}, fmt.Errorf("wasm.pure_function_shape")
	}
	if _, err := pureType(graph, result.Reference); err != nil {
		return nil, nil, wire.ID{}, err
	}
	types := make(map[wire.ID]string, len(parameters.List))
	ids := make([]wire.ID, len(parameters.List))
	for index, item := range parameters.List {
		if item.Tag != 6 {
			return nil, nil, wire.ID{}, fmt.Errorf("wasm.pure_parameter_membership")
		}
		parameter, ok := graph.Entities[item.Reference]
		typeValue, typeErr := field(parameter, 0x9121)
		position, positionErr := field(parameter, 0x9122)
		if !ok || parameter.Schema != identity(0x9012) || typeErr != nil || positionErr != nil || typeValue.Tag != 6 || position.Tag != 3 || position.Unsigned != uint64(index) {
			return nil, nil, wire.ID{}, fmt.Errorf("wasm.pure_parameter_order")
		}
		valueType, err := pureType(graph, typeValue.Reference)
		if err != nil {
			return nil, nil, wire.ID{}, err
		}
		types[parameter.ID] = valueType.name
		ids[index] = parameter.ID
	}
	return types, ids, body.Reference, nil
}

func expandCallBlock(graph *wire.Envelope, id wire.ID, members map[wire.ID]wire.Entity, parameters []wire.ID, stack map[wire.ID]bool, budget *int) error {
	block, ok := graph.Entities[id]
	statements, err := field(block, 0x9800)
	if !ok || block.Schema != identity(0x9080) || err != nil || statements.Tag != 7 || len(statements.List) != 1 || statements.List[0].Tag != 6 {
		return fmt.Errorf("wasm.pure_call_block")
	}
	statement, ok := graph.Entities[statements.List[0].Reference]
	if !ok {
		return fmt.Errorf("wasm.pure_statement_missing")
	}
	switch statement.Schema {
	case identity(0x9081):
		values, err := field(statement, 0x9810)
		if err != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
			return fmt.Errorf("wasm.pure_return_values")
		}
		return expandCallExpression(graph, values.List[0].Reference, members, parameters, stack, budget)
	case identity(0x90c0):
		condition, a := field(statement, 0x9c00)
		thenValue, b := field(statement, 0x9c01)
		elseValue, c := field(statement, 0x9c02)
		if a != nil || b != nil || c != nil || condition.Tag != 6 || thenValue.Tag != 6 || elseValue.Tag != 6 {
			return fmt.Errorf("wasm.pure_if_fields")
		}
		if err := expandCallExpression(graph, condition.Reference, members, parameters, stack, budget); err != nil {
			return err
		}
		if err := expandCallBlock(graph, thenValue.Reference, members, parameters, stack, budget); err != nil {
			return err
		}
		return expandCallBlock(graph, elseValue.Reference, members, parameters, stack, budget)
	default:
		return fmt.Errorf("wasm.pure_call_terminal")
	}
}

func expandCallExpression(graph *wire.Envelope, id wire.ID, members map[wire.ID]wire.Entity, parameters []wire.ID, stack map[wire.ID]bool, budget *int) error {
	if *budget == 0 {
		return fmt.Errorf("wasm.pure_call_graph_too_large")
	}
	*budget--
	expression, ok := graph.Entities[id]
	if !ok {
		return fmt.Errorf("wasm.pure_expression_missing")
	}
	if expression.Schema != identity(0x9060) {
		for _, child := range expressionChildren(expression) {
			if err := expandCallExpression(graph, child, members, parameters, stack, budget); err != nil {
				return err
			}
		}
		return nil
	}
	calleeValue, a := field(expression, 0x9600)
	argumentsValue, b := field(expression, 0x9601)
	callee, exists := members[calleeValue.Reference]
	if a != nil || b != nil || calleeValue.Tag != 6 || argumentsValue.Tag != 7 || !exists {
		return fmt.Errorf("wasm.pure_call_target")
	}
	if stack[callee.ID] {
		return fmt.Errorf("wasm.pure_recursive_call")
	}
	_, calleeParameters, calleeBody, err := pureFunctionShape(*graph, callee)
	if err != nil || len(argumentsValue.List) != len(calleeParameters) {
		return fmt.Errorf("wasm.pure_call_arity")
	}
	substitutions := make(map[wire.ID]wire.ID, len(calleeParameters))
	for index, argument := range argumentsValue.List {
		if argument.Tag != 6 {
			return fmt.Errorf("wasm.pure_call_argument")
		}
		if err := expandCallExpression(graph, argument.Reference, members, parameters, stack, budget); err != nil {
			return err
		}
		substitutions[calleeParameters[index]] = argument.Reference
	}
	resultExpression, err := singleReturnExpression(*graph, calleeBody)
	if err != nil {
		return err
	}
	stack[callee.ID] = true
	err = cloneCallExpression(graph, resultExpression, id, id, substitutions, members, calleeParameters, stack, budget)
	delete(stack, callee.ID)
	return err
}

func singleReturnExpression(graph wire.Envelope, body wire.ID) (wire.ID, error) {
	block, ok := graph.Entities[body]
	statements, err := field(block, 0x9800)
	if !ok || block.Schema != identity(0x9080) || err != nil || statements.Tag != 7 || len(statements.List) != 1 || statements.List[0].Tag != 6 {
		return wire.ID{}, fmt.Errorf("wasm.pure_callee_not_expression")
	}
	statement, ok := graph.Entities[statements.List[0].Reference]
	values, valueErr := field(statement, 0x9810)
	if !ok || statement.Schema != identity(0x9081) || valueErr != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
		return wire.ID{}, fmt.Errorf("wasm.pure_callee_not_expression")
	}
	return values.List[0].Reference, nil
}

func cloneCallExpression(graph *wire.Envelope, source, target, call wire.ID, substitutions map[wire.ID]wire.ID, members map[wire.ID]wire.Entity, parameters []wire.ID, stack map[wire.ID]bool, budget *int) error {
	entity, ok := graph.Entities[source]
	if !ok {
		return fmt.Errorf("wasm.pure_expression_missing")
	}
	if entity.Schema == identity(0x9013) {
		parameter, err := field(entity, 0x9130)
		replacement, exists := substitutions[parameter.Reference]
		if err != nil || parameter.Tag != 6 || !exists {
			return fmt.Errorf("wasm.pure_callee_parameter")
		}
		entity = cloneWireEntity(graph.Entities[replacement])
		entity.ID = target
		graph.Entities[target] = entity
		return nil
	}
	entity = cloneWireEntity(entity)
	entity.ID = target
	for fieldID, value := range entity.Fields {
		if value.Tag == 6 && isExpressionEntity(*graph, value.Reference) {
			child := derivedCallID(call, value.Reference)
			if err := cloneCallExpression(graph, value.Reference, child, call, substitutions, members, parameters, stack, budget); err != nil {
				return err
			}
			value.Reference = child
			entity.Fields[fieldID] = value
		} else if value.Tag == 7 {
			for index, item := range value.List {
				if item.Tag == 6 && isExpressionEntity(*graph, item.Reference) {
					child := derivedCallID(call, item.Reference)
					if err := cloneCallExpression(graph, item.Reference, child, call, substitutions, members, parameters, stack, budget); err != nil {
						return err
					}
					value.List[index].Reference = child
				}
			}
			entity.Fields[fieldID] = value
		}
	}
	graph.Entities[target] = entity
	return expandCallExpression(graph, target, members, parameters, stack, budget)
}

func isExpressionEntity(graph wire.Envelope, id wire.ID) bool {
	entity, ok := graph.Entities[id]
	if !ok {
		return false
	}
	return entity.Schema != identity(0x9010) && entity.Schema != identity(0x9020) && entity.Schema != identity(0x9040) && entity.Schema != identity(0x9011) && entity.Schema != identity(0x9012) && entity.Schema != identity(0x9080) && entity.Schema != identity(0x9081) && entity.Schema != identity(0x90d0)
}

func derivedCallID(call, source wire.ID) wire.ID {
	digest := sha256.Sum256(append(append([]byte("seme.wasm.call-inline.v1\x00"), call[:]...), source[:]...))
	var id wire.ID
	id[0] = 0x80
	copy(id[1:], digest[:15])
	return id
}

func cloneWireEntity(entity wire.Entity) wire.Entity {
	fields := make(map[wire.ID]wire.Value, len(entity.Fields))
	for id, value := range entity.Fields {
		if value.Tag == 7 {
			value.List = append([]wire.Value(nil), value.List...)
		}
		fields[id] = value
	}
	entity.Fields = fields
	return entity
}

func cloneWireGraph(graph wire.Envelope) wire.Envelope {
	clone := graph
	clone.Entities = make(map[wire.ID]wire.Entity, len(graph.Entities))
	for id, entity := range graph.Entities {
		clone.Entities[id] = cloneWireEntity(entity)
	}
	return clone
}
