// Package wasmtarget implements the first deliberately scoped canonical
// Seme-to-WebAssembly backend. It consumes only a validated semantic plan.
package wasmtarget

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"seme.local/reference/wire"
)

func Lower(plan wire.Envelope) ([]byte, error) {
	plans := bySchema(plan, 0xc014)
	boundaries := bySchema(plan, 0xc015)
	functions := bySchema(plan, 0x9011)
	calls := bySchema(plan, 0x9060)
	effects := bySchema(plan, 0x15)
	if len(plans) != 1 || len(functions) != 2 || len(calls) != 1 || len(effects) != 1 {
		return nil, fmt.Errorf("wasm.profile_cardinality")
	}
	if value, err := field(plans[0], 0xc144); err != nil || value.Tag != 2 {
		return nil, fmt.Errorf("wasm.plan_not_executable")
	}
	if len(boundaries) != 1 {
		return nil, fmt.Errorf("wasm.boundary_cardinality")
	}
	resolutionList, err := field(plans[0], 0xc142)
	if err != nil || len(resolutionList.List) != 2 {
		return nil, fmt.Errorf("wasm.resolutions")
	}
	for _, item := range resolutionList.List {
		resolution, ok := plan.Entities[item.Reference]
		if !ok || resolution.Schema != identity(0xc013) {
			return nil, fmt.Errorf("wasm.resolution_missing")
		}
		fidelity, err := field(resolution, 0xc133)
		if err != nil || fidelity.Unsigned != 2 {
			return nil, fmt.Errorf("wasm.resolution_not_adapted")
		}
		if selected, err := field(resolution, 0xc132); err != nil || selected.Tag != 6 {
			return nil, fmt.Errorf("wasm.resolution_rule_missing")
		}
	}
	transport, err := field(boundaries[0], 0xc153)
	if err != nil {
		return nil, err
	}
	transportEntity, ok := plan.Entities[transport.Reference]
	if !ok || transportEntity.Schema != identity(0xb013) {
		return nil, fmt.Errorf("wasm.transport_missing")
	}
	transportName, err := field(transportEntity, 0xb130)
	if err != nil || string(transportName.Bytes) != "seme.pulp.log-v1" {
		return nil, fmt.Errorf("wasm.unsupported_transport")
	}
	effectName, err := field(effects[0], 0x150)
	if err != nil || string(effectName.Bytes) != "observability.log" {
		return nil, fmt.Errorf("wasm.unsupported_effect")
	}
	callee, err := field(calls[0], 0x9600)
	if err != nil {
		return nil, fmt.Errorf("wasm.call_callee")
	}
	helper, ok := plan.Entities[callee.Reference]
	if !ok || helper.Schema != identity(0x9011) {
		return nil, fmt.Errorf("wasm.call_callee_missing")
	}
	var entry wire.Entity
	for _, function := range functions {
		if function.ID != helper.ID {
			entry = function
		}
	}
	if entry.ID == (wire.ID{}) {
		return nil, fmt.Errorf("wasm.entry_function")
	}
	helperInstructions, err := validateDecisionHelper(plan, helper)
	if err != nil {
		return nil, err
	}
	layout, err := validateFunction(plan, entry, helper)
	if err != nil {
		return nil, err
	}
	layout.helperInstructions = helperInstructions
	return module(layout)
}

type applicationLayout struct {
	helperInstructions []byte
	requestOffsets     [3]uint64
	requestHeaderSize  uint64
	responseBoolOffset uint64
	responseHeaderSize uint64
	errorMessage       []byte
}

