package wasmtarget

import (
	"fmt"

	"seme.local/reference/wire"
)

// normalizePureLocals validates lexical scope and replaces immutable reads with
// their already-validated pure initializer graph. This is an execution
// optimization only: the certified input retains explicit single-evaluation
// bindings, and effectful initializers are outside the pure profile.
func normalizePureLocals(graph wire.Envelope, body wire.ID, parameterTypes map[wire.ID]string) (wire.Envelope, error) {
	if len(bySchema(graph, 0x90d0))+len(bySchema(graph, 0x90d1))+len(bySchema(graph, 0x90d2)) == 0 {
		return graph, nil
	}
	copyGraph := graph
	copyGraph.Entities = make(map[wire.ID]wire.Entity, len(graph.Entities))
	for id, entity := range graph.Entities {
		copyGraph.Entities[id] = entity
	}
	budget := 4096
	if err := normalizeLocalBlock(&copyGraph, body, map[wire.ID]localInitializer{}, parameterTypes, map[wire.ID]bool{}, &budget); err != nil {
		return wire.Envelope{}, err
	}
	return copyGraph, nil
}

type localInitializer struct {
	expression wire.ID
	valueType  string
}

func normalizeLocalBlock(graph *wire.Envelope, id wire.ID, inherited map[wire.ID]localInitializer, parameterTypes map[wire.ID]string, visiting map[wire.ID]bool, budget *int) error {
	if *budget == 0 || visiting[id] {
		return fmt.Errorf("wasm.pure_local_control_cycle_or_size")
	}
	*budget--
	visiting[id] = true
	defer delete(visiting, id)
	block, ok := graph.Entities[id]
	if !ok || block.Schema != identity(0x9080) {
		return fmt.Errorf("wasm.pure_local_block")
	}
	statements, err := field(block, 0x9800)
	if err != nil || statements.Tag != 7 || len(statements.List) == 0 {
		return fmt.Errorf("wasm.pure_local_statements")
	}
	visible := cloneInitializers(inherited)
	seen := map[wire.ID]bool{}
	var terminal wire.Value
	for index, item := range statements.List {
		if item.Tag != 6 {
			return fmt.Errorf("wasm.pure_local_statement_reference")
		}
		statement, exists := graph.Entities[item.Reference]
		if !exists {
			return fmt.Errorf("wasm.pure_local_statement_missing")
		}
		if statement.Schema == identity(0x90d1) {
			if terminal.Tag != 0 {
				return fmt.Errorf("wasm.pure_local_after_terminal")
			}
			bindingValue, fieldErr := field(statement, 0x9d10)
			if fieldErr != nil || bindingValue.Tag != 6 || seen[bindingValue.Reference] {
				return fmt.Errorf("wasm.pure_local_binding_reference")
			}
			binding, exists := graph.Entities[bindingValue.Reference]
			if !exists || binding.Schema != identity(0x90d0) {
				return fmt.Errorf("wasm.pure_local_binding_missing")
			}
			name, nameErr := field(binding, 0x9d00)
			typeValue, typeErr := field(binding, 0x9d01)
			initializer, initializerErr := field(binding, 0x9d02)
			if nameErr != nil || name.Tag != 5 || typeErr != nil || typeValue.Tag != 6 || initializerErr != nil || initializer.Tag != 6 {
				return fmt.Errorf("wasm.pure_local_binding_fields")
			}
			valueTypeName := ""
			valueType, typeErr := pureType(*graph, typeValue.Reference)
			if typeErr == nil {
				valueTypeName = valueType.name
			} else if recordType, ok := graph.Entities[typeValue.Reference]; ok && recordType.Schema == identity(0x9030) {
				valueTypeName = "record"
			} else {
				return typeErr
			}
			if err := normalizeLocalExpression(graph, initializer.Reference, visible, map[wire.ID]bool{}, budget); err != nil {
				return err
			}
			containsDeferred := containsExpressionSchema(*graph, initializer.Reference, identity(0x9060), map[wire.ID]bool{}) ||
				containsExpressionSchema(*graph, initializer.Reference, identity(0x9032), map[wire.ID]bool{}) ||
				containsExpressionSchema(*graph, initializer.Reference, identity(0x9033), map[wire.ID]bool{})
			if !containsDeferred {
				checkBudget := 4096
				if err := validatePureExpression(*graph, initializer.Reference, valueTypeName, parameterTypes, map[wire.ID]bool{}, &checkBudget); err != nil {
					return err
				}
			}
			visible[binding.ID] = localInitializer{expression: initializer.Reference, valueType: valueTypeName}
			seen[binding.ID] = true
			continue
		}
		if index != len(statements.List)-1 {
			return fmt.Errorf("wasm.pure_local_terminal_order")
		}
		terminal = item
		if statement.Schema == identity(0x9081) {
			values, valueErr := field(statement, 0x9810)
			if valueErr != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
				return fmt.Errorf("wasm.pure_return_values")
			}
			if err := normalizeLocalExpression(graph, values.List[0].Reference, visible, map[wire.ID]bool{}, budget); err != nil {
				return err
			}
		} else if statement.Schema == identity(0x90c0) {
			condition, a := field(statement, 0x9c00)
			thenValue, b := field(statement, 0x9c01)
			elseValue, c := field(statement, 0x9c02)
			if a != nil || b != nil || c != nil || condition.Tag != 6 || thenValue.Tag != 6 || elseValue.Tag != 6 {
				return fmt.Errorf("wasm.pure_if_fields")
			}
			if err := normalizeLocalExpression(graph, condition.Reference, visible, map[wire.ID]bool{}, budget); err != nil {
				return err
			}
			if err := normalizeLocalBlock(graph, thenValue.Reference, visible, parameterTypes, visiting, budget); err != nil {
				return err
			}
			if err := normalizeLocalBlock(graph, elseValue.Reference, visible, parameterTypes, visiting, budget); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("wasm.pure_local_terminal")
		}
	}
	if terminal.Tag == 0 {
		return fmt.Errorf("wasm.pure_local_missing_terminal")
	}
	fields := make(map[wire.ID]wire.Value, len(block.Fields))
	for key, value := range block.Fields {
		fields[key] = value
	}
	fields[identity(0x9800)] = wire.Value{Tag: 7, List: []wire.Value{terminal}}
	block.Fields = fields
	graph.Entities[id] = block
	return nil
}

