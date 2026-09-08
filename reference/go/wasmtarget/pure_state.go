package wasmtarget

import (
	"fmt"

	"seme.local/reference/wire"
)

// validateStateScopes certifies declaration-before-use and lexical visibility
// for mutable places before any target code is emitted.
func validateStateScopes(graph wire.Envelope, blockID wire.ID, inherited map[wire.ID]bool, declared map[wire.ID]bool, visiting map[wire.ID]bool, budget *int) error {
	if *budget == 0 || visiting[blockID] {
		return fmt.Errorf("wasm.pure_state_scope_cycle_or_size")
	}
	*budget--
	visiting[blockID] = true
	defer delete(visiting, blockID)
	block, ok := graph.Entities[blockID]
	statements, err := field(block, 0x9800)
	if !ok || block.Schema != identity(0x9080) || err != nil || statements.Tag != 7 || len(statements.List) == 0 {
		return fmt.Errorf("wasm.pure_state_scope_block")
	}
	visible := cloneVisibility(inherited)
	terminal := false
	for index, item := range statements.List {
		if item.Tag != 6 || terminal {
			return fmt.Errorf("wasm.pure_state_scope_statement")
		}
		statement, exists := graph.Entities[item.Reference]
		if !exists {
			return fmt.Errorf("wasm.pure_state_scope_statement")
		}
		switch statement.Schema {
		case identity(0x90e1):
			placeValue, fieldErr := field(statement, 0x9e10)
			if fieldErr != nil || placeValue.Tag != 6 || visible[placeValue.Reference] || declared[placeValue.Reference] {
				return fmt.Errorf("wasm.pure_place_declaration_scope")
			}
			place, exists := graph.Entities[placeValue.Reference]
			if !exists || place.Schema != identity(0x90e0) {
				return fmt.Errorf("wasm.pure_place_declaration_scope")
			}
			initializer, fieldErr := field(place, 0x9e02)
			if fieldErr != nil || initializer.Tag != 6 {
				return fmt.Errorf("wasm.pure_place_initializer_scope")
			}
			if err := validateStateExpression(graph, initializer.Reference, visible, map[wire.ID]bool{}, budget); err != nil {
				return err
			}
			visible[place.ID] = true
			declared[place.ID] = true
		case identity(0x90e3):
			placeValue, placeErr := field(statement, 0x9e30)
			value, valueErr := field(statement, 0x9e31)
			if placeErr != nil || valueErr != nil || placeValue.Tag != 6 || value.Tag != 6 || !visible[placeValue.Reference] {
				return fmt.Errorf("wasm.pure_assignment_scope")
			}
			if err := validateStateExpression(graph, value.Reference, visible, map[wire.ID]bool{}, budget); err != nil {
				return err
			}
		case identity(0x90e4):
			condition, conditionErr := field(statement, 0x9e40)
			body, bodyErr := field(statement, 0x9e41)
			if conditionErr != nil || bodyErr != nil || condition.Tag != 6 || body.Tag != 6 {
				return fmt.Errorf("wasm.pure_while_scope")
			}
			if err := validateStateExpression(graph, condition.Reference, visible, map[wire.ID]bool{}, budget); err != nil {
				return err
			}
			if err := validateStateScopes(graph, body.Reference, visible, declared, visiting, budget); err != nil {
				return err
			}
		case identity(0x9081):
			values, valueErr := field(statement, 0x9810)
			if valueErr != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 || index != len(statements.List)-1 {
				return fmt.Errorf("wasm.pure_state_return_scope")
			}
			if err := validateStateExpression(graph, values.List[0].Reference, visible, map[wire.ID]bool{}, budget); err != nil {
				return err
			}
			terminal = true
		default:
			return fmt.Errorf("wasm.pure_state_scope_statement")
		}
	}
	return nil
}

func validateStateExpression(graph wire.Envelope, id wire.ID, visible map[wire.ID]bool, visiting map[wire.ID]bool, budget *int) error {
	if *budget == 0 || visiting[id] {
		return fmt.Errorf("wasm.pure_state_expression_cycle_or_size")
	}
	*budget--
	visiting[id] = true
	defer delete(visiting, id)
	expression, ok := graph.Entities[id]
	if !ok {
		return fmt.Errorf("wasm.pure_state_expression_missing")
	}
	if expression.Schema == identity(0x90e2) {
		place, err := field(expression, 0x9e20)
		if err != nil || place.Tag != 6 || !visible[place.Reference] {
			return fmt.Errorf("wasm.pure_place_read_scope")
		}
		return nil
	}
	for _, child := range expressionChildren(expression) {
		if err := validateStateExpression(graph, child, visible, visiting, budget); err != nil {
			return err
		}
	}
	return nil
}

func cloneVisibility(source map[wire.ID]bool) map[wire.ID]bool {
	result := make(map[wire.ID]bool, len(source))
	for id, visible := range source {
		result[id] = visible
	}
	return result
}