func validateFunction(graph wire.Envelope, function, helper wire.Entity) (applicationLayout, error) {
	var layout applicationLayout
	parameters, err := field(function, 0x9111)
	if err != nil || len(parameters.List) != 1 {
		return layout, fmt.Errorf("wasm.function_parameters")
	}
	parameter, ok := graph.Entities[parameters.List[0].Reference]
	if !ok || parameter.Schema != identity(0x9012) {
		return layout, fmt.Errorf("wasm.parameter_missing")
	}
	position, err := field(parameter, 0x9122)
	if err != nil || position.Unsigned != 0 {
		return layout, fmt.Errorf("wasm.parameter_order")
	}
	requestType, err := referenced(graph, parameter, 0x9121, 0x9030)
	if err != nil {
		return layout, err
	}
	requestFields, err := validateRecord(graph, requestType, "AdmitRequest", []recordFieldProfile{{"Current", 0x9010, 8}, {"Delta", 0x9010, 8}, {"Limit", 0x9010, 8}, {"Subject", 0x9040, 4}, {"Evidence", 0x9041, 4}})
	if err != nil {
		return layout, err
	}
	for index, item := range requestFields[:3] {
		layout.requestOffsets[index] = item.offset
	}
	layout.requestHeaderSize = requestFields[len(requestFields)-1].offset + requestFields[len(requestFields)-1].size
	resultValue, err := field(function, 0x9112)
	if err != nil {
		return layout, fmt.Errorf("wasm.function_result")
	}
	resultType, ok := graph.Entities[resultValue.Reference]
	if !ok || resultType.Schema != identity(0x9042) {
		return layout, fmt.Errorf("wasm.function_result")
	}
	okType, okErr := field(resultType, 0x9400)
	errorType, errorErr := field(resultType, 0x9401)
	responseType, responseOK := graph.Entities[okType.Reference]
	errorTypeEntity, errorOK := graph.Entities[errorType.Reference]
	if okErr != nil || errorErr != nil || !responseOK || responseType.Schema != identity(0x9030) || !errorOK || errorTypeEntity.Schema != identity(0x9030) {
		return layout, fmt.Errorf("wasm.result_profile")
	}
	responseFields, err := validateRecord(graph, responseType, "AdmitResponse", []recordFieldProfile{{"Accepted", 0x9020, 1}, {"Subject", 0x9040, 4}, {"Evidence", 0x9041, 4}})
	if err != nil {
		return layout, err
	}
	layout.responseBoolOffset = 1 + responseFields[0].offset
	layout.responseHeaderSize = 1 + responseFields[len(responseFields)-1].offset + responseFields[len(responseFields)-1].size
	errorFields, err := validateRecord(graph, errorTypeEntity, "AdmitError", []recordFieldProfile{{"Message", 0x9040, 4}})
	if err != nil {
		return layout, err
	}
	conditional, err := referenced(graph, function, 0x9113, 0x9052)
	if err != nil {
		return layout, err
	}
	condition, err := referenced(graph, conditional, 0x9520, 0x9051)
	if err != nil {
		return layout, err
	}
	conditionLeft, err := referenced(graph, condition, 0x9510, 0x9032)
	if err != nil {
		return layout, err
	}
	conditionField, conditionFieldErr := field(conditionLeft, 0x9321)
	if conditionFieldErr != nil || conditionField.Reference != requestFields[3].entity.ID {
		return layout, fmt.Errorf("wasm.error_condition_field")
	}
	resultError, err := referenced(graph, conditional, 0x9521, 0x9044)
	if err != nil {
		return layout, err
	}
	resultErrorType, typeErr := field(resultError, 0x9420)
	if typeErr != nil || resultErrorType.Reference != resultType.ID {
		return layout, fmt.Errorf("wasm.result_error_type")
	}
	errorConstruct, err := referenced(graph, resultError, 0x9421, 0x9033)
	if err != nil {
		return layout, err
	}
	errorConstructType, errorConstructTypeErr := field(errorConstruct, 0x9330)
	errorValues, errorValuesErr := field(errorConstruct, 0x9331)
	if errorConstructTypeErr != nil || errorValuesErr != nil || errorConstructType.Reference != errorTypeEntity.ID || len(errorValues.List) != len(errorFields) {
		return layout, fmt.Errorf("wasm.error_construct")
	}
	errorLiteral, ok := graph.Entities[errorValues.List[0].Reference]
	if !ok || errorLiteral.Schema != identity(0x9050) {
		return layout, fmt.Errorf("wasm.error_literal")
	}
	errorMessage, err := field(errorLiteral, 0x9500)
	if err != nil || len(errorMessage.Bytes) == 0 || len(errorMessage.Bytes) > 4096 {
		return layout, fmt.Errorf("wasm.error_message")
	}
	layout.errorMessage = append([]byte(nil), errorMessage.Bytes...)
	resultOk, err := referenced(graph, conditional, 0x9522, 0x9043)
	if err != nil {
		return layout, err
	}
	resultOKType, resultTypeErr := field(resultOk, 0x9410)
	if resultTypeErr != nil || resultOKType.Reference != resultType.ID {
		return layout, fmt.Errorf("wasm.result_ok_type")
	}
	body, err := referenced(graph, resultOk, 0x9411, 0x9033)
	if err != nil {
		return layout, err
	}
	constructedType, err := field(body, 0x9330)
	if err != nil || constructedType.Reference != responseType.ID {
		return layout, fmt.Errorf("wasm.response_construct_type")
	}
	values, err := field(body, 0x9331)
	if err != nil || len(values.List) != 3 {
		return layout, fmt.Errorf("wasm.response_construct_values")
	}
	call, ok := graph.Entities[values.List[0].Reference]
	if !ok || call.Schema != identity(0x9060) {
		return layout, fmt.Errorf("wasm.response_value")
	}
	callee, calleeErr := field(call, 0x9600)
	arguments, argumentsErr := field(call, 0x9601)
	if calleeErr != nil || argumentsErr != nil || callee.Reference != helper.ID || len(arguments.List) != 3 {
		return layout, fmt.Errorf("wasm.helper_call")
	}
	for index, argument := range arguments.List {
		expression, exists := graph.Entities[argument.Reference]
		if !exists || expression.Schema != identity(0x9032) {
			return layout, fmt.Errorf("wasm.helper_argument")
		}
		read, err := referenced(graph, expression, 0x9320, 0x9013)
		if err != nil {
			return layout, err
		}
		readParameter, err := field(read, 0x9130)
		selectedField, selectedErr := field(expression, 0x9321)
		if err != nil || selectedErr != nil || readParameter.Reference != parameter.ID || selectedField.Reference != requestFields[index].entity.ID {
			return layout, fmt.Errorf("wasm.expression_record_field")
		}
	}
	for index, fieldIndex := range []int{3, 4} {
		value, ok := graph.Entities[values.List[index+1].Reference]
		if !ok || value.Schema != identity(0x9032) {
			return layout, fmt.Errorf("wasm.response_variable_value")
		}
		selected, selectedErr := field(value, 0x9321)
		if selectedErr != nil || selected.Reference != requestFields[fieldIndex].entity.ID {
			return layout, fmt.Errorf("wasm.response_variable_field")
		}
	}
	return layout, nil
}