func containsExpressionSchema(graph wire.Envelope, id, schema wire.ID, seen map[wire.ID]bool) bool {
	if seen[id] {
		return false
	}
	seen[id] = true
	entity, ok := graph.Entities[id]
	if !ok {
		return false
	}
	if entity.Schema == schema {
		return true
	}
	for _, child := range expressionChildren(entity) {
		if containsExpressionSchema(graph, child, schema, seen) {
			return true
		}
	}
	return false
}

func normalizeLocalExpression(graph *wire.Envelope, id wire.ID, visible map[wire.ID]localInitializer, visiting map[wire.ID]bool, budget *int) error {
	if *budget == 0 || visiting[id] {
		return fmt.Errorf("wasm.pure_local_expression_cycle_or_size")
	}
	*budget--
	visiting[id] = true
	defer delete(visiting, id)
	expression, ok := graph.Entities[id]
	if !ok {
		return fmt.Errorf("wasm.pure_local_expression_missing")
	}
	if expression.Schema == identity(0x90d2) {
		binding, err := field(expression, 0x9d20)
		initializer, visibleNow := visible[binding.Reference]
		if err != nil || binding.Tag != 6 || !visibleNow {
			return fmt.Errorf("wasm.pure_local_read_scope")
		}
		if err := normalizeLocalExpression(graph, initializer.expression, visible, visiting, budget); err != nil {
			return err
		}
		replacement := graph.Entities[initializer.expression]
		replacement.ID = id
		graph.Entities[id] = replacement
		return nil
	}
	for _, child := range expressionChildren(expression) {
		if err := normalizeLocalExpression(graph, child, visible, visiting, budget); err != nil {
			return err
		}
	}
	return nil
}

func expressionChildren(entity wire.Entity) []wire.ID {
	var fields []wire.ID
	var children []wire.ID
	switch entity.Schema {
	case identity(0x9014):
		fields = []wire.ID{identity(0x9140), identity(0x9141)}
	case identity(0x9021):
		fields = []wire.ID{identity(0x9160), identity(0x9161)}
	case identity(0x9090):
		fields = []wire.ID{identity(0x9900), identity(0x9901)}
	case identity(0x90a0):
		fields = []wire.ID{identity(0x9a00), identity(0x9a01)}
	case identity(0x90b1):
		fields = []wire.ID{identity(0x9b10), identity(0x9b11)}
	case identity(0x90c1):
		fields = []wire.ID{identity(0x9c10), identity(0x9c11)}
	case identity(0x90c2):
		fields = []wire.ID{identity(0x9c20), identity(0x9c21)}
	case identity(0x90c3):
		fields = []wire.ID{identity(0x9c30), identity(0x9c31)}
	case identity(0x90e3):
		fields = []wire.ID{identity(0x9e31)}
	case identity(0x90e4):
		fields = []wire.ID{identity(0x9e40)}
	case identity(0x9060):
		if arguments, ok := entity.Fields[identity(0x9601)]; ok && arguments.Tag == 7 {
			for _, argument := range arguments.List {
				if argument.Tag == 6 {
					children = append(children, argument.Reference)
				}
			}
		}
	case identity(0x9032):
		fields = []wire.ID{identity(0x9320)}
	case identity(0x9033):
		if values, ok := entity.Fields[identity(0x9331)]; ok && values.Tag == 7 {
			for _, value := range values.List {
				if value.Tag == 6 {
					children = append(children, value.Reference)
				}
			}
		}
	}
	for _, id := range fields {
		if value, ok := entity.Fields[id]; ok && value.Tag == 6 {
			children = append(children, value.Reference)
		}
	}
	return children
}

func cloneInitializers(source map[wire.ID]localInitializer) map[wire.ID]localInitializer {
	result := make(map[wire.ID]localInitializer, len(source))
	for id, value := range source {
		result[id] = value
	}
	return result
}