func validateDecisionHelper(graph wire.Envelope, function wire.Entity) ([]byte, error) {
	parameters, err := field(function, 0x9111)
	if err != nil || len(parameters.List) != 3 {
		return nil, fmt.Errorf("wasm.helper_parameters")
	}
	parameterLocals := make(map[wire.ID]byte, 3)
	for index, reference := range parameters.List {
		parameter, ok := graph.Entities[reference.Reference]
		if !ok || parameter.Schema != identity(0x9012) {
			return nil, fmt.Errorf("wasm.helper_parameter")
		}
		position, positionErr := field(parameter, 0x9122)
		if positionErr != nil || position.Unsigned != uint64(index) || validateIntegerTypeReference(graph, parameter, 0x9121) != nil {
			return nil, fmt.Errorf("wasm.helper_parameter_type")
		}
		parameterLocals[parameter.ID] = byte(index)
	}
	if _, err := referenced(graph, function, 0x9112, 0x9020); err != nil {
		return nil, fmt.Errorf("wasm.helper_result")
	}
	comparison, err := referenced(graph, function, 0x9113, 0x9021)
	if err != nil {
		return nil, err
	}
	if validateIntegerTypeReference(graph, comparison, 0x9162) != nil {
		return nil, fmt.Errorf("wasm.helper_integer_type")
	}
	left, leftErr := field(comparison, 0x9160)
	right, rightErr := field(comparison, 0x9161)
	if leftErr != nil || rightErr != nil {
		return nil, fmt.Errorf("wasm.helper_comparison")
	}
	used := make(map[byte]bool, 3)
	visiting := make(map[wire.ID]bool)
	budget := 256
	leftInstructions, err := lowerHelperInteger(graph, left.Reference, parameterLocals, used, visiting, &budget)
	if err != nil {
		return nil, err
	}
	rightInstructions, err := lowerHelperInteger(graph, right.Reference, parameterLocals, used, visiting, &budget)
	if err != nil {
		return nil, err
	}
	if len(used) != len(parameterLocals) {
		return nil, fmt.Errorf("wasm.helper_parameter_usage")
	}
	instructions := append(leftInstructions, rightInstructions...)
	return append(instructions, 0x57), nil // i64.le_s
}

func lowerHelperInteger(graph wire.Envelope, id wire.ID, parameterLocals map[wire.ID]byte, used map[byte]bool, visiting map[wire.ID]bool, budget *int) ([]byte, error) {
	if *budget == 0 {
		return nil, fmt.Errorf("wasm.helper_expression_too_large")
	}
	*budget--
	if visiting[id] {
		return nil, fmt.Errorf("wasm.helper_expression_cycle")
	}
	visiting[id] = true
	defer delete(visiting, id)
	expression, ok := graph.Entities[id]
	if !ok {
		return nil, fmt.Errorf("wasm.helper_expression_missing")
	}
	switch expression.Schema {
	case identity(0x9013):
		parameter, err := field(expression, 0x9130)
		local, ok := parameterLocals[parameter.Reference]
		if err != nil || !ok {
			return nil, fmt.Errorf("wasm.helper_parameter_reference")
		}
		used[local] = true
		return []byte{0x20, local}, nil
	case identity(0x9070):
		bits, bitsErr := field(expression, 0x9700)
		if bitsErr != nil || validateIntegerTypeReference(graph, expression, 0x9701) != nil {
			return nil, fmt.Errorf("wasm.helper_integer_literal")
		}
		var instructions bytes.Buffer
		instructions.WriteByte(0x42) // i64.const
		sleb(&instructions, int64(bits.Unsigned))
		return instructions.Bytes(), nil
	case identity(0x9014):
		if validateIntegerTypeReference(graph, expression, 0x9142) != nil {
			return nil, fmt.Errorf("wasm.helper_integer_type")
		}
		left, leftErr := field(expression, 0x9140)
		right, rightErr := field(expression, 0x9141)
		if leftErr != nil || rightErr != nil {
			return nil, fmt.Errorf("wasm.helper_addition")
		}
		leftInstructions, err := lowerHelperInteger(graph, left.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		rightInstructions, err := lowerHelperInteger(graph, right.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		instructions := append(leftInstructions, rightInstructions...)
		return append(instructions, 0x7c), nil // i64.add
	case identity(0x9090):
		if validateIntegerTypeReference(graph, expression, 0x9902) != nil {
			return nil, fmt.Errorf("wasm.helper_integer_type")
		}
		left, leftErr := field(expression, 0x9900)
		right, rightErr := field(expression, 0x9901)
		if leftErr != nil || rightErr != nil {
			return nil, fmt.Errorf("wasm.helper_multiplication")
		}
		leftInstructions, err := lowerHelperInteger(graph, left.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		rightInstructions, err := lowerHelperInteger(graph, right.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		instructions := append(leftInstructions, rightInstructions...)
		return append(instructions, 0x7e), nil // i64.mul
	case identity(0x90a0):
		if validateIntegerTypeReference(graph, expression, 0x9a02) != nil {
			return nil, fmt.Errorf("wasm.helper_integer_type")
		}
		left, leftErr := field(expression, 0x9a00)
		right, rightErr := field(expression, 0x9a01)
		if leftErr != nil || rightErr != nil {
			return nil, fmt.Errorf("wasm.helper_subtraction")
		}
		leftInstructions, err := lowerHelperInteger(graph, left.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		rightInstructions, err := lowerHelperInteger(graph, right.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		instructions := append(leftInstructions, rightInstructions...)
		return append(instructions, 0x7d), nil // i64.sub
	case identity(0x90f4):
		collection, collectionErr := field(expression, 0x9f40)
		index, indexErr := field(expression, 0x9f41)
		if collectionErr != nil || indexErr != nil || collection.Tag != 6 || index.Tag != 6 {
			return nil, fmt.Errorf("wasm.helper_index_read")
		}
		collectionEntity, exists := graph.Entities[collection.Reference]
		if !exists {
			return nil, fmt.Errorf("wasm.helper_index_collection")
		}
		if collectionEntity.Schema == identity(0x9013) {
			parameter, err := field(collectionEntity, 0x9130)
			local, ok := parameterLocals[parameter.Reference]
			if err != nil || !ok {
				return nil, fmt.Errorf("wasm.helper_index_parameter")
			}
			parameterEntity, ok := graph.Entities[parameter.Reference]
			if !ok {
				return nil, fmt.Errorf("wasm.helper_index_parameter")
			}
			typeValue, err := field(parameterEntity, 0x9121)
			if err != nil || typeValue.Tag != 6 {
				return nil, fmt.Errorf("wasm.helper_index_parameter_type")
			}
			arrayType, err := pureType(graph, typeValue.Reference)
			if err != nil || !strings.HasPrefix(arrayType.name, "fixed-array:i64:") {
				return nil, fmt.Errorf("wasm.helper_index_parameter_type")
			}
			length := int64(arrayType.size / 8)
			indexCode, err := lowerHelperInteger(graph, index.Reference, parameterLocals, used, visiting, budget)
			if err != nil {
				return nil, err
			}
			instructions := append([]byte{}, indexCode...)
			instructions = append(instructions, 0x42)
			var limit bytes.Buffer
			sleb(&limit, length)
			instructions = append(instructions, limit.Bytes()...)
			instructions = append(instructions, 0x5a, 0x04, 0x40, 0x00, 0x0b, 0x20, local)
			instructions = append(instructions, indexCode...)
			instructions = append(instructions, 0xa7, 0x41, 0x08, 0x6c, 0x6a, 0x29, 0x03, 0x00)
			used[local] = true
			return instructions, nil
		}
		values, err := fixedI64ArrayValues(graph, collection.Reference)
		if err != nil {
			return nil, err
		}
		var instructions []byte
		for position, value := range values {
			indexCode, err := lowerHelperInteger(graph, index.Reference, parameterLocals, used, visiting, budget)
			if err != nil {
				return nil, err
			}
			valueCode, err := lowerHelperInteger(graph, value, parameterLocals, used, visiting, budget)
			if err != nil {
				return nil, err
			}
			instructions = append(instructions, indexCode...)
			instructions = append(instructions, 0x42)
			var constant bytes.Buffer
			sleb(&constant, int64(position))
			instructions = append(instructions, constant.Bytes()...)
			instructions = append(instructions, 0x51, 0x04, 0x7e)
			instructions = append(instructions, valueCode...)
			instructions = append(instructions, 0x05)
		}
		instructions = append(instructions, 0x00)
		for range values {
			instructions = append(instructions, 0x0b)
		}
		return instructions, nil
	case identity(0x90f7):
		collection, initial, accumulator, element, body, err := foldFields(graph, expression)
		if err != nil {
			return nil, err
		}
		if err := validateI64AddFoldBody(graph, body, accumulator, element); err != nil {
			return nil, err
		}
		instructions, err := lowerHelperInteger(graph, initial, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		collectionEntity, ok := graph.Entities[collection]
		if !ok {
			return nil, fmt.Errorf("wasm.fold_collection")
		}
		var values []wire.ID
		var parameter wire.ID
		if collectionEntity.Schema == identity(0x90f3) {
			values, err = fixedI64ArrayValues(graph, collection)
			if err != nil {
				return nil, err
			}
		} else if collectionEntity.Schema == identity(0x9013) {
			parameterValue, fieldErr := field(collectionEntity, 0x9130)
			if fieldErr != nil || parameterValue.Tag != 6 {
				return nil, fmt.Errorf("wasm.fold_parameter")
			}
			parameter = parameterValue.Reference
		} else {
			return nil, fmt.Errorf("wasm.fold_collection")
		}
		if parameter != (wire.ID{}) {
			local, ok := parameterLocals[parameter]
			if !ok {
				return nil, fmt.Errorf("wasm.fold_parameter")
			}
			parameterEntity := graph.Entities[parameter]
			typeValue, _ := field(parameterEntity, 0x9121)
			arrayType, err := pureType(graph, typeValue.Reference)
			if err != nil {
				return nil, err
			}
			for position := uint64(0); position < arrayType.size/8; position++ {
				instructions = append(instructions, 0x20, local, 0x29, 0x03)
				var offset bytes.Buffer
				uleb(&offset, position*8)
				instructions = append(instructions, offset.Bytes()...)
				instructions = append(instructions, 0x7c)
			}
			used[local] = true
			return instructions, nil
		}
		for _, value := range values {
			valueCode, err := lowerHelperInteger(graph, value, parameterLocals, used, visiting, budget)
			if err != nil {
				return nil, err
			}
			instructions = append(instructions, valueCode...)
			instructions = append(instructions, 0x7c)
		}
		return instructions, nil
	case identity(0x90f9):
		collection, err := field(expression, 0x9f90)
		if err != nil || collection.Tag != 6 {
			return nil, fmt.Errorf("wasm.helper_collection_length")
		}
		collectionEntity, ok := graph.Entities[collection.Reference]
		if !ok {
			return nil, fmt.Errorf("wasm.helper_collection_length")
		}
		if collectionEntity.Schema == identity(0x90f3) {
			values, err := fixedI64ArrayValues(graph, collection.Reference)
			if err != nil {
				return nil, err
			}
			var constant bytes.Buffer
			constant.WriteByte(0x42)
			sleb(&constant, int64(len(values)))
			return constant.Bytes(), nil
		}
		parameter, err := field(collectionEntity, 0x9130)
		local, exists := parameterLocals[parameter.Reference]
		if err != nil || collectionEntity.Schema != identity(0x9013) || !exists {
			return nil, fmt.Errorf("wasm.helper_collection_length")
		}
		parameterEntity := graph.Entities[parameter.Reference]
		typeValue, err := field(parameterEntity, 0x9121)
		valueType, typeErr := pureType(graph, typeValue.Reference)
		if err != nil || typeErr != nil {
			return nil, fmt.Errorf("wasm.helper_collection_length_type")
		}
		used[local] = true
		if valueType.name == "slice:i64" {
			return []byte{0x20, local, 0x42, 0x20, 0x88}, nil
		}
		if strings.HasPrefix(valueType.name, "fixed-array:i64:") {
			var constant bytes.Buffer
			constant.WriteByte(0x42)
			sleb(&constant, int64(valueType.size/8))
			return constant.Bytes(), nil
		}
		return nil, fmt.Errorf("wasm.helper_collection_length_type")
	case identity(0x90fa):
		collection, collectionErr := field(expression, 0x9fa0)
		index, indexErr := field(expression, 0x9fa1)
		if collectionErr != nil || indexErr != nil || collection.Tag != 6 || index.Tag != 6 {
			return nil, fmt.Errorf("wasm.helper_dynamic_index")
		}
		collectionEntity, ok := graph.Entities[collection.Reference]
		if !ok || collectionEntity.Schema != identity(0x9013) {
			return nil, fmt.Errorf("wasm.helper_dynamic_index_collection")
		}
		parameter, err := field(collectionEntity, 0x9130)
		local, exists := parameterLocals[parameter.Reference]
		if err != nil || !exists {
			return nil, fmt.Errorf("wasm.helper_dynamic_index_parameter")
		}
		parameterEntity := graph.Entities[parameter.Reference]
		typeValue, err := field(parameterEntity, 0x9121)
		valueType, typeErr := pureType(graph, typeValue.Reference)
		if err != nil || typeErr != nil || valueType.name != "slice:i64" {
			return nil, fmt.Errorf("wasm.helper_dynamic_index_type")
		}
		indexCode, err := lowerHelperInteger(graph, index.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		instructions := append([]byte{}, indexCode...)
		instructions = append(instructions, 0x20, local, 0x42, 0x20, 0x88, 0x5a, 0x04, 0x40, 0x00, 0x0b)
		instructions = append(instructions, 0x20, local, 0xa7)
		instructions = append(instructions, indexCode...)
		instructions = append(instructions, 0xa7, 0x41, 0x03, 0x74, 0x6a, 0x29, 0x03, 0x00)
		used[local] = true
		return instructions, nil
	default:
		return nil, fmt.Errorf("wasm.helper_integer_expression")
	}
}

func lowerHelperBoolean(graph wire.Envelope, id wire.ID, parameterLocals map[wire.ID]byte, used map[byte]bool, visiting map[wire.ID]bool, budget *int) ([]byte, error) {
	if *budget == 0 {
		return nil, fmt.Errorf("wasm.helper_expression_too_large")
	}
	*budget--
	if visiting[id] {
		return nil, fmt.Errorf("wasm.helper_expression_cycle")
	}
	visiting[id] = true
	defer delete(visiting, id)
	expression, ok := graph.Entities[id]
	if !ok {
		return nil, fmt.Errorf("wasm.helper_expression_missing")
	}
	switch expression.Schema {
	case identity(0x9013):
		parameter, err := field(expression, 0x9130)
		local, ok := parameterLocals[parameter.Reference]
		if err != nil || !ok {
			return nil, fmt.Errorf("wasm.helper_parameter_reference")
		}
		used[local] = true
		return []byte{0x20, local}, nil
	case identity(0x90b0):
		literal, err := field(expression, 0x9b00)
		if err != nil || (literal.Tag != 1 && literal.Tag != 2) {
			return nil, fmt.Errorf("wasm.helper_boolean_literal")
		}
		value := byte(0)
		if literal.Tag == 2 {
			value = 1
		}
		return []byte{0x41, value}, nil // i32.const
	case identity(0x90b1):
		left, leftErr := field(expression, 0x9b10)
		right, rightErr := field(expression, 0x9b11)
		if leftErr != nil || rightErr != nil {
			return nil, fmt.Errorf("wasm.helper_boolean_and")
		}
		leftInstructions, err := lowerHelperBoolean(graph, left.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		rightInstructions, err := lowerHelperBoolean(graph, right.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		// if (result i32) evaluates the right operand only when left is true.
		instructions := append(leftInstructions, 0x04, 0x7f)
		instructions = append(instructions, rightInstructions...)
		return append(instructions, 0x05, 0x41, 0x00, 0x0b), nil
	case identity(0x90c1):
		left, leftErr := field(expression, 0x9c10)
		right, rightErr := field(expression, 0x9c11)
		if leftErr != nil || rightErr != nil {
			return nil, fmt.Errorf("wasm.helper_boolean_or")
		}
		leftInstructions, err := lowerHelperBoolean(graph, left.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		rightInstructions, err := lowerHelperBoolean(graph, right.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		// if (result i32) evaluates the right operand only when left is false.
		instructions := append(leftInstructions, 0x04, 0x7f, 0x41, 0x01, 0x05)
		instructions = append(instructions, rightInstructions...)
		return append(instructions, 0x0b), nil
	case identity(0x90c2):
		left, leftErr := field(expression, 0x9c20)
		right, rightErr := field(expression, 0x9c21)
		if leftErr != nil || rightErr != nil {
			return nil, fmt.Errorf("wasm.helper_string_equal")
		}
		leftText, err := constantUTF8String(graph, left.Reference, map[wire.ID]bool{}, budget)
		if err != nil {
			return nil, err
		}
		rightText, err := constantUTF8String(graph, right.Reference, map[wire.ID]bool{}, budget)
		if err != nil {
			return nil, err
		}
		value := byte(0)
		if bytes.Equal(leftText, rightText) {
			value = 1
		}
		return []byte{0x41, value}, nil
	case identity(0x9021):
		left, leftErr := field(expression, 0x9160)
		right, rightErr := field(expression, 0x9161)
		if leftErr != nil || rightErr != nil || validateIntegerTypeReference(graph, expression, 0x9162) != nil {
			return nil, fmt.Errorf("wasm.helper_integer_comparison")
		}
		leftInstructions, err := lowerHelperInteger(graph, left.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		rightInstructions, err := lowerHelperInteger(graph, right.Reference, parameterLocals, used, visiting, budget)
		if err != nil {
			return nil, err
		}
		instructions := append(leftInstructions, rightInstructions...)
		return append(instructions, 0x57), nil // i64.le_s
	default:
		return nil, fmt.Errorf("wasm.helper_boolean_expression")
	}
}

func constantUTF8String(graph wire.Envelope, id wire.ID, visiting map[wire.ID]bool, budget *int) ([]byte, error) {
	if *budget == 0 || visiting[id] {
		return nil, fmt.Errorf("wasm.helper_string_cycle_or_size")
	}
	*budget--
	visiting[id] = true
	defer delete(visiting, id)
	expression, ok := graph.Entities[id]
	if !ok {
		return nil, fmt.Errorf("wasm.helper_string_missing")
	}
	switch expression.Schema {
	case identity(0x9050):
		literal, err := field(expression, 0x9500)
		if err != nil || literal.Tag != 5 || !utf8.Valid(literal.Bytes) || len(literal.Bytes) > 4096 {
			return nil, fmt.Errorf("wasm.helper_string_literal")
		}
		return append([]byte(nil), literal.Bytes...), nil
	case identity(0x90c3):
		left, leftErr := field(expression, 0x9c30)
		right, rightErr := field(expression, 0x9c31)
		if leftErr != nil || rightErr != nil {
			return nil, fmt.Errorf("wasm.helper_string_concat")
		}
		leftText, err := constantUTF8String(graph, left.Reference, visiting, budget)
		if err != nil {
			return nil, err
		}
		rightText, err := constantUTF8String(graph, right.Reference, visiting, budget)
		if err != nil {
			return nil, err
		}
		if len(leftText) > 4096-len(rightText) {
			return nil, fmt.Errorf("wasm.helper_string_too_large")
		}
		return append(leftText, rightText...), nil
	default:
		return nil, fmt.Errorf("wasm.helper_string_expression")
	}
}

type recordFieldProfile struct {
	name   string
	schema uint64
	size   uint64
}
type recordFieldLayout struct {
	entity wire.Entity
	offset uint64
	size   uint64
}

func validateRecord(graph wire.Envelope, record wire.Entity, name string, profile []recordFieldProfile) ([]recordFieldLayout, error) {
	nameValue, err := field(record, 0x9300)
	fieldsValue, fieldsErr := field(record, 0x9301)
	if err != nil || fieldsErr != nil || string(nameValue.Bytes) != name || len(fieldsValue.List) != len(profile) {
		return nil, fmt.Errorf("wasm.record_profile:%s", name)
	}
	layout := make([]recordFieldLayout, len(profile))
	var offset uint64
	for index, expected := range profile {
		entity, ok := graph.Entities[fieldsValue.List[index].Reference]
		if !ok || entity.Schema != identity(0x9031) {
			return nil, fmt.Errorf("wasm.record_field_missing:%s", expected.name)
		}
		fieldName, nameErr := field(entity, 0x9310)
		fieldType, typeErr := field(entity, 0x9311)
		fieldIndex, indexErr := field(entity, 0x9312)
		typeEntity, typeOK := graph.Entities[fieldType.Reference]
		if nameErr != nil || typeErr != nil || indexErr != nil || string(fieldName.Bytes) != expected.name || fieldIndex.Unsigned != uint64(index) || !typeOK || typeEntity.Schema != identity(expected.schema) {
			return nil, fmt.Errorf("wasm.record_field_profile:%s", expected.name)
		}
		if expected.schema == 0x9010 {
			if err := validateIntegerTypeReference(graph, entity, 0x9311); err != nil {
				return nil, err
			}
		}
		layout[index] = recordFieldLayout{entity, offset, expected.size}
		offset += expected.size
	}
	return layout, nil
}

func validateIntegerTypeReference(graph wire.Envelope, entity wire.Entity, fieldID uint64) error {
	typeValue, err := field(entity, fieldID)
	if err != nil {
		return err
	}
	typeEntity, ok := graph.Entities[typeValue.Reference]
	if !ok || typeEntity.Schema != identity(0x9010) {
		return fmt.Errorf("wasm.integer_type_missing")
	}
	width, widthErr := field(typeEntity, 0x9100)
	signed, signedErr := field(typeEntity, 0x9101)
	overflow, overflowErr := field(typeEntity, 0x9102)
	if widthErr != nil || signedErr != nil || overflowErr != nil || width.Unsigned != 64 || signed.Tag != 2 || overflow.Unsigned != 0 {
		return fmt.Errorf("wasm.unsupported_integer_profile")
	}
	return nil
}

func referenced(graph wire.Envelope, entity wire.Entity, fieldID, schema uint64) (wire.Entity, error) {
	value, err := field(entity, fieldID)
	if err != nil {
		return wire.Entity{}, err
	}
	target, ok := graph.Entities[value.Reference]
	if !ok || target.Schema != identity(schema) {
		return wire.Entity{}, fmt.Errorf("wasm.reference_schema:%x", fieldID)
	}
	return target, nil
}

func field(entity wire.Entity, low uint64) (wire.Value, error) {
	value, ok := entity.Fields[identity(low)]
	if !ok {
		return wire.Value{}, fmt.Errorf("wasm.field_missing:%x", low)
	}
	return value, nil
}
func bySchema(graph wire.Envelope, low uint64) []wire.Entity {
	want := identity(low)
	var values []wire.Entity
	for _, entity := range graph.Entities {
		if entity.Schema == want {
			values = append(values, entity)
		}
	}
	return values
}
func identity(low uint64) wire.ID {
	var value wire.ID
	for index := 15; index >= 8; index-- {
		value[index] = byte(low)
		low >>= 8
	}
	return value
}
